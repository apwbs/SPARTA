package teeserver_requester

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sparta/src/utils/attestation"
	"sparta/src/utils/https"
)

func VerifyTEE(expectedMeasurement string, SKIPATTESTATION bool) bool {
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

	return true
}
