package server

import (
	"bytes"
	"io"
	"testing"
)

func TestServerLogWriter(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		ignoreLogs []string
		want       string
		wantErr    bool
	}{
		{
			name:       "No logs ignored",
			input:      "Info: server started",
			ignoreLogs: nil,
			want:       "[HTTP Server] Info: server started\n",
			wantErr:    false,
		},
		{
			name:       "Ignore specific log",
			input:      "Debug: verbose log",
			ignoreLogs: []string{"Debug: verbose log"},
			want:       "",
			wantErr:    false,
		},
		{
			name:       "Ignore multiple logs",
			input:      "Trace: very detailed log",
			ignoreLogs: []string{"Debug: verbose log", "Trace: very detailed log"},
			want:       "",
			wantErr:    false,
		},
		{
			name:       "Error writing to log",
			input:      "Error: failed to write",
			ignoreLogs: []string{"Error: failed to write"},
			want:       "",
			wantErr:    true, // Simulate an error in writing to the log
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := newServerLogger(&buf, tt.ignoreLogs...)

			if tt.wantErr {
				logger.SetOutput(io.Discard)
			}

			logger.Print(tt.input)

			if got := buf.String(); got != tt.want {
				t.Errorf("serverLogWriter.Write() = %v, want %v", got, tt.want)
			}
		})
	}
}
