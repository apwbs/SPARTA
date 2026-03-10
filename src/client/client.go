package main

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sparta/src/utils/clientEncryption"
	"sparta/src/utils/simple_attestation"
	"time"
)

var TESTMODE bool

func main() {
	// Define flags for function selection and other parameters
	signerArg := flag.String("signer_id", "", "signer ID")
	serverAddr := flag.String("server_address", "localhost:8075", "server address")
	inputFiles := flag.String("input_file", "", "path to JSON input file")
	processInstanceID := flag.String("process_id", "", "process instance ID")
	messageID := flag.String("message_id", "", "message ID")
	functionName := flag.String("function", "", "function to call")
	measurement := flag.String("measurement", "", "expected measurement of the server tee")
	certificate := flag.String("certificate", "", "path to certificate file")
	ipnsKey := flag.String("ipnsKey", "", "path to certificate file")

	TESTMODESTR := flag.String("memory_test", "", "Test mode")

	setFunction := flag.Bool("set_function", false, "set a function name to call")
	getFunction := flag.Bool("get_function", false, "get the function name")
	decisionFunction := flag.Bool("decision_function", false, "get the decision function name")

	flag.Parse()

	TESTMODE, _ = strconv.ParseBool(*TESTMODESTR)

	// Check mandatory parameters
	if *signerArg == "" {
		fmt.Println("Error: Signer ID is required.")
		os.Exit(1)
	}
	if *serverAddr == "" {
		fmt.Println("Error: Server Address is required.")
		os.Exit(1)
	}
	if *functionName == "" {
		fmt.Println("Error: Function Name is required.")
		os.Exit(1)
	}
	if *measurement == "" {
		fmt.Println("Error: Measurement is required.")
		os.Exit(1)
	}
	if *certificate == "" {
		fmt.Println("Error: Certificate file is required.")
		os.Exit(1)
	}

	// Handle the specified function
	switch {
	case *setFunction:
		setFunctionHandler(*signerArg, *serverAddr, *measurement, *certificate, *processInstanceID, *functionName, *inputFiles, *ipnsKey)
	case *getFunction:
		getFunctionHandler(*signerArg, *serverAddr, *measurement, *certificate, *processInstanceID, *functionName, *messageID)
	case *decisionFunction:
		decisionFunctionHandler(*signerArg, *serverAddr, *measurement, *certificate, *processInstanceID, *functionName, *ipnsKey)
	default:
		fmt.Println("No function specified.")
	}
}

func setFunctionHandler(signerArg, middlewareAddr, expectedMeasurement, certificateFile, processInstanceID, functionName, inputFile, ipnsKey string) {
	var teePublicKey *ecdh.PublicKey

	_, _, teePublicKey = simple_attestation.RemoteAttestation("https://localhost:8075", []byte(expectedMeasurement))

	// Generate the client's ECDH key pair
	curve := ecdh.X25519()
	clientPrivateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Println("Error generating ECDH private key:", err)
		return
	}
	clientPublicKey := clientPrivateKey.PublicKey()

	// Compute the shared secret using the TEE's public key
	sharedSecret, err := clientPrivateKey.ECDH(teePublicKey)
	if err != nil {
		fmt.Println("Error computing shared secret:", err)
		return
	}
	fmt.Printf("Shared Secret: %x\n", sharedSecret)

	// Derive a symmetric key from the shared secret using SHA-256
	symmetricKey := sha256.Sum256(sharedSecret)

	// Load the private key from the "privkey" folder
	privateKeyPath := "../client/privkey/privateKey.pem" // When runned from the sh file
	privateKey, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		fmt.Println("Error loading private key:", err)
		return
	}

	// Sign the clientPublicKey with the loaded private key
	clientPublicKeyBytes := clientPublicKey.Bytes()
	hash := sha256.Sum256(clientPublicKeyBytes) // Hash the clientPublicKey for signing
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		fmt.Println("Error signing the client public key:", err)
		return
	}

	// Encode the signature (r, s) into a format that can be sent
	signature := struct {
		R, S *big.Int
	}{
		R: r,
		S: s,
	}
	signatureBytes, err := asn1.Marshal(signature)
	if err != nil {
		fmt.Println("Error marshaling the signature:", err)
		return
	}

	// Read the input file
	inputData, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Println("Error reading input file:", err)
		return
	}

	// Encrypt the input file using AES-GCM
	encryptedData, err := clientEncryption.EncryptWithAESGCM(symmetricKey[:], inputData)
	if err != nil {
		fmt.Println("Error encrypting input file:", err)
		return
	}

	// Read the certificate file content (CA-issued certificate)
	certContent, err := os.ReadFile(certificateFile)
	if err != nil {
		fmt.Println("Error reading certificate file:", err)
		return
	}

	// Prepare the payload
	payload := map[string]string{
		"ipns_key":          ipnsKey,
		"function_name":     functionName,
		"client_public_key": base64.StdEncoding.EncodeToString(clientPublicKeyBytes), // DH Public key
		"signature":         base64.StdEncoding.EncodeToString(signatureBytes),       // Signature of clientPublicKey
		"file_extension":    filepath.Ext(inputFile),                                 // File extension
		"certificate":       string(certContent),                                     // CA-issued certificate
		"encrypted_data":    base64.StdEncoding.EncodeToString(encryptedData),        // Encrypted input
	}

	// Serialize the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling payload:", err)
		return
	}

	// Connect to the middleware over TCP
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		fmt.Println("Error connecting to middleware:", err)
		return
	}
	defer conn.Close()

	// Prefix the payload with its length (8 bytes)
	lengthPrefix := make([]byte, 8)
	binary.BigEndian.PutUint64(lengthPrefix, uint64(len(payloadBytes)))

	// Send the length prefix
	_, err = conn.Write(lengthPrefix)
	if err != nil {
		fmt.Println("Error sending length prefix:", err)
		return
	}

	// Send the payload in chunks
	chunkSize := 8192 // 8 KB chunks
	for i := 0; i < len(payloadBytes); i += chunkSize {
		end := i + chunkSize
		if end > len(payloadBytes) {
			end = len(payloadBytes)
		}
		_, err := conn.Write(payloadBytes[i:end])
		if err != nil {
			fmt.Println("Error sending data chunk to middleware:", err)
			return
		}
	}
	fmt.Printf("Payload size sent: %d bytes\n", len(payloadBytes))

	// Receive acknowledgment from the middleware
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading acknowledgment from middleware:", err)
		return
	}
	fmt.Printf("Acknowledgment from middleware: %s\n", string(buffer[:n]))

	// Poll for response from middleware
	var responseBytes []byte
	tempBuffer := make([]byte, 4096) // 4 KB chunks

	for {
		n, err := conn.Read(tempBuffer)

		// Append the read bytes to the response buffer
		if n > 0 {
			responseBytes = append(responseBytes, tempBuffer[:n]...)
		}

		// Handle errors
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Middleware closed the connection. Assuming response is complete.")
				break // Middleware closed the connection after sending the response
			}
			fmt.Printf("Error reading response from middleware: %v\n", err)
			return
		}

		// If no data was read, continue waiting
		if n == 0 {
			continue
		}

		// If we detect the end of the response, break
		if n < len(tempBuffer) {
			break // Likely end of response
		}
	}

	// Convert response bytes to a string
	response := string(responseBytes)
	if response == "" {
		fmt.Println("Error: Empty response from middleware.")
	} else {
		fmt.Printf("Response from middleware: %s\n", response)
	}
}

func getFunctionHandler(signerArg, middlewareAddr, expectedMeasurement, certificateFile, processInstanceID, functionName, messageID string) {
	// Perform Remote Attestation
	_, _, _ = simple_attestation.RemoteAttestation("https://localhost:8075", []byte(expectedMeasurement))

	// Read the certificate file content (CA-issued certificate)
	certContent, err := os.ReadFile(certificateFile)
	if err != nil {
		fmt.Println("Error reading certificate file:", err)
		return
	}

	// Prepare the payload for the GET request
	payload := map[string]string{
		"function_name": functionName,
		"message_id":    messageID,
		"certificate":   string(certContent),
	}

	// Serialize the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling payload: %v\n", err)
		return
	}

	// Connect to the middleware via TCP
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		fmt.Printf("Error connecting to middleware: %v\n", err)
		return
	}
	defer conn.Close()

	// Send the GET payload to the middleware
	_, err = conn.Write(payloadBytes)
	if err != nil {
		fmt.Printf("Error sending data to middleware: %v\n", err)
		return
	}

	// Wait for acknowledgment from the middleware
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("Error reading acknowledgment from middleware: %v\n", err)
		return
	}
	fmt.Printf("Acknowledgment from middleware: %s\n", string(buffer[:n]))

	// Poll for a response from the middleware
	var responseBytes []byte
	tempBuffer := make([]byte, 4096) // 4 KB chunks
	for {
		n, err := conn.Read(tempBuffer)

		// Append read bytes to the response buffer
		if n > 0 {
			responseBytes = append(responseBytes, tempBuffer[:n]...)
		}

		// Handle errors and EOF
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Middleware closed the connection. Assuming response is complete.")
				break
			}
			fmt.Printf("Error reading response from middleware: %v\n", err)
			return
		}

		// Detect end of response
		if n == 0 || n < len(tempBuffer) {
			break
		}
	}

	// Ensure responseBytes does not include extra metadata
	response := string(responseBytes)
	if response == "" {
		fmt.Println("Error: Empty response from middleware.")
		return
	}

	// Parse response for file extension and handle accordingly
	fileExtension := ""
	var fileContent []byte

	parts := strings.SplitN(response, "---------", 2) // Split into content and metadata
	if len(parts) == 2 {
		fileExtension = parts[1]
		fileContent = responseBytes[:len(parts[0])]
	} else {
		// If no separator, treat the entire response as plain text or JSON
		fileContent = responseBytes
	}

	// Handle binary file saving
	if fileExtension != ".json" && fileExtension != "" {
		fileName := "response_file" + fileExtension

		// Create and write the response to a file
		file, err := os.Create(fileName)
		if err != nil {
			fmt.Printf("Error creating response file: %v\n", err)
			return
		}
		defer file.Close()

		_, err = file.Write(fileContent)
		if err != nil {
			fmt.Printf("Error saving response file: %v\n", err)
			return
		}

		fmt.Println("Response saved as:", fileName)
	} else {
		// Print the response if it’s a JSON or plain string
		fmt.Println("Response Body:", string(fileContent))
	}
}

func decisionFunctionHandler(signerArg, middlewareAddr, expectedMeasurement, certificateFile, processInstanceID, functionName, ipnsKey string) {
	if TESTMODE {
		println("TESTMODE - CLIENT STARTED AT: ", time.Now().UnixMilli())
	}
	_, _, _ = simple_attestation.RemoteAttestation("https://localhost:8075", []byte(expectedMeasurement))

	// Read the certificate file content (CA-issued certificate)
	certContent, err := os.ReadFile(certificateFile)
	if err != nil {
		fmt.Println("Error reading certificate file:", err)
		return
	}

	// Prepare the payload for the decision request
	payload := map[string]string{
		"ipns_key":      ipnsKey,
		"function_name": functionName,        // Decision function name
		"certificate":   string(certContent), // Certificate file
	}

	// Serialize the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling payload: %v\n", err)
		return
	}

	// Connect to the middleware via TCP
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		fmt.Printf("Error connecting to middleware: %v\n", err)
		return
	}
	defer conn.Close()

	// Prefix the payload with its length (8 bytes)
	lengthPrefix := make([]byte, 8)
	binary.BigEndian.PutUint64(lengthPrefix, uint64(len(payloadBytes)))

	// Send the length prefix
	_, err = conn.Write(lengthPrefix)
	if err != nil {
		fmt.Println("Error sending length prefix:", err)
		return
	}

	// Send the payload in chunks
	chunkSize := 8192 // 8 KB chunks
	for i := 0; i < len(payloadBytes); i += chunkSize {
		end := i + chunkSize
		if end > len(payloadBytes) {
			end = len(payloadBytes)
		}
		_, err := conn.Write(payloadBytes[i:end])
		if err != nil {
			fmt.Println("Error sending data chunk to middleware:", err)
			return
		}
	}
	fmt.Printf("Payload size sent: %d bytes\n", len(payloadBytes))

	// Receive acknowledgment from the middleware
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("Error reading acknowledgment from middleware: %v\n", err)
		return
	}
	fmt.Printf("Acknowledgment from middleware: %s\n", string(buffer[:n]))

	// Poll for a response from middleware
	var responseBytes []byte
	tempBuffer := make([]byte, 4096) // 4 KB chunks
	for {
		n, err := conn.Read(tempBuffer)

		// Append the read bytes to the response buffer
		if n > 0 {
			responseBytes = append(responseBytes, tempBuffer[:n]...)
		}

		// Handle errors and EOF
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Middleware closed the connection. Assuming response is complete.")
				break
			}
			fmt.Printf("Error reading response from middleware: %v\n", err)
			return
		}

		// Detect end of response
		if n == 0 || n < len(tempBuffer) {
			break
		}
	}

	// Convert the response bytes to a string and print it
	response := string(responseBytes)
	if response == "" {
		fmt.Println("Error: Empty response from middleware.")
	} else {
		fmt.Printf("Response from middleware: %s\n", response)
	}
}

func loadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	// Read the private key file
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading private key file: %w", err)
	}

	// Decode the PEM block
	block, _ := pem.Decode(keyBytes)
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return nil, fmt.Errorf("invalid private key file: no PEM block found or incorrect type")
	}

	// Parse the private key
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing private key: %w", err)
	}

	return privateKey, nil
}
