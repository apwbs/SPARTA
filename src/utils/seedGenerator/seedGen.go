package seedGeneration

import (
	"errors"
	"fmt"
	"os"
	"sparta/src/utils/sealing"

	"github.com/theckman/go-securerandom"
)

func GenerateSeed() (string, error) {
	if _, err := os.Stat("seed/seed.txt"); err == nil {
		return "", errors.New("seed already exists")
	} else if !os.IsNotExist(err) {
		return "", err
	}

	rStr, _ := securerandom.Base64OfBytes(32)
	seed := rStr

	sealedSeed, _ := sealing.NewSealing([]byte(seed), []byte(""), true)

	err := WriteSealedSeed(sealedSeed)
	if err != nil {
		return "", err
	}

	return seed, nil
}

func WriteSealedSeed(sealedSeed []byte) error {
	err := os.WriteFile("seed/seed.txt", sealedSeed, 0644)
	if err != nil {
		fmt.Println("Errore nella scrittura del file seed:", err)
		return err
	}
	return nil
}

func GetKey() []byte {
	sealedSeed, err := os.ReadFile("./seed/seed.txt")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	seed, _ := sealing.Unseal(sealedSeed, []byte(""))
	return seed
}
