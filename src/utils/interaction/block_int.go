package blockchain

import (
	"context"
	"crypto/ecdsa"
	"crypto/tls"
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
	ethereumNodeURL := "HTTP://127.0.0.1:8545"
	contractAddress := "0xbCdAceB705e2E576127DcdEd134d459D44C6a343"

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

func SetDocument(messageID int64, functionName, ipfsHash string) error {
	contractInstance, client, _ := ExecuteTransaction()

	privateKey := "3429d1a2c8c923d12ae254f8ee270a51326355bbcce902d87b622daba211fcbe"

	var functionNameBytes [32]byte
	copy(functionNameBytes[:], functionName)

	base64String := base64.StdEncoding.EncodeToString([]byte(ipfsHash))
	midpoint := len(base64String) / 2
	firstHalf := base64String[:midpoint]
	secondHalf := base64String[midpoint:]

	var firstHalfBytes [32]byte
	copy(firstHalfBytes[:], firstHalf)

	var secondHalfBytes [32]byte
	copy(secondHalfBytes[:], secondHalf)

	privateKeyECDSA, err := hexToECDSA(privateKey)
	if err != nil {
		return fmt.Errorf("error converting private key to ECDSA: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKeyECDSA, big.NewInt(1337))
	if err != nil {
		return fmt.Errorf("error creating transaction auth: %v", err)
	}

	tx, err := contractInstance.SetDocument(auth, functionNameBytes, uint64(messageID), firstHalfBytes, secondHalfBytes)
	if err != nil {
		return fmt.Errorf("error executing transaction: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	receipt, err := bind.WaitMined(ctx, client, tx) // Replace 'client' with 'ethClient'
	if err != nil {
		return fmt.Errorf("error waiting for transaction to be mined: %v", err)
	}

	fmt.Println("Transaction mined:", receipt.TxHash.Hex())

	fmt.Println("----------------------------------------------------------------------------------------------------------------------------")

	return nil
}

func GetDocument(functionName string, messageID int64) (string, error) {
	contractInstance, _, _ := ExecuteTransaction()
	callOpts := &bind.CallOpts{}

	var functionNameBytes [32]byte
	copy(functionNameBytes[:], functionName)

	bytesData, err := contractInstance.GetDocument(callOpts, functionNameBytes, uint64(messageID))
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
