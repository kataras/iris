package server

import (
	"bytes"
	"strings"
	"testing"
)

// Mock io.Writer to capture the output.
type mockWriter struct {
	bytes.Buffer
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	return m.Buffer.Write(p)
}

// TestPrintIPsWildcardIP tests the printIPs function when serverNetworkAddr is a wildcard IP.
func TestPrintIPsWildcardIP(t *testing.T) {
	var mw mockWriter
	printIPs(&mw, "0.0.0.0:8080", "http://localhost:8080")
	output := mw.String()
	if !strings.Contains(output, "> Network:") {
		t.Errorf("Expected '> Network:' but got %s", output)
	}
}

// TestPrintIPsLocalhost tests the printIPs function when serverURL is localhost.
func TestPrintIPsLocalhost(t *testing.T) {
	var mw mockWriter
	printIPs(&mw, "127.0.0.1:8080", "http://localhost:8080")
	output := mw.String()
	if strings.Contains(output, "> Local:") {
		t.Errorf("Expected without local but got %s", output)
	}
}

// TestPrintIPsPublicURL tests the printIPs function when serverURL is a public URL.
func TestPrintIPsPublicURL(t *testing.T) {
	var mw mockWriter
	printIPs(&mw, "127.0.0.1:8080", "https://iris-go.com")
	output := mw.String()
	if strings.Contains(output, "> Public:") {
		t.Errorf("Expected without public but got %s", output)
	}
}

// TestPrintIPsMultipleURIs tests the printIPs function when there are multiple URIs to print.
func TestPrintIPsMultipleURIs(t *testing.T) {
	var mw mockWriter
	printIPs(&mw, "0.0.0.0:8080", "https://iris-go.com")
	output := mw.String()
	if !(strings.Count(output, "> Network:") >= 1) {
		t.Errorf("Expected at least one '> Network:' but got %s", output)
	}

	if strings.Count(output, "> Public:") != 1 {
		t.Errorf("Expected one '> Public:' but got %s", output)
	}
}
