package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client

const (
	requestQueue   = "request_queue"
	responseQueue  = "response_queue"
	middlewareAddr = ":9000"
)

func init() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}

type ClientPayload struct {
	FunctionName    string `json:"function_name"`
	Certificate     string `json:"certificate"`
	EncryptedData   string `json:"encrypted_data"`
	FileExtension   string `json:"file_extension"`
	IPNSKey         string `json:"ipns_key"`
	ClientPublicKey string `json:"client_public_key"`
	Signature       string `json:"signature"`
}

type QueuePayload struct {
	ClientID string        `json:"client_id"`
	Data     ClientPayload `json:"data"`
}

// Handle client requests: receive, enqueue, and poll for responses
func handleClient(conn net.Conn, wg *sync.WaitGroup) {
	defer conn.Close()
	defer wg.Done()

	clientID := conn.RemoteAddr().String()
	fmt.Printf("[Middleware] Handling client %s...\n", clientID)

	// Read the length prefix
	lengthPrefix := make([]byte, 8)
	_, err := conn.Read(lengthPrefix)
	if err != nil {
		fmt.Printf("[Middleware] Error reading length prefix: %v\n", err)
		return
	}
	payloadLength := binary.BigEndian.Uint64(lengthPrefix)

	// Read the payload bytes
	var dataBytes []byte
	tempBuffer := make([]byte, 8192)
	for uint64(len(dataBytes)) < payloadLength {
		n, err := conn.Read(tempBuffer)
		if n > 0 {
			dataBytes = append(dataBytes, tempBuffer[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("[Middleware] Error reading data: %v\n", err)
			return
		}
	}

	fmt.Printf("[Middleware] Payload size received: %d bytes\n", len(dataBytes))

	// Parse raw client payload into ClientPayload struct
	var clientPayload ClientPayload
	if err := json.Unmarshal(dataBytes, &clientPayload); err != nil {
		fmt.Printf("[Middleware] Error parsing client payload: %v\n", err)
		return
	}

	// Wrap into QueuePayload with client ID
	payloadStruct := QueuePayload{
		ClientID: clientID,
		Data:     clientPayload,
	}

	// Push to Redis
	payload, _ := json.Marshal(payloadStruct)
	err = redisClient.LPush(redisClient.Context(), requestQueue, payload).Err()
	if err != nil {
		fmt.Printf("[Middleware] Error enqueuing request: %v\n", err)
		conn.Write([]byte("Failed to enqueue request."))
		return
	}

	conn.Write([]byte("Request received and queued."))
	fmt.Printf("[Middleware] Polling for response for %s...\n", clientID)

	// Poll response from Redis
	for {
		res, err := redisClient.BRPop(redisClient.Context(), 30*time.Second, responseQueue).Result()
		if err != nil {
			fmt.Printf("[Middleware] Timeout or error while waiting for response: %v\n", err)
			conn.Write([]byte("Timeout waiting for response."))
			return
		}

		var response map[string]string
		err = json.Unmarshal([]byte(res[1]), &response)
		if err != nil {
			fmt.Printf("[Middleware] Error parsing response JSON: %v\n", err)
			continue
		}

		if response["client_id"] == clientID {
			_, writeErr := conn.Write([]byte(response["response"]))
			if writeErr != nil {
				fmt.Printf("[Middleware] Error sending response to client %s: %v\n", clientID, writeErr)
			} else {
				fmt.Printf("[Middleware] Response sent to %s successfully.\n", clientID)
			}
			fmt.Println("----------------------------------------------------------------------------------------------------------------------------")
			return
		}

		fmt.Printf("[Middleware] Response mismatch. Expected: %s, Got: %s\n", clientID, response["client_id"])
	}
}

// Start the middleware server
func startMiddleware() {
	listener, err := net.Listen("tcp", middlewareAddr)
	if err != nil {
		fmt.Printf("[Middleware] Error starting server: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("[Middleware] Listening on %s\n", middlewareAddr)
	fmt.Println("----------------------------------------------------------------------------------------------------------------------------")

	var wg sync.WaitGroup

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("[Middleware] Connection error: %v\n", err)
			continue
		}

		// Handle the client request and response in a single goroutine
		wg.Add(1)
		go handleClient(conn, &wg)
	}
}

func main() {
	startMiddleware()
}
