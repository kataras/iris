package versioning

import (
	"time"

	"github.com/kataras/iris/context"
)

// DeprecationOptions describes the deprecation headers key-values.
// - "X-API-Warn": options.WarnMessage
// - "X-API-Deprecation-Date": context.FormatTime(ctx, options.DeprecationDate))
// - "X-API-Deprecation-Info": options.DeprecationInfo
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

// Deprecated marks a specific handler as a deprecated.
// Deprecated can be used to tell the clients that
// a newer version of that specific resource is available instead.
func Deprecated(handler context.Handler, options DeprecationOptions) context.Handler {
	if options.WarnMessage == "" {
		options.WarnMessage = DefaultDeprecationOptions.WarnMessage
	}

	return func(ctx context.Context) {
		handler(ctx)
		ctx.Header("X-API-Warn", options.WarnMessage)

		if !options.DeprecationDate.IsZero() {
			ctx.Header("X-API-Deprecation-Date", context.FormatTime(ctx, options.DeprecationDate))
		}

		if options.DeprecationInfo != "" {
			ctx.Header("X-API-Deprecation-Info", options.DeprecationInfo)
		}
	}
}
