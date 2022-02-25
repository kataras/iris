package cors

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

func init() {
	context.SetHandlerName("iris/middleware/basicauth.*", "iris.cors")
}

var (
	// ErrOriginNotAllowed is given to the error handler
	// when the error is caused because an origin was not allowed to pass through.
	ErrOriginNotAllowed = errors.New("origin not allowed")

	// AllowAnyOrigin allows all origins to pass.
	AllowAnyOrigin = func(_ iris.Context, _ string) bool {
		return true
	}

	// DefaultErrorHandler is the default error handler which
	// fires forbidden status (403) on disallowed origins.
	DefaultErrorHandler = func(ctx iris.Context, _ error) {
		ctx.StopWithStatus(iris.StatusForbidden)
	}

	// DefaultOriginExtractor is the default method which
	// an origin is extracted. It returns the value of the request's "Origin" header.
	DefaultOriginExtractor = func(ctx iris.Context) string {
		return ctx.GetHeader("Origin")
	}
)

type (
	// ExtractOriginFunc describes the function which should return the request's origin.
	ExtractOriginFunc = func(ctx iris.Context) string

	// AllowOriginFunc describes the function which is called when the
	// middleware decides if the request's origin should be allowed or not.
	AllowOriginFunc = func(ctx iris.Context, origin string) bool

	// HandleErrorFunc describes the function which is fired
	// when a request by a specific (or empty) origin was not allowed to pass through.
	HandleErrorFunc = func(ctx iris.Context, err error)

	// CORS holds the customizations developers can
	// do on the cors middleware.
	//
	// Read more at: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS.
	CORS struct {
		extractOriginFunc ExtractOriginFunc
		allowOriginFunc   AllowOriginFunc
		errorHandler      HandleErrorFunc

		allowCredentialsValue string
		exposeHeadersValue    string
		allowHeadersValue     string
		allowMethodsValue     string

		maxAgeSecondsValue string
	}
)

// ExtractOriginFunc sets the function which should return the request's origin.
func (c *CORS) ExtractOriginFunc(fn ExtractOriginFunc) *CORS {
	c.extractOriginFunc = fn
	return c
}

// AllowOriginFunc sets the function which decides if an origin(domain) is allowed
// to continue or not.
//
// Read more at: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#access-control-allow-origin.
func (c *CORS) AllowOriginFunc(fn AllowOriginFunc) *CORS {
	c.allowOriginFunc = fn
	return c
}

// HandleErrorFunc sets the function which is called
// when an error of origin not allowed is fired.
func (c *CORS) HandleErrorFunc(fn HandleErrorFunc) *CORS {
	c.errorHandler = fn
	return c
}

// DisallowCredentials sets the "Access-Control-Allow-Credentials" header to false.
//
// Read more at: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#access-control-allow-credentials.
func (c *CORS) DisallowCredentials() *CORS {
	c.allowCredentialsValue = "false"
	return c
}

// ExposeHeaders sets the "Access-Control-Expose-Headers" header value.
//
// Read more at: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#access-control-expose-headers.
func (c *CORS) ExposeHeaders(headers ...string) *CORS {
	c.exposeHeadersValue = strings.Join(headers, ", ")
	return c
}

// AllowHeaders sets the "Access-Control-Allow-Headers" header value.
//
// Read more at: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#access-control-allow-headers.
func (c *CORS) AllowHeaders(headers ...string) *CORS {
	c.allowHeadersValue = strings.Join(headers, ", ")
	return c
}

// MaxAge sets the "Access-Control-Max-Age" header value.
//
// Read more at: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#access-control-max-age.
func (c *CORS) MaxAge(d time.Duration) *CORS {
	c.maxAgeSecondsValue = strconv.FormatFloat(d.Seconds(), 'E', -1, 64)
	return c
}

// Handler method returns the Iris CORS Handler with basic features.
// Note that the caller should NOT modify any of the CORS instance fields afterwards.
func (c *CORS) Handler() iris.Handler {
	return func(ctx iris.Context) {
		origin := c.extractOriginFunc(ctx)
		if !c.allowOriginFunc(ctx, origin) {
			c.errorHandler(ctx, ErrOriginNotAllowed)
			return
		}

		if origin == "" { // if we allow empty origins, set it to wildcard.
			origin = "*"
		}

		ctx.Header("Access-Control-Allow-Origin", origin)
		ctx.Header("Access-Control-Allow-Credentials", c.allowCredentialsValue)
		// 08 July 2021 Mozzila updated the following document: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
		ctx.Header("Referrer-Policy", "no-referrer-when-downgrade")
		ctx.Header("Access-Control-Expose-Headers", c.exposeHeadersValue)
		if ctx.Method() == iris.MethodOptions {
			ctx.Header("Access-Control-Allow-Methods", "*")
			ctx.Header("Access-Control-Allow-Headers", c.allowHeadersValue)
			ctx.Header("Access-Control-Max-Age", c.maxAgeSecondsValue)
			ctx.StatusCode(iris.StatusNoContent)
			return
		}

		ctx.Next()
	}
}

// New returns the default CORS middleware.
// For a more advanced type of protection middleware with more options
// please refer to: https://github.com/iris-contrib/middleware repository instead.
func New() *CORS {
	return &CORS{
		extractOriginFunc: DefaultOriginExtractor,
		allowOriginFunc:   AllowAnyOrigin,
		errorHandler:      DefaultErrorHandler,

		allowCredentialsValue: "true",
		exposeHeadersValue:    "*, Authorization, X-Authorization",
		allowHeadersValue:     "*",
		// This field cannot be modified by the end-developer,
		// as we have another type of controlling the HTTP verbs per handler.
		allowMethodsValue:  "*",
		maxAgeSecondsValue: "86400",
	}
}
