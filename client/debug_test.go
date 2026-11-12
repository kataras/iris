package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type recordingLogger struct {
	mu    sync.Mutex
	lines []string
}

func (l *recordingLogger) Debugf(format string, args ...any) {
	l.mu.Lock()
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
	l.mu.Unlock()
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial refused")
}

// TestDebugNilLoggerDoesNotPanic: Debug(nil) falls back to golog, on a normal
// request and on a transport error, where the old implementation dereferenced
// a nil response.
func TestDebugNilLoggerDoesNotPanic(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"pong"`))
	})

	c := New(Handler(mux), Debug(nil))
	if _, err := c.BindJSON[string](context.Background(), http.MethodGet, "http://in.memory/ping", nil); err != nil {
		t.Fatal(err)
	}

	failing := New(Transport(failingTransport{}), Debug(nil))
	if _, err := failing.BindJSON[string](context.Background(), http.MethodGet, "http://in.memory/ping", nil); err == nil {
		t.Fatal("expected the transport error")
	}
}

// TestDebugCustomLoggerReceivesBothDumps.
func TestDebugCustomLoggerReceivesBothDumps(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"pong"`))
	})

	logger := &recordingLogger{}
	c := New(Handler(mux), Debug(logger))
	if _, err := c.BindJSON[string](context.Background(), http.MethodGet, "http://in.memory/ping", nil); err != nil {
		t.Fatal(err)
	}

	joined := strings.Join(logger.lines, "\n")
	if !strings.Contains(joined, "GET /ping") || !strings.Contains(joined, "pong") {
		t.Fatalf("expected the request and the response dumps, got:\n%s", joined)
	}
}
