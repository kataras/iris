// Package cors implements Cross-Origin Resource Sharing.
//
// Its entry point is New() (or `Strict()`, which refuses to be permissive by
// accident), which returns a *CORS rather than a Handler: the policy is built
// with chained setters such as AllowOrigin and AllowHeaders, and its `Handler`
// method is registered with app.UseRouter because it also answers preflight
// requests.
package cors

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/v14/context"
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
	DefaultErrorHandler = func(ctx *context.Context, err error) {
		ctx.StopWithError(http.StatusForbidden, err)
	}

	// DefaultOriginExtractor is the default method which
	// an origin is extracted. It returns the value of the request's "Origin" header
	// and always true, means that it allows empty origin headers as well.
	DefaultOriginExtractor = func(ctx *context.Context) (string, bool) {
		header := ctx.Header(originRequestHeader)
		return header, true
	}

	// StrictOriginExtractor is an ExtractOriginFunc type
	// which is a bit more strictly than the DefaultOriginExtractor.
	// It allows only non-empty "Origin" header values to be passed.
	// If the header is missing, the middleware will not allow the execution
	// of the next handler(s).
	StrictOriginExtractor = func(ctx *context.Context) (string, bool) {
		header := ctx.Header(originRequestHeader)
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

		allowCredentialsValue    string
		allowCredentialsExplicit bool // true when the developer explicitly enabled credentials; allows them even with a "*" origin.
		exposeHeadersValue       string
		allowHeadersValue        string
		allowMethodsValue        string
		maxAgeSecondsValue       string
		referrerPolicyValue      string

		skipLocalRequests bool // if true then skip local requests, defaults to false.
	}
)

// New returns the default CORS middleware.
// For a more advanced type of protection middleware with more options
// please refer to: https://github.com/iris-contrib/middleware repository instead.
//
// SECURITY WARNING: the defaults allow ANY origin and advertise
// "Access-Control-Allow-Credentials: true" for it.
//
// A page on any domain can then run
//
//	fetch("https://your-app/api/me", {credentials: "include"})
//
// and read the authenticated response, because the browser attaches the user's
// cookies and this middleware echoes the attacker's origin back with credentials
// allowed. Unless your API is genuinely public and unauthenticated:
//
//   - restrict the origins with AllowOrigin/AllowOrigins, and/or
//   - use `Strict()` instead of `New()`, which advertises credentials only when
//     you ask for them with AllowCredentials().
//
// Example Code:
//
//	import "github.com/kataras/iris/v14/middleware/cors"
//	import "github.com/kataras/iris/v14/rest/errors"
//
//	app.UseRouter(cors.New().
//	    HandleErrorFunc(func(ctx iris.Context, err error) {
//	        errors.FailedPrecondition.Err(ctx, err)
//	    }).
//	    ExtractOriginFunc(cors.StrictOriginExtractor).
//	    ReferrerPolicy(cors.NoReferrerWhenDowngrade).
//	    AllowOrigin("domain1.com,domain2.com,domain3.com").
//	    Handler())
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
		skipLocalRequests:   false,
	}
}

// Strict returns a CORS middleware configured exactly like `New`, except that
// credentials are NOT advertised unless you enable them explicitly with
// `AllowCredentials()`.
//
// This is the recommended entry point for anything that serves authenticated
// responses. With `New()`, a reflected origin is answered with
// "Access-Control-Allow-Credentials: true", which lets any site read a logged-in
// user's data cross-origin. `Strict()` closes that by default and still lets you
// opt in per deployment:
//
//	app.UseRouter(cors.Strict().
//	    AllowOrigin("https://app.example.com").
//	    AllowCredentials().
//	    Handler())
func Strict() *CORS {
	c := New()
	c.allowCredentialsValue = "false"
	return c
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
//
// To accept any origin pass "*".
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
// Matches are exact or with wildcards.
// Examples:
// - "https://example.com" matches exactly that origin
// - "https://*.example.com" matches any subdomain of example.com but not the domain itself
// - "https://www.example.*" matches any TLD for www.example
// - "https://www.example*" matches any domain starting with "https://www.example"
// - "*" matches any origin.
func (c *CORS) AllowOrigins(origins ...string) *CORS {
	if slices.Contains(origins, "*") {
		// If AllowOrigins called with asterix, it is a missuse of this
		// middleware (set AllowAnyOrigin instead).
		return c.AllowOriginFunc(AllowAnyOrigin)
		// panic("wildcard is not allowed, use AllowOriginFunc(AllowAnyOrigin) instead")
		// No ^ let's register a function which allows all and continue.
	}

	patterns := MergeOriginPatterns(origins)
	return c.AllowOriginFunc(func(ctx *context.Context, origin string) bool {
		for _, pattern := range patterns {
			if matchOriginPattern(origin, pattern) {
				return true
			}
		}

		return false
	})
}

// HandleErrorFunc sets the function which is called
// when an error of origin not allowed is fired.
func (c *CORS) HandleErrorFunc(fn HandleErrorFunc) *CORS {
	c.errorHandler = fn
	return c
}

// AllowCredentials sets the "Access-Control-Allow-Credentials" header to true and
// marks credentials as explicitly enabled.
//
// By default credentials are advertised for a specific origin but suppressed for a
// wildcard ("*") origin, because browsers reject the "*" + credentials combination.
// Call AllowCredentials to advertise credentials even with a wildcard origin when
// you have a deliberate reason to (for example a non-browser API client).
//
// Read more at: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#access-control-allow-credentials.
func (c *CORS) AllowCredentials() *CORS {
	c.allowCredentialsValue = "true"
	c.allowCredentialsExplicit = true
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
	// Note(@kataras): Access-Control-Max-Age is delta-seconds - digits only.
	// FormatFloat with 'E' rendered 24h as "8.64E+04", which browsers discard,
	// falling back to a ~5 second cache so that every request preflights again.
	seconds := int64(d.Seconds())
	if seconds < 0 {
		seconds = 0
	}

	c.maxAgeSecondsValue = strconv.FormatInt(seconds, 10)
	return c
}

// SkipLocalRequests sets whether the CORS middleware should skip origin checks for local requests.
//
// If set to true, requests originating from local addresses (such as 127.0.0.1 or ::1) will bypass CORS validation
// and proceed without origin checks.
// This is useful during development or when trusted local services interact with the server.
//
// By default, this option is false, meaning all requests (including local ones) are subject to CORS checks.
func (c *CORS) SkipLocalRequests(skip bool) *CORS {
	c.skipLocalRequests = skip
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
	allowAllMethodsValue   = "GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD, CONNECT, TRACE, QUERY"
	allowHeadersHeader     = "Access-Control-Allow-Headers"
	maxAgeHeader           = "Access-Control-Max-Age"
	varyHeader             = "Vary"
)

func (c *CORS) addVaryHeaders(ctx *context.Context) {
	ctx.AddHeader(varyHeader, originRequestHeader)

	if ctx.Method() == http.MethodOptions {
		ctx.AddHeader(varyHeader, requestMethodHeader)
		ctx.AddHeader(varyHeader, requestHeadersHeader)
	}
}

// allowedHeadersFor resolves the "Access-Control-Allow-Headers" value for a
// preflight request.
//
// Per the Fetch standard a literal "*" is NOT a wildcard for a credentialed
// request, so advertising it fails every preflight that asks for Authorization -
// while this middleware simultaneously exposes Authorization through
// "Access-Control-Expose-Headers". Echoing back what the client asked for keeps
// the permissive intent and actually works with credentials.
func (c *CORS) allowedHeadersFor(ctx *context.Context) string {
	if c.allowHeadersValue != "*" {
		return c.allowHeadersValue
	}

	if requested := ctx.Header(requestHeadersHeader); requested != "" {
		return requested
	}

	return c.allowHeadersValue
}

// Handler method returns the Iris CORS Handler with basic features.
// Note that the caller should NOT modify any of the CORS instance fields afterwards.
func (c *CORS) Handler() context.Handler {
	return func(ctx *context.Context) {
		c.addVaryHeaders(ctx) // add vary headers at any case.

		origin := ""
		if c.skipLocalRequests && ctx.IsLocal() {
			// Skip local requests, no origin checks, but we should set the headers for browsers.
			referer := strings.TrimSuffix(ctx.Request().Referer(), "/")
			if referer == "" {
				scheme := ctx.Request().URL.Scheme
				if scheme == "" {
					scheme = "http"
				}

				host := ctx.Request().URL.Host
				if host == "" {
					host = "localhost"
				}

				referer = fmt.Sprintf("%s://%s", scheme, host)
			}

			origin = referer
		} else {
			extractedOrigin, ok := c.extractOriginFunc(ctx)
			if !ok {
				c.errorHandler(ctx, ErrOriginNotAllowed)
				return
			}

			if extractedOrigin == "" {
				// Not a CORS request at all: a plain navigation, a health check,
				// curl, or any same-origin GET - none of which send an Origin
				// header. Answer with the wildcard, as before, but do NOT run it
				// through the origin allowlist.
				//
				// Note(@kataras): the substituted "*" used to be validated like a
				// real origin, and no allowlist pattern matches "*". So the moment
				// anyone called AllowOrigin/AllowOrigins, every ordinary browser
				// navigation and every health check started answering 403 - which
				// pushed people back to AllowAnyOrigin to get their site working.
				//
				// Whether these requests are acceptable at all is the extractor's
				// call: DefaultOriginExtractor returns ("", true) to permit them,
				// StrictOriginExtractor returns ("", false) to reject them, and the
				// !ok branch above already handled that.
				origin = "*"
			} else {
				if !c.allowOriginFunc(ctx, extractedOrigin) {
					c.errorHandler(ctx, ErrOriginNotAllowed)
					return
				}

				origin = extractedOrigin
			}
		}

		ctx.SetHeader(allowOriginHeader, origin)
		// Advertise credentials for a specific origin as before. For a wildcard
		// ("*") origin, suppress them by default because browsers reject the
		// "*" + credentials combination; advertise them only when the developer
		// explicitly opted in via AllowCredentials.
		if origin != "*" || c.allowCredentialsExplicit {
			ctx.SetHeader(allowCredentialsHeader, c.allowCredentialsValue)
		}
		// 08 July 2021 Mozzila updated the following document: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
		ctx.SetHeader(referrerPolicyHeader, c.referrerPolicyValue)
		ctx.SetHeader(exposeHeadersHeader, c.exposeHeadersValue)
		if ctx.Method() == http.MethodOptions {
			ctx.SetHeader(allowMethodsHeader, allowAllMethodsValue)
			ctx.SetHeader(allowHeadersHeader, c.allowedHeadersFor(ctx))
			ctx.SetHeader(maxAgeHeader, c.maxAgeSecondsValue)
			ctx.StatusCode(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
