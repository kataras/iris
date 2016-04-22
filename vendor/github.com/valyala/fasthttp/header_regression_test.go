package fasthttp

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestIssue28ResponseWithoutBodyNoContentType(t *testing.T) {
	var r Response

	// Empty response without content-type
	s := r.String()
	if strings.Contains(s, "Content-Type") {
		t.Fatalf("unexpected Content-Type found in response header with empty body: %q", s)
	}

	// Explicitly set content-type
	r.Header.SetContentType("foo/bar")
	s = r.String()
	if !strings.Contains(s, "Content-Type: foo/bar\r\n") {
		t.Fatalf("missing explicitly set content-type for empty response: %q", s)
	}

	// Non-empty response.
	r.Reset()
	r.SetBodyString("foobar")
	s = r.String()
	if !strings.Contains(s, fmt.Sprintf("Content-Type: %s\r\n", defaultContentType)) {
		t.Fatalf("missing default content-type for non-empty response: %q", s)
	}

	// Non-empty response with custom content-type.
	r.Header.SetContentType("aaa/bbb")
	s = r.String()
	if !strings.Contains(s, "Content-Type: aaa/bbb\r\n") {
		t.Fatalf("missing custom content-type: %q", s)
	}
}

func TestIssue6RequestHeaderSetContentType(t *testing.T) {
	testIssue6RequestHeaderSetContentType(t, "GET")
	testIssue6RequestHeaderSetContentType(t, "POST")
	testIssue6RequestHeaderSetContentType(t, "PUT")
	testIssue6RequestHeaderSetContentType(t, "PATCH")
}

func testIssue6RequestHeaderSetContentType(t *testing.T, method string) {
	contentType := "application/json"
	contentLength := 123

	var h RequestHeader
	h.SetMethod(method)
	h.SetRequestURI("http://localhost/test")
	h.SetContentType(contentType)
	h.SetContentLength(contentLength)

	issue6VerifyRequestHeader(t, &h, contentType, contentLength, method)

	s := h.String()

	var h1 RequestHeader

	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := h1.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	issue6VerifyRequestHeader(t, &h1, contentType, contentLength, method)
}

func issue6VerifyRequestHeader(t *testing.T, h *RequestHeader, contentType string, contentLength int, method string) {
	if string(h.ContentType()) != contentType {
		t.Fatalf("unexpected content-type: %q. Expecting %q. method=%q", h.ContentType(), contentType, method)
	}
	if string(h.Method()) != method {
		t.Fatalf("unexpected method: %q. Expecting %q", h.Method(), method)
	}
	if method != "GET" {
		if h.ContentLength() != contentLength {
			t.Fatalf("unexpected content-length: %d. Expecting %d. method=%q", h.ContentLength(), contentLength, method)
		}
	} else if h.ContentLength() != 0 {
		t.Fatalf("unexpected content-length for GET method: %d. Expecting 0", h.ContentLength())
	}
}
