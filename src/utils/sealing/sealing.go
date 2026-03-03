package sealing

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"

	"github.com/edgelesssys/ego/enclave"
)

const keyInfoLengthLength = 4

var sealer interface {
	GetUniqueSealKey() (key, keyInfo []byte, err error)
	GetProductSealKey() (key, keyInfo []byte, err error)
	GetSealKey(keyInfo []byte) ([]byte, error)
} = enclaveSealer{}

type enclaveSealer struct{}

func (enclaveSealer) GetUniqueSealKey() (key, keyInfo []byte, err error) {
	return enclave.GetUniqueSealKey()
}

func (enclaveSealer) GetProductSealKey() (key, keyInfo []byte, err error) {
	return enclave.GetProductSealKey()
}

func (enclaveSealer) GetSealKey(keyInfo []byte) ([]byte, error) {
	return enclave.GetSealKey(keyInfo)
}

func NewSealing(plaintext []byte, additionalData []byte, useProductKey bool) ([]byte, error) {
	var sealKey, keyInfo []byte
	var err error

	if useProductKey {
		sealKey, keyInfo, err = sealer.GetProductSealKey()
	} else {
		sealKey, keyInfo, err = sealer.GetUniqueSealKey()
	}
	if err != nil {
		return nil, err
	}

	return seal(plaintext, sealKey, keyInfo, additionalData)
}

func SealWithUniqueKey(plaintext []byte, additionalData []byte) ([]byte, error) {
	sealKey, keyInfo, err := sealer.GetUniqueSealKey()
	if err != nil {
		return nil, err
	}

	return seal(plaintext, sealKey, keyInfo, additionalData)
}

func SealWithProductKey(plaintext []byte, additionalData []byte) ([]byte, error) {
	sealKey, keyInfo, err := sealer.GetProductSealKey()
	if err != nil {
		return nil, err
	}

	return seal(plaintext, sealKey, keyInfo, additionalData)
}

func Unseal(ciphertext []byte, additionalData []byte) ([]byte, error) {
	if len(ciphertext) <= keyInfoLengthLength {
		return nil, errors.New("ciphertext is too short")
	}
	keyInfoLength := binary.LittleEndian.Uint32(ciphertext)
	ciphertext = ciphertext[keyInfoLengthLength:]

	if !(0 < keyInfoLength && int(keyInfoLength) < len(ciphertext)) {
		return nil, errors.New("ciphertext contains invalid key info length")
	}
	keyInfo, ciphertext := ciphertext[:keyInfoLength], ciphertext[keyInfoLength:]

	sealKey, err := sealer.GetSealKey(keyInfo)
	if err != nil {
		return nil, err
	}

	return decrypt(ciphertext, sealKey, additionalData)
}

func seal(plaintext []byte, sealKey []byte, keyInfo []byte, additionalData []byte) ([]byte, error) {
	ciphertext, err := encrypt(plaintext, sealKey, additionalData)
	if err != nil {
		return nil, err
	}

	keyInfoLength := make([]byte, keyInfoLengthLength)
	binary.LittleEndian.PutUint32(keyInfoLength, uint32(len(keyInfo)))
	keyInfoEncoded := append(keyInfoLength, keyInfo...)

	return append(keyInfoEncoded, ciphertext...), nil
}

func encrypt(plaintext []byte, key []byte, additionalData []byte) ([]byte, error) {
	aesgcm, err := getCipher(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return aesgcm.Seal(nonce, nonce, plaintext, additionalData), nil
}

func getCipher(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func decrypt(ciphertext []byte, key []byte, additionalData []byte) ([]byte, error) {
	aesgcm, err := getCipher(key)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) <= nonceSize {
		return nil, errors.New("ciphertext is too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	return aesgcm.Open(nil, nonce, ciphertext, additionalData)
}
