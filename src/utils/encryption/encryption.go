package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	seedGeneration "sparta/src/utils/seedGenerator"

	"github.com/edgelesssys/ego/enclave"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/sha3"
)

var sealer interface {
	GetUniqueSealKey() (key, keyInfo []byte, err error)
} = enclaveSealer{}

type enclaveSealer struct{}

func GenerateSymmetricKeyTest(seedBytes []byte, messageId int64) []byte {
	seed := sha3Hash(string(seedBytes), strconv.FormatInt(messageId, 10))
	key := generateSymmetricKey([]byte(seed), 32)
	return key
}

func EncryptDataTest(key []byte, body string) (string, error) {
	ciphertext, err := encryptAESGCM([]byte(body), key)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return hex.EncodeToString(ciphertext), nil
}

func DecryptDataTest(key []byte, ciphertext string) (string, error) {
	decodedCiphertext, err := hex.DecodeString(ciphertext)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	plaintext, err := decryptAESGCM(decodedCiphertext, key)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return string(plaintext), nil
}

func NewEncryptData(seedBytes []byte, messageId int64, body string) (string, error) {
	seed := sha3Hash(string(seedBytes), strconv.FormatInt(messageId, 10))
	key := generateSymmetricKey([]byte(seed), 32)
	ciphertext, err := encryptAESGCM([]byte(body), key)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return hex.EncodeToString(ciphertext), nil
}

func EncryptData(messageId int64, body string) (string, error) {
	seedBytes := seedGeneration.GetKey()
	seed := sha3Hash(string(seedBytes), strconv.FormatInt(messageId, 10))
	key := generateSymmetricKey([]byte(seed), 32)
	ciphertext, err := encryptAESGCM([]byte(body), key)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return hex.EncodeToString(ciphertext), nil
}

func NewDecryptData(seedBytes []byte, messageId int64, ciphertext string) (string, error) {
	decodedCiphertext, err := hex.DecodeString(ciphertext)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	seed := sha3Hash(string(seedBytes), strconv.FormatInt(messageId, 10))
	key := generateSymmetricKey([]byte(seed), 32)
	plaintext, err := decryptAESGCM(decodedCiphertext, key)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return string(plaintext), nil
}

func DecryptData(messageId int64, ciphertext string) (string, error) {
	decodedCiphertext, err := hex.DecodeString(ciphertext)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	seedBytes := seedGeneration.GetKey()
	seed := sha3Hash(string(seedBytes), strconv.FormatInt(messageId, 10))
	key := generateSymmetricKey([]byte(seed), 32)
	plaintext, err := decryptAESGCM(decodedCiphertext, key)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return string(plaintext), nil
}

func (enclaveSealer) GetUniqueSealKey() (key, keyInfo []byte, err error) {
	return enclave.GetUniqueSealKey()
}

func sha3Hash(inputs ...string) string {
	concatenated := ""
	for _, input := range inputs {
		concatenated += input
	}
	hash := sha3.New256()
	_, _ = hash.Write([]byte(concatenated))
	sha3 := hash.Sum(nil)

	return fmt.Sprintf("%x", sha3)
}

func generateSymmetricKey(seed []byte, keyLen int) []byte {
	// Use HKDF with SHA-256 to derive key from seed
	hkdf := hkdf.New(sha256.New, seed, nil, nil)
	key := make([]byte, keyLen)
	_, _ = io.ReadFull(hkdf, key)

	return key
}

func encryptAESGCM(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ciphertext := aesGCM.Seal(nil, nonce, plaintext, nil)
	ciphertext = append(nonce, ciphertext...) // Prepend nonce to ciphertext

	return ciphertext, nil
}

func decryptAESGCM(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func DecryptWithAESGCMClientInput(key, ciphertext []byte) ([]byte, error) {
	// Extract nonce from the ciphertext
	nonceSize := 12 // Standard nonce size for AES-GCM
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Create a new AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM mode: %w", err)
	}

	// Decrypt the ciphertext
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}
