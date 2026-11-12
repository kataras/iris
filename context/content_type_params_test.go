package context

import (
	"fmt"
	"strings"
	"testing"
)

// A multipart boundary is matched byte-for-byte against the body delimiters
// (RFC 2046 section 5.1.1) and browsers send mixed case, so lowercasing it
// yields a message no client can parse.
func TestParseContentTypeKeepsTheBoundaryCase(t *testing.T) {
	const boundary = "----WebKitFormBoundaryAbC123xYz"

	ct, err := ParseContentType("Multipart/Form-Data; Boundary=" + boundary)
	if err != nil {
		t.Fatalf("ParseContentType() error = %v", err)
	}

	if ct.Boundary != boundary {
		t.Errorf("Boundary = %q, want %q", ct.Boundary, boundary)
	}

	if ct.MediaType != "multipart/form-data" {
		t.Errorf("MediaType = %q, want it lowercased", ct.MediaType)
	}

	if got, want := ct.String(), "multipart/form-data; boundary="+boundary; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestParseContentTypeLowercasesWhatIsCaseInsensitive(t *testing.T) {
	ct, err := ParseContentType("TEXT/HTML; CHARSET=UTF-8; Profile=UrnExample")
	if err != nil {
		t.Fatalf("ParseContentType() error = %v", err)
	}

	if ct.MediaType != "text/html" {
		t.Errorf("MediaType = %q, want %q", ct.MediaType, "text/html")
	}

	if ct.Charset != "utf-8" {
		t.Errorf("Charset = %q, want %q", ct.Charset, "utf-8")
	}

	// The parameter name is case-insensitive, its value is not.
	if got, ok := ct.Params["profile"]; !ok || got != "UrnExample" {
		t.Errorf("Params[profile] = %q (present: %t), want %q", got, ok, "UrnExample")
	}
}

// The header is client-supplied and net/http accepts it up to MaxHeaderBytes,
// so an uncapped parameter map is an allocation amplifier on every read.
func TestParseContentTypeCapsTheParameters(t *testing.T) {
	var b strings.Builder
	b.WriteString("text/plain")
	for i := range MaxContentTypeParams * 10 {
		fmt.Fprintf(&b, "; p%d=%d", i, i)
	}

	ct, err := ParseContentType(b.String())
	if err != nil {
		t.Fatalf("ParseContentType() error = %v", err)
	}

	if len(ct.Params) > MaxContentTypeParams {
		t.Errorf("kept %d parameters, want at most %d", len(ct.Params), MaxContentTypeParams)
	}

	if ct.MediaType != "text/plain" {
		t.Errorf("MediaType = %q, want %q", ct.MediaType, "text/plain")
	}
}

func TestParseContentTypeRejectsAnInvalidVersion(t *testing.T) {
	for _, value := range []string{"-3", "abc", "0", "9999999999999999999999"} {
		ct, err := ParseContentType("application/json; version=" + value)
		if err != nil {
			t.Fatalf("ParseContentType(version=%s) error = %v", value, err)
		}

		if ct.Version != 0 {
			t.Errorf("version=%s produced Version = %d, want 0", value, ct.Version)
		}

		if strings.Contains(ct.String(), "version") {
			t.Errorf("version=%s was echoed back into %q", value, ct.String())
		}
	}
}

// The media type is the only part of the input worth reporting, and reporting
// the whole header reflects client-controlled text into the error.
func TestParseContentTypeErrorDoesNotEchoTheInput(t *testing.T) {
	_, err := ParseContentType("; charset=<script>alert(1)</script>")
	if err == nil {
		t.Fatal("ParseContentType() with no media type returned no error")
	}

	if strings.Contains(err.Error(), "script") {
		t.Errorf("the error reflected the input: %q", err.Error())
	}
}
