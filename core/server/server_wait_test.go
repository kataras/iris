package server

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestServerWait tests the Server's Wait method.
func TestServerWait(t *testing.T) {
	// Create a ServerOptions instance with the desired settings.
	options := ServerOptions{
		Network: "tcp",
		Addr:    ":http",
	}

	if runsInCI {
		options.Addr = ":8080"
	}

	// Create a new Server instance with the options.
	server := NewServer(options)
	// Define a simple handler that responds with 200 OK.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Start the server in a separate goroutine since it is blocking.
	go func() {
		if err := server.ListenAndServe(handler); err != nil {
			t.Errorf("ListenAndServe() failed: %v", err)
		}
	}()

	// Test the Wait method.
	t.Run("Wait for server to start", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Call the Wait method to block until the server is up and running.
		if err := server.Wait(ctx); err != nil {
			t.Fatalf("Wait() failed: %v", err)
		}

		// Make a request to the test server to ensure it's responding.
		resp, err := http.Get(server.String())
		if err != nil {
			t.Fatalf("Could not make GET request after Wait: %v", err)
		}
		defer resp.Body.Close()

		// Check if the response status code is 200 OK.
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code 200 after Wait, got %v", resp.StatusCode)
		}
	})

	// Shutdown the server gracefully.
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(ctxShutdown); err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}
}
