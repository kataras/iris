package client

import (
	"net/http"

	"github.com/kataras/httpclient"
)

// ExtractError returns the response wrapped inside an APIError, reading and
// closing its body. A nil response yields the zero APIError.
func ExtractError(resp *http.Response) APIError {
	return httpclient.ExtractError(resp)
}

// GetError reports whether the given "err" is, or wraps, an APIError. It
// reports false for an APIError carrying no response, so that callers can read
// apiErr.Response without a nil check.
func GetError(err error) (APIError, bool) {
	return httpclient.GetError(err)
}

// GetErrorCode reads an error, which should be an APIError, and returns its
// status code. It returns zero when "err" is nil, is not an APIError, or
// carries no response. Check for zero before using the result as a status.
func GetErrorCode(err error) int {
	return httpclient.GetErrorCode(err)
}
