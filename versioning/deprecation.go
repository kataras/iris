package versioning

import (
	"time"

	"github.com/kataras/iris/context"
)

type DeprecationOptions struct {
	WarnMessage     string
	DeprecationDate time.Time
	DeprecationInfo string
}

func (opts DeprecationOptions) ShouldHandle() bool {
	return opts.WarnMessage != "" || !opts.DeprecationDate.IsZero() || opts.DeprecationInfo != ""
}

var DefaultDeprecationOptions = DeprecationOptions{
	WarnMessage: "WARNING! You are using a deprecated version of this API.",
}

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
