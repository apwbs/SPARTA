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

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func ExecuteTransaction() (contractInstance *contract.Contract, client *ethclient.Client, err error) {
	ethereumNodeURL := "HTTP://127.0.0.1:7545"
	contractAddress := "0x225a676dfe1c104369F0622BEE8aFCbFD2436856"

	pemCert, err := os.ReadFile("ca/certificate.pem")

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(pemCert)
	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: certPool}}}

	ethClient, err := rpc.DialOptions(context.Background(), ethereumNodeURL, rpc.WithHTTPClient(httpClient))
	ethClient1 := ethclient.NewClient(ethClient)
	if err != nil {
		return nil, nil, err
	}

	contractInstance, _ = bindContract(common.HexToAddress(contractAddress), ethClient1)
	if err != nil {
		return nil, nil, err
	}

	return contractInstance, ethClient1, nil
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