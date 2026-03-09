package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

var (
	blockchainAddressOID = asn1.ObjectIdentifier{1, 2, 3, 4, 5, 6, 7, 8, 1}
	attributesOID        = asn1.ObjectIdentifier{1, 2, 3, 4, 5, 6, 7, 8, 2}
)

func LoadPublicKey(filename string) (*ecdsa.PublicKey, error) {
	pemData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	ecdsaPubKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast parsed public key to ECDSA")
	}
	return ecdsaPubKey, nil
}

func GenerateKeyPair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func GenerateCertificate(template, parent *x509.Certificate, pubKey *ecdsa.PublicKey, privKey *ecdsa.PrivateKey) ([]byte, error) {
	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, pubKey, privKey)
	if err != nil {
		return nil, err
	}
	return certDER, nil
}

func SavePEM(data []byte, filename string) error {
	pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})
	return ioutil.WriteFile(filename, pemData, 0644)
}

func SavePrivateKey(privKey *ecdsa.PrivateKey, filename string) error {
	privKeyBytes, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return err
	}
	privKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privKeyBytes})
	return ioutil.WriteFile(filename, privKeyPEM, 0644)
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// SetupCA creates or loads a CA cert+key pair.
func SetupCA(caCertFilename, caPrivateKeyFilename string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	var caPrivKey *ecdsa.PrivateKey
	var caTemplate *x509.Certificate

	if fileExists(caCertFilename) && fileExists(caPrivateKeyFilename) {
		fmt.Println("CA already exists on disk:", caCertFilename, "and", caPrivateKeyFilename)
		return nil, nil, nil
	}

	fmt.Println("Generating new CA certificate and private key...")

	caPrivKey, _ = GenerateKeyPair()
	caPubKey := &caPrivKey.PublicKey

	caTemplate = &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // Valid for 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	caCertDER, err := GenerateCertificate(caTemplate, caTemplate, caPubKey, caPrivKey)
	if err != nil {
		return nil, nil, fmt.Errorf("error generating CA certificate: %v", err)
	}

	if err := SavePEM(caCertDER, caCertFilename); err != nil {
		return nil, nil, fmt.Errorf("error saving CA certificate: %v", err)
	}
	fmt.Println("CA Certificate generated and saved as", caCertFilename)

	if err := SavePrivateKey(caPrivKey, caPrivateKeyFilename); err != nil {
		return nil, nil, fmt.Errorf("error saving CA private key: %v", err)
	}
	fmt.Println("CA Private Key generated and saved as", caPrivateKeyFilename)

	return caTemplate, caPrivKey, nil
}

func LoadPEM(filename string) ([]byte, error) {
	pemData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	return block.Bytes, nil
}

// CheckCertificate verifies a certificate against the local CA cert and extracts attributes.
func CheckCertificate(certificate []byte) (bool, string, string) {
	caCertDER, err := LoadPEM("src/certauth/pubkey/ca_cert.pem")
	if err != nil {
		fmt.Println("Error loading CA certificate:", err)
		return false, "", ""
	}
	parsedCaCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		fmt.Println("Error parsing CA certificate:", err)
		return false, "", ""
	}

	parsedDoCert, err := x509.ParseCertificate(certificate)
	if err != nil {
		fmt.Println("Error parsing DO certificate:", err)
		return false, "", ""
	}

	if err = parsedDoCert.CheckSignatureFrom(parsedCaCert); err != nil {
		fmt.Println("Certificate verification failed:", err)
		return false, "", ""
	}
	// fmt.Println("Certificate successfully verified by TEE")

	var attributes []string
	for _, ext := range parsedDoCert.Extensions {
		if ext.Id.Equal(attributesOID) {
			if _, err := asn1.Unmarshal(ext.Value, &attributes); err != nil {
				fmt.Println("Error unmarshaling attributes:", err)
				return false, "", ""
			}
		}
	}
	return true, "", fmt.Sprintf("%v", attributes)
}

// GenerateClientCertificate generates a client cert signed by the CA, using the client's public key.
func GenerateClientCertificate(userPublicKeyFile, blockchainAddress, attributesStr, caCertFilename, caPrivateKeyFilename string) ([]byte, string, error) {
	if userPublicKeyFile == "" {
		return nil, "", fmt.Errorf("certificate public key path is required (-certificate)")
	}
	if attributesStr == "" {
		return nil, "", fmt.Errorf("attributes list is required (-attributes)")
	}

	// Load client public key
	doPubKey, err := LoadPublicKey(userPublicKeyFile)
	if err != nil {
		return nil, "", fmt.Errorf("error loading client public key: %v", err)
	}

	// Load CA cert
	caCertPEM, err := ioutil.ReadFile(caCertFilename)
	if err != nil {
		return nil, "", fmt.Errorf("error loading CA certificate: %v", err)
	}
	block, _ := pem.Decode(caCertPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, "", fmt.Errorf("failed to decode PEM block containing CA certificate")
	}
	caTemplate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing CA certificate: %v", err)
	}

	// Load CA private key
	caPrivKeyPEM, err := ioutil.ReadFile(caPrivateKeyFilename)
	if err != nil {
		return nil, "", fmt.Errorf("error loading CA private key: %v", err)
	}
	block, _ = pem.Decode(caPrivKeyPEM)
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return nil, "", fmt.Errorf("failed to decode PEM block containing CA private key")
	}
	caPrivKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing CA private key: %v", err)
	}

	// Marshal optional blockchain address + required attributes
	var blockchainAddressValue, attributesValue []byte

	if blockchainAddress != "" {
		blockchainAddressValue, err = asn1.Marshal(blockchainAddress)
		if err != nil {
			return nil, "", fmt.Errorf("error marshaling blockchain address: %v", err)
		}
	}

	attributes := []string{attributesStr}
	attributesValue, err = asn1.Marshal(attributes)
	if err != nil {
		return nil, "", fmt.Errorf("error marshaling attributes: %v", err)
	}

	// Create client certificate template
	doTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "DO",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtraExtensions: []pkix.Extension{
			{Id: blockchainAddressOID, Value: blockchainAddressValue},
			{Id: attributesOID, Value: attributesValue},
		},
	}

	// Sign client certificate with CA
	doCertDER, err := GenerateCertificate(doTemplate, caTemplate, doPubKey, caPrivKey)
	if err != nil {
		return nil, "", fmt.Errorf("error generating client certificate: %v", err)
	}

	// Save as certificate/user_cert.pem in the parent folder of the public-key directory
	doPublicKeyDir := filepath.Dir(userPublicKeyFile)           // .../someDirContainingPublicKey
	parentDir := filepath.Dir(doPublicKeyDir)                   // one level up
	certDir := filepath.Join(parentDir, "certificate")          // .../parent/certificate
	doCertPath := filepath.Join(certDir, "user_cert.pem")       // .../parent/certificate/user_cert.pem

	// Ensure the directory exists
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return nil, "", fmt.Errorf("error creating certificate directory: %v", err)
	}

	if err := SavePEM(doCertDER, doCertPath); err != nil {
		return nil, "", fmt.Errorf("error saving client certificate: %v", err)
	}

	return doCertDER, doCertPath, nil
}

func main() {
	// New flags to split operations
	startCA := flag.Bool("start_ca", false, "initialize/load the CA certificate and private key only")
	genClientCert := flag.Bool("gen_client_cert", false, "generate a client certificate signed by the CA")

	// Existing flags (used for client cert generation)
	userPublicKeyFile := flag.String("certificate", "", "path to client public key file (PEM)")
	blockchainAddressFlag := flag.String("blockchain_address", "", "blockchain address (optional)")
	attributesFlag := flag.String("attributes", "", "attributes (required for client cert generation)")

	// Filenames
	caCertFilename := "src/certauth/pubkey/ca_cert.pem"
	caPrivateKeyFilename := "src/certauth/privkey/ca_cert_private_key.pem"

	flag.Parse()

	// Require exactly one mode (or at least one)
	if !*startCA && !*genClientCert {
		fmt.Println("Error: choose one mode: -start_ca or -gen_client_cert")
		flag.Usage()
		os.Exit(1)
	}
	if *startCA && *genClientCert {
		fmt.Println("Error: choose only one mode at a time: -start_ca OR -gen_client_cert")
		flag.Usage()
		os.Exit(1)
	}

	// Mode 1: only initialize/load CA
	if *startCA {
		_, _, err := SetupCA(caCertFilename, caPrivateKeyFilename)
		if err != nil {
			fmt.Println("Error setting up CA:", err)
			os.Exit(1)
		}
		fmt.Println("CA is ready.")
		return
	}

	// Mode 2: generate client certificate
	if *genClientCert {
		if *userPublicKeyFile == "" {
			fmt.Println("Error: -certificate is required for -gen_client_cert")
			os.Exit(1)
		}
		if *attributesFlag == "" {
			fmt.Println("Error: -attributes is required for -gen_client_cert")
			os.Exit(1)
		}

		doCertDER, doCertPath, err := GenerateClientCertificate(
			*userPublicKeyFile,
			*blockchainAddressFlag,
			*attributesFlag,
			caCertFilename,
			caPrivateKeyFilename,
		)
		if err != nil {
			fmt.Println("Error generating client certificate:", err)
			os.Exit(1)
		}
		fmt.Println("Client certificate generated and saved as", doCertPath)

		// Optional verification print (kept from your original behavior)
		// success, blockchainAddress, attrs := CheckCertificate(doCertDER)
		success, _, attrs := CheckCertificate(doCertDER)
		if success {
			// fmt.Println("Blockchain Address:", blockchainAddress)
			fmt.Println("Attributes:", attrs)
		} else {
			fmt.Println("Failed to verify or retrieve information from the certificate.")
		}
		return
	}
}