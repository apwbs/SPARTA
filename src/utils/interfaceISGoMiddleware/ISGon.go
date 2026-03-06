package interfaceISGoMiddleware

import (
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	mrand "math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"
	decisionFunctions "sparta/src/decisionfunctions"
	"sparta/src/utils/encryption"
	"sparta/src/utils/ipfs"
	"sparta/src/utils/isgonHelper"
	seedGeneration "sparta/src/utils/seedGenerator"
	blockchain "sparta/src/utils/interaction"
	structs "sparta/src/utils/structures"
	"time"

	"github.com/Knetic/govaluate"
	shell "github.com/ipfs/go-ipfs-api"
)

var TESTMODE bool

func CreateBootstrapCertificate() ([]byte, crypto.PrivateKey) {
	// Load the ECDSA private key from the file (same as normal)
	ecdsaPrivateKey, err := isgonHelper.LoadPrivateKey("privkey/privateKey.pem")
	if err != nil {
		fmt.Println("Error loading ECDSA private key:", err)
		return nil, nil
	}

	// Create a template WITHOUT using the shared seed.
	template := &x509.Certificate{
		SerialNumber: &big.Int{},
		Subject:      pkix.Name{CommonName: "localhost"},
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"localhost"},
		// Optional: include a marker extension so you know it's bootstrap
		ExtraExtensions: []pkix.Extension{
			{
				Id:       asn1.ObjectIdentifier{1, 2, 3, 4, 5, 6, 7, 8, 99}, // your own OID
				Critical: false,
				Value:    []byte("BOOTSTRAP"),
			},
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &ecdsaPrivateKey.PublicKey, ecdsaPrivateKey)
	if err != nil {
		fmt.Println("Failed to create bootstrap certificate:", err)
		return nil, nil
	}

	return certDER, ecdsaPrivateKey
}

func CreateCertificate() ([]byte, crypto.PrivateKey) {
	// Shared secret seed known to all TEEs
	seedBytes := seedGeneration.GetKey()

	// Generate deterministic DH key pair
	_, ecdhPublicKey, err := isgonHelper.GenerateDeterministicDHKeyPair(seedBytes)
	if err != nil {
		fmt.Println("Error generating DH key pair:", err)
		return nil, nil
	}

	// Properly encode the ECDH public key in PKIX format
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(ecdhPublicKey)
	if err != nil {
		fmt.Println("Error marshaling ECDH public key:", err)
		return nil, nil
	}

	// Load the ECDSA private key from the file
	ecdsaPrivateKey, err := isgonHelper.LoadPrivateKey("privkey/privateKey.pem")
	if err != nil {
		fmt.Println("Error loading ECDSA private key:", err)
		return nil, nil
	}

	// Create the certificate template
	template := &x509.Certificate{
		SerialNumber: &big.Int{},
		Subject:      pkix.Name{CommonName: "localhost"},
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"localhost"},

		// Embed the ECDH public key as an extra extension
		ExtraExtensions: []pkix.Extension{
			{
				Id:       asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}, // OID for ECDH public key
				Critical: false,
				Value:    publicKeyBytes,
			},
		},
	}

	// Create the self-signed certificate using the ECDSA private key
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &ecdsaPrivateKey.PublicKey, ecdsaPrivateKey)
	if err != nil {
		fmt.Println("Failed to create certificate:", err)
		return nil, nil
	}

	return certDER, ecdsaPrivateKey
}

func CheckCertificate(certificate []byte) (bool, string) {
	// Load the CA certificate
	caCertDER, err := isgonHelper.LoadPEM("../certauth/pubkey/ca_cert.pem")
	if err != nil {
		fmt.Println("Error loading CA certificate:", err)
		return false, ""
	}
	parsedCaCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		fmt.Println("Error parsing CA certificate:", err)
		return false, ""
	}

	// Decode the DO certificate from PEM to DER
	block, _ := pem.Decode(certificate)
	if block == nil || block.Type != "CERTIFICATE" {
		fmt.Println("Error decoding PEM certificate or invalid type")
		return false, ""
	}
	doCertDER := block.Bytes

	// Parse the DO certificate
	parsedDoCert, err := x509.ParseCertificate(doCertDER)
	if err != nil {
		fmt.Println("Error parsing DO certificate:", err)
		return false, ""
	}

	// Verify the certificate's signature
	if err = parsedDoCert.CheckSignatureFrom(parsedCaCert); err != nil {
		fmt.Println("Certificate verification failed:", err)
		return false, ""
	}
	fmt.Println("Certificate successfully verified by TEE")

	// Extract the blockchain address and attributes
	//blockchainAddressOID := asn1.ObjectIdentifier{1, 2, 3, 4, 5, 6, 7, 8, 1}
	attributesOID := asn1.ObjectIdentifier{1, 2, 3, 4, 5, 6, 7, 8, 2}

	//var blockchainAddress string
	var attributes []string
	for _, ext := range parsedDoCert.Extensions {
		// if ext.Id.Equal(blockchainAddressOID) {
		// 	// if _, err := asn1.Unmarshal(ext.Value, &blockchainAddress); err != nil {
		// 	// 	fmt.Println("Error unmarshaling blockchain address:", err)
		// 	// 	return false, "", ""
		// 	// }
		// } else
		if ext.Id.Equal(attributesOID) {
			if _, err := asn1.Unmarshal(ext.Value, &attributes); err != nil {
				fmt.Println("Error unmarshaling attributes:", err)
				return false, ""
			}
		}
	}
	return true, strings.Join(attributes, ", ")
}

func CheckCallability(policy, attributes string) bool {
	parsedAttributes := isgonHelper.ParseAttributes(attributes)

	filterCondition, _ := isgonHelper.ParseFEELContext(policy)

	// Create a new govaluate expression
	expression, err := govaluate.NewEvaluableExpression(filterCondition)
	if err != nil {
		fmt.Println("Error creating expression:", err)
		return false
	}

	// Evaluate the expression with the provided attributes
	result, err := expression.Evaluate(parsedAttributes)
	if err != nil {
		fmt.Println("Error evaluating expression:", err)
		return false
	}

	fmt.Printf("CheckCallability Result: %v\n", result)
	return result.(bool)

}

func EncryptRawData(fileString, fileExtension string) (int64, string) {
	sh := shell.NewShell("localhost:5001")

	now := time.Now()
	nowInt := now.Format("20060102150405")
	nowIntVal, _ := time.Parse("20060102150405", nowInt)
	source := mrand.NewSource(nowIntVal.UnixNano())
	rng := mrand.New(source)
	messageID := rng.Int63n(1<<63 - 1)
	fmt.Println("messageID: ", messageID)

	encryptedData, _ := encryption.EncryptData(messageID, fileString)

	ipfsData := IPFSData{
		Ciphertext: encryptedData,
		Extension:  fileExtension,
	}

	ipfsJSON, err := json.Marshal(ipfsData)
	if err != nil {
		fmt.Println("Error creating JSON file for IPFS:", err)
		os.Exit(1)
	}

	ipfsHash, err := ipfs.UploadToIPFS(sh, ipfsJSON)
	if err != nil {
		fmt.Println("Error updating JSON file to IPFS:", err)
		os.Exit(1)
	}
	fmt.Println("IPFS Hash: ", ipfsHash)

	return messageID, ipfsHash

}

type IPFSData struct {
	Ciphertext interface{} `json:"ciphertext"`
	Extension  interface{} `json:"extension"`
}

func DecryptRawData(messageID int, functionName string) (string, string, error) {
	data := isgonHelper.RetrieveCiphertext(int64(messageID), functionName)

	decryptedString, err := encryption.DecryptData(int64(messageID), data.Ciphertext.(string))
	if err != nil {
		return "error during decryption", "", err
	}

	fileExtension := data.Extension.(string)
	fmt.Println("fileExtension:", fileExtension)

	fmt.Println("----------------------------------------------------------------------------------------------------------------------------")

	return decryptedString, fileExtension, nil
}

func NewEncryptDocument(inputData, structName string) (int64, string) {
	sh := shell.NewShell("localhost:5001")

	structType, exists := structs.StructRegistry[structName]
	if !exists {
		fmt.Printf("Struct %s not found in registry\n", structName)
		return 0, ""
	}

	// Create a slice dynamically
	sliceType := reflect.SliceOf(structType)
	newStructPtr := reflect.New(sliceType).Interface()

	// Parse JSON into the slice
	err := json.Unmarshal([]byte(inputData), newStructPtr)
	if err != nil {
		fmt.Println("Error parsing input JSON into Person structs:", err)
		return 0, ""
	}

	// Dereference the pointer to get the slice
	newStruct := reflect.ValueOf(newStructPtr).Elem()
	if newStruct.Kind() != reflect.Slice {
		fmt.Println("Expected a slice after dereferencing")
		return 0, ""
	}

	if newStruct.Len() > 1 {
		fmt.Println("Multiple entries detected. Only the accepted. MAYBE WE WANT TO CHANGE THIS. LETS TALK ABOUT IT!")
		os.Exit(1)
	}

	now := time.Now()
	nowInt := now.Format("20060102150405")
	nowIntVal, _ := time.Parse("20060102150405", nowInt)
	source := mrand.NewSource(nowIntVal.UnixNano())
	rng := mrand.New(source)
	messageID := rng.Int63n(1<<63 - 1)
	fmt.Println("messageID: ", messageID)

	dataToEncrypt := newStruct.Index(0).Interface()
	dataToEncryptJSON, err := json.Marshal(dataToEncrypt)

	encryptedData, _ := encryption.EncryptData(messageID, string(dataToEncryptJSON))

	ipfsData := IPFSData{
		// Add ID of the invoker that you find in the Certificate released by the CA
		Ciphertext: encryptedData,
		Extension:  ".json",
	}

	ipfsJSON, err := json.Marshal(ipfsData)
	if err != nil {
		fmt.Println("Error creating JSON file for IPFS:", err)
		os.Exit(1)
	}

	ipfsHash, err := ipfs.UploadToIPFS(sh, ipfsJSON)
	if err != nil {
		fmt.Println("Error updating JSON file to IPFS:", err)
		os.Exit(1)
	}
	fmt.Println("IPFS Hash: ", ipfsHash)

	return messageID, ipfsHash
}

func NewDecryptDocuments(messageID int, functionName, structName string) (string, error) {
	data := isgonHelper.RetrieveCiphertext(int64(messageID), functionName)

	// Retrieve the struct type dynamically
	structType, exists := structs.StructRegistry[structName]
	if !exists {
		return "", fmt.Errorf("Struct %s not found in registry", structName)
	}

	// Check if the ID of the invoker inside the file corresponds to the ID in the certificate
	decryptedData, err := encryption.DecryptData(int64(messageID), data.Ciphertext.(string))
	if err != nil {
		return "error during decryption", err
	}

	newElement := reflect.New(structType).Elem()
	err = json.Unmarshal([]byte(decryptedData), newElement.Addr().Interface())

	fmt.Println("----------------------------------------------------------------------------------------------------------------------------")

	return decryptedData, nil
}

func ParseSetRequestFromQueue(payload map[string]string) ([]byte, string, string, string, string, error) {
	// Extract fields from the payload
	certificate := []byte(payload["certificate"])
	encryptedInputDataBase64 := payload["encrypted_data"]
	fileExtension := payload["file_extension"]
	functionName := payload["function_name"]
	ipnsKey := payload["ipns_key"]

	functionName = strings.TrimPrefix(strings.TrimPrefix(string(functionName), "set"), "get")

	// Parse the certificate
	block, _ := pem.Decode(certificate)
	if block == nil || block.Type != "CERTIFICATE" {
		fmt.Println("Error decoding PEM certificate or invalid type")
		return nil, "", "", "", "", fmt.Errorf("invalid PEM certificate format")
	}
	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		fmt.Printf("Error parsing certificate: %v\n", err)
		return nil, "", "", "", "", err
	}

	// Extract the ECDSA public key from the certificate
	certPublicKey, ok := parsedCert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		fmt.Println("Certificate public key is not an ECDSA public key")
		return nil, "", "", "", "", fmt.Errorf("invalid public key type in certificate")
	}

	// Decode the client's public key from the payload
	clientPublicKeyBytes, err := base64.StdEncoding.DecodeString(payload["client_public_key"])
	if err != nil {
		fmt.Printf("Error decoding client public key: %v\n", err)
		return nil, "", "", "", "", err
	}

	// Compute the hash of the client's public key (same as the client did)
	hash := sha256.Sum256(clientPublicKeyBytes)

	// Decode the ECDSA signature (r, s) from ASN.1 format
	var signature struct {
		R, S *big.Int
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(payload["signature"])
	if err != nil {
		fmt.Printf("Error decoding signature: %v\n", err)
		return nil, "", "", "", "", err
	}

	if _, err := asn1.Unmarshal(signatureBytes, &signature); err != nil {
		fmt.Printf("Error unmarshaling signature: %v\n", err)
		return nil, "", "", "", "", err
	}

	// Verify the signature using the public key extracted from the certificate
	if !ecdsa.Verify(certPublicKey, hash[:], signature.R, signature.S) {
		fmt.Println("Signature verification failed")
		return nil, "", "", "", "", fmt.Errorf("signature verification failed")
	}

	fmt.Println("Signature successfully verified using the certificate's public key")

	// Decode encrypted input data
	encryptedInputData, err := base64.StdEncoding.DecodeString(encryptedInputDataBase64)
	if err != nil {
		fmt.Printf("Error decoding encrypted input data: %v\n", err)
		return nil, "", "", "", "", err
	}

	// Generate deterministic server key
	seedBytes := seedGeneration.GetKey()
	serverPrivateKey, _, err := isgonHelper.GenerateDeterministicDHKeyPair(seedBytes)
	if err != nil {
		fmt.Printf("Error generating server private key: %v\n", err)
		return nil, "", "", "", "", err
	}

	// Reconstruct the client's public key
	clientPublicKey, err := ecdh.X25519().NewPublicKey(clientPublicKeyBytes)
	if err != nil {
		fmt.Printf("Error reconstructing client public key: %v\n", err)
		return nil, "", "", "", "", err
	}

	// Compute shared secret
	sharedSecret, err := serverPrivateKey.ECDH(clientPublicKey)
	if err != nil {
		fmt.Printf("Error computing shared secret: %v\n", err)
		return nil, "", "", "", "", err
	}
	fmt.Printf("Shared Secret: %x\n", sharedSecret)

	// Derive symmetric key using SHA-256
	symmetricKey := sha256.Sum256(sharedSecret)

	// Decrypt the encrypted input data using the symmetric key
	decryptedData, err := encryption.DecryptWithAESGCMClientInput(symmetricKey[:], encryptedInputData)
	if err != nil {
		fmt.Printf("Error decrypting input data: %v\n", err)
		return nil, "", "", "", "", err
	}

	// Convert decrypted data to a string
	fileString := string(decryptedData)

	return certificate, functionName, fileString, fileExtension, ipnsKey, nil
}

func ParseSetRequestFromQueueBytes(payload map[string]string) ([]byte, string, []byte, string, string, error) {
	// Extract fields from the payload
	certificate := []byte(payload["certificate"])
	encryptedInputDataBase64 := payload["encrypted_data"]
	fileExtension := payload["file_extension"]
	functionName := payload["function_name"]
	ipnsKey := payload["ipns_key"]

	functionName = strings.TrimPrefix(strings.TrimPrefix(string(functionName), "set"), "get")

	// Parse the certificate
	block, _ := pem.Decode(certificate)
	if block == nil || block.Type != "CERTIFICATE" {
		fmt.Println("Error decoding PEM certificate or invalid type")
		return nil, "", nil, "", "", fmt.Errorf("invalid PEM certificate format")
	}
	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		fmt.Printf("Error parsing certificate: %v\n", err)
		return nil, "", nil, "", "", err
	}

	// Extract the ECDSA public key from the certificate
	certPublicKey, ok := parsedCert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		fmt.Println("Certificate public key is not an ECDSA public key")
		return nil, "", nil, "", "", fmt.Errorf("invalid public key type in certificate")
	}

	// Decode the client's public key from the payload
	clientPublicKeyBytes, err := base64.StdEncoding.DecodeString(payload["client_public_key"])
	if err != nil {
		fmt.Printf("Error decoding client public key: %v\n", err)
		return nil, "", nil, "", "", err
	}

	// Compute the hash of the client's public key (same as the client did)
	hash := sha256.Sum256(clientPublicKeyBytes)

	// Decode the ECDSA signature (r, s) from ASN.1 format
	var signature struct {
		R, S *big.Int
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(payload["signature"])
	if err != nil {
		fmt.Printf("Error decoding signature: %v\n", err)
		return nil, "", nil, "", "", err
	}

	if _, err := asn1.Unmarshal(signatureBytes, &signature); err != nil {
		fmt.Printf("Error unmarshaling signature: %v\n", err)
		return nil, "", nil, "", "", err
	}

	// Verify the signature using the public key extracted from the certificate
	if !ecdsa.Verify(certPublicKey, hash[:], signature.R, signature.S) {
		fmt.Println("Signature verification failed")
		return nil, "", nil, "", "", fmt.Errorf("signature verification failed")
	}

	fmt.Println("Signature successfully verified using the certificate's public key")

	// Decode encrypted input data
	encryptedInputData, err := base64.StdEncoding.DecodeString(encryptedInputDataBase64)
	if err != nil {
		fmt.Printf("Error decoding encrypted input data: %v\n", err)
		return nil, "", nil, "", "", err
	}

	// Generate deterministic server key
	seedBytes := seedGeneration.GetKey()
	serverPrivateKey, _, err := isgonHelper.GenerateDeterministicDHKeyPair(seedBytes)
	if err != nil {
		fmt.Printf("Error generating server private key: %v\n", err)
		return nil, "", nil, "", "", err
	}

	// Reconstruct the client's public key
	clientPublicKey, err := ecdh.X25519().NewPublicKey(clientPublicKeyBytes)
	if err != nil {
		fmt.Printf("Error reconstructing client public key: %v\n", err)
		return nil, "", nil, "", "", err
	}

	// Compute shared secret
	sharedSecret, err := serverPrivateKey.ECDH(clientPublicKey)
	if err != nil {
		fmt.Printf("Error computing shared secret: %v\n", err)
		return nil, "", nil, "", "", err
	}
	fmt.Printf("Shared Secret: %x\n", sharedSecret)

	// Derive symmetric key using SHA-256
	symmetricKey := sha256.Sum256(sharedSecret)

	// Decrypt the encrypted input data using the symmetric key
	decryptedData, err := encryption.DecryptWithAESGCMClientInput(symmetricKey[:], encryptedInputData)
	if err != nil {
		fmt.Printf("Error decrypting input data: %v\n", err)
		return nil, "", nil, "", "", err
	}

	return certificate, functionName, decryptedData, fileExtension, ipnsKey, nil
}


func ParseGetRequestFromQueue(payload map[string]string) ([]byte, string, int, error) {
	// Extract fields from the payload
	certificate := []byte(payload["certificate"])
	functionName := payload["function_name"]
	messageIDString := payload["message_id"]

	// Convert messageID to an integer
	messageID, err := strconv.Atoi(messageIDString)
	if err != nil {
		fmt.Printf("Error converting message ID to integer: %v\n", err)
		return nil, "", 0, err
	}

	// Normalize the functionName (removing "set" and "get" prefixes)
	functionName = strings.TrimPrefix(functionName, "set")
	functionName = strings.TrimPrefix(functionName, "get")

	return certificate, functionName, messageID, nil
}

func ParseDecisionRequestFromQueue(payload map[string]string) ([]byte, string, string, error) {
	// Extract fields from the payload
	certificate := []byte(payload["certificate"])
	functionName := payload["function_name"]
	ipnsKey := payload["ipns_key"]

	// Ensure the certificate and functionName exist
	if len(certificate) == 0 {
		err := fmt.Errorf("certificate field is empty or missing")
		fmt.Println(err)
		return nil, "", "", err
	}
	if functionName == "" {
		err := fmt.Errorf("function_name field is empty or missing")
		fmt.Println(err)
		return nil, "", "", err
	}

	return certificate, functionName, ipnsKey, nil
}

type EncryptedData struct {
	Randomness    int64  `json:"Randomness"`
	EncryptedData string `json:"EncryptedData"`
}

type SimpleBatch struct {
	Previous string          `json:"previous"`
	Entries  []EncryptedData `json:"entries"`
}

func EncryptAndUploadLinkedBytes(fileBytes []byte, structName, ipnsKey string) {
	sh := shell.NewShell("localhost:5001")

	// Retrieve the publicKey from IPNS
	publicKeyIPNS := ipfs.RetrieveKey(sh, ipnsKey)
	if publicKeyIPNS == "" {
		fmt.Println("Error: could not retrieve IPNS key")
		return
	}
	fmt.Println("IPNS Key Retrieved:", publicKeyIPNS)

	// Step 1: Parse input data
	structType, exists := structs.StructRegistry[structName]
	if !exists {
		fmt.Println("Error: Struct not found in registry")
		return
	}

	sliceType := reflect.SliceOf(structType)
	newStructPtr := reflect.New(sliceType).Interface()

	if err := json.Unmarshal(fileBytes, newStructPtr); err != nil {
		fmt.Println("Error parsing input JSON:", err)
		return
	}

	newStruct := reflect.ValueOf(newStructPtr).Elem()
	if newStruct.Kind() != reflect.Slice {
		fmt.Println("Error: Expected slice")
		return
	}

	// Step 2: Encrypt records
	seedBytes := seedGeneration.GetKey()
	var encryptedEntries []EncryptedData

	startEncryption := time.Now()

	for i := 0; i < newStruct.Len(); i++ {
		randomness, err := isgonHelper.GenerateRandomness()
		if err != nil {
			fmt.Println("Randomness error:", err)
			return
		}
		record := newStruct.Index(i).Interface()
		recordJSON, _ := json.Marshal(record)

		encrypted, err := encryption.NewEncryptData(seedBytes, randomness, string(recordJSON))
		if err != nil {
			fmt.Println("Encryption error:", err)
			return
		}

		encryptedEntries = append(encryptedEntries, EncryptedData{
			Randomness:    randomness,
			EncryptedData: encrypted,
		})
	}

	encryptionTime := time.Since(startEncryption)
	fmt.Printf("Encryption time: %s\n", encryptionTime)

	// Step 3: Create chained batch
	prevCID, _ := ipfs.RetrieveFromIPNS(sh, publicKeyIPNS)
	fmt.Println("Batch CID from IPNS:", prevCID)

	batch := SimpleBatch{
		Previous: prevCID,
		Entries:  encryptedEntries,
	}

	batchJSON, err := json.Marshal(batch)
	if err != nil {
		fmt.Println("Marshal error:", err)
		return
	}

	// Upload to IPFS
	cid, err := ipfs.UploadToIPFS(sh, batchJSON)
	if err != nil {
		fmt.Println("Upload to IPFS failed:", err)
		return
	}
	fmt.Println("Uploaded batch CID:", cid)

	// Update IPNS
	if err := ipfs.UploadToIPNS(sh, cid, publicKeyIPNS); err != nil {
		fmt.Println("IPNS update failed:", err)
		return
	}
	fmt.Println("IPNS now points to CID:", cid)
}

func DecryptLinkedLog(keyName, structName string) ([]interface{}, error) {
	sh := shell.NewShell("localhost:5001")
	var results []interface{}

	// publicKeyIPNS := ipfs.RetrieveKey(sh, keyName)
	// if publicKeyIPNS == "" {
	// 	return nil, fmt.Errorf("failed to retrieve public key for keyName: %s", keyName)
	// }
	// fmt.Println("publicKeyIPNS:",publicKeyIPNS)

	blockchainKeyIPNS, _ := blockchain.GetIPNSKey(keyName)
	if blockchainKeyIPNS == "" {
		return nil, fmt.Errorf("failed to retrieve public key for keyName: %s", keyName)
	}
	fmt.Println("blockchainKeyIPNS:", blockchainKeyIPNS)

	cid, err := ipfs.RetrieveFromIPNS(sh, blockchainKeyIPNS)
	fmt.Println("CID from IPNS:", cid)
	if err != nil || cid == "" {
		return nil, fmt.Errorf("failed to resolve IPNS head for key %s: %v", keyName, err)
	}

	structType, ok := structs.StructRegistry[structName]
	if !ok {
		return nil, fmt.Errorf("struct type %s not found in registry", structName)
	}

	seedBytes := seedGeneration.GetKey()
	batchCounter := 0
	decryptionTimes := make(map[int]time.Duration)
	entryIndex := 0

	for cid != "" {
		batchCounter++

		data, err := ipfs.FetchDataFromIPFS(sh, cid)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch CID %s: %v", cid, err)
		}

		// Use json.RawMessage to defer allocation
		var rawBatch struct {
			Previous string            `json:"previous"`
			Entries  []json.RawMessage `json:"entries"`
		}
		if err := json.Unmarshal(data, &rawBatch); err != nil {
			return nil, fmt.Errorf("failed to unmarshal batch %s: %v", cid, err)
		}

		for _, raw := range rawBatch.Entries {
			var enc EncryptedData
			if err := json.Unmarshal(raw, &enc); err != nil {
				fmt.Println("Warning: failed to parse encrypted entry:", err)
				continue
			}

			test := time.Now()
			plain, err := encryption.NewDecryptData(seedBytes, enc.Randomness, enc.EncryptedData)
			elapsed := time.Since(test)
			decryptionTimes[entryIndex] = elapsed
			entryIndex++
			if err != nil {
				fmt.Println("Warning: failed to decrypt entry:", err)
				continue
			}

			newElem := reflect.New(structType).Interface()
			if err := json.Unmarshal([]byte(plain), newElem); err != nil {
				fmt.Println("Warning: failed to unmarshal decrypted entry:", err)
				continue
			}

			results = append(results, newElem)
		}

		cid = rawBatch.Previous
	}

	var total time.Duration
	for _, t := range decryptionTimes {
		total += t
	}
	fmt.Printf("Total decryption time: %v\n", total)
	return results, nil
}

func printRecordAsJSON(v any, maxBytes int) {
    b, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        fmt.Println("Cannot marshal record to JSON:", err)
        return
    }
    if maxBytes > 0 && len(b) > maxBytes {
        fmt.Printf("%s...\n(truncated, %d bytes total)\n", b[:maxBytes], len(b))
        return
    }
    fmt.Println(string(b))
}

func RetrieveStructSliceLinkedLog(structName, keyName string) (interface{}, error) {
	records, err := DecryptLinkedLog(keyName, structName)

	if err != nil {
		return nil, fmt.Errorf("error retrieving or decrypting: %v", err)
	}

	fmt.Printf("Decrypted %d records for %s\n", len(records), structName)

	return records, nil
}

func Decision(functionName, structName, ipnsKey string) string {
	// Retrieve the struct slice dynamically
	structSliceInterface, err := RetrieveStructSliceLinkedLog(structName, ipnsKey)
	if err != nil {
		return "error in getting the struct back"
	}

	// Convert to reflect.Value
	structSlice := reflect.ValueOf(structSliceInterface)

	// Ensure it's a slice
	if structSlice.Kind() != reflect.Slice {
		fmt.Println("Error: structSlice is not a slice")
		return "error: structSlice is not a slice"
	}

	// Lookup receiver for this decision
	recv, ok := decisionFunctions.DecisionRegistry[functionName]
	if !ok {
		fmt.Println("Error: decision not registered:", functionName)
		return "Error: Method not found"
	}

	// Lookup method on receiver
	method := reflect.ValueOf(recv).MethodByName(functionName)
	if !method.IsValid() {
		fmt.Println("Error: method not found on registered receiver:", functionName)
		return "Error: Method not found"
	}

	fmt.Println("Found method via DecisionRegistry:", functionName)
	return executeMethod(method, structSlice, ipnsKey, functionName)
}

// Helper function to execute the method dynamically
func executeMethod(method reflect.Value, structSlice reflect.Value, ipnsKey, functionName string) string {
	// Pre-allocate a list to store prepared args
	argsList := make([][]reflect.Value, structSlice.Len()) // Each element is a slice of reflect.Value

	// First loop: Prepare args
	for i := 0; i < structSlice.Len(); i++ {
		// Get the person element at index i
		dataInput := structSlice.Index(i).Interface()

		// Create inputs from the person dynamically and handle the error
		inputs, err := isgonHelper.CreateInputsFromDataInput(dataInput)
		if err != nil {
			fmt.Println("Error creating inputs:", err)
			continue // Skip this iteration if there's an error
		}

		// Store the args for this input
		inputValue := reflect.ValueOf(inputs)
		if inputValue.IsValid() {
			argsList[i] = []reflect.Value{inputValue}
		}
	}

	// Second loop: Measure time for calling the function
	start111 := time.Now()
	// for i := 0; i < len(argsList); i++ {
	// 	_ = method.Call(argsList[i])
	// }
	for i := 0; i < len(argsList); i++ {
		fmt.Printf("Result for input #%d:\n", i)
		results := method.Call(argsList[i])
		for j, result := range results {
			fmt.Printf("  [%d] %v\n", j, result.Interface())
		}
		fmt.Println(strings.Repeat("-", 40))
	}
	elapsed111 := time.Since(start111)
	fmt.Printf("Total time taken for method.Call(args): %s\n", elapsed111)

	fmt.Println("----------------------------------------------------------------------------------------------------------------------------")

	return "All decisions made"
}

func DecisionWithAggregation(functionName string, structSlice reflect.Value, additional map[string]interface{}, decryptionTime, aggregationTime time.Duration, ipnsKey string) string {
	// Ensure structSlice is a slice
	if structSlice.Kind() != reflect.Slice {
		fmt.Println("Error: structSlice is not a slice")
		return "error: structSlice is not a slice"
	}

	// // First Attempt: Try Rules2425262728Aggregation{}
	// m1 := decisionFunctions.Rules2425262728Aggregation{}
	// value1 := reflect.ValueOf(m1)
	// method1 := value1.MethodByName(functionName)

	// if method1.IsValid() {
	// 	fmt.Println("Found method in Rules2425262728Aggregation{}")
	// 	return executeMethodWithAggregation(method1, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // First Attempt: Try Rules8WithAggregation{}
	// m8 := decisionFunctions.Rules8WithAggregation{}
	// value8 := reflect.ValueOf(m8)
	// method8 := value8.MethodByName(functionName)

	// if method8.IsValid() {
	// 	fmt.Println("Found method in Rules8WithAggregation{}")
	// 	return executeMethodWithAggregation(method8, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // First Attempt: Try Rules9WithAggregation{}
	// m9 := decisionFunctions.Rules9WithAggregation{}
	// value9 := reflect.ValueOf(m9)
	// method9 := value9.MethodByName(functionName)

	// if method9.IsValid() {
	// 	fmt.Println("Found method in Rules9WithAggregation{}")
	// 	return executeMethodWithAggregation(method9, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // First Attempt: Try Rules10WithAggregation{}
	// m10 := decisionFunctions.Rules10WithAggregation{}
	// value10 := reflect.ValueOf(m10)
	// method10 := value10.MethodByName(functionName)

	// if method10.IsValid() {
	// 	fmt.Println("Found method in Rules10WithAggregation{}")
	// 	return executeMethodWithAggregation(method10, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // First Attempt: Try Rules11WithAggregation{}
	// m11 := decisionFunctions.Rules11WithAggregation{}
	// value11 := reflect.ValueOf(m11)
	// method11 := value11.MethodByName(functionName)

	// if method11.IsValid() {
	// 	fmt.Println("Found method in Rules11WithAggregation{}")
	// 	return executeMethodWithAggregation(method11, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // First Attempt: Try Rules12WithAggregation{}
	// m12 := decisionFunctions.Rules12WithAggregation{}
	// value12 := reflect.ValueOf(m12)
	// method12 := value12.MethodByName(functionName)

	// if method12.IsValid() {
	// 	fmt.Println("Found method in Rules12WithAggregation{}")
	// 	return executeMethodWithAggregation(method12, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // First Attempt: Try Rules13WithAggregation{}
	// m13 := decisionFunctions.Rules13WithAggregation{}
	// value13 := reflect.ValueOf(m13)
	// method13 := value13.MethodByName(functionName)

	// if method13.IsValid() {
	// 	fmt.Println("Found method in Rules13WithAggregation{}")
	// 	return executeMethodWithAggregation(method13, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // First Attempt: Try Rules14WithAggregation{}
	// m14 := decisionFunctions.Rules14WithAggregation{}
	// value14 := reflect.ValueOf(m14)
	// method14 := value14.MethodByName(functionName)

	// if method14.IsValid() {
	// 	fmt.Println("Found method in Rules14WithAggregation{}")
	// 	return executeMethodWithAggregation(method14, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // First Attempt: Try Rules15WithAggregation{}
	// m15 := decisionFunctions.Rules15WithAggregation{}
	// value15 := reflect.ValueOf(m15)
	// method15 := value15.MethodByName(functionName)

	// if method15.IsValid() {
	// 	fmt.Println("Found method in Rules15WithAggregation{}")
	// 	return executeMethodWithAggregation(method15, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // First Attempt: Try Rules16WithAggregation{}
	// m16 := decisionFunctions.Rules16WithAggregation{}
	// value16 := reflect.ValueOf(m16)
	// method16 := value16.MethodByName(functionName)

	// if method16.IsValid() {
	// 	fmt.Println("Found method in Rules16WithAggregation{}")
	// 	return executeMethodWithAggregation(method16, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Second Attempt: Try Rules17WithAggregation{}
	// m17 := decisionFunctions.Rules17WithAggregation{}
	// value17 := reflect.ValueOf(m17)
	// method17 := value17.MethodByName(functionName)

	// if method17.IsValid() {
	// 	fmt.Println("Found method in Rules17WithAggregation{}")
	// 	return executeMethodWithAggregation(method17, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Second Attempt: Try Rules18WithAggregation{}
	// m18 := decisionFunctions.Rules18WithAggregation{}
	// value18 := reflect.ValueOf(m18)
	// method18 := value18.MethodByName(functionName)

	// if method18.IsValid() {
	// 	fmt.Println("Found method in Rules18WithAggregation{}")
	// 	return executeMethodWithAggregation(method18, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Second Attempt: Try Rules19WithAggregation{}
	// m19 := decisionFunctions.Rules19WithAggregation{}
	// value19 := reflect.ValueOf(m19)
	// method19 := value19.MethodByName(functionName)

	// if method19.IsValid() {
	// 	fmt.Println("Found method in Rules19WithAggregation{}")
	// 	return executeMethodWithAggregation(method19, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Third Attempt: Try Rules20WithAggregation{}
	// m20 := decisionFunctions.Rules20WithAggregation{}
	// value20 := reflect.ValueOf(m20)
	// method20 := value20.MethodByName(functionName)

	// if method20.IsValid() {
	// 	fmt.Println("Found method in Rules20WithAggregation{}")
	// 	return executeMethodWithAggregation(method20, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Third Attempt: Try Rules21WithAggregation{}
	// m21 := decisionFunctions.Rules21WithAggregation{}
	// value21 := reflect.ValueOf(m21)
	// method21 := value21.MethodByName(functionName)

	// if method21.IsValid() {
	// 	fmt.Println("Found method in Rules21WithAggregation{}")
	// 	return executeMethodWithAggregation(method21, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Third Attempt: Try Rules22WithAggregation{}
	// m22 := decisionFunctions.Rules22WithAggregation{}
	// value22 := reflect.ValueOf(m22)
	// method22 := value22.MethodByName(functionName)

	// if method22.IsValid() {
	// 	fmt.Println("Found method in Rules22WithAggregation{}")
	// 	return executeMethodWithAggregation(method22, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Third Attempt: Try Rules23WithAggregation{}
	// m23 := decisionFunctions.Rules23WithAggregation{}
	// value23 := reflect.ValueOf(m23)
	// method23 := value23.MethodByName(functionName)

	// if method23.IsValid() {
	// 	fmt.Println("Found method in Rules23WithAggregation{}")
	// 	return executeMethodWithAggregation(method23, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Third Attempt: Try Rules24WithAggregation{}
	// m24 := decisionFunctions.Rules24WithAggregation{}
	// value24 := reflect.ValueOf(m24)
	// method24 := value24.MethodByName(functionName)

	// if method24.IsValid() {
	// 	fmt.Println("Found method in Rules24WithAggregation{}")
	// 	return executeMethodWithAggregation(method24, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Third Attempt: Try Rules25WithAggregation{}
	// m25 := decisionFunctions.Rules25WithAggregation{}
	// value25 := reflect.ValueOf(m25)
	// method25 := value25.MethodByName(functionName)

	// if method25.IsValid() {
	// 	fmt.Println("Found method in Rules25WithAggregation{}")
	// 	return executeMethodWithAggregation(method25, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Third Attempt: Try Rules26WithAggregation{}
	// m26 := decisionFunctions.Rules26WithAggregation{}
	// value26 := reflect.ValueOf(m26)
	// method26 := value26.MethodByName(functionName)

	// if method26.IsValid() {
	// 	fmt.Println("Found method in Rules26WithAggregation{}")
	// 	return executeMethodWithAggregation(method26, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Third Attempt: Try Rules27WithAggregation{}
	// m27 := decisionFunctions.Rules27WithAggregation{}
	// value27 := reflect.ValueOf(m27)
	// method27 := value27.MethodByName(functionName)

	// if method27.IsValid() {
	// 	fmt.Println("Found method in Rules27WithAggregation{}")
	// 	return executeMethodWithAggregation(method27, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Third Attempt: Try Rules28WithAggregation{}
	// m28 := decisionFunctions.Rules28WithAggregation{}
	// value28 := reflect.ValueOf(m28)
	// method28 := value28.MethodByName(functionName)

	// if method28.IsValid() {
	// 	fmt.Println("Found method in Rules28WithAggregation{}")
	// 	return executeMethodWithAggregation(method28, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// // Fourth Attempt: Try SeventhMy{}
	// m4 := decisionFunctions.SeventhMy{}
	// value4 := reflect.ValueOf(m4)
	// method4 := value4.MethodByName(functionName)

	// if method4.IsValid() {
	// 	fmt.Println("Found method in SeventhMy{}")
	// 	return executeMethodWithAggregation(method4, structSlice, additional, decryptionTime, aggregationTime, ipnsKey)
	// }

	// If no valid method is found
	fmt.Println("Error: Method not found in any struct!")
	return "Error: Method not found"
}

// Helper function to execute the method dynamically with aggregation
func executeMethodWithAggregation(method reflect.Value, structSlice reflect.Value, additional map[string]interface{}, decryptionTime, aggregationTime time.Duration, ipnsKey string) string {
	// Prepare argument list
	argsList := make([][]reflect.Value, structSlice.Len())

	// First loop: Prepare arguments
	for i := 0; i < structSlice.Len(); i++ {
		// Get struct element at index i
		dataInput := structSlice.Index(i).Interface()

		// Create inputs dynamically and handle errors
		inputs, err := isgonHelper.CreateInputsFromDataInput(dataInput)
		if err != nil {
			fmt.Println("Error creating inputs:", err)
			continue // Skip iteration on error
		}

		// Merge additional parameters if provided
		if additional != nil {
			inputs = isgonHelper.MergeMaps(inputs, additional)
			// fmt.Printf("Merged Inputs at index [%d]: %+v\n", i, inputs)
			// fmt.Printf("Merged Inputs at index [%d]: %+v\n", i, len(inputs))
		}

		// Store valid input arguments
		inputValue := reflect.ValueOf(inputs)
		if inputValue.IsValid() {
			argsList[i] = []reflect.Value{inputValue}
		}
	}

	var numKeys int

	for i := 0; i < len(argsList); i++ {
		for _, val := range argsList[i] {
			if vMap, ok := val.Interface().(map[string]interface{}); ok {
				numKeys = len(vMap)
				break
			}
		}
		if numKeys > 0 {
			break
		}
	}

	// fmt.Printf("🔢 Number of keys in the first entry: %d\n", numKeys)
	// fmt.Printf("The keys are: %v\n", argsList[0][0].Interface())

	// Second loop: Execute method calls and measure time
	startTime := time.Now()
	for i := 0; i < len(argsList); i++ {
		_ = method.Call(argsList[i])
	}
	elapsedTime := time.Since(startTime)

	// Log execution time
	fmt.Printf("Total time taken for method.Call(args): %s\n", elapsedTime)
	fmt.Println("len of structSlice:", structSlice.Len())

	parts := strings.Split(ipnsKey, "_")
	numColumns, _ := strconv.Atoi(parts[2])

	structSliceLen := structSlice.Len()
	resultValueCheck := structSliceLen == numColumns

	red := "\033[31m"
	reset := "\033[0m"

	fmt.Println(red+"len of struct equal to number of rows of the key:"+reset, strconv.FormatBool(resultValueCheck))

	fmt.Println("----------------------------------------------------------------------------------------------------------------------------")

	return "All patient priorities processed"
}

func NewPerformAggregation(feelExpr string, structSlice reflect.Value) (float64, time.Duration, error) {

	// Parse the input to extract the filter condition and aggregation expression
	filterCondition, aggregationExpr, err := isgonHelper.NewParseInput(feelExpr)
	if err != nil {
		return 0, 0, fmt.Errorf("error parsing input: %w", err)
	}
	// fmt.Println("\nExtracted Filter Condition:", filterCondition)
	// fmt.Println("Aggregation Expression:", aggregationExpr)

	// Evaluate the filter condition to get filtered elements
	filteredElements, err := isgonHelper.EvaluateFilterCondition(filterCondition, structSlice)
	if err != nil {
		return 0, 0, fmt.Errorf("error evaluating filter condition: %w", err)
	}
	// HARDCODING THE AGGREGATION EXPRESSION FOR TESTING
	if aggregationExpr == "max(allPlans)" {
		// fmt.Println("Aggregation Expression is hardcoded for testing")
		// fmt.Println("Filtered Elements:", filteredElements)

		// Ensure structSlice is a slice
		if structSlice.Kind() != reflect.Slice {
			// fmt.Println("Error: structSlice is not a slice")
			return 0, 0, fmt.Errorf("structSlice is not a slice")
		}

		// Map to count occurrences of each Plan value
		planCounts := make(map[string]int)

		startTimeAggregation := time.Now()
		for i := 0; i < structSlice.Len(); i++ {
			elem := structSlice.Index(i)

			// Unwrap interface or pointer
			for elem.Kind() == reflect.Interface || elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}

			if elem.Kind() != reflect.Struct {
				// fmt.Printf("Skipping non-struct at index %d (kind: %s)\n", i, elem.Kind())
				continue
			}

			// fmt.Printf("Inspecting struct at index %d: %+v\n", i, elem.Interface())

			planField := elem.FieldByName("Plan")
			if !planField.IsValid() || planField.Kind() != reflect.String {
				// fmt.Printf("'Plan' field missing or not a string at index %d\n", i)
				continue
			}

			planValue := planField.String()
			// fmt.Printf("Raw Plan value: %s\n", planValue)

			if len(planValue) > 2 && planValue[:2] == "PL" {
				planValue = planValue[2:]
				// fmt.Printf("Normalized Plan value: %s\n", planValue)
			}

			planCounts[planValue]++
			// fmt.Printf("Counted Plan %s: now %d\n", planValue, planCounts[planValue])
		}

		// Find the most frequent Plan value
		var mostFrequentPlan string
		maxCount := 0

		for plan, count := range planCounts {
			if count > maxCount {
				mostFrequentPlan = plan
				maxCount = count
			}
		}
		aggregationTime := time.Since(startTimeAggregation)
		// fmt.Printf("Aggregation took: %v\n", aggregationTime)

		// Convert the most frequent Plan value to float64
		mostFrequentPlanFloat, err := strconv.ParseFloat(mostFrequentPlan, 64)
		if err != nil {
			// fmt.Println("Error converting most frequent Plan value to float64:", err)
			return 0, 0, fmt.Errorf("error converting most frequent Plan value to float64: %w", err)
		}

		// Print the most frequent Plan value as float64
		// fmt.Println("Most Frequent Plan Value as float64:", mostFrequentPlanFloat)

		return mostFrequentPlanFloat, aggregationTime, nil

	} else {
		// Call NewPerformAggregationWithDynamicField with the filtered elements and aggregation expression
		result, err := isgonHelper.NewPerformAggregationWithDynamicField(reflect.ValueOf(filteredElements), aggregationExpr)
		if err != nil {
			return 0, 0, fmt.Errorf("error performing aggregation: %w", err)
		}

		return result, 0, nil
	}

}

