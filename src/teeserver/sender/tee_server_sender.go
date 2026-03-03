package teeserver_sender

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"mime/multipart"
	"net/http"
	"sparta/src/utils/attestation"
	"sparta/src/utils/https"
	seedGeneration "sparta/src/utils/seedGenerator"
)

func SendSeed(expectedMeasurement string, SKIPATTESTATION bool) {
	var certBytes []byte
	if !SKIPATTESTATION {
		fmt.Println("Doing remote attestation")
		certBytes, _ = attestation.RemoteAttestation("https://localhost:8078", []byte(expectedMeasurement), "../tee/pubkey/publicKey.pem")
	} else {
		fmt.Println("Skipping remote attestation")
		tlsConfig := &tls.Config{InsecureSkipVerify: true}
		certBytes = https.HttpGet(tlsConfig, "https://localhost:8078/cert")
	}

	tlsConfig := &tls.Config{RootCAs: x509.NewCertPool(), ServerName: "localhost"}
	cert, _ := x509.ParseCertificate(certBytes)
	tlsConfig.RootCAs.AddCert(cert)

	seed, err := seedGeneration.GenerateSeed()
	if err != nil {
		if err.Error() == "seed already exists" {
			fmt.Println("Seed already exists. No new seed was generated.")
		} else {
			fmt.Printf("An error occurred: %v\n", err)
		}
		return
	}

	// Prepare the multipart form data
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add the seed field
	err = writer.WriteField("seed", seed)
	if err != nil {
		fmt.Println("Error adding field to multipart writer:", err)
		return
	}

	// Close the writer to finalize the form
	err = writer.Close()
	if err != nil {
		fmt.Println("Error closing multipart writer:", err)
		return
	}

	// Set up the POST request
	req, err := http.NewRequest("POST", "https://localhost:8078/secret", &requestBody)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "Go HTTP client")

	// Send the request
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making HTTP request:", err)
		return
	}
	defer resp.Body.Close()

	// Handle the response
	fmt.Printf("Response Status: %s\n", resp.Status)
}
