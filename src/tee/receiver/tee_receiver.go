package tee_receiver

import (
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	teeRequester "sparta/src/tee/requester"
	"sparta/src/utils/interfaceISGoMiddleware"
	"sparta/src/utils/sealing"
	seedGeneration "sparta/src/utils/seedGenerator"

	"github.com/edgelesssys/ego/enclave"
)

type TEEReceiver struct {
	server *http.Server
}

var (
	teeServerMeasurement string = ""
	seedExchangeEnabled  bool   = false
)

func StartTEE(exchangeSeed bool) *TEEReceiver {
	seedExchangeEnabled = exchangeSeed

	var cert []byte
	var priv interface{}

	// Choose certificate creation mode
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
	s := &TEEReceiver{}

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

	handler := http.NewServeMux()
	handler.HandleFunc("/caCert", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Sending CA certificate")
		w.Write(caCert)
	})
	handler.HandleFunc("/cert", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Sending RA certificate")
		w.Write(cert)
	})
	handler.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Sending report")
		w.Write(report)
	})

	// Only allow /secret in bootstrap mode
	handler.HandleFunc("/secret", s.handleKey)

	tlsCfg := tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{cert},
				PrivateKey:  priv,
			},
		},
	}

	s.server = &http.Server{
		Addr:      "0.0.0.0:8078",
		TLSConfig: &tlsCfg,
		Handler:   handler,
	}
	return s
}

func (s *TEEReceiver) Start(measurement string) error {
	fmt.Println("TEE Server Receiver started")
	teeServerMeasurement = measurement
	return s.server.ListenAndServeTLS("", "")
}

// Graceful stop (used after seed exchange, if you want to exit immediately).
func (s *TEEReceiver) Stop() {
	if s == nil || s.server == nil {
		return
	}
	fmt.Println("Stopping TEE server receiver...")
	_ = s.server.Close()
}

func (s *TEEReceiver) handleKey(w http.ResponseWriter, r *http.Request) {
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