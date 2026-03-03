package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	teeReceiver "sparta/src/tee/receiver"
	teeSender "sparta/src/tee/sender"
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

	measurement := flag.String("measurement", "", "expected measurement of the peer TEE")
	exchangeSeed := flag.Bool("exchange_seed", false, "bootstrap mode: exchange shared seed with peer")
	seedRole := flag.Int("seed_role", 1, "seed sender selector: 1=teeserver sends, 2=tee sends")

	flag.Parse()

	if *measurement == "" {
		fmt.Println("Error: -measurement is required.")
		os.Exit(1)
	}

	if *exchangeSeed {
		if *seedRole != 1 && *seedRole != 2 {
			fmt.Println("Error: invalid -seed_role. Use 1 (teeserver sends) or 2 (tee sends).")
			os.Exit(1)
		}
		fmt.Printf("BOOTSTRAP: exchange_seed enabled (seed_role=%d)\n", *seedRole)
	} else {
		fmt.Println("NORMAL: seed exchange disabled (expects seed already present).")
	}

	// Start receiver inside enclave (bootstrap cert if exchangeSeed=true).
	receiver := teeReceiver.StartTEE(*exchangeSeed)

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := receiver.Start(*measurement)
		// server.Close() triggers http.ErrServerClosed (normal)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("Server error:", err)
		}
	}()

	// --- BOOTSTRAP STOP LOGIC ---
	if *exchangeSeed {
		if *seedRole == 2 {
			fmt.Println("BOOTSTRAP: tee is SENDER -> sending seed to peer")

			// Peer is teeserver; wait until its /caCert endpoint is reachable.
			peerCACertURL := "https://localhost:8075/caCert"
			if err := waitForPeer(peerCACertURL, 20*time.Second); err != nil {
				fmt.Println("Error:", err)
				receiver.Stop()
				<-done
				os.Exit(1)
			}

			teeSender.SendSeed(*measurement, false)

			// Sender side done: stop our server and exit.
			fmt.Println("BOOTSTRAP complete (sender): stopping.")
			receiver.Stop()

			<-done
			return
		}

		// Receiver: /secret handler will store seed and call receiver.Stop() inside enclave.
		fmt.Println("BOOTSTRAP: tee is RECEIVER -> waiting for seed from teeserver")
		<-done
		fmt.Println("BOOTSTRAP complete (receiver): stopping.")
		return
	}

	// Normal mode: keep running forever.
	select {}
}