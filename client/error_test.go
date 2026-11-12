package client

import (
	"net/http"
	"strings"
	"testing"
)

// An APIError is not always built from a live round trip: server code
// constructs them too, without a Request or a Body. Every accessor must
// tolerate that.
func TestAPIErrorZeroValue(t *testing.T) {
	var zero APIError
	if s := zero.Error(); s != "" {
		t.Fatalf("zero APIError text = %q, want empty", s)
	}
	if code := GetErrorCode(zero); code != 0 {
		t.Fatalf("GetErrorCode(zero) = %d, want 0 (unknown)", code)
	}
}

func TestAPIErrorResponseWithoutRequest(t *testing.T) {
	err := APIError{
		Response: &http.Response{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway"},
		Body:     []byte(`{"message":"upstream"}`),
	}

	s := err.Error()
	if !strings.Contains(s, "Bad Gateway") || !strings.Contains(s, "upstream") {
		t.Fatalf("unexpected text: %q", s)
	}
	if code := GetErrorCode(err); code != http.StatusBadGateway {
		t.Fatalf("GetErrorCode = %d, want 502", code)
	}
}

func TestExtractErrorNilResponse(t *testing.T) {
	if got := ExtractError(nil); got.Response != nil || len(got.Body) != 0 {
		t.Fatalf("ExtractError(nil) = %+v, want the zero value", got)
	}

	resp := &http.Response{StatusCode: http.StatusInternalServerError}
	got := ExtractError(resp) // nil Body.
	if got.Response != resp || len(got.Body) != 0 {
		t.Fatalf("ExtractError(body-less response) = %+v", got)
	}
}
