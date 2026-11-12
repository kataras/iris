package context

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ContentType represents the structured parts of the Content-Type header.
type ContentType struct {
	MediaType string            // The MIME type of the resource or the data.
	Charset   string            // The character encoding standard.
	Boundary  string            // The boundary parameter for multipart entities.
	Version   int               // The version parameter for the content type.
	Params    map[string]string // Any other parameters in the Content-Type. May be nil if there are none.

	// raw     string
	NoSniff bool // Whether to set the X-Content-Type-Options header to nosniff, it's used by setters methods and it's not part of the content-type literal.
	Once    bool // Whether to set the Content-Type header only once, it's used by setters methods and it's not part of the content-type literal.
}

// String returns the raw Content-Type header value.
// It will be constructed from the MediaType, Charset, Boundary, and Params fields.
// The constructed value will be in the format "media-type; charset=charset; boundary=boundary; version=1; key=value",
// e.g. application/json; charset=utf-8; version=2.
// The media-type is required, and the charset and boundary are optional.
// The key-value pairs are the other parameters in the Content-Type.
// The key-value pairs are separated by a semicolon.
// The key and value are separated by an equal sign.
// Only the media type, the parameter names and the charset are lowercased;
// parameter values such as the multipart boundary are kept verbatim.
func (ct ContentType) String() string {
	// if ct.raw != "" {
	// 	return ct.raw
	// }
	// return ct.Encode()
	return ct.toString()
}

// Encode encodes the ContentType to a string.
// It rebuilds the string value, it should be called each modification of the ContentType structure.
// func (ct *ContentType) Encode() string {
// 	ct.raw = ct.toString()
// 	return ct.raw
// }

func (ct *ContentType) toString() string {
	// Fill the Content-Type string with the media type and parameters.
	//
	// Only the media type, the parameter names and the charset are lowercased.
	// RFC 2045 section 5.1 makes those case-insensitive, but parameter values in
	// general are not: a multipart boundary is a literal delimiter matched
	// byte-for-byte against the body (RFC 2046 section 5.1.1), and browsers send
	// mixed case, so lowercasing it produces a response no client can parse.
	var contentTypeValue strings.Builder
	contentTypeValue.WriteString(strings.ToLower(ct.MediaType))

	// Include Charset if it's not empty.
	if ct.Charset != "" {
		fmt.Fprintf(&contentTypeValue, "; charset=%s", strings.ToLower(ct.Charset))
	}

	// Include Boundary if it's not empty.
	if ct.Boundary != "" {
		fmt.Fprintf(&contentTypeValue, "; boundary=%s", ct.Boundary)
	}

	if ct.Version > 0 {
		fmt.Fprintf(&contentTypeValue, "; version=%d", ct.Version)
	}

	// Include other parameters.
	for k, v := range ct.Params {
		fmt.Fprintf(&contentTypeValue, "; %s=%s", strings.ToLower(k), v)
	}

	return contentTypeValue.String()
}

// ContentTypeOption is a function that modifies a ContentType.
// It is used on `SetContentType` method.
//
// Available options:
// - ContentTypeCharset
// - ContentTypeNoSniff
// - ContentTypeOnce
type ContentTypeOption func(*ContentType)

// ContentTypeCharset sets the charset parameter of the Content-Type.
func ContentTypeCharset(charset string) ContentTypeOption {
	return func(ct *ContentType) {
		ct.Charset = charset
	}
}

// ContentTypeNoSniff sets the X-Content-Type-Options header to nosniff.
func ContentTypeNoSniff(ct *ContentType) {
	ct.NoSniff = true
}

// ContentTypeOnce sets the Content-Type header only once.
func ContentTypeOnce(ct *ContentType) {
	ct.Once = true
}

var (
	// ErrNoContentType is returned when the Content-Type is required but not provided.
	// It is used in the ParseContentType function.
	ErrNoContentType = fmt.Errorf("content-type is required but was not provided")
	// ErrNoContentTypeMediaType is returned when the media-type is required but not provided.
	// It is used in the ParseContentType function.
	ErrNoContentTypeMediaType = fmt.Errorf("media-type is required but was not provided")
)

// MaxContentTypeParams is the highest number of parameters ParseContentType
// keeps from a single Content-Type header. Anything beyond it is discarded.
//
// The header is client-supplied and net/http accepts it up to MaxHeaderBytes
// (1MB by default), so without a cap a request of ";a=1;b=1;..." would have
// every read of Context.ContentType build a map of hundreds of thousands of
// attacker-chosen keys. No real content type carries more than a handful.
const MaxContentTypeParams = 32

// ParseContentType takes a Content-Type string and returns a ContentType struct.
//
// The media type, the parameter names and the charset are lowercased, as those
// are case-insensitive. Every other parameter value is kept verbatim, because
// values such as a multipart boundary are matched literally against the body.
func ParseContentType(contentType string) (ContentType, error) {
	if contentType == "" {
		return ContentType{}, ErrNoContentType
	}

	// ct := ContentType{raw: contentType}
	ct := ContentType{}

	if !strings.ContainsRune(contentType, ';') { // fast path.
		ct.MediaType = strings.ToLower(strings.TrimSpace(contentType))
		return ct, nil
	}

	for param := range strings.SplitSeq(contentType, ";") {
		before, after, hasValue := strings.Cut(strings.TrimSpace(param), "=")
		if !hasValue {
			ct.MediaType = strings.ToLower(strings.TrimSpace(before))
			continue
		}

		key := strings.ToLower(strings.TrimSpace(before))
		value := strings.TrimSpace(after)

		switch key {
		case "charset":
			ct.Charset = strings.ToLower(value)
		case "boundary":
			ct.Boundary = value
		case "version":
			// A malformed or out-of-range version is dropped rather than echoed
			// back into the response header by String.
			if v, err := strconv.Atoi(value); err == nil && v > 0 {
				ct.Version = v
			}
		default:
			if len(ct.Params) >= MaxContentTypeParams {
				continue
			}

			if ct.Params == nil {
				ct.Params = make(map[string]string)
			}
			ct.Params[key] = value
		}
	}

	if ct.MediaType == "" {
		return ContentType{}, ErrNoContentTypeMediaType
	}

	return ct, nil
}

var (
	// DefaultCharset is the default charset to use if none is provided.
	DefaultCharset = "utf-8"
	// MediaTypesRequiringCharset is a list of media types that typically require a charset parameter.
	MediaTypesRequiringCharset = map[string]struct{}{ // see Application.MimeToContentType too.
		"text/html":                         {},
		"text/css":                          {},
		"text/plain":                        {},
		"text/xml":                          {},
		"text/javascript":                   {},
		"application/xml":                   {},
		"application/problem+xml":           {},
		"application/x-yaml":                {},
		"application/xhtml+xml":             {},
		"application/atom+xml":              {},
		"application/rss+xml":               {},
		"application/soap+xml":              {},
		"application/svg+xml":               {},
		"application/x-www-form-urlencoded": {},
	}
)

const contentTypeOptionsHeaderKey = "X-Content-Type-Options"

// This function checks the media type and decides whether to include the charset parameter.
// For JSON and problem+json, it doesn’t add the charset since they are UTF-8 by default.
// For XML, YAML, and problem+xml, it includes UTF-8 as the charset if it’s not already specified.
// The function then constructs the full Content-Type value and sets it in the response header.
func setContentType(w http.ResponseWriter, ct ContentType, defaultCharset string) {
	if ct.Charset == "" { // check only on write, not receive or parse or string.
		// Check if the media type is in the list of types that require a charset.
		if _, ok := MediaTypesRequiringCharset[ct.MediaType]; ok {
			// Include charset if not provided.
			if defaultCharset == "" {
				defaultCharset = DefaultCharset
			}

			ct.Charset = defaultCharset
			// ct.Encode() // refresh.
		}
	}

	writeContentType(w, ct.String())
	if ct.NoSniff {
		w.Header().Set(contentTypeOptionsHeaderKey, "nosniff")
	}
}

func writeContentType(w http.ResponseWriter, contentType string) {
	w.Header().Set(ContentTypeHeaderKey, contentType)
}
