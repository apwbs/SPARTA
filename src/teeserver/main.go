package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	teeServerReceiver "sparta/src/teeserver/receiver"
	teeServerSender "sparta/src/teeserver/sender"
	blockchain "sparta/src/utils/interaction"
)

// waitForPeer blocks until the peer is up (we just need /caCert to respond).
func waitForPeer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	// During bootstrap you typically use insecure TLS until you pin via RA.
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 2 * time.Second,
	}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("peer not reachable at %s within %s", url, timeout)
}

func main() {
	fmt.Println("starting")

	measurement := flag.String("measurement", "", "expected measurement of the peer TEE (required only with -exchange_seed)")
	exchangeSeed := flag.Bool("exchange_seed", false, "bootstrap mode: exchange shared seed with peer")
	seedRole := flag.Int("seed_role", 1, "seed sender selector: 1=teeserver sends, 2=tee sends")

	// blockchain mode flags
	doBlockchain := flag.Bool("blockchain", false, "call blockchain.SetIPNSKey and exit")
	keyName := flag.String("key_name", "", "key name for SetIPNSKey")
	ipnsKey := flag.String("ipns_key", "", "IPNS/IPFS key string to store via SetIPNSKey")

	flag.Parse()

	// 1) BLOCKCHAIN mode: do it and exit (no measurement needed)
	if *doBlockchain {
		if *keyName == "" {
			fmt.Println("Error: -key_name is required when using -blockchain")
			os.Exit(1)
		}
		if *ipnsKey == "" {
			fmt.Println("Error: -ipns_key is required when using -blockchain")
			os.Exit(1)
		}

		fmt.Printf("[blockchain] SetIPNSKey(key_name=%s, ipns_key=%s)\n", *keyName, *ipnsKey)
		if err := blockchain.SetIPNSKey(*keyName, *ipnsKey); err != nil {
			fmt.Println("[blockchain] Error:", err)
			os.Exit(1)
		}
		fmt.Println("[blockchain] Done.")
		return
	}

	// 2) Bootstrap vs normal daemon
	if *exchangeSeed {
		// bootstrap requires measurement
		if *measurement == "" {
			fmt.Println("Error: -measurement is required when using -exchange_seed")
			os.Exit(1)
		}
		if *seedRole != 1 && *seedRole != 2 {
			fmt.Println("Error: invalid -seed_role. Use 1 (teeserver sends) or 2 (tee sends).")
			os.Exit(1)
		}
		fmt.Printf("BOOTSTRAP: exchange_seed enabled (seed_role=%d)\n", *seedRole)
	} else {
		// normal mode: no measurement required
		fmt.Println("NORMAL: starting server and waiting for requests.")
	}

	// Start receiver inside enclave (bootstrap cert if exchangeSeed=true).
	receiver := teeServerReceiver.StartTEE(*exchangeSeed)

	done := make(chan struct{})
	go func() {
		defer close(done)

		var err error
		if *exchangeSeed {
			// bootstrap: you DO want measurement checking
			err = receiver.Start(*measurement)
		} else {
			// normal daemon: start without measurement checks
			// You must implement StartNoMeasurement() in the receiver package.
			err = receiver.StartNoMeasurement()
			// If you prefer, and Start("") is acceptable in your receiver:
			// err = receiver.Start("")
		}

		// server.Close() triggers http.ErrServerClosed (normal)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("Server error:", err)
		}
	}()

	// --- BOOTSTRAP STOP LOGIC ---
	if *exchangeSeed {
		if *seedRole == 1 {
			fmt.Println("BOOTSTRAP: teeserver is SENDER -> sending seed to peer")

			peerCACertURL := "https://localhost:8078/caCert"
			if err := waitForPeer(peerCACertURL, 20*time.Second); err != nil {
				fmt.Println("Error:", err)
				receiver.Stop()
				<-done
				os.Exit(1)
			}

			teeServerSender.SendSeed(*measurement, false)

			// Sender side done: stop our server and exit.
			fmt.Println("BOOTSTRAP complete (sender): stopping.")
			receiver.Stop()

			<-done
			return
		}

		// Receiver: /secret handler will store seed and call receiver.Stop() inside enclave.
		fmt.Println("BOOTSTRAP: teeserver is RECEIVER -> waiting for seed from tee")
		<-done
		fmt.Println("BOOTSTRAP complete (receiver): stopping.")
		return
	}

	// Normal mode: keep running forever.
	select {}
}