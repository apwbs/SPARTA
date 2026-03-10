package blockchain

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"encoding/base64"
	"os"
	"sort"
	"strings"
	"time"

	"sparta/src/utils/contract"

	"github.com/joho/godotenv"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// -----------------------------------------------------------------------------
// ENV helpers
// -----------------------------------------------------------------------------

func loadEnv() {
	// Safe if missing; keeps existing environment variables.
	_ = godotenv.Load()
}

func mustGetEnv(key string) (string, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return "", fmt.Errorf("missing %s in environment/.env", key)
	}
	return v, nil
}

func getPrivateKeyNo0x() (string, error) {
	pk, err := mustGetEnv("CCU_BLOCKCHAIN_PRIVATE_KEY")
	if err != nil {
		return "", err
	}
	pk = strings.TrimPrefix(pk, "0x")
	pk = strings.TrimPrefix(pk, "0X")
	return pk, nil
}

// IPNS_KEY_PATIENT -> Patient
// IPNS_KEY_PATIENT_LIGHT -> PatientLight
func envVarToKeyName(envVar string) (string, error) {
	const prefix = "IPNS_KEY_"
	if !strings.HasPrefix(envVar, prefix) {
		return "", fmt.Errorf("invalid IPNS env var (missing %s): %s", prefix, envVar)
	}

	suffix := strings.TrimPrefix(envVar, prefix) // e.g., PATIENT_LIGHT
	parts := strings.Split(strings.ToLower(suffix), "_")

	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, ""), nil
}

// -----------------------------------------------------------------------------
// Ethereum/contract wiring
// -----------------------------------------------------------------------------

func ExecuteTransaction() (contractInstance *contract.Contract, client *ethclient.Client, err error) {
	loadEnv()

	ethereumNodeURL, err := mustGetEnv("ETHEREUM_NODE_URL")
	if err != nil {
		return nil, nil, err
	}
	contractAddress, err := mustGetEnv("CONTRACT_ADDRESS_SPARTA")
	if err != nil {
		return nil, nil, err
	}

	caCertPath := strings.TrimSpace(os.Getenv("CA_CERT_PATH"))
	if caCertPath == "" {
		caCertPath = "../certauth/pubkey/ca_cert.pem"
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

func hexToECDSA(privateKeyNo0x string) (*ecdsa.PrivateKey, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyNo0x)
	if err != nil {
		return nil, err
	}
	return crypto.ToECDSA(privateKeyBytes)
}

// -----------------------------------------------------------------------------
// IPNS key storage (Set)
// -----------------------------------------------------------------------------

// SetAllIPNSKeys scans environment/.env for all non-empty variables that start with IPNS_KEY_
// and stores each one on-chain using a keyName derived from the variable name.
func SetAllIPNSKeys() error {
	loadEnv()

	// Collect all IPNS_KEY_* vars that have a non-empty value
	var vars []string
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := parts[0]
		if !strings.HasPrefix(k, "IPNS_KEY_") {
			continue
		}
		if strings.TrimSpace(os.Getenv(k)) == "" {
			continue
		}
		vars = append(vars, k)
	}

	sort.Strings(vars)

	if len(vars) == 0 {
		return fmt.Errorf("no non-empty IPNS_KEY_* entries found in environment/.env")
	}

	for _, v := range vars {
		if err := SetIPNSKeyFromEnvVar(v); err != nil {
			return err
		}
	}

	return nil
}

// SetIPNSKeyFromEnvVar reads ipnsKey from envVar (e.g., IPNS_KEY_PATIENT_LIGHT),
// derives keyName (PatientLight), and stores it on-chain.
func SetIPNSKeyFromEnvVar(envVar string) error {
	loadEnv()

	contractInstance, client, err := ExecuteTransaction()
	if err != nil {
		return err
	}

	// Private key
	pk, err := getPrivateKeyNo0x()
	if err != nil {
		return err
	}

	// IPNS key value
	ipnsKey := strings.TrimSpace(os.Getenv(envVar))
	if ipnsKey == "" {
		return fmt.Errorf("%s not set in environment/.env", envVar)
	}

	// Derive on-chain keyName from env var name
	keyName, err := envVarToKeyName(envVar)
	if err != nil {
		return err
	}

	// bytes32 keyName
	var keyNameBytes [32]byte
	copy(keyNameBytes[:], []byte(keyName))

	// bytes32 halves of ipnsKey (raw ASCII/UTF-8), max 64 bytes
	firstHalfBytes, secondHalfBytes, err := splitStringTo2xBytes32(ipnsKey)
	if err != nil {
		return err
	}

	privateKeyECDSA, err := hexToECDSA(pk)
	if err != nil {
		return fmt.Errorf("error converting private key to ECDSA: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKeyECDSA, big.NewInt(1337))
	if err != nil {
		return fmt.Errorf("error creating transaction auth: %v", err)
	}

	fmt.Printf("[blockchain] Storing keyName=%s envVar=%s\n", keyName, envVar)

	tx, err := contractInstance.SetIPNSKey(auth, keyNameBytes, firstHalfBytes, secondHalfBytes)
	if err != nil {
		return fmt.Errorf("error executing transaction: %v", err)
	}

	fmt.Printf("[blockchain] tx submitted: %s\n", tx.Hash().Hex())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return fmt.Errorf("error waiting for transaction to be mined: %v", err)
	}

	fmt.Printf("[blockchain] tx mined: %s  status=%d  block=%d  gasUsed=%d\n",
		receipt.TxHash.Hex(), receipt.Status, receipt.BlockNumber.Uint64(), receipt.GasUsed)

	fmt.Printf("[blockchain] Stored %s (%s) on-chain\n", keyName, envVar)
	return nil
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

// -----------------------------------------------------------------------------
// IPNS key retrieval (Get)
// -----------------------------------------------------------------------------

// GetIPNSKey returns the stored IPNS key string for a given keyName (e.g., "PatientLight").
// Solidity returns 64 bytes (p1||p2) padded with zeros; we trim trailing zeros and return string.
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

	plain := bytes.TrimRight(bytesData, "\x00")
	return string(plain), nil
}

// -----------------------------------------------------------------------------
// Utilities
// -----------------------------------------------------------------------------

func splitStringTo2xBytes32(s string) (a [32]byte, b [32]byte, err error) {
	raw := []byte(s)
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