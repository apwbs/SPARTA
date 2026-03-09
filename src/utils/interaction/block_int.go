package blockchain

import (
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"sparta/src/utils/contract"
	"time"
	"strings"

	"github.com/joho/godotenv"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func ExecuteTransaction() (contractInstance *contract.Contract, client *ethclient.Client, err error) {
	// Load .env (safe if missing)
	_ = godotenv.Load()

	ethereumNodeURL := strings.TrimSpace(os.Getenv("ETHEREUM_NODE_URL"))
	contractAddress := strings.TrimSpace(os.Getenv("CONTRACT_ADDRESS_SPARTA"))
	caCertPath := strings.TrimSpace(os.Getenv("CA_CERT_PATH"))

	if ethereumNodeURL == "" {
		return nil, nil, fmt.Errorf("missing ETHEREUM_NODE_URL in environment/.env")
	}
	if contractAddress == "" {
		return nil, nil, fmt.Errorf("missing CONTRACT_ADDRESS_SPARTA in environment/.env")
	}
	if caCertPath == "" {
		// default if you want
		caCertPath = "src/certauth/pubkey/ca_cert.pem"
	}

	pemCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading CA cert (%s): %v", caCertPath, err)
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(pemCert); !ok {
		return nil, nil, fmt.Errorf("failed to append CA cert from PEM (%s)", caCertPath)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: certPool},
		},
	}

	rpcClient, err := rpc.DialOptions(context.Background(), ethereumNodeURL, rpc.WithHTTPClient(httpClient))
	if err != nil {
		return nil, nil, err
	}

	ethClient := ethclient.NewClient(rpcClient)

	contractInstance, err = bindContract(common.HexToAddress(contractAddress), ethClient)
	if err != nil {
		return nil, nil, err
	}

	return contractInstance, ethClient, nil
}

func bindContract(address common.Address, backend bind.ContractBackend) (*contract.Contract, error) {
	return contract.NewContract(address, backend)
}

func hexToECDSA(privateKey string) (*ecdsa.PrivateKey, error) {
	privateKeyBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, err
	}
	return crypto.ToECDSA(privateKeyBytes)
}

func SetIPNSKey(keyName, ipnsKey string) error {
	contractInstance, client, _ := ExecuteTransaction()

	privateKey := "70fe281af5b213e0926dc1d25a80686f3b672370845a16d5e5a072ca611ed3ad"

	// keyName -> bytes32 (as you already do elsewhere)
	var keyNameBytes [32]byte
	copy(keyNameBytes[:], []byte(keyName))

	// ipnsKey -> two bytes32 WITHOUT base64
	firstHalfBytes, secondHalfBytes, err := splitStringTo2xBytes32(ipnsKey)
	if err != nil {
		return err
	}

	privateKeyECDSA, err := hexToECDSA(privateKey)
	if err != nil {
		return fmt.Errorf("error converting private key to ECDSA: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKeyECDSA, big.NewInt(1337))
	if err != nil {
		return fmt.Errorf("error creating transaction auth: %v", err)
	}

	tx, err := contractInstance.SetIPNSKey(auth, keyNameBytes, firstHalfBytes, secondHalfBytes)
	if err != nil {
		return fmt.Errorf("error executing transaction: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	_, err = bind.WaitMined(ctx, client, tx)
	if err != nil {
		return fmt.Errorf("error waiting for transaction to be mined: %v", err)
	}

	return nil
}

func GetIPNSKey(keyName string) (string, error) {
	contractInstance, _, err := ExecuteTransaction()
	if err != nil {
		return "", err
	}

	callOpts := &bind.CallOpts{}

	var keyNameBytes [32]byte
	copy(keyNameBytes[:], []byte(keyName))

	_, bytesData, err := contractInstance.GetIPNSKey(callOpts, keyNameBytes)
	if err != nil {
		return "", err
	}

	// joined is 64 bytes, right-padded with zeros
	plain := bytes.TrimRight(bytesData, "\x00")
	return string(plain), nil
}

func splitStringTo2xBytes32(s string) (a [32]byte, b [32]byte, err error) {
	raw := []byte(s) // ASCII/UTF-8
	if len(raw) > 64 {
		return a, b, fmt.Errorf("ipnsKey too long: %d bytes (max 64)", len(raw))
	}
	copy(a[:], raw[:min(32, len(raw))])
	if len(raw) > 32 {
		copy(b[:], raw[32:])
	}
	return a, b, nil
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func GetDocument(functionName string, messageID int64) (string, error) {
	contractInstance, _, _ := ExecuteTransaction()
	callOpts := &bind.CallOpts{}

	var functionNameBytes [32]byte
	copy(functionNameBytes[:], functionName)

	_, bytesData, err := contractInstance.GetIPNSKey(callOpts, functionNameBytes)
	if err != nil {
		return "", err
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(string(bytesData))
	if err != nil {
		return "", err
	}

	originalString := string(decodedBytes)

	return originalString, nil
}