package iris

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/geekypanda/httpcache"
	"github.com/iris-contrib/letsencrypt"
	"github.com/kataras/go-errors"
	"golang.org/x/crypto/acme/autocert"
)

const (
	// MethodGet "GET"
	MethodGet = "GET"
	// MethodPost "POST"
	MethodPost = "POST"
	// MethodPut "PUT"
	MethodPut = "PUT"
	// MethodDelete "DELETE"
	MethodDelete = "DELETE"
	// MethodConnect "CONNECT"
	MethodConnect = "CONNECT"
	// MethodHead "HEAD"
	MethodHead = "HEAD"
	// MethodPatch "PATCH"
	MethodPatch = "PATCH"
	// MethodOptions "OPTIONS"
	MethodOptions = "OPTIONS"
	// MethodTrace "TRACE"
	MethodTrace = "TRACE"
	// MethodNone is a Virtual method
	// to store the "offline" routes
	// in the mux's tree
	MethodNone = "NONE"
)

var (
	// AllMethods contains all the http valid methods:
	// "GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE"
	AllMethods = [...]string{MethodGet, MethodPost, MethodPut, MethodDelete, MethodConnect, MethodHead, MethodPatch, MethodOptions, MethodTrace}
)

// HTTP status codes.
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

var statusText = map[int]string{
	StatusContinue:           "Continue",
	StatusSwitchingProtocols: "Switching Protocols",
	StatusProcessing:         "Processing",

	StatusOK:                   "OK",
	StatusCreated:              "Created",
	StatusAccepted:             "Accepted",
	StatusNonAuthoritativeInfo: "Non-Authoritative Information",
	StatusNoContent:            "No Content",
	StatusResetContent:         "Reset Content",
	StatusPartialContent:       "Partial Content",
	StatusMultiStatus:          "Multi-Status",
	StatusAlreadyReported:      "Already Reported",
	StatusIMUsed:               "IM Used",

	StatusMultipleChoices:   "Multiple Choices",
	StatusMovedPermanently:  "Moved Permanently",
	StatusFound:             "Found",
	StatusSeeOther:          "See Other",
	StatusNotModified:       "Not Modified",
	StatusUseProxy:          "Use Proxy",
	StatusTemporaryRedirect: "Temporary Redirect",
	StatusPermanentRedirect: "Permanent Redirect",

	StatusBadRequest:                   "Bad Request",
	StatusUnauthorized:                 "Unauthorized",
	StatusPaymentRequired:              "Payment Required",
	StatusForbidden:                    "Forbidden",
	StatusNotFound:                     "Not Found",
	StatusMethodNotAllowed:             "Method Not Allowed",
	StatusNotAcceptable:                "Not Acceptable",
	StatusProxyAuthRequired:            "Proxy Authentication Required",
	StatusRequestTimeout:               "Request Timeout",
	StatusConflict:                     "Conflict",
	StatusGone:                         "Gone",
	StatusLengthRequired:               "Length Required",
	StatusPreconditionFailed:           "Precondition Failed",
	StatusRequestEntityTooLarge:        "Request Entity Too Large",
	StatusRequestURITooLong:            "Request URI Too Long",
	StatusUnsupportedMediaType:         "Unsupported Media Type",
	StatusRequestedRangeNotSatisfiable: "Requested Range Not Satisfiable",
	StatusExpectationFailed:            "Expectation Failed",
	StatusTeapot:                       "I'm a teapot",
	StatusUnprocessableEntity:          "Unprocessable Entity",
	StatusLocked:                       "Locked",
	StatusFailedDependency:             "Failed Dependency",
	StatusUpgradeRequired:              "Upgrade Required",
	StatusPreconditionRequired:         "Precondition Required",
	StatusTooManyRequests:              "Too Many Requests",
	StatusRequestHeaderFieldsTooLarge:  "Request Header Fields Too Large",
	StatusUnavailableForLegalReasons:   "Unavailable For Legal Reasons",

	StatusInternalServerError:           "Internal Server Error",
	StatusNotImplemented:                "Not Implemented",
	StatusBadGateway:                    "Bad Gateway",
	StatusServiceUnavailable:            "Service Unavailable",
	StatusGatewayTimeout:                "Gateway Timeout",
	StatusHTTPVersionNotSupported:       "HTTP Version Not Supported",
	StatusVariantAlsoNegotiates:         "Variant Also Negotiates",
	StatusInsufficientStorage:           "Insufficient Storage",
	StatusLoopDetected:                  "Loop Detected",
	StatusNotExtended:                   "Not Extended",
	StatusNetworkAuthenticationRequired: "Network Authentication Required",
}

// StatusText returns a text for the HTTP status code. It returns the empty
// string if the code is unknown.
func StatusText(code int) string {
	return statusText[code]
}

// errHandler returns na error with message: 'Passed argument is not func(*Context) neither an object which implements the iris.Default.Handler with Serve(ctx *Context)
// It seems to be a  +type Points to: +pointer.'
var errHandler = errors.New(`
Passed argument is not an iris.Handler (or func(*iris.Context)) neither one of these types:
  - http.Handler
  - func(w http.ResponseWriter, r *http.Request)
  - func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)
  ---------------------------------------------------------------------
It seems to be a  %T points to: %v`)

type (
	// Handler the main Iris Handler interface.
	Handler interface {
		Serve(ctx *Context) // iris-specific
	}

	// HandlerFunc type is an adapter to allow the use of
	// ordinary functions as HTTP handlers.  If f is a function
	// with the appropriate signature, HandlerFunc(f) is a
	// Handler that calls f.
	HandlerFunc func(*Context)
	// Middleware is just a slice of Handler []func(c *Context)
	Middleware []Handler

	// HandlerAPI empty interface used for .API
	HandlerAPI interface{}
)

// Serve implements the Handler, is like ServeHTTP but for Iris
func (h HandlerFunc) Serve(ctx *Context) {
	h(ctx)
}

// ToNativeHandler converts an iris handler to http.Handler
func ToNativeHandler(s *Framework, h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := s.AcquireCtx(w, r)
		h.Serve(ctx)
		s.ReleaseCtx(ctx)
	})
}

// ToHandler converts different type styles of handlers that you
// used to use (usually with third-party net/http middleware) to an iris.HandlerFunc.
//
// Supported types:
// - .ToHandler(h http.Handler)
// - .ToHandler(func(w http.ResponseWriter, r *http.Request))
// - .ToHandler(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc))
func ToHandler(handler interface{}) HandlerFunc {
	switch handler.(type) {
	case HandlerFunc:
		{
			//
			//it's already an iris handler
			//
			return handler.(HandlerFunc)
		}

	case http.Handler:
		//
		// handlerFunc.ServeHTTP(w,r)
		//
		{
			h := handler.(http.Handler)
			return func(ctx *Context) {
				h.ServeHTTP(ctx.ResponseWriter, ctx.Request)
			}
		}

	case func(http.ResponseWriter, *http.Request):
		{
			//
			// handlerFunc(w,r)
			//
			return ToHandler(http.HandlerFunc(handler.(func(http.ResponseWriter, *http.Request))))
		}

	case func(http.ResponseWriter, *http.Request, http.HandlerFunc):
		{
			//
			// handlerFunc(w,r, http.HandlerFunc)
			//
			return toHandlerNextHTTPHandlerFunc(handler.(func(http.ResponseWriter, *http.Request, http.HandlerFunc)))
		}

	default:
		{
			//
			// No valid handler passed
			//
			panic(errHandler.Format(handler, handler))
		}

	}

}

func toHandlerNextHTTPHandlerFunc(h func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)) HandlerFunc {
	return HandlerFunc(func(ctx *Context) {
		// take the next handler in route's chain
		nextIrisHandler := ctx.NextHandler()
		if nextIrisHandler != nil {
			executed := false // we need to watch this in order to StopExecution from all next handlers
			// if this next handler is not executed by the third-party net/http next-style middleware.
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextIrisHandler.Serve(ctx)
				executed = true
			})

			h(ctx.ResponseWriter, ctx.Request, nextHandler)

			// after third-party middleware's job:
			if executed {
				// if next is executed then increment the ctx.Pos manually
				// in order to the next handler not to be executed twice.
				ctx.Pos++
			} else {
				// otherwise StopExecution from all next handlers.
				ctx.StopExecution()
			}
			return
		}

		// if not next handler found then this is not a 'valid' middleware but
		// some middleware may don't care about next,
		// so we just execute the handler with an empty net.
		h(ctx.ResponseWriter, ctx.Request, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	})
}

// convertToHandlers just make []HandlerFunc to []Handler, although HandlerFunc and Handler are the same
// we need this on some cases we explicit want a interface Handler, it is useless for users.
func convertToHandlers(handlersFn []HandlerFunc) []Handler {
	hlen := len(handlersFn)
	mlist := make([]Handler, hlen)
	for i := 0; i < hlen; i++ {
		mlist[i] = Handler(handlersFn[i])
	}
	return mlist
}

// joinMiddleware uses to create a copy of all middleware and return them in order to use inside the node
func joinMiddleware(middleware1 Middleware, middleware2 Middleware) Middleware {
	nowLen := len(middleware1)
	totalLen := nowLen + len(middleware2)
	// create a new slice of middleware in order to store all handlers, the already handlers(middleware) and the new
	newMiddleware := make(Middleware, totalLen)
	//copy the already middleware to the just created
	copy(newMiddleware, middleware1)
	//start from there we finish, and store the new middleware too
	copy(newMiddleware[nowLen:], middleware2)
	return newMiddleware
}

const (
	// parameterStartByte is very used on the node, it's just contains the byte for the ':' rune/char
	parameterStartByte = byte(':')
	// slashByte is just a byte of '/' rune/char
	slashByte = byte('/')
	// slash is just a string of "/"
	slash = "/"
	// matchEverythingByte is just a byte of '*" rune/char
	matchEverythingByte = byte('*')

	isRoot entryCase = iota
	hasParams
	matchEverything
)

type (
	// entryCase is the type which the type of muxEntryusing in order to determinate what type (parameterized, anything, static...) is the perticular node
	entryCase uint8

	// muxEntry is the node of a tree of the routes,
	// in order to learn how this is working, google 'trie' or watch this lecture: https://www.youtube.com/watch?v=uhAUk63tLRM
	// this method is used by the BSD's kernel also
	muxEntry struct {
		part        string
		entryCase   entryCase
		hasWildNode bool
		tokens      string
		nodes       []*muxEntry
		middleware  Middleware
		precedence  uint64
		paramsLen   uint8
	}
)

var (
	errMuxEntryConflictsWildcard           = errors.New("Router: Path's part: '%s' conflicts with wildcard '%s' in the route path: '%s' !")
	errMuxEntryMiddlewareAlreadyExists     = errors.New("Router: Middleware were already registered for the path: '%s' !")
	errMuxEntryInvalidWildcard             = errors.New("Router: More than one wildcard found in the path part: '%s' in route's path: '%s' !")
	errMuxEntryConflictsExistingWildcard   = errors.New("Router: Wildcard for route path: '%s' conflicts with existing children in route path: '%s' !")
	errMuxEntryWildcardUnnamed             = errors.New("Router: Unnamed wildcard found in path: '%s' !")
	errMuxEntryWildcardInvalidPlace        = errors.New("Router: Wildcard is only allowed at the end of the path, in the route path: '%s' !")
	errMuxEntryWildcardConflictsMiddleware = errors.New("Router: Wildcard  conflicts with existing middleware for the route path: '%s' !")
	errMuxEntryWildcardMissingSlash        = errors.New("Router: No slash(/) were found before wildcard in the route path: '%s' !")
)

// getParamsLen returns the parameters length from a given path
func getParamsLen(path string) uint8 {
	var n uint
	for i := 0; i < len(path); i++ {
		if path[i] != ':' && path[i] != '*' { // ParameterStartByte & MatchEverythingByte
			continue
		}
		n++
	}
	if n >= 255 {
		return 255
	}
	return uint8(n)
}

// findLower returns the smaller number between a and b
func findLower(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// add adds a muxEntry to the existing muxEntry or to the tree if no muxEntry has the prefix of
func (e *muxEntry) add(path string, middleware Middleware) error {
	fullPath := path
	e.precedence++
	numParams := getParamsLen(path)

	if len(e.part) > 0 || len(e.nodes) > 0 {
	loop:
		for {
			if numParams > e.paramsLen {
				e.paramsLen = numParams
			}

			i := 0
			max := findLower(len(path), len(e.part))
			for i < max && path[i] == e.part[i] {
				i++
			}

			if i < len(e.part) {
				node := muxEntry{
					part:        e.part[i:],
					hasWildNode: e.hasWildNode,
					tokens:      e.tokens,
					nodes:       e.nodes,
					middleware:  e.middleware,
					precedence:  e.precedence - 1,
				}

				for i := range node.nodes {
					if node.nodes[i].paramsLen > node.paramsLen {
						node.paramsLen = node.nodes[i].paramsLen
					}
				}

				e.nodes = []*muxEntry{&node}
				e.tokens = string([]byte{e.part[i]})
				e.part = path[:i]
				e.middleware = nil
				e.hasWildNode = false
			}

			if i < len(path) {
				path = path[i:]

				if e.hasWildNode {
					e = e.nodes[0]
					e.precedence++

					if numParams > e.paramsLen {
						e.paramsLen = numParams
					}
					numParams--

					if len(path) >= len(e.part) && e.part == path[:len(e.part)] {

						if len(e.part) >= len(path) || path[len(e.part)] == slashByte {
							continue loop
						}
					}
					return errMuxEntryConflictsWildcard.Format(path, e.part, fullPath)
				}

				c := path[0]

				if e.entryCase == hasParams && c == slashByte && len(e.nodes) == 1 {
					e = e.nodes[0]
					e.precedence++
					continue loop
				}
				for i := range e.tokens {
					if c == e.tokens[i] {
						i = e.precedenceTo(i)
						e = e.nodes[i]
						continue loop
					}
				}

				if c != parameterStartByte && c != matchEverythingByte {

					e.tokens += string([]byte{c})
					node := &muxEntry{
						paramsLen: numParams,
					}
					e.nodes = append(e.nodes, node)
					e.precedenceTo(len(e.tokens) - 1)
					e = node
				}
				e.addNode(numParams, path, fullPath, middleware)
				return nil

			} else if i == len(path) {
				if e.middleware != nil {
					return errMuxEntryMiddlewareAlreadyExists.Format(fullPath)
				}
				e.middleware = middleware
			}
			return nil
		}
	} else {
		e.addNode(numParams, path, fullPath, middleware)
		e.entryCase = isRoot
	}
	return nil
}

// addNode adds a muxEntry as children to other muxEntry
func (e *muxEntry) addNode(numParams uint8, path string, fullPath string, middleware Middleware) error {
	var offset int

	for i, max := 0, len(path); numParams > 0; i++ {
		c := path[i]
		if c != parameterStartByte && c != matchEverythingByte {
			continue
		}

		end := i + 1
		for end < max && path[end] != slashByte {
			switch path[end] {
			case parameterStartByte, matchEverythingByte:
				/*
				   panic("only one wildcard per path segment is allowed, has: '" +
				   	path[i:] + "' in path '" + fullPath + "'")
				*/
				return errMuxEntryInvalidWildcard.Format(path[i:], fullPath)
			default:
				end++
			}
		}

		if len(e.nodes) > 0 {
			return errMuxEntryConflictsExistingWildcard.Format(path[i:end], fullPath)
		}

		if end-i < 2 {
			return errMuxEntryWildcardUnnamed.Format(fullPath)
		}

		if c == parameterStartByte {

			if i > 0 {
				e.part = path[offset:i]
				offset = i
			}

			child := &muxEntry{
				entryCase: hasParams,
				paramsLen: numParams,
			}
			e.nodes = []*muxEntry{child}
			e.hasWildNode = true
			e = child
			e.precedence++
			numParams--

			if end < max {
				e.part = path[offset:end]
				offset = end

				child := &muxEntry{
					paramsLen:  numParams,
					precedence: 1,
				}
				e.nodes = []*muxEntry{child}
				e = child
			}

		} else {
			if end != max || numParams > 1 {
				return errMuxEntryWildcardInvalidPlace.Format(fullPath)
			}

			if len(e.part) > 0 && e.part[len(e.part)-1] == '/' {
				return errMuxEntryWildcardConflictsMiddleware.Format(fullPath)
			}

			i--
			if path[i] != slashByte {
				return errMuxEntryWildcardMissingSlash.Format(fullPath)
			}

			e.part = path[offset:i]

			child := &muxEntry{
				hasWildNode: true,
				entryCase:   matchEverything,
				paramsLen:   1,
			}
			e.nodes = []*muxEntry{child}
			e.tokens = string(path[i])
			e = child
			e.precedence++

			child = &muxEntry{
				part:       path[i:],
				entryCase:  matchEverything,
				paramsLen:  1,
				middleware: middleware,
				precedence: 1,
			}
			e.nodes = []*muxEntry{child}

			return nil
		}
	}

	e.part = path[offset:]
	e.middleware = middleware

	return nil
}

// get is used by the Router, it finds and returns the correct muxEntry for a path
func (e *muxEntry) get(path string, ctx *Context) (mustRedirect bool) {
loop:
	for {
		if len(path) > len(e.part) {
			if path[:len(e.part)] == e.part {
				path = path[len(e.part):]

				if !e.hasWildNode {
					c := path[0]
					for i := range e.tokens {
						if c == e.tokens[i] {
							e = e.nodes[i]
							continue loop
						}
					}

					mustRedirect = (path == slash && e.middleware != nil)
					return
				}

				e = e.nodes[0]
				switch e.entryCase {
				case hasParams:

					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					ctx.Set(e.part[1:], path[:end])

					if end < len(path) {
						if len(e.nodes) > 0 {
							path = path[end:]
							e = e.nodes[0]
							continue loop
						}

						mustRedirect = (len(path) == end+1)
						return
					}
					if ctx.Middleware = e.middleware; ctx.Middleware != nil {
						return
					} else if len(e.nodes) == 1 {
						e = e.nodes[0]
						mustRedirect = (e.part == slash && e.middleware != nil)
					}

					return

				case matchEverything:

					ctx.Set(e.part[2:], path)
					ctx.Middleware = e.middleware
					return

				default:
					return
				}
			}
		} else if path == e.part {
			if ctx.Middleware = e.middleware; ctx.Middleware != nil {
				return
			}

			if path == slash && e.hasWildNode && e.entryCase != isRoot {
				mustRedirect = true
				return
			}

			for i := range e.tokens {
				if e.tokens[i] == slashByte {
					e = e.nodes[i]
					mustRedirect = (len(e.part) == 1 && e.middleware != nil) ||
						(e.entryCase == matchEverything && e.nodes[0].middleware != nil)
					return
				}
			}

			return
		}

		mustRedirect = (path == slash) ||
			(len(e.part) == len(path)+1 && e.part[len(path)] == slashByte &&
				path == e.part[:len(e.part)-1] && e.middleware != nil)
		return
	}
}

// precedenceTo just adds the priority of this muxEntry by an index
func (e *muxEntry) precedenceTo(index int) int {
	e.nodes[index].precedence++
	_precedence := e.nodes[index].precedence

	newindex := index
	for newindex > 0 && e.nodes[newindex-1].precedence < _precedence {
		tmpN := e.nodes[newindex-1]
		e.nodes[newindex-1] = e.nodes[newindex]
		e.nodes[newindex] = tmpN

		newindex--
	}

	if newindex != index {
		e.tokens = e.tokens[:newindex] +
			e.tokens[index:index+1] +
			e.tokens[newindex:index] + e.tokens[index+1:]
	}

	return newindex
}

// cachedMuxEntry is just a wrapper for the Cache functionality
// it seems useless but I prefer to keep the cached handler on its own memory stack,
// reason:  no clojures hell in the Cache function
type cachedMuxEntry struct {
	cachedHandler http.Handler
}

func newCachedMuxEntry(s *Framework, bodyHandler HandlerFunc, expiration time.Duration) *cachedMuxEntry {
	httphandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := s.AcquireCtx(w, r)
		bodyHandler.Serve(ctx)
		s.ReleaseCtx(ctx)
	})

	cachedHandler := httpcache.Cache(httphandler, expiration)
	return &cachedMuxEntry{
		cachedHandler: cachedHandler,
	}
}

func (c *cachedMuxEntry) Serve(ctx *Context) {
	c.cachedHandler.ServeHTTP(ctx.ResponseWriter, ctx.Request)
}

type (
	// Route contains some useful information about a route
	Route interface {
		// Name returns the name of the route
		Name() string
		// Subdomain returns the subdomain,if any
		Subdomain() string
		// Method returns the http method
		Method() string
		// SetMethod sets the route's method
		// requires re-build of the iris.Router
		SetMethod(string)

		// Path returns the path
		Path() string

		// staticPath returns the static part of the path
		StaticPath() string

		// SetPath changes/sets the path for this route
		SetPath(string)
		// Middleware returns the slice of Handler([]Handler) registered to this route
		Middleware() Middleware
		// SetMiddleware changes/sets the middleware(handler(s)) for this route
		SetMiddleware(Middleware)
		// IsOnline returns true if the route is marked as "online" (state)
		IsOnline() bool
	}

	route struct {
		// if no name given then it's the subdomain+path
		name           string
		subdomain      string
		method         string
		path           string
		staticPath     string
		middleware     Middleware
		formattedPath  string
		formattedParts int
	}

	bySubdomain []*route
)

// Sorting happens when the mux's request handler initialized
func (s bySubdomain) Len() int {
	return len(s)
}
func (s bySubdomain) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s bySubdomain) Less(i, j int) bool {
	return len(s[i].Subdomain()) > len(s[j].Subdomain())
}

var _ Route = &route{}

func newRoute(method string, subdomain string, path string, middleware Middleware) *route {
	r := &route{name: path + subdomain, method: method, subdomain: subdomain, path: path, middleware: middleware}
	r.formatPath()
	r.calculateStaticPath()
	return r
}

func (r *route) formatPath() {
	// we don't care about performance here.
	n1Len := strings.Count(r.path, ":")
	isMatchEverything := len(r.path) > 0 && r.path[len(r.path)-1] == matchEverythingByte
	if n1Len == 0 && !isMatchEverything {
		// its a static
		return
	}
	if n1Len == 0 && isMatchEverything {
		//if we have something like: /mypath/anything/* -> /mypatch/anything/%v
		r.formattedPath = r.path[0:len(r.path)-2] + "%v"
		r.formattedParts++
		return
	}

	tempPath := r.path
	splittedN1 := strings.Split(r.path, "/")

	for _, v := range splittedN1 {
		if len(v) > 0 {
			if v[0] == ':' || v[0] == matchEverythingByte {
				r.formattedParts++
				tempPath = strings.Replace(tempPath, v, "%v", -1) // n1Len, but let it we don't care about performance here.
			}
		}

	}
	r.formattedPath = tempPath
}

func (r *route) calculateStaticPath() {
	for i := 0; i < len(r.path); i++ {
		if r.path[i] == matchEverythingByte || r.path[i] == parameterStartByte {
			r.staticPath = r.path[0 : i-1] // stop at the first dynamic path symbol and set the static path to its [0:previous]
			return
		}
	}
	// not a dynamic symbol found, set its static path to its path.
	r.staticPath = r.path
}

func (r *route) setName(newName string) Route {
	r.name = newName
	return r
}

func (r route) Name() string {
	return r.name
}

func (r route) Subdomain() string {
	return r.subdomain
}

func (r route) Method() string {
	return r.method
}

func (r *route) SetMethod(method string) {
	r.method = method
}

func (r route) Path() string {
	return r.path
}

func (r route) StaticPath() string {
	return r.staticPath
}

func (r *route) SetPath(s string) {
	r.path = s
}

func (r route) Middleware() Middleware {
	return r.middleware
}

func (r *route) SetMiddleware(m Middleware) {
	r.middleware = m
}

func (r route) IsOnline() bool {
	return r.method != MethodNone
}

// RouteConflicts checks for route's middleware conflicts
func RouteConflicts(r *route, with string) bool {
	for _, h := range r.middleware {
		if m, ok := h.(interface {
			Conflicts() string
		}); ok {
			if c := m.Conflicts(); c == with {
				return true
			}
		}
	}
	return false
}

func (r *route) hasCors() bool {
	return RouteConflicts(r, "httpmethod")
}

const (
	// subdomainIndicator where './' exists in a registered path then it contains subdomain
	subdomainIndicator = "./"
	// dynamicSubdomainIndicator where a registered path starts with '*.' then it contains a dynamic subdomain, if subdomain == "*." then its dynamic
	dynamicSubdomainIndicator = "*."
)

type (
	muxTree struct {
		method string
		// subdomain is empty for default-hostname routes,
		// ex: mysubdomain.
		subdomain string
		entry     *muxEntry
	}

	serveMux struct {
		garden        []*muxTree
		lookups       []*route
		maxParameters uint8

		onLookup func(Route)

		api           *muxAPI
		errorHandlers map[int]Handler
		logger        *log.Logger
		// the main server host's name, ex:  localhost, 127.0.0.1, 0.0.0.0, iris-go.com
		hostname string
		// if any of the trees contains not empty subdomain
		hosts bool
		// if false then the /something it's not the same as /something/
		// defaults to true
		correctPath bool
		// if enabled then the router checks and fires an error for 405 http status method not allowed too if no method compatible method was found
		// by default is false
		fireMethodNotAllowed bool
		mu                   sync.Mutex
	}
)

func newServeMux(logger *log.Logger) *serveMux {
	mux := &serveMux{
		lookups:              make([]*route, 0),
		errorHandlers:        make(map[int]Handler, 0),
		hostname:             DefaultServerHostname, // these are changing when the server is up
		correctPath:          !DefaultDisablePathCorrection,
		fireMethodNotAllowed: false,
		logger:               logger,
	}

	return mux
}

func (mux *serveMux) setHostname(h string) {
	mux.hostname = h
}

func (mux *serveMux) setCorrectPath(b bool) {
	mux.correctPath = b
}

func (mux *serveMux) setFireMethodNotAllowed(b bool) {
	mux.fireMethodNotAllowed = b
}

// registerError registers a handler to a http status
func (mux *serveMux) registerError(statusCode int, handler Handler) {
	mux.mu.Lock()
	func(statusCode int, handler Handler) {
		mux.errorHandlers[statusCode] = HandlerFunc(func(ctx *Context) {
			if w, ok := ctx.IsRecording(); ok {
				w.Reset()
			}
			ctx.SetStatusCode(statusCode)
			handler.Serve(ctx)
		})
	}(statusCode, handler)
	mux.mu.Unlock()
}

// fireError fires an error
func (mux *serveMux) fireError(statusCode int, ctx *Context) {
	mux.mu.Lock()
	errHandler := mux.errorHandlers[statusCode]
	if errHandler == nil {
		errHandler = HandlerFunc(func(ctx *Context) {
			if w, ok := ctx.IsRecording(); ok {
				w.Reset()
			}
			ctx.SetStatusCode(statusCode)
			ctx.WriteString(statusText[statusCode])
		})
		mux.errorHandlers[statusCode] = errHandler
	}
	mux.mu.Unlock()
	errHandler.Serve(ctx)
}

func (mux *serveMux) getTree(method string, subdomain string) *muxTree {
	for i := range mux.garden {
		t := mux.garden[i]
		if t.method == method && t.subdomain == subdomain {
			return t
		}
	}
	return nil
}

func (mux *serveMux) register(method string, subdomain string, path string, middleware Middleware) *route {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if subdomain != "" {
		mux.hosts = true
	}

	// add to the lookups, it's just a collection of routes information
	lookup := newRoute(method, subdomain, path, middleware)
	if mux.onLookup != nil {
		mux.onLookup(lookup)
	}
	mux.lookups = append(mux.lookups, lookup)

	return lookup

}

// build collects all routes info and adds them to the registry in order to be served from the request handler
// this happens once(except when a route changes its state) when server is setting the mux's handler.
func (mux *serveMux) build() (methodEqual func(string, string) bool) {

	sort.Sort(bySubdomain(mux.lookups))
	// clear them for any case
	// build may called internally to re-build the routes.
	// re-build happens from BuildHandler() when a route changes its state
	// from offline to online or from online to offline
	mux.garden = mux.garden[0:0]
	// this is not used anywhere for now, but keep it here.
	mux.maxParameters = 0

	for i := range mux.lookups {
		r := mux.lookups[i]
		// add to the registry tree
		tree := mux.getTree(r.method, r.subdomain)
		if tree == nil {
			//first time we register a route to this method with this domain
			tree = &muxTree{method: r.method, subdomain: r.subdomain, entry: &muxEntry{}}
			mux.garden = append(mux.garden, tree)
		}
		// I decide that it's better to explicit give subdomain and a path to it than registeredPath(mysubdomain./something) now its: subdomain: mysubdomain., path: /something
		// we have different tree for each of subdomains, now you can use everything you can use with the normal paths ( before you couldn't set /any/*path)
		if err := tree.entry.add(r.path, r.middleware); err != nil {
			mux.logger.Panic(err)
		}

		if mp := tree.entry.paramsLen; mp > mux.maxParameters {
			mux.maxParameters = mp
		}
	}

	methodEqual = func(reqMethod string, treeMethod string) bool {
		return reqMethod == treeMethod
	}
	// check for cors conflicts FIRST in order to put them in OPTIONS tree also
	for i := range mux.lookups {
		r := mux.lookups[i]
		if r.hasCors() {
			// cors middleware is updated also, ref: https://github.com/kataras/iris/issues/461
			methodEqual = func(reqMethod string, treeMethod string) bool {
				// preflights
				return reqMethod == MethodOptions || reqMethod == treeMethod
			}
			break
		}
	}

	return

}

func (mux *serveMux) lookup(routeName string) *route {
	for i := range mux.lookups {
		if r := mux.lookups[i]; r.name == routeName {
			return r
		}
	}
	return nil
}

//THESE ARE FROM Go Authors
var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	// "&#34;" is shorter than "&quot;".
	`"`, "&#34;",
	// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
	"'", "&#39;",
)

// HTMLEscape returns a string which has no valid html code
func HTMLEscape(s string) string {
	return htmlReplacer.Replace(s)
}

// BuildHandler the default Iris router when iris.Router is nil
//
// NOTE: Is called and re-set to the iris.Router when
// a route changes its state from "online" to "offline" or "offline" to "online"
// look iris.None(...) for more
// and: https://github.com/kataras/iris/issues/585
func (mux *serveMux) BuildHandler() HandlerFunc {

	// initialize the router once
	methodEqual := mux.build()

	return func(context *Context) {
		routePath := context.Path()
		for i := range mux.garden {
			tree := mux.garden[i]
			if !methodEqual(context.Request.Method, tree.method) {
				continue
			}

			if mux.hosts && tree.subdomain != "" {
				// context.VirtualHost() is a slow method because it makes
				// string.Replaces but user can understand that if subdomain then server will have some nano/or/milleseconds performance cost
				requestHost := context.VirtualHostname()
				if requestHost != mux.hostname {
					//println(requestHost + " != " + mux.hostname)
					// we have a subdomain
					if strings.Index(tree.subdomain, dynamicSubdomainIndicator) != -1 {
					} else {
						//println(requestHost + " = " + mux.hostname)
						// mux.host = iris-go.com:8080, the subdomain for example is api.,
						// so the host must be api.iris-go.com:8080
						if tree.subdomain+mux.hostname != requestHost {
							// go to the next tree, we have a subdomain but it is not the correct
							continue
						}

					}
				} else {
					//("it's subdomain but the request is the same as the listening addr mux.host == requestHost =>" + mux.host + "=" + requestHost + " ____ and tree's subdomain was: " + tree.subdomain)
					continue
				}
			}

			mustRedirect := tree.entry.get(routePath, context) // pass the parameters here for 0 allocation
			if context.Middleware != nil {
				// ok we found the correct route, serve it and exit entirely from here
				//ctx.Request.Header.SetUserAgentBytes(DefaultUserAgent)
				context.Do()
				return
			} else if mustRedirect && mux.correctPath { // && context.Method() == MethodConnect {
				reqPath := routePath
				pathLen := len(reqPath)

				if pathLen > 1 {
					if reqPath[pathLen-1] == '/' {
						reqPath = reqPath[:pathLen-1] //remove the last /
					} else {
						//it has path prefix, it doesn't ends with / and it hasn't be found, then just add the slash
						reqPath = reqPath + "/"
					}

					urlToRedirect := reqPath

					statusForRedirect := StatusMovedPermanently //	StatusMovedPermanently, this document is obselte, clients caches this.
					if tree.method == MethodPost ||
						tree.method == MethodPut ||
						tree.method == MethodDelete {
						statusForRedirect = StatusTemporaryRedirect //	To maintain POST data
					}

					context.Redirect(urlToRedirect, statusForRedirect)
					// RFC2616 recommends that a short note "SHOULD" be included in the
					// response because older user agents may not understand 301/307.
					// Shouldn't send the response for POST or HEAD; that leaves GET.
					if tree.method == MethodGet {
						note := "<a href=\"" + HTMLEscape(urlToRedirect) + "\">Moved Permanently</a>.\n"
						context.WriteString(note)
					}
					return
				}
			}
			// not found
			break
		}
		// https://github.com/kataras/iris/issues/469
		if mux.fireMethodNotAllowed {
			for i := range mux.garden {
				tree := mux.garden[i]
				if !methodEqual(context.Method(), tree.method) {
					continue
				}
			}
			mux.fireError(StatusMethodNotAllowed, context)
			return
		}
		mux.fireError(StatusNotFound, context)
	}
}

var (
	errPortAlreadyUsed = errors.New("Port is already used")
	errRemoveUnix      = errors.New("Unexpected error when trying to remove unix socket file. Addr: %s | Trace: %s")
	errChmod           = errors.New("Cannot chmod %#o for %q: %s")
	errCertKeyMissing  = errors.New("You should provide certFile and keyFile for TLS/SSL")
	errParseTLS        = errors.New("Couldn't load TLS, certFile=%q, keyFile=%q. Trace: %s")
)

// TCPKeepAlive returns a new tcp keep alive Listener
func TCPKeepAlive(addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", ParseHost(addr))
	if err != nil {
		return nil, err
	}
	return TCPKeepAliveListener{ln.(*net.TCPListener)}, err
}

// TCP4 returns a new tcp4 Listener
func TCP4(addr string) (net.Listener, error) {
	return net.Listen("tcp4", ParseHost(addr))
}

// UNIX returns a new unix(file) Listener
func UNIX(addr string, mode os.FileMode) (net.Listener, error) {
	if errOs := os.Remove(addr); errOs != nil && !os.IsNotExist(errOs) {
		return nil, errRemoveUnix.Format(addr, errOs.Error())
	}

	listener, err := net.Listen("unix", addr)
	if err != nil {
		return nil, errPortAlreadyUsed.AppendErr(err)
	}

	if err = os.Chmod(addr, mode); err != nil {
		return nil, errChmod.Format(mode, addr, err.Error())
	}

	return listener, nil
}

// TLS returns a new TLS Listener
func TLS(addr, certFile, keyFile string) (net.Listener, error) {

	if certFile == "" || keyFile == "" {
		return nil, errCertKeyMissing
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errParseTLS.Format(certFile, keyFile, err)
	}

	return CERT(addr, cert)
}

// CERT returns a listener which contans tls.Config with the provided certificate, use for ssl
func CERT(addr string, cert tls.Certificate) (net.Listener, error) {
	ln, err := TCP4(addr)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		PreferServerCipherSuites: true,
	}
	return tls.NewListener(ln, tlsConfig), nil
}

// LETSENCRYPT returns a new Automatic TLS Listener using letsencrypt.org service
// receives two parameters, the first is the domain of the server
// and the second is optionally, the cache file, if you skip it then the cache directory is "./letsencrypt.cache"
// if you want to disable cache file then simple give it a value of empty string ""
//
// supports localhost domains for testing,
// but I recommend you to use the LETSENCRYPTPROD if you gonna to use it on production
func LETSENCRYPT(addr string, cacheFileOptional ...string) (net.Listener, error) {
	if portIdx := strings.IndexByte(addr, ':'); portIdx == -1 {
		addr += ":443"
	}

	ln, err := TCP4(addr)
	if err != nil {
		return nil, err
	}

	cacheFile := "./letsencrypt.cache"
	if len(cacheFileOptional) > 0 {
		cacheFile = cacheFileOptional[0]
	}

	var m letsencrypt.Manager

	if cacheFile != "" {
		if err = m.CacheFile(cacheFile); err != nil {
			return nil, err
		}
	}

	tlsConfig := &tls.Config{GetCertificate: m.GetCertificate}
	tlsLn := tls.NewListener(ln, tlsConfig)

	return tlsLn, nil
}

// LETSENCRYPTPROD returns a new Automatic TLS Listener using letsencrypt.org service
// receives two parameters, the first is the domain of the server
// and the second is optionally, the cache directory, if you skip it then the cache directory is "./certcache"
// if you want to disable cache directory then simple give it a value of empty string ""
//
// does NOT supports localhost domains for testing, use LETSENCRYPT instead.
//
// this is the recommended function to use when you're ready for production state
func LETSENCRYPTPROD(addr string, cacheDirOptional ...string) (net.Listener, error) {
	if portIdx := strings.IndexByte(addr, ':'); portIdx == -1 {
		addr += ":443"
	}

	ln, err := TCP4(addr)
	if err != nil {
		return nil, err
	}

	cacheDir := "./certcache"
	if len(cacheDirOptional) > 0 {
		cacheDir = cacheDirOptional[0]
	}

	m := autocert.Manager{
		Prompt: autocert.AcceptTOS,
	} // HostPolicy is missing, if user wants it, then she/he should manually
	// configure the autocertmanager and use the `iris.Serve` to pass that listener

	if cacheDir == "" {
		// then the user passed empty by own will, then I guess she/he doesnt' want any cache directory
	} else {
		m.Cache = autocert.DirCache(cacheDir)
	}

	tlsConfig := &tls.Config{GetCertificate: m.GetCertificate}
	tlsLn := tls.NewListener(ln, tlsConfig)

	return tlsLn, nil
}

// TCPKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections.
// Dead TCP connections (e.g. closing laptop mid-download) eventually
// go away
// It is not used by default if you want to pass a keep alive listener
// then just pass the child listener, example:
// listener := iris.TCPKeepAliveListener{iris.TCP4(":8080").(*net.TCPListener)}
type TCPKeepAliveListener struct {
	*net.TCPListener
}

// Accept implements the listener and sets the keep alive period which is 3minutes
func (ln TCPKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// ParseHost tries to convert a given string to an address which is compatible with net.Listener and server
func ParseHost(addr string) string {
	// check if addr has :port, if not do it +:80 ,we need the hostname for many cases
	a := addr
	if a == "" {
		// check for os environments
		if oshost := os.Getenv("ADDR"); oshost != "" {
			a = oshost
		} else if oshost := os.Getenv("HOST"); oshost != "" {
			a = oshost
		} else if oshost := os.Getenv("HOSTNAME"); oshost != "" {
			a = oshost
			// check for port also here
			if osport := os.Getenv("PORT"); osport != "" {
				a += ":" + osport
			}
		} else if osport := os.Getenv("PORT"); osport != "" {
			a = ":" + osport
		} else {
			a = ":http"
		}
	}
	if portIdx := strings.IndexByte(a, ':'); portIdx == 0 {
		if a[portIdx:] == ":https" {
			a = DefaultServerHostname + ":443"
		} else {
			// if contains only :port	,then the : is the first letter, so we dont have setted a hostname, lets set it
			a = DefaultServerHostname + a
		}
	}

	/* changed my mind, don't add 80, this will cause problems on unix listeners, and it's not really necessary because we take the port using parsePort
	if portIdx := strings.IndexByte(a, ':'); portIdx < 0 {
		// missing port part, add it
		a = a + ":80"
	}*/

	return a
}

// ParseHostname receives an addr of form host[:port] and returns the hostname part of it
// ex: localhost:8080 will return the `localhost`, mydomain.com:8080 will return the 'mydomain'
func ParseHostname(addr string) string {
	idx := strings.IndexByte(addr, ':')
	if idx == 0 {
		// only port, then return 0.0.0.0
		return "0.0.0.0"
	} else if idx > 0 {
		return addr[0:idx]
	}
	// it's already hostname
	return addr
}

// ParsePort receives an addr of form host[:port] and returns the port part of it
// ex: localhost:8080 will return the `8080`, mydomain.com will return the '80'
func ParsePort(addr string) int {
	if portIdx := strings.IndexByte(addr, ':'); portIdx != -1 {
		afP := addr[portIdx+1:]
		p, err := strconv.Atoi(afP)
		if err == nil {
			return p
		} else if afP == "https" { // it's not number, check if it's :https
			return 443
		}
	}
	return 80
}

const (
	// SchemeHTTPS returns "https://" (full)
	SchemeHTTPS = "https://"
	// SchemeHTTP returns "http://" (full)
	SchemeHTTP = "http://"
)

// ParseScheme returns the scheme based on the host,addr,domain
// Note: the full scheme not just http*,https* *http:// *https://
func ParseScheme(domain string) string {
	// pure check
	if strings.HasPrefix(domain, SchemeHTTPS) || ParsePort(domain) == 443 {
		return SchemeHTTPS
	}
	return SchemeHTTP
}

// ProxyHandler returns a new net/http.Handler which works as 'proxy', maybe doesn't suits you look its code before using that in production
var ProxyHandler = func(redirectSchemeAndHost string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// override the handler and redirect all requests to this addr
		redirectTo := redirectSchemeAndHost
		fakehost := r.URL.Host
		path := r.URL.EscapedPath()
		if strings.Count(fakehost, ".") >= 3 { // propably a subdomain, pure check but doesn't matters don't worry
			if sufIdx := strings.LastIndexByte(fakehost, '.'); sufIdx > 0 {
				// check if the last part is a number instead of .com/.gr...
				// if it's number then it's propably is 0.0.0.0 or 127.0.0.1... so it shouldn' use  subdomain
				if _, err := strconv.Atoi(fakehost[sufIdx+1:]); err != nil {
					// it's not number then process the try to parse the subdomain
					redirectScheme := ParseScheme(redirectSchemeAndHost)
					realHost := strings.Replace(redirectSchemeAndHost, redirectScheme, "", 1)
					redirectHost := strings.Replace(fakehost, fakehost, realHost, 1)
					redirectTo = redirectScheme + redirectHost + path
					http.Redirect(w, r, redirectTo, StatusMovedPermanently)
					return
				}
			}
		}
		if path != "/" {
			redirectTo += path
		}
		if redirectTo == r.URL.String() {
			return
		}

		//	redirectTo := redirectSchemeAndHost + r.RequestURI

		http.Redirect(w, r, redirectTo, StatusMovedPermanently)
	}
}

// Proxy not really a proxy, it's just
// starts a server listening on proxyAddr but redirects all requests to the redirectToSchemeAndHost+$path
// nothing special, use it only when you want to start a secondary server which its only work is to redirect from one requested path to another
//
// returns a close function
func Proxy(proxyAddr string, redirectSchemeAndHost string) func() error {
	proxyAddr = ParseHost(proxyAddr)

	// override the handler and redirect all requests to this addr
	h := ProxyHandler(redirectSchemeAndHost)
	prx := New(OptionDisableBanner(true))
	prx.Router = h

	go prx.Listen(proxyAddr)
	if ok := <-prx.Available; !ok {
		prx.Logger.Panic("Unexpected error: proxy server cannot start, please report this as bug!!")
	}

	return func() error { return prx.Close() }
}
