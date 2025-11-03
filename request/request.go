package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

func main() {
	// --- 1. Define and parse command-line flags ---
	// This allows you to easily change the URL and request count from the terminal.
	// Example: go run main.go -n=50 -url="https://api.example.com"
	url := flag.String("url", "http://localhost:8080", "The URL to request")
	keepalive := flag.Bool("keepalive", false, "Whether to enable keepalive in http connections")
	numRequests := flag.Int("n", 10, "Number of parallel requests to make")
	ms := flag.Int("ms", 2000, "Ms")
	duration := time.Duration(*ms) * time.Millisecond
	flag.Parse()

	fmt.Printf("Starting %d parallel requests to %s...\n", *numRequests, *url)

	// --- 2. Create a reusable HTTP client ---
	// It's much more efficient to create one client with a custom transport
	// than using http.Get() in a loop (which uses the DefaultClient).
	// This allows for better connection pooling and control.
	client := &http.Client{
		Transport: &http.Transport{
			// Set pool size to be at least the number of requests
			MaxIdleConns:    *numRequests,
			MaxConnsPerHost: *numRequests,
			// A reasonable timeout for idle connections
			IdleConnTimeout: 30 * time.Second,
			DisableKeepAlives: !*keepalive,
		},
			// A total timeout for each request
		Timeout: duration,
	}

	// --- 3. Start the infinite loop ---
	// This loop will continuously run batches of parallel requests.
	fmt.Println("Starting request loop. Press Ctrl+C to stop.")
	batchNumber := 1
	for {
		fmt.Printf("\n--- Starting Batch %d ---\n", batchNumber)

		// --- 4. Use a WaitGroup (re-created for each batch) ---
		// A WaitGroup is used to wait for a collection of goroutines to finish.
		// The main goroutine calls Add to set the number of goroutines to wait for.
		// Each goroutine calls Done when it finishes.
		//
		var wg sync.WaitGroup

		start := time.Now()

		// --- 5. Launch Goroutines for the batch ---
		for i := 0; i < *numRequests; i++ {
			// Add 1 to the WaitGroup counter for each goroutine we're about to start.
			// It's important to do this *before* launching the goroutine.
			wg.Add(1)

			// Launch a new goroutine (a lightweight thread)
			// We pass 'i+1' as an ID for logging purposes.
			go makeRequest(client, *url, i+1, &wg)
		}

		// --- 6. Wait for all requests in the batch ---
		fmt.Println("Waiting for all requests in this batch to complete...")
		// wg.Wait() blocks the main goroutine until the WaitGroup counter is zero.
		wg.Wait()

		duration := time.Since(start)
		fmt.Printf("Batch %d: All %d requests completed in %v\n", batchNumber, *numRequests, duration)

		batchNumber++
	}
}

// makeRequest performs a single HTTP GET request and signals to the WaitGroup
// when it's complete.
func makeRequest(client *http.Client, url string, id int, wg *sync.WaitGroup) {
	// Defer wg.Done() to ensure it's called when this function exits,
	// no matter what (even if it panics or returns early on an error).
	defer wg.Done()

	log.Printf("[Request %d] Starting...\n", id)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("x-mgc-test-id", uuid.New().String())
	// Perform the HTTP GET request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Request %d] ERROR: %v\n", id, err)
		return
	}

	// Defer closing the response body.
	// This is crucial to prevent resource (connection) leaks.
	defer resp.Body.Close()

	// We must read and discard the response body to allow the
	// underlying connection to be reused. io.Copy to io.Discard
	// is the most efficient way to do this.
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		log.Printf("[Request %d] ERROR reading body: %v\n", id, err)
		return
	}

	log.Printf("[Request %d] Finished with status: %s\n", id, resp.Status)
}
