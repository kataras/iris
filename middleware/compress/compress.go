// Package compress negotiates gzip, deflate, brotli, snappy, s2 and zstd for
// the response and transparently decompresses a compressed request body.
//
// It has no settings, so following the plain-handler rule its entry points are
// the `Handler` and `Safe` functions themselves (`Safe` refuses an unbounded
// request body); `Limited(maxDecompressedSize)` is the one variant that takes a
// setting and therefore returns a Handler.
package compress

import (
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/kataras/iris/v14/context"
)

// The available builtin compression algorithms.
const (
	GZIP    = "gzip"
	DEFLATE = "deflate"
	BROTLI  = "br"
	SNAPPY  = "snappy"
	S2      = "s2"
	ZSTD    = "zstd"

	// IDENTITY no transformation whatsoever.
	IDENTITY = "identity"
)

// AllEncodings is a slice of default content encodings.
// See `AcquireResponseWriter`.
// ZSTD is last on purpose: on equal q-values the first offer wins,
// so clients sending a plain "Accept-Encoding: gzip, zstd" keep getting gzip.
var AllEncodings = []string{GZIP, DEFLATE, BROTLI, SNAPPY, S2, ZSTD}

var (
	// ErrNoEncoding returned when the request's Accept-Encoding or Content-Encoding was not found
	// or was IDENTITY.
	ErrNoEncoding = errors.New("compress: no encoding found")

	// ErrNotSupportedCompression returned when the request's Accept-Encoding (compress) or
	// Content-Encoding (decompress) was not found in the server's supported
	// compression algorithms or the header is empty or the header was IDENTITY.
	// Check with errors.Is.
	ErrNotSupportedCompression = errors.New("compress: no compression algorithm found for the request")
)

// GetAcceptEncoding extracts the best available encoding from the request
// for response writer. It's the encoding the client supports so,
// server can decide which algorithm to use to compress.
//
// Reports whether the request has an encoding that is supported by the server.
// If the request does not accept encoding then it returns an empty string, false and nil error,
// If the request accepts encoding but it's not supported by the server
// it returns an empty string, false and ErrNotSupportedCompression error.
func GetAcceptEncoding(r *http.Request, offers ...string) (string, bool, error) {
	if len(offers) == 0 {
		offers = AllEncodings
	}

	acceptEncoding := r.Header[context.AcceptEncodingHeaderKey]
	encoding := parseEncoding(acceptEncoding, offers)
	if err := shouldCompress(encoding); err != nil {
		if err == ErrNoEncoding {
			return "", false, nil
		}

		return "", false, err
	}

	return encoding, true, nil
}

// GetContentEncoding extracts the best available encoding from the request
// for request body reader. It's the encoding the request sent data to the server.
// If the request has no encoding then it returns an empty string, false and nil error,
// If the request has encoding but it's not supported by the server
// it returns an empty string, false and ErrNotSupportedCompression error.
func GetContentEncoding(r *http.Request, offers ...string) (string, bool, error) {
	if len(offers) == 0 {
		offers = AllEncodings
	}

	contentEncoding := r.Header[context.ContentEncodingHeaderKey]
	encoding := parseEncoding(contentEncoding, offers)
	if err := shouldCompress(encoding); err != nil {
		if err == ErrNoEncoding {
			return "", false, nil
		}

		return "", false, err
	}

	return encoding, true, nil
}

func parseEncoding(headerValue []string, offers []string) string {
	if len(headerValue) == 0 {
		return ""
	}

	encoding := context.NegotiateAcceptHeader(headerValue, offers, IDENTITY)
	return encoding
}

func shouldCompress(encoding string) error {
	if encoding == "" || encoding == IDENTITY {
		return ErrNoEncoding
	}

	if slices.Contains(AllEncodings, encoding) {
		return nil
	}

	return fmt.Errorf("%w: %s", ErrNotSupportedCompression, encoding)
}
