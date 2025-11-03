package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"server/metrics"
)

var (
	mockResponseBytes []byte
)

func init() {
	// init() runs once when the program starts.
	// We create our mock response map
	mockResp := map[string]string{
		"status":  "ok",
		"message": "This is a fast mock response!",
	}

	// We marshal it to JSON *once*
	var err error
	mockResponseBytes, err = json.Marshal(mockResp)
	if err != nil {
		// If this fails, we can't run the server.
		log.Fatalf("Fatal Error: Failed to marshal mock response: %v", err)
	}
}

// mockHandler is our high-performance request handler.
func mockHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Set the content type header
	w.Header().Set("Content-Type", "application/json")

	// 2. Write the status code
	w.WriteHeader(http.StatusOK)

	// 3. Write the pre-computed response bytes.
	// This is the fastest way to send a static response.
	w.Write(mockResponseBytes)
}

func main() {
	// Register our fast handler for all routes
	http.Handle("/", metrics.PrometheusMiddleware(http.HandlerFunc(mockHandler), "root"))
	http.Handle("/metrics", promhttp.Handler())

	const port = ":8080"
	fmt.Printf("Starting high-performance mock server on http://localhost%s\n", port)

	// http.ListenAndServe automatically handles each request in a new goroutine,
	// so it's highly concurrent by default.
	// We use log.Fatal to crash the app if the server fails to start
	// (e.g., if the port is already in use).
	log.Fatal(http.ListenAndServe(port, nil))
}
