package tee_server_receiver

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	teeRequester "sparta/src/teeserver/requester"
	"sparta/src/utils/interfaceISGoMiddleware"
	"sparta/src/utils/sealing"
	seedGeneration "sparta/src/utils/seedGenerator"

	"github.com/edgelesssys/ego/enclave"
	"github.com/go-redis/redis/v8"
)
	
const (
	requestQueue  = "request_queue"
	responseQueue = "response_queue"
)

// Global Redis client
var redisClient *redis.Client

func init() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Redis server address
	})
}

type USGonServerReceiver struct {
	server *http.Server
}

var (
	teeServerMeasurement string = ""
	seedExchangeEnabled  bool   = false
)


// StartUSGonServer now supports BOTH modes:
//
// - exchangeSeed=true  => BOOTSTRAP mode:
//     * uses CreateBootstrapCertificate() (no seed needed)
//     * exposes /secret endpoint (to receive the seed)
//     * DOES NOT start readQueue() because normal workload assumes the deterministic cert/seed exists
//
// - exchangeSeed=false => NORMAL mode:
//     * uses CreateCertificate() (seed required)
//     * /secret is disabled
//     * starts readQueue()

// StartTEE creates the enclave HTTPS server.
// - exchangeSeed=true  => BOOTSTRAP cert + /secret enabled
// - exchangeSeed=false => NORMAL deterministic cert + /secret disabled
func StartTEE(exchangeSeed bool) *USGonServerReceiver {
	seedExchangeEnabled = exchangeSeed

	var cert []byte
	var priv interface{}

	if seedExchangeEnabled {
		cert, priv = interfaceISGoMiddleware.CreateBootstrapCertificate()
		fmt.Println("Using BOOTSTRAP certificate (seed not required).")
	} else {
		cert, priv = interfaceISGoMiddleware.CreateCertificate()
		fmt.Println("Using NORMAL deterministic certificate (seed required).")
	}

	if cert == nil || priv == nil {
		fmt.Println("Error: certificate creation failed")
		os.Exit(1)
	}

	hash := sha256.Sum256(cert)
	s := &USGonServerReceiver{}

	report, err := enclave.GetRemoteReport(hash[:])
	if err != nil {
		fmt.Println(err)
	}

	// Read CA certificate for attestation
	caCert, err := os.ReadFile("certificate/user_cert.pem")
	if err != nil {
		fmt.Println("Error reading CA certificate:", err)
		os.Exit(1)
	}

	// HTTP handlers
	handler := http.NewServeMux()

	// Remote attestation endpoints
	handler.HandleFunc("/caCert", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Sending CA certificate")
		_, _ = w.Write(caCert)
	})
	handler.HandleFunc("/cert", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Sending RA certificate")
		_, _ = w.Write(cert)
	})
	handler.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Sending report")
		_, _ = w.Write(report)
	})

	// Seed exchange endpoint (gated inside handleKey)
	handler.HandleFunc("/secret", s.handleKey)

	tlsCfg := tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{cert},
				PrivateKey:  priv,
			},
		},
	}

	// Start processing requests from middleware ONLY in normal mode
	if !seedExchangeEnabled {
		go readQueue()
	} else {
		fmt.Println("BOOTSTRAP mode: readQueue() NOT started (waiting for /secret).")
	}

	s.server = &http.Server{
		Addr:      "0.0.0.0:8075",
		TLSConfig: &tlsCfg,
		Handler:   handler,
	}
	return s
}

// Start sets the expected peer measurement (used by VerifyTEE) and starts HTTPS server.
func (s *USGonServerReceiver) Start(measurement string) error {
	fmt.Println("TEE Server Receiver started")
	teeServerMeasurement = measurement
	return s.server.ListenAndServeTLS("", "")
}

// Graceful stop (used after seed exchange, if you want to exit immediately).
func (s *USGonServerReceiver) Stop() {
	if s == nil || s.server == nil {
		return
	}
	fmt.Println("Stopping TEE server receiver...")
	_ = s.server.Close()
}

// -------------------------
// Seed exchange receiver
// -------------------------
func (s *USGonServerReceiver) handleKey(w http.ResponseWriter, r *http.Request) {
	// Gate seed exchange endpoint
	if !seedExchangeEnabled {
		http.Error(w, "Seed exchange disabled (run with -exchange_seed)", http.StatusForbidden)
		return
	}

	// Verify the peer TEE
	valid := teeRequester.VerifyTEE(teeServerMeasurement, false)
	if !valid {
		http.Error(w, "TEE verification failed", http.StatusUnauthorized)
		return
	}

	// Validate Content-Type
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		fmt.Println("Unsupported Content-Type:", contentType)
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	mr, err := r.MultipartReader()
	if err != nil {
		fmt.Println("Error creating multipart reader:", err)
		http.Error(w, "Failed to read multipart data", http.StatusBadRequest)
		return
	}

	var seedBytes []byte
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading part:", err)
			http.Error(w, "Failed to read multipart data", http.StatusInternalServerError)
			return
		}

		switch part.FormName() {
		case "seed":
			seed, err := io.ReadAll(part)
			if err != nil || len(seed) == 0 {
				fmt.Println("Error reading seed field:", err)
				http.Error(w, "Invalid seed field", http.StatusBadRequest)
				return
			}
			seedBytes = seed
		default:
			fmt.Println("Unknown form field:", part.FormName())
		}
	}

	if len(seedBytes) == 0 {
		fmt.Println("Seed not provided")
		http.Error(w, "Seed not provided", http.StatusBadRequest)
		return
	}

	// Seal and store
	sealedSeed, err := sealing.NewSealing(seedBytes, []byte(""), true)
	if err != nil {
		fmt.Println("Error sealing seed:", err)
		http.Error(w, "Failed to seal seed", http.StatusInternalServerError)
		return
	}

	if err := seedGeneration.WriteSealedSeed(sealedSeed); err != nil {
		fmt.Println("Error storing sealed seed:", err)
		http.Error(w, "Failed to store sealed seed", http.StatusInternalServerError)
		return
	}

	// Respond first
	_, _ = w.Write([]byte("Seed stored successfully"))

	// Flush if possible (helps avoid cutting the connection)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Then stop the HTTPS server so Start() returns and main unblocks on <-done
	go s.Stop()
}

// -------------------------
// Redis request processing
// -------------------------
type EncryptedPayload struct {
	ClientID string `json:"client_id"`
	Data     string `json:"data"`
}

type ClientPayload struct {
	FunctionName    string `json:"function_name"`
	Certificate     string `json:"certificate"`
	EncryptedData   string `json:"encrypted_data"`
	FileExtension   string `json:"file_extension"`
	IPNSKey         string `json:"ipns_key"`
	ClientPublicKey string `json:"client_public_key"`
	Signature       string `json:"signature"`
}

type QueuePayload struct {
	ClientID string        `json:"client_id"`
	Data     ClientPayload `json:"data"`
}

func readQueue() {
	for {
		res, err := redisClient.BRPop(redisClient.Context(), 0, requestQueue).Result()
		if err != nil {
			fmt.Printf("[Server] Error dequeuing request: %!v(MISSING)\n", err)
			continue
		}

		var payload QueuePayload
		if err := json.Unmarshal([]byte(res[1]), &payload); err != nil {
			fmt.Printf("[Server] Error parsing payload: %!v(MISSING)\n", err)
			continue
		}

		response := handleFunction(payload.Data.FunctionName, map[string]string{
			"function_name":     payload.Data.FunctionName,
			"certificate":       payload.Data.Certificate,
			"encrypted_data":    payload.Data.EncryptedData,
			"file_extension":    payload.Data.FileExtension,
			"ipns_key":          payload.Data.IPNSKey,
			"client_public_key": payload.Data.ClientPublicKey,
			"signature":         payload.Data.Signature,
		})

		responsePayload, _ := json.Marshal(map[string]string{
			"client_id": payload.ClientID,
			"response":  response,
		})
		if err := redisClient.LPush(redisClient.Context(), responseQueue, responsePayload).Err(); err != nil {
			fmt.Printf("[Server] Error enqueuing response: %!v(MISSING)\n", err)
		} else {
			fmt.Printf("[Server] Enqueued response for client: %!s(MISSING)\n", payload.ClientID)
			fmt.Println("----------------------------------------------------------------------------------------------------------------------------")
		}
	}
}

func handleFunction(functionName string, payload map[string]string) string {
	switch functionName {
	case "PriorityRE":
		return decidePatientPrioritizationWithAggrHandler(payload)
	case "setWritePatientData":
		return setWritePatientDataHandler(payload)
	default:
		return "Unknown function: " + functionName
	}
}

func decidePatientPrioritizationWithAggrHandler(payload map[string]string) string {
	certificate, functionName, ipnsKey, _ := interfaceISGoMiddleware.ParseDecisionRequestFromQueue(payload)
	certificateValidity, attributes := interfaceISGoMiddleware.CheckCertificate(certificate)
	if certificateValidity {
		callable := interfaceISGoMiddleware.CheckCallability(`{accessPolicy: (Role="MedicalHub" and Country="Italy")}`, attributes)
		if callable {
			interfaceISGoMiddleware.Decision(functionName, "PatientLight", ipnsKey+"Light")
			return "Decision performed successfully"
		}
		return "Access policy not satisfied"
	}
	return "Invalid certificate"
}
				
func setWritePatientDataHandler(payload map[string]string) string {
	certificate, _, fileBytes, _, ipnsKey, _ := interfaceISGoMiddleware.ParseSetRequestFromQueueBytes(payload)
	certificateValidity, attributes := interfaceISGoMiddleware.CheckCertificate(certificate)
	if certificateValidity {
		callable := interfaceISGoMiddleware.CheckCallability(`{accessPolicy: (Country="Italy" and (Role="Professor" or (Role="Student" and Company="Sapienza")))}`, attributes)
		if callable {
			interfaceISGoMiddleware.EncryptAndUploadLinkedBytes(fileBytes, "Patient", ipnsKey)
			interfaceISGoMiddleware.EncryptAndUploadLinkedBytes(fileBytes, "PatientLight", ipnsKey+"Light")
			return "Encryption of document performed successfully"
		}
		return "Access policy not satisfied"
	}
	return "Invalid certificate"
}
				