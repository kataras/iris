package cors

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"
)

func init() {
	context.SetHandlerName("iris/middleware/cors.*", "iris.cors")
}

var (
	// ErrOriginNotAllowed is given to the error handler
	// when the error is caused because an origin was not allowed to pass through.
	ErrOriginNotAllowed = errors.New("origin not allowed")

	// AllowAnyOrigin allows all origins to pass.
	AllowAnyOrigin = func(_ *context.Context, _ string) bool {
		return true
	}

	// DefaultErrorHandler is the default error handler which
	// fires forbidden status (403) on disallowed origins.
	DefaultErrorHandler = func(ctx *context.Context, _ error) {
		ctx.StopWithStatus(http.StatusForbidden)
	}

	// DefaultOriginExtractor is the default method which
	// an origin is extracted. It returns the value of the request's "Origin" header
	// and always true, means that it allows empty origin headers as well.
	DefaultOriginExtractor = func(ctx *context.Context) (string, bool) {
		header := ctx.GetHeader(originRequestHeader)
		return header, true
	}

	// StrictOriginExtractor is an ExtractOriginFunc type
	// which is a bit more strictly than the DefaultOriginExtractor.
	// It allows only non-empty "Origin" header values to be passed.
	// If the header is missing, the middleware will not allow the execution
	// of the next handler(s).
	StrictOriginExtractor = func(ctx *context.Context) (string, bool) {
		header := ctx.GetHeader(originRequestHeader)
		return header, header != ""
	}
)

type (
	// ExtractOriginFunc describes the function which should return the request's origin or false.
	ExtractOriginFunc = func(ctx *context.Context) (string, bool)

	// AllowOriginFunc describes the function which is called when the
	// middleware decides if the request's origin should be allowed or not.
	AllowOriginFunc = func(ctx *context.Context, origin string) bool

	// HandleErrorFunc describes the function which is fired
	// when a request by a specific (or empty) origin was not allowed to pass through.
	HandleErrorFunc = func(ctx *context.Context, err error)

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
		maxAgeSecondsValue    string
		referrerPolicyValue   string
	}
)

// New returns the default CORS middleware.
// For a more advanced type of protection middleware with more options
// please refer to: https://github.com/iris-contrib/middleware repository instead.
//
// Example Code:
//
//		import "github.com/kataras/iris/v12/middleware/cors"
//	 import "github.com/kataras/iris/v12/x/errors"
//
//	 app.UseRouter(cors.New().
//	     HandleErrorFunc(func(ctx iris.Context, err error) {
//	         errors.FailedPrecondition.Err(ctx, err)
//	     }).
//	     ExtractOriginFunc(cors.StrictOriginExtractor).
//	     ReferrerPolicy(cors.NoReferrerWhenDowngrade).
//	     AllowOrigin("domain1.com,domain2.com,domain3.com").
//	     Handler())
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
		allowMethodsValue:   "*",
		maxAgeSecondsValue:  "86400",
		referrerPolicyValue: NoReferrerWhenDowngrade.String(),
	}
}

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

// AllowOrigin calls the "AllowOriginFunc" method
// and registers a function which accepts any incoming
// request with origin of the given "originLine".
// The originLine can contain one or more domains separated by comma.
// See "AllowOrigins" to set a list of strings instead.
func (c *CORS) AllowOrigin(originLine string) *CORS {
	return c.AllowOrigins(strings.Split(originLine, ",")...)
}

// AllowOriginMatcherFunc sets the allow origin func without iris.Context
// as its first parameter, i.e. a regular expression.
func (c *CORS) AllowOriginMatcherFunc(fn func(origin string) bool) *CORS {
	return c.AllowOriginFunc(func(ctx *context.Context, origin string) bool {
		return fn(origin)
	})
}

// AllowOriginRegex calls the "AllowOriginFunc" method
// and registers a function which accepts any incoming
// request with origin that matches at least one of the given "regexpLines".
func (c *CORS) AllowOriginRegex(regexpLines ...string) *CORS {
	matchers := make([]func(string) bool, 0, len(regexpLines))
	for _, line := range regexpLines {
		matcher := regexp.MustCompile(line).MatchString
		matchers = append(matchers, matcher)
	}

	return c.AllowOriginFunc(func(ctx *context.Context, origin string) bool {
		for _, m := range matchers {
			if m(origin) {
				return true
			}
		}

		return false
	})
}

// AllowOrigins calls the "AllowOriginFunc" method
// and registers a function which accepts any incoming
// request with origin of one of the given "origins".
func (c *CORS) AllowOrigins(origins ...string) *CORS {
	allowOrigins := make(map[string]struct{}, len(origins)) // read-only at serve time.
	for _, origin := range origins {
		if origin == "*" {
			// If AllowOrigins called with asterix, it is a missuse of this
			// middleware (set AllowAnyOrigin instead).
			allowOrigins = nil
			return c.AllowOriginFunc(AllowAnyOrigin)
			// panic("wildcard is not allowed, use AllowOriginFunc(AllowAnyOrigin) instead")
			// No ^ let's register a function which allows all and continue.
		}

		origin = strings.TrimSpace(origin)
		allowOrigins[origin] = struct{}{}
	}

	return c.AllowOriginFunc(func(ctx *context.Context, origin string) bool {
		_, allow := allowOrigins[origin]
		return allow
	})
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

// ReferrerPolicy type for referrer-policy header value.
type ReferrerPolicy string

// All available referrer policies.
// Read more at: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy.
const (
	NoReferrer                  ReferrerPolicy = "no-referrer"
	NoReferrerWhenDowngrade     ReferrerPolicy = "no-referrer-when-downgrade"
	Origin                      ReferrerPolicy = "origin"
	OriginWhenCrossOrigin       ReferrerPolicy = "origin-when-cross-origin"
	SameOrigin                  ReferrerPolicy = "same-origin"
	StrictOrigin                ReferrerPolicy = "strict-origin"
	StrictOriginWhenCrossOrigin ReferrerPolicy = "strict-origin-when-cross-origin"
	UnsafeURL                   ReferrerPolicy = "unsafe-url"
)

// String returns the text representation of the "r" ReferrerPolicy.
func (r ReferrerPolicy) String() string {
	return string(r)
}

// ReferrerPolicy sets the "Referrer-Policy" header value.
// Defaults to "no-referrer-when-downgrade".
//
// Read more at: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
// and https://developer.mozilla.org/en-US/docs/Web/Security/Referer_header:_privacy_and_security_concerns.
func (c *CORS) ReferrerPolicy(referrerPolicy ReferrerPolicy) *CORS {
	c.referrerPolicyValue = referrerPolicy.String()
	return c
}

// MaxAge sets the "Access-Control-Max-Age" header value.
//
// Read more at: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#access-control-max-age.
func (c *CORS) MaxAge(d time.Duration) *CORS {
	c.maxAgeSecondsValue = strconv.FormatFloat(d.Seconds(), 'E', -1, 64)
	return c
}

const (
	originRequestHeader    = "Origin"
	allowOriginHeader      = "Access-Control-Allow-Origin"
	allowCredentialsHeader = "Access-Control-Allow-Credentials"
	referrerPolicyHeader   = "Referrer-Policy"
	exposeHeadersHeader    = "Access-Control-Expose-Headers"
	requestMethodHeader    = "Access-Control-Request-Method"
	requestHeadersHeader   = "Access-Control-Request-Headers"
	allowMethodsHeader     = "Access-Control-Allow-Methods"
	allowAllMethodsValue   = "*"
	allowHeadersHeader     = "Access-Control-Allow-Headers"
	maxAgeHeader           = "Access-Control-Max-Age"
	varyHeader             = "Vary"
)

func (c *CORS) addVaryHeaders(ctx *context.Context) {
	ctx.Header(varyHeader, originRequestHeader)

	if ctx.Method() == http.MethodOptions {
		ctx.Header(varyHeader, requestMethodHeader)
		ctx.Header(varyHeader, requestHeadersHeader)
	}
}

// Handler method returns the Iris CORS Handler with basic features.
// Note that the caller should NOT modify any of the CORS instance fields afterwards.
func (c *CORS) Handler() context.Handler {
	return func(ctx *context.Context) {
		c.addVaryHeaders(ctx) // add vary headers at any case.

		origin, ok := c.extractOriginFunc(ctx)
		if !ok || !c.allowOriginFunc(ctx, origin) {
			c.errorHandler(ctx, ErrOriginNotAllowed)
			return
		}

		if origin == "" { // if we allow empty origins, set it to wildcard.
			origin = "*"
		}

		ctx.Header(allowOriginHeader, origin)
		ctx.Header(allowCredentialsHeader, c.allowCredentialsValue)
		// 08 July 2021 Mozzila updated the following document: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
		ctx.Header(referrerPolicyHeader, c.referrerPolicyValue)
		ctx.Header(exposeHeadersHeader, c.exposeHeadersValue)
		if ctx.Method() == http.MethodOptions {
			ctx.Header(allowMethodsHeader, allowAllMethodsValue)
			ctx.Header(allowHeadersHeader, c.allowHeadersValue)
			ctx.Header(maxAgeHeader, c.maxAgeSecondsValue)
			ctx.StatusCode(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
