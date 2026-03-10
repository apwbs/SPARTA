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
	fmt.Println("/caCert ricevuto qui")

	deadline := time.Now().Add(timeout)

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
	senderRole := flag.String("sender_role", "", "required with -exchange_seed: ccu1 or ccu2")
	doBlockchain := flag.Bool("blockchain", false, "call blockchain.SetAllIPNSKeys and exit")

	flag.Parse()

	if *doBlockchain {
		if err := blockchain.SetAllIPNSKeys(); err != nil {
			fmt.Println("[blockchain] Error:", err)
			os.Exit(1)
		}
		fmt.Println("[blockchain] Done.")
		return
	}

	if *exchangeSeed {
		if *measurement == "" {
			fmt.Println("Error: -measurement is required when using -exchange_seed")
			os.Exit(1)
		}
		if *senderRole == "" {
			fmt.Println("Error: -sender_role is required when using -exchange_seed")
			os.Exit(1)
		}
		if *senderRole != "ccu1" && *senderRole != "ccu2" {
			fmt.Println("Error: invalid -sender_role. Use ccu1 or ccu2.")
			os.Exit(1)
		}
		fmt.Printf("BOOTSTRAP: exchange_seed enabled (sender_role=%s)\n", *senderRole)
	} else {
		fmt.Println("NORMAL: starting server and waiting for requests.")
	}

	// In teeserver, sender means global role 1.
	isSender := *exchangeSeed && *senderRole == "ccu1"

	receiver := teeServerReceiver.StartTEE(*exchangeSeed)

	done := make(chan struct{})
	go func() {
		defer close(done)

		var err error
		if *exchangeSeed {
			err = receiver.Start(*measurement)
		} else {
			err = receiver.StartNoMeasurement()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("Server error:", err)
		}
	}()

	if *exchangeSeed {
		if isSender {
			fmt.Println("BOOTSTRAP: teeserver is SENDER -> attesting tee and sending seed")

			peerCACertURL := "https://localhost:8078/caCert"
			if err := waitForPeer(peerCACertURL, 20*time.Second); err != nil {
				fmt.Println("Error:", err)
				receiver.Stop()
				<-done
				os.Exit(1)
			}

			teeServerSender.SendSeed(*measurement, false)

			fmt.Println("BOOTSTRAP complete (sender): stopping.")
			receiver.Stop()
			<-done
			return
		}

		fmt.Println("BOOTSTRAP: teeserver is RECEIVER -> waiting for seed from tee")
		<-done
		fmt.Println("BOOTSTRAP complete (receiver): stopping.")
		return
	}

	select {}
}