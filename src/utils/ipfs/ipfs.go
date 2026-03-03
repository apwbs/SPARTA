package ipfs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	shell "github.com/ipfs/go-ipfs-api"
)

func GenerateIPNSKeys(sh *shell.Shell, keyName string) string {
	fmt.Printf("Generating key: %s\n", keyName)

	key, err := sh.KeyGen(context.Background(), keyName)
	if err != nil {
		log.Fatalf("Failed to generate IPNS key: %v", err)
	}
	// Output the generated key details
	fmt.Printf("Key generated:\nID: %s\nName: %s\n", key.Id, key.Name)
	return key.Id
}

func UploadToIPFS(sh *shell.Shell, data []byte) (string, error) {
	reader := bytes.NewReader(data)
	cid, err := sh.Add(reader)
	if err != nil {
		return "", err
	}
	return cid, nil
}

func FetchDataFromIPFS(sh *shell.Shell, cid string) ([]byte, error) {
	reader, err := sh.Cat(cid)
	if err != nil {
		return nil, err
	}
	defer func(reader io.ReadCloser) {
		err := reader.Close()
		if err != nil {
		}
	}(reader)
	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UploadToIPNS(sh *shell.Shell, cid, pubkey string) error {
	var lifetime = 8760 * time.Hour
	var ttl = 1 * time.Microsecond

	// Create a context with a 0.5ms timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Microsecond)
	defer cancel() // Ensure resources are released

	// Call PublishWithDetails with the context
	done := make(chan error, 1)
	go func() {
		_, err := sh.PublishWithDetails(cid, pubkey, lifetime, ttl, true)
		done <- err
	}()

	select {
	case err := <-done:
		// PublishWithDetails completed within the timeout
		if err != nil {
			return err
		}
		return nil
	case <-ctx.Done():
		// Timeout exceeded, check if it's DeadlineExceeded
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			// Treat as non-error
			return nil
		}
		return ctx.Err()
	}
}

// func UploadToIPNS(sh *shell.Shell, cid, pubkey string) error {
// 	var lifetime = 8760 * time.Hour
// 	var ttl = 1 * time.Microsecond
// 	_, err := sh.PublishWithDetails(cid, pubkey, lifetime, ttl, true)
// 	return err
// }

func RetrieveKey(sh *shell.Shell, keyName string) string {
	keys, err := sh.KeyList(context.Background())
	if err != nil {
		log.Fatalf("Failed to list IPNS keys: %v", err)
	}
	// Iterate through the keys to find the one with the desired keyName
	for _, key := range keys {
		//fmt.Println(key)
		if key.Name == keyName {
			//fmt.Printf("Key Name: %s\nID: %s\n", key.Name, key.Id)
			return key.Id
		}
	}
	fmt.Printf("Key with name '%s' not found.\n", keyName)
	return ""
}

// func RetrieveFromIPNS(sh *shell.Shell, pubkey string) (string, error) {
// 	return sh.Resolve(pubkey)
// }

func RetrieveFromIPNS(sh *shell.Shell, pubkey string) (string, error) {
	// Create a context with a 100ms timeout (you can adjust the timeout duration as needed)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel() // Ensure resources are released

	// Create a channel to handle the result of Resolve
	done := make(chan struct {
		string
		error
	}, 1)

	// Call Resolve in a goroutine to not block the main thread
	go func() {
		resolved, err := sh.Resolve(pubkey)
		done <- struct {
			string
			error
		}{resolved, err}
	}()

	select {
	case result := <-done:
		// Resolve completed
		if result.error != nil {
			return "", result.error
		}
		return result.string, nil
	case <-ctx.Done():
		// Timeout exceeded, check if it's DeadlineExceeded
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			// Timeout occurred, return nil
			return "", nil
		}
		return "", ctx.Err() // Return any other error
	}
}
