package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
)

// GenerateKeyPair generates a public/private key pair using ECDSA.
func GenerateKeyPair() (*ecdsa.PrivateKey, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

// SavePEM saves the given public key to a PEM file.
func SavePEM(pubKey *ecdsa.PublicKey, filename string) error {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return err
	}
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes})
	return os.WriteFile(filename, pubKeyPEM, 0644)
}

func SavePrivateKeyPEM(privKey *ecdsa.PrivateKey, filename string) error {
	privKeyBytes, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return err
	}
	privKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privKeyBytes})
	return os.WriteFile(filename, privKeyPEM, 0600) // Use restrictive permissions
}

func main() {
	output_Path_File := flag.String("output_path_file", "", "path where to save the public key file")
	output_Path_File_private_key := flag.String("output_path_file_private_key", "", "path where to save the private key file")

	flag.Parse()

	if *output_Path_File == "" {
		fmt.Println("Error: Output path to store User PublicKeyFile is required.")
		os.Exit(1)
	}

	if *output_Path_File_private_key == "" {
		fmt.Println("Error: Output path to store User PrivateKeyFile is required.")
		os.Exit(1)
	}

	// Step 1: Generate key pair for DO
	userPrivKey, err := GenerateKeyPair()
	if err != nil {
		fmt.Println("Error generating USER key pair:", err)
		return
	}
	userPubKey := &userPrivKey.PublicKey

	// Step 2: Save the DO public key to a file
	err = SavePEM(userPubKey, *output_Path_File)
	if err != nil {
		fmt.Println("Error saving USER public key:", err)
		return
	}
	fmt.Println("Public Key generated and saved")

	err = SavePrivateKeyPEM(userPrivKey, *output_Path_File_private_key)
	if err != nil {
		fmt.Println("Error saving USER private key:", err)
		return
	}
	fmt.Println("Private Key generated and saved")

}
