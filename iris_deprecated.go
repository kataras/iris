// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iris

// The below code will be removed at the next release.
// It's here to make your overall experience more familiar with the APIs you used before Iris.
import (
	"net"
	"net/url"
	"os"

	"github.com/cdren/iris/cache"
	"github.com/cdren/iris/context"
	"github.com/cdren/iris/core/handlerconv"
	"github.com/cdren/iris/core/host"
	"github.com/cdren/iris/core/nettools"
)

// ToHandler converts native http.Handler & http.HandlerFunc to context.Handler.
//
// Supported form types:
// 		 .ToHandler(h http.Handler)
// 		 .ToHandler(func(w http.ResponseWriter, r *http.Request))
// 		 .ToHandler(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc))
//
// Deprecated. Use the "core/handlerconv" package instead, equivalent to `ToHandler` is its `FromStd` function.
func ToHandler(handler interface{}) context.Handler {
	return handlerconv.FromStd(handler)
}

// Cache provides cache capabilities to a route's handler.
// Usage:
// Get("/", iris.Cache(time.Duration(10*time.Second)), func(ctx context.Context){
//    ctx.Writef("Hello, world!") // or a template or anything else
// })
//
// Deprecated. Use "github.com/cdren/iris/cache" sub-package which contains the full features instead.
var Cache = cache.Handler

// CheckErr is the old `Must`. It panics on errors as expected with
// the old listen functions, change of this method will affect only ListenXXX functions.
//
// Its only callers are the following, deprecated, listen functions.
var CheckErr = func(err error) {
	if err != nil {
		panic(err)
	}
}

// Serve serves incoming connections from the given listener.
//
// Serve blocks until the given listener returns permanent error.
//
// Deprecated. Use `Run` instead.
func (app *Application) Serve(l net.Listener) error {
	return app.Run(Listener(l))
}

// Listen starts the standalone http server
// which listens to the addr parameter which as the form of
// host:port
//
// Deprecated. Use `Run` instead.
func (app *Application) Listen(addr string) {
	CheckErr(app.Run(Addr(addr)))
}

// ListenUNIX starts the server listening to the new requests using a 'socket file'.
// Note: this works only on unix.
//
// Deprecated. Use `Run` instead.
func (app *Application) ListenUNIX(socketFile string, mode os.FileMode) {
	l, err := nettools.UNIX(socketFile, mode)
	CheckErr(err)

	CheckErr(app.Run(Listener(l)))
}

// ListenTLS starts the secure server with provided certificates,
// if you use this method the requests of the form of 'http://' will fail
// only https:// connections are allowed
// which listens to the addr parameter which as the form of
// host:port
//
//
// Deprecated. Use `Run` instead.
func (app *Application) ListenTLS(addr string, certFile, keyFile string) {
	CheckErr(app.Run(TLS(addr, certFile, keyFile)))
}

// ListenLETSENCRYPT starts the server listening at the specific nat address
// using key & certification taken from the letsencrypt.org 's servers
// it's also starts a second 'http' server to redirect all 'http://$ADDR/$PATH' to the' https://$ADDR/$PATH'
// it creates a cache file to store the certifications, for performance reasons, this file by-default is "./certcache"
// if you skip the second parameter then the cache file is "./letsencrypt.cache"
// if you want to disable cache then simple pass as second argument an empty empty string ""
//
// Deprecated. Use `Run` instead.
func (app *Application) ListenLETSENCRYPT(addr string, cacheDirOptional ...string) {
	l, err := nettools.LETSENCRYPT(addr, addr, cacheDirOptional...)
	CheckErr(err)
	// create the redirect server to redirect http://... to https://...
	hostname := nettools.ResolveHostname(addr)
	proxyAddr := hostname + ":80"
	target, err := url.Parse("https://" + hostname)
	CheckErr(err)
	go host.NewProxy(proxyAddr, target).ListenAndServe()

	CheckErr(app.Run(Listener(l)))
}

// OnStatusCode registers an error http status code
// based on the "statusCode" >= 400.
// The handler is being wrapepd by a generic
// handler which will try to reset
// the body if recorder was enabled
// and/or disable the gzip if gzip response recorder
// was active.
//
// Deprecated. Use `OnErrorCode` instead.
func (app *Application) OnStatusCode(statusCode int, handler context.Handler) {
	app.OnErrorCode(statusCode, handler)
}

// HTTP status codes as registered with IANA.
// See: http://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
// Raw Copy from the net/http std package in order to recude the import path of "net/http" for the users.
//
// These may or may not stay.
const (
	StatusContinue           = 100 // RFC 7231, 6.2.1
	StatusSwitchingProtocols = 101 // RFC 7231, 6.2.2
	StatusProcessing         = 102 // RFC 2518, 10.1

	StatusOK                   = 200 // RFC 7231, 6.3.1
	StatusCreated              = 201 // RFC 7231, 6.3.2
	StatusAccepted             = 202 // RFC 7231, 6.3.3
	StatusNonAuthoritativeInfo = 203 // RFC 7231, 6.3.4
	StatusNoContent            = 204 // RFC 7231, 6.3.5
	StatusResetContent         = 205 // RFC 7231, 6.3.6
	StatusPartialContent       = 206 // RFC 7233, 4.1
	StatusMultiStatus          = 207 // RFC 4918, 11.1
	StatusAlreadyReported      = 208 // RFC 5842, 7.1
	StatusIMUsed               = 226 // RFC 3229, 10.4.1

	StatusMultipleChoices   = 300 // RFC 7231, 6.4.1
	StatusMovedPermanently  = 301 // RFC 7231, 6.4.2
	StatusFound             = 302 // RFC 7231, 6.4.3
	StatusSeeOther          = 303 // RFC 7231, 6.4.4
	StatusNotModified       = 304 // RFC 7232, 4.1
	StatusUseProxy          = 305 // RFC 7231, 6.4.5
	_                       = 306 // RFC 7231, 6.4.6 (Unused)
	StatusTemporaryRedirect = 307 // RFC 7231, 6.4.7
	StatusPermanentRedirect = 308 // RFC 7538, 3

	StatusBadRequest                   = 400 // RFC 7231, 6.5.1
	StatusUnauthorized                 = 401 // RFC 7235, 3.1
	StatusPaymentRequired              = 402 // RFC 7231, 6.5.2
	StatusForbidden                    = 403 // RFC 7231, 6.5.3
	StatusNotFound                     = 404 // RFC 7231, 6.5.4
	StatusMethodNotAllowed             = 405 // RFC 7231, 6.5.5
	StatusNotAcceptable                = 406 // RFC 7231, 6.5.6
	StatusProxyAuthRequired            = 407 // RFC 7235, 3.2
	StatusRequestTimeout               = 408 // RFC 7231, 6.5.7
	StatusConflict                     = 409 // RFC 7231, 6.5.8
	StatusGone                         = 410 // RFC 7231, 6.5.9
	StatusLengthRequired               = 411 // RFC 7231, 6.5.10
	StatusPreconditionFailed           = 412 // RFC 7232, 4.2
	StatusRequestEntityTooLarge        = 413 // RFC 7231, 6.5.11
	StatusRequestURITooLong            = 414 // RFC 7231, 6.5.12
	StatusUnsupportedMediaType         = 415 // RFC 7231, 6.5.13
	StatusRequestedRangeNotSatisfiable = 416 // RFC 7233, 4.4
	StatusExpectationFailed            = 417 // RFC 7231, 6.5.14
	StatusTeapot                       = 418 // RFC 7168, 2.3.3
	StatusUnprocessableEntity          = 422 // RFC 4918, 11.2
	StatusLocked                       = 423 // RFC 4918, 11.3
	StatusFailedDependency             = 424 // RFC 4918, 11.4
	StatusUpgradeRequired              = 426 // RFC 7231, 6.5.15
	StatusPreconditionRequired         = 428 // RFC 6585, 3
	StatusTooManyRequests              = 429 // RFC 6585, 4
	StatusRequestHeaderFieldsTooLarge  = 431 // RFC 6585, 5
	StatusUnavailableForLegalReasons   = 451 // RFC 7725, 3

	StatusInternalServerError           = 500 // RFC 7231, 6.6.1
	StatusNotImplemented                = 501 // RFC 7231, 6.6.2
	StatusBadGateway                    = 502 // RFC 7231, 6.6.3
	StatusServiceUnavailable            = 503 // RFC 7231, 6.6.4
	StatusGatewayTimeout                = 504 // RFC 7231, 6.6.5
	StatusHTTPVersionNotSupported       = 505 // RFC 7231, 6.6.6
	StatusVariantAlsoNegotiates         = 506 // RFC 2295, 8.1
	StatusInsufficientStorage           = 507 // RFC 4918, 11.5
	StatusLoopDetected                  = 508 // RFC 5842, 7.2
	StatusNotExtended                   = 510 // RFC 2774, 7
	StatusNetworkAuthenticationRequired = 511 // RFC 6585, 6
)

// These may or may not stay, you can use net/http's constants too.
const (
	MethodGet     = "GET"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodHead    = "HEAD"
	MethodPatch   = "PATCH"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
	// MethodNone is declared at iris.go, it will stay.
)
