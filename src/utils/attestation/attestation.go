package attestation

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sparta/src/utils/https"

	"github.com/edgelesssys/ego/attestation"
	"github.com/edgelesssys/ego/attestation/tcbstatus"
	"github.com/edgelesssys/ego/enclave"
)

/*Method invoked to execute the remote attestation of the Secure Miner hosted in a given address*/
func RemoteAttestation(serverAddr string, expectedMeasurement []byte, expectedPeerPubKeyPath string) ([]byte, []byte) {
	tlsConfig := &tls.Config{InsecureSkipVerify: true} // Skip TLS verification temporarily
	caCertBytes := httpGet(tlsConfig, serverAddr+"/caCert")

	isValid, attributes, certPublicKey := checkCertificate(caCertBytes, expectedPeerPubKeyPath)
	if !isValid {
		panic(errors.New("CA Certificate verification failed"))
	}

	// Validate attributes
	if err := validateAttributes(attributes); err != nil {
		panic(err)
	}

	minerCertBytes := https.HttpGet(tlsConfig, serverAddr+"/cert")

	// Validate the certificate
	_, err := validateCertificate(minerCertBytes, certPublicKey, expectedPeerPubKeyPath)
	if err != nil {
		panic(errors.New("TLS certificate is not valid: " + err.Error()))
	}

	//Create a TLS config that uses the server certificate as root CA so that future connections to the server can be verified.
	cert, err := x509.ParseCertificate(minerCertBytes)
	if err != nil {
		panic(errors.New("Failed to parse certificate: " + err.Error()))
	}

	minerTLSConfig := &tls.Config{RootCAs: x509.NewCertPool(), ServerName: "localhost"}
	minerTLSConfig.RootCAs.AddCert(cert)

	// Get the report via attested TLS channel
	reportBytes := httpGet(minerTLSConfig, serverAddr+"/report")

	// Verify the report
	if err := verifyReport(reportBytes, minerCertBytes, expectedMeasurement); err != nil {
		panic(err)
	}

	fmt.Println("TEE Remote Attestation completed")
	return minerCertBytes, reportBytes
}

// verifyReport verifies that the report is signed by a functioning Intel SGX TEE and matches the expected measurement.
func verifyReport(reportBytes []byte, certBytes []byte, expectedMeasurement []byte) error {
	// Verify the report validity and extract the report data.
	report, err := enclave.VerifyRemoteReport(reportBytes)
	if err == attestation.ErrTCBLevelInvalid {
		fmt.Printf("Warning: TCB level is invalid: %v\n%v\n", report.TCBStatus, tcbstatus.Explain(report.TCBStatus))
		fmt.Println("Ignoring TCB level issue for this sample.")
	} else if err != nil {
		return err
	}

	// Verify the UniqueID or expected measurement
	if !bytes.Equal([]byte(hex.EncodeToString(report.UniqueID)), expectedMeasurement) {
		return errors.New("the report measurement does not match the expected one")
	}

	// Verify that the report data matches the server's TLS certificate
	hash := sha256.Sum256(certBytes)
	if !bytes.Equal(report.Data[:len(hash)], hash[:]) {
		return errors.New("report data does not match the certificate's hash")
	}

	return nil
}

/*Verify that the certificate valid*/
func validateCertificate(certBytes []byte, CAcertPublicKey *ecdsa.PublicKey, expectedPeerPubKeyPath string) (error, error) {
	// Parse the certificate
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, err
	}

	// Check if the public key is able to verify the certificate's signature
	if err := cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature); err != nil {
		return nil, errors.New("certificate signature is not valid")
	}

	// Load the public key from the file
	publicKey, err := loadPublicKey(expectedPeerPubKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	// Verify that the public key inside the certificate matches the loaded public key
	certPublicKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("certificate public key is not an ECDSA public key")
	}

	loadedPublicKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("loaded public key is not an ECDSA public key")
	}

	// Compare the public key coordinates (X and Y) with the loaded public key
	if certPublicKey.X.Cmp(loadedPublicKey.X) != 0 || certPublicKey.Y.Cmp(loadedPublicKey.Y) != 0 {
		return nil, errors.New("certificate public key does not match the loaded public key")
	}

	// Compare the public key inside the certificate with the public key certified by the CA
	if certPublicKey.X.Cmp(CAcertPublicKey.X) != 0 || certPublicKey.Y.Cmp(CAcertPublicKey.Y) != 0 {
		return nil, errors.New("certificate public key does not match the public key certified by the CA")
	}

	// All checks passed
	return nil, nil
}

// httpGet sends an HTTP GET request to the given URL with the specified TLS configuration.
func httpGet(tlsConfig *tls.Config, url string) []byte {
	client := http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		panic(resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body
}

func checkCertificate(certificate []byte, expectedPeerPubKeyPath string) (bool, string, *ecdsa.PublicKey) {
	// Step 1: Load and parse the CA certificate
	caCertDER, err := loadPEM("../certauth/pubkey/ca_cert.pem")
	if err != nil {
		fmt.Println("Error loading CA certificate:", err)
		return false, "", nil
	}
	parsedCaCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		fmt.Println("Error parsing CA certificate:", err)
		return false, "", nil
	}

	// Step 2: Decode the server's certificate (DO certificate) from PEM to DER
	block, _ := pem.Decode(certificate)
	if block == nil || block.Type != "CERTIFICATE" {
		fmt.Println("Error decoding PEM certificate or invalid type")
		return false, "", nil
	}
	doCertDER := block.Bytes

	// Step 3: Parse the server's certificate (DO certificate)
	parsedDoCert, err := x509.ParseCertificate(doCertDER)
	if err != nil {
		fmt.Println("Error parsing DO certificate:", err)
		return false, "", nil
	}

	// Step 4: Verify the server certificate's signature against the CA's public key
	if err = parsedDoCert.CheckSignatureFrom(parsedCaCert); err != nil {
		fmt.Println("Certificate verification failed:", err)
		return false, "", nil
	}

	// Step 5: Load the server's public key from the file
	// Modify the path to the public key file as needed
	publicKey, err := loadPublicKey(expectedPeerPubKeyPath)
	if err != nil {
		fmt.Println("Error loading public key:", err)
		return false, "", nil
	}

	// Step 6: Verify the public key in the certificate matches the loaded public key
	certPublicKey, ok := parsedDoCert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		fmt.Println("Certificate public key is not an ECDSA public key")
		return false, "", nil
	}

	loadedPublicKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		fmt.Println("Loaded public key is not an ECDSA public key")
		return false, "", nil
	}

	// Compare the coordinates of the public keys
	if certPublicKey.X.Cmp(loadedPublicKey.X) != 0 || certPublicKey.Y.Cmp(loadedPublicKey.Y) != 0 {
		fmt.Println("Loaded public key does not match the certificate's public key")
		return false, "", nil
	}

	// Step 8: Extract and return the attributes (optional)
	attributesOID := asn1.ObjectIdentifier{1, 2, 3, 4, 5, 6, 7, 8, 2}
	var attributes []string
	for _, ext := range parsedDoCert.Extensions {
		if ext.Id.Equal(attributesOID) {
			if _, err := asn1.Unmarshal(ext.Value, &attributes); err != nil {
				fmt.Println("Error unmarshaling attributes:", err)
				return false, "", nil
			}
		}
	}

	return true, strings.Join(attributes, ", "), certPublicKey
}

func validateAttributes(attributes string) error {
	// Check if "certified=CCU<number>" exists in the attributes
	pattern := `certified=CCU\d+`
	matched, err := regexp.MatchString(pattern, attributes)
	if err != nil {
		return fmt.Errorf("error matching attributes pattern: %v", err)
	}
	if !matched {
		return errors.New("attributes validation failed: required 'certified=CCU<number>' not found")
	}
	return nil
}

func loadPEM(filename string) ([]byte, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block from file: %s", filename)
	}
	return block.Bytes, nil
}

func loadPublicKey(path string) (crypto.PublicKey, error) {
	// Read the public key file
	keyBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading public key file: %w", err)
	}

	// Decode the PEM block
	block, _ := pem.Decode(keyBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("invalid public key file: no PEM block found or incorrect type")
	}

	// Parse the public key
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing public key: %w", err)
	}

	return publicKey, nil
}
