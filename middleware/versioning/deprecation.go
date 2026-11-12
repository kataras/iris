package versioning

import (
	"maps"
	"net/http"
	"time"
)

// The response header keys when a resource is deprecated by the server.
const (
	APIWarnHeader            = "X-Api-Warn"
	APIDeprecationDateHeader = "X-Api-Deprecation-Date"
	APIDeprecationInfoHeader = "X-Api-Deprecation-Info"
)

// DeprecationOptions describes the deprecation headers key-values.
// - "X-Api-Warn": options.WarnMessage
// - "X-Api-Deprecation-Date": options.DeprecationDate.Format(http.TimeFormat)
// - "X-Api-Deprecation-Info": options.DeprecationInfo
type DeprecationOptions struct {
	WarnMessage     string    `json:"warn_message" yaml:"WarnMessage"`
	DeprecationDate time.Time `json:"deprecation_date" yaml:"DeprecationDate"`
	DeprecationInfo string    `json:"deprecation_info" yaml:"DeprecationInfo"`
}

// ShouldHandle reports whether the deprecation headers should be present or no.
func (opts DeprecationOptions) ShouldHandle() bool {
	return opts.WarnMessage != "" || !opts.DeprecationDate.IsZero() || opts.DeprecationInfo != ""
}

// DefaultDeprecationOptions are the default deprecation options,
// it defaults the "X-API-Warn" header to a generic message.
var DefaultDeprecationOptions = DeprecationOptions{
	WarnMessage: "WARNING! You are using a deprecated version of this API.",
}

// Headers returns the deprecation headers based on the options.
func (opts *DeprecationOptions) Headers() http.Header {
	h := make(http.Header)
	if opts.WarnMessage != "" {
		h.Set(APIWarnHeader, opts.WarnMessage)
	}

	if !opts.DeprecationDate.IsZero() {
		h.Set(APIDeprecationDateHeader, opts.DeprecationDate.Format(http.TimeFormat))
	}

	if opts.DeprecationInfo != "" {
		h.Set(APIDeprecationInfoHeader, opts.DeprecationInfo)
	}

	return h
}

// WriteTo writes the deprecation headers to the response writer.
func (opts *DeprecationOptions) WriteTo(w http.ResponseWriter) {
	h := opts.Headers()
	rh := w.Header()
	maps.Copy(rh, h)
}
