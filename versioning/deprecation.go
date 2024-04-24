package versioning

import (
	"time"

	"github.com/kataras/iris/v12/context"
)

// The response header keys when a resource is deprecated by the server.
const (
	APIWarnHeader            = "X-Api-Warn"
	APIDeprecationDateHeader = "X-Api-Deprecation-Date"
	APIDeprecationInfoHeader = "X-Api-Deprecation-Info"
)

// DeprecationOptions describes the deprecation headers key-values.
// - "X-Api-Warn": options.WarnMessage
// - "X-Api-Deprecation-Date": context.FormatTime(ctx, options.DeprecationDate))
// - "X-Api-Deprecation-Info": options.DeprecationInfo
type DeprecationOptions struct {
	WarnMessage     string
	DeprecationDate time.Time
	DeprecationInfo string
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

// WriteDeprecated writes the deprecated response headers
// based on the given "options".
// It can be used inside a middleware.
//
// See `Deprecated` to wrap an existing handler instead.
func WriteDeprecated(ctx *context.Context, options DeprecationOptions) {
	if options.WarnMessage == "" {
		options.WarnMessage = DefaultDeprecationOptions.WarnMessage
	}

	ctx.Header(APIWarnHeader, options.WarnMessage)

	if !options.DeprecationDate.IsZero() {
		ctx.Header(APIDeprecationDateHeader, context.FormatTime(ctx, options.DeprecationDate))
	}

	if options.DeprecationInfo != "" {
		ctx.Header(APIDeprecationInfoHeader, options.DeprecationInfo)
	}
}

// Deprecated wraps an existing API handler and
// marks it as a deprecated one.
// Deprecated can be used to tell the clients that
// a newer version of that specific resource is available instead.
func Deprecated(handler context.Handler, options DeprecationOptions) context.Handler {
	return func(ctx *context.Context) {
		WriteDeprecated(ctx, options)
		handler(ctx)
	}
}
