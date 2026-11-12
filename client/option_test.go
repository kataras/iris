package client

import (
	"context"
	"net/http"
	"testing"
)

// TestHandlerOptionWiresTheHandler is the regression test for the bug where
// the Handler option dropped its argument, leaving the transport's handler
// nil and panicking on the first request.
func TestHandlerOptionWiresTheHandler(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`"pong"`))
	})

	c := New(Handler(mux))

	out, err := c.BindJSON[string](context.Background(), http.MethodGet, "http://in.memory/ping", nil)
	if err != nil {
		t.Fatalf("request through the handler transport failed: %v", err)
	}
	if out != "pong" {
		t.Fatalf("got %q, want %q", out, "pong")
	}
}
