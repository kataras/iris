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

// TestPrintIPsWildcardServerURL tests that the printIPs function prints localhost
// when serverURL carries the wildcard IP, a browser cannot open "http://0.0.0.0:8080".
// These are the values a Server gives for an "0.0.0.0:PORT" address, see #2175.
func TestPrintIPsWildcardServerURL(t *testing.T) {
	tests := []struct {
		serverNetworkAddr string
		serverURL         string
		expected          string
	}{
		{"0.0.0.0:8080", "http://0.0.0.0:8080", "http://localhost:8080"},
		{"0.0.0.0:80", "http://0.0.0.0", "http://localhost"},
		{"0.0.0.0:8443", "https://0.0.0.0:8443", "https://localhost:8443"},
		{"0.0.0.0:443", "https://0.0.0.0", "https://localhost"},
		// Not the wildcard IP, it is printed as it is.
		{"127.0.0.1:8080", "http://127.0.0.1:8080", "http://127.0.0.1:8080"},
	}

	for _, test := range tests {
		var mw mockWriter
		printIPs(&mw, test.serverNetworkAddr, test.serverURL)
		output := mw.String()

		// With "> Network:" lines the server URL goes to the "> Local:" line,
		// without them (no private IPv4 on this machine) it is the only one.
		expectedLine := "Now listening on: " + test.expected + "\n"
		if strings.Contains(output, "> Network:") {
			expectedLine = "> Local:    " + test.expected + "\n"
		}

		if !strings.Contains(output, expectedLine) {
			t.Errorf("printIPs(%q, %q): expected %q but got %s", test.serverNetworkAddr, test.serverURL, expectedLine, output)
		}
		if test.expected != test.serverURL && strings.Contains(output, test.serverURL) {
			t.Errorf("printIPs(%q, %q): expected without the wildcard URL but got %s", test.serverNetworkAddr, test.serverURL, output)
		}
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
