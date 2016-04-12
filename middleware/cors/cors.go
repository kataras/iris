// Copyright (c) 2016, Iris Team
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Cors credits goes to @keuller

package cors

import (
	"github.com/kataras/iris"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const toLower = 'a' - 'A'

type converter func(string) string

type CorsOptions struct {
	// AllowedOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// An origin may contain a wildcard (*) to replace 0 or more characters
	// (i.e.: http://*.domain.com). Usage of wildcards implies a small performance penality.
	// Only one wildcard can be used per origin.
	// Default value is ["*"]
	AllowedOrigins []string
	// AllowOriginFunc is a custom function to validate the origin. It take the origin
	// as argument and returns true if allowed or false otherwise. If this option is
	// set, the content of AllowedOrigins is ignored.
	AllowOriginFunc func(origin string) bool
	// AllowedMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is simple methods (GET and POST)
	AllowedMethods []string
	// AllowedHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	// If the special "*" value is present in the list, all headers will be allowed.
	// Default value is [] but "Origin" is always appended to the list.
	AllowedHeaders []string

	AllowedHeadersAll bool

	// ExposedHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification
	ExposedHeaders []string
	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool
	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached
	MaxAge int
	// OptionsPassthrough instructs preflight to let other potential next handlers to
	// process the OPTIONS method. Turn this on if your application handles OPTIONS.
	OptionsPassthrough bool
	// Debugging flag adds additional output to debug server side CORS issues
	Debug bool
}

type cors struct {
	Log     *log.Logger
	Options CorsOptions
}

func (c *cors) Serve(ctx *iris.Context) {
	if ctx.MethodString() == "OPTIONS" {
		c.handlePreflight(ctx)
		// Preflight requests are standalone and should stop the chain as some other
		// middleware may not handle OPTIONS requests correctly. One typical example
		// is authentication middleware ; OPTIONS requests won't carry authentication
		// headers (see #1)
		if c.Options.OptionsPassthrough {
			ctx.Next()
		}
		return
	}

	c.handleActualRequest(ctx)
	ctx.Next()
}

func New(opts CorsOptions) *cors {
	c := &cors{nil, opts}

	if opts.Debug {
		c.Log = log.New(os.Stdout, "[iris::cors] ", log.LstdFlags)
	}

	// Allowed Headers
	if len(opts.AllowedHeaders) == 0 {
		// Use sensible defaults
		c.Options.AllowedHeaders = []string{"Origin", "Accept", "Content-Type"}
	} else {
		// Origin is always appended as some browsers will always request for this header at preflight
		c.Options.AllowedHeaders = convert(append(opts.AllowedHeaders, "Origin"), http.CanonicalHeaderKey)
		for _, h := range opts.AllowedHeaders {
			if h == "*" {
				c.Options.AllowedHeadersAll = true
				c.Options.AllowedHeaders = nil
				break
			}
		}
	}

	// Allowed Methods
	if len(opts.AllowedMethods) == 0 {
		// Default is spec's "simple" methods
		c.Options.AllowedMethods = []string{"GET", "POST"}
	} else {
		c.Options.AllowedMethods = convert(opts.AllowedMethods, strings.ToUpper)
	}

	return c
}

func DefaultCors() *cors {
	return New(CorsOptions{})
}

func Cors() *cors {
	return DefaultCors()
}

// handlePreflight handles pre-flight CORS requests
func (c *cors) handlePreflight(ctx *iris.Context) {
	r := ctx.RequestCtx.Request
	headers := ctx.RequestCtx.Response.Header
	origin := iris.BytesToString(r.Header.Peek("Origin"))

	if ctx.MethodString() != "OPTIONS" {
		// c.logf("  Preflight aborted: %s!=OPTIONS", r.Method)
		return
	}
	// Always set Vary headers
	// see https://github.com/rs/cors/issues/10,
	//     https://github.com/rs/cors/commit/dbdca4d95feaa7511a46e6f1efb3b3aa505bc43f#commitcomment-12352001
	headers.Add("Vary", "Origin")
	headers.Add("Vary", "Access-Control-Request-Method")
	headers.Add("Vary", "Access-Control-Request-Headers")

	if origin == "" {
		// c.logf("  Preflight aborted: empty origin")
		return
	}
	if !c.isOriginAllowed(origin) {
		// c.logf("  Preflight aborted: origin '%s' not allowed", origin)
		return
	}

	reqMethod := iris.BytesToString(r.Header.Peek("Access-Control-Request-Method"))
	if !c.IsMethodAllowed(reqMethod) {
		// c.logf("  Preflight aborted: method '%s' not allowed", reqMethod)
		return
	}
	reqHeaders := parseHeaderList(iris.BytesToString(r.Header.Peek("Access-Control-Request-Headers")))
	if !c.areHeadersAllowed(reqHeaders) {
		// c.logf("  Preflight aborted: headers '%v' not allowed", reqHeaders)
		return
	}
	headers.Set("Access-Control-Allow-Origin", origin)
	// Spec says: Since the list of methods can be unbounded, simply returning the method indicated
	// by Access-Control-Request-Method (if supported) can be enough
	headers.Set("Access-Control-Allow-Methods", strings.ToUpper(reqMethod))
	if len(reqHeaders) > 0 {
		// Spec says: Since the list of headers can be unbounded, simply returning supported headers
		// from Access-Control-Request-Headers can be enough
		headers.Set("Access-Control-Allow-Headers", strings.Join(reqHeaders, ", "))
	}
	if c.Options.AllowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
	if c.Options.MaxAge > 0 {
		headers.Set("Access-Control-Max-Age", strconv.Itoa(c.Options.MaxAge))
	}
}

// handleActualRequest handles simple cross-origin requests, actual request or redirects
func (c *cors) handleActualRequest(ctx *iris.Context) {
	r := ctx.Request
	headers := ctx.RequestCtx.Response.Header
	origin := iris.BytesToString(r.Header.Peek("Origin"))

	if ctx.MethodString() == "OPTIONS" {
		c.logf("  Actual request no headers added: method == %s", ctx.MethodString())
		return
	}

	// Always set Vary, see https://github.com/rs/cors/issues/10
	headers.Add("Vary", "Origin")
	if origin == "" {
		c.logf("  Actual request no headers added: missing origin")
		return
	}

	if !c.isOriginAllowed(origin) {
		c.logf("  Actual request no headers added: origin '%s' not allowed", origin)
		return
	}

	// Note that spec does define a way to specifically disallow a simple method like GET or
	// POST. Access-Control-Allow-Methods is only used for pre-flight requests and the
	// spec doesn't instruct to check the allowed methods for simple cross-origin requests.
	// We think it's a nice feature to be able to have control on those methods though.
	if !c.IsMethodAllowed(ctx.MethodString()) {
		c.logf("  Actual request no headers added: method '%s' not allowed", ctx.MethodString())
		return
	}

	headers.Set("Access-Control-Allow-Origin", origin)
	if len(c.Options.ExposedHeaders) > 0 {
		headers.Set("Access-Control-Expose-Headers", strings.Join(c.Options.ExposedHeaders, ", "))
	}

	if c.Options.AllowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
	c.logf("  Actual response added headers: %v", headers)
}

// convenience method. checks if debugging is turned on before printing
func (c *cors) logf(format string, a ...interface{}) {
	if c.Log != nil {
		c.Log.Printf(format, a...)
	}
}

// isOriginAllowed checks if a given origin is allowed to perform cross-domain requests
// on the endpoint
func (c *cors) isOriginAllowed(origin string) bool {
	if c.Options.AllowOriginFunc != nil {
		return c.Options.AllowOriginFunc(origin)
	}
	if len(c.Options.AllowedOrigins) == 0 {
		return true
	}
	origin = strings.ToLower(origin)
	for _, o := range c.Options.AllowedOrigins {
		if o == origin {
			return true
		}
	}
	// for _, w := range c.Options.AllowedOrigins {
	// 	if w.match(origin) {
	// 		return true
	// 	}
	// }
	return false
}

// IsMethodAllowed checks if a given method can be used as part of a cross-domain request
// on the endpoing
// Capitalize by @thesyncim
func (c *cors) IsMethodAllowed(method string) bool {
	if len(c.Options.AllowedMethods) == 0 {
		// If no method allowed, always return false, even for preflight request
		return false
	}
	method = strings.ToUpper(method)
	if method == "OPTIONS" {
		// Always allow preflight requests
		return true
	}
	for _, m := range c.Options.AllowedMethods {
		if m == method {
			return true
		}
	}
	return false
}

// areHeadersAllowed checks if a given list of headers are allowed to used within
// a cross-domain request.
func (c *cors) areHeadersAllowed(requestedHeaders []string) bool {
	if c.Options.AllowedHeadersAll || len(requestedHeaders) == 0 {
		return true
	}
	for _, header := range requestedHeaders {
		header = http.CanonicalHeaderKey(header)
		found := false
		for _, h := range c.Options.AllowedHeaders {
			if h == header {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// parseHeaderList tokenize + normalize a string containing a list of headers
func parseHeaderList(headerList string) []string {
	l := len(headerList)
	h := make([]byte, 0, l)
	upper := true
	// Estimate the number headers in order to allocate the right splice size
	t := 0
	for i := 0; i < l; i++ {
		if headerList[i] == ',' {
			t++
		}
	}
	headers := make([]string, 0, t)
	for i := 0; i < l; i++ {
		b := headerList[i]
		if b >= 'a' && b <= 'z' {
			if upper {
				h = append(h, b-toLower)
			} else {
				h = append(h, b)
			}
		} else if b >= 'A' && b <= 'Z' {
			if !upper {
				h = append(h, b+toLower)
			} else {
				h = append(h, b)
			}
		} else if b == '-' || b == '_' || (b >= '0' && b <= '9') {
			h = append(h, b)
		}

		if b == ' ' || b == ',' || i == l-1 {
			if len(h) > 0 {
				// Flush the found header
				headers = append(headers, string(h))
				h = h[:0]
				upper = true
			}
		} else {
			upper = b == '-' || b == '_'
		}
	}
	return headers
}

// convert converts a list of string using the passed converter function
func convert(s []string, c converter) []string {
	out := []string{}
	for _, i := range s {
		out = append(out, c(i))
	}
	return out
}
