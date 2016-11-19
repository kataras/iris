package iris

import (
	"bytes"
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
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
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
)

var (
	// AllMethods "GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE"
	AllMethods = [...]string{MethodGet, MethodPost, MethodPut, MethodDelete, MethodConnect, MethodHead, MethodPatch, MethodOptions, MethodTrace}

	/* methods as []byte, these are really used by iris */

	// MethodGetBytes "GET"
	MethodGetBytes = []byte(MethodGet)
	// MethodPostBytes "POST"
	MethodPostBytes = []byte(MethodPost)
	// MethodPutBytes "PUT"
	MethodPutBytes = []byte(MethodPut)
	// MethodDeleteBytes "DELETE"
	MethodDeleteBytes = []byte(MethodDelete)
	// MethodConnectBytes "CONNECT"
	MethodConnectBytes = []byte(MethodConnect)
	// MethodHeadBytes "HEAD"
	MethodHeadBytes = []byte(MethodHead)
	// MethodPatchBytes "PATCH"
	MethodPatchBytes = []byte(MethodPatch)
	// MethodOptionsBytes "OPTIONS"
	MethodOptionsBytes = []byte(MethodOptions)
	// MethodTraceBytes "TRACE"
	MethodTraceBytes = []byte(MethodTrace)
	/* */
)

const (
	// StatusContinue http status '100'
	StatusContinue = 100
	// StatusSwitchingProtocols http status '101'
	StatusSwitchingProtocols = 101
	// StatusOK http status '200'
	StatusOK = 200
	// StatusCreated http status '201'
	StatusCreated = 201
	// StatusAccepted http status '202'
	StatusAccepted = 202
	// StatusNonAuthoritativeInfo http status '203'
	StatusNonAuthoritativeInfo = 203
	// StatusNoContent http status '204'
	StatusNoContent = 204
	// StatusResetContent http status '205'
	StatusResetContent = 205
	// StatusPartialContent http status '206'
	StatusPartialContent = 206
	// StatusMultipleChoices http status '300'
	StatusMultipleChoices = 300
	// StatusMovedPermanently http status '301'
	StatusMovedPermanently = 301
	// StatusFound http status '302'
	StatusFound = 302
	// StatusSeeOther http status '303'
	StatusSeeOther = 303
	// StatusNotModified http status '304'
	StatusNotModified = 304
	// StatusUseProxy http status '305'
	StatusUseProxy = 305
	// StatusTemporaryRedirect http status '307'
	StatusTemporaryRedirect = 307
	// StatusBadRequest http status '400'
	StatusBadRequest = 400
	// StatusUnauthorized http status '401'
	StatusUnauthorized = 401
	// StatusPaymentRequired http status '402'
	StatusPaymentRequired = 402
	// StatusForbidden http status '403'
	StatusForbidden = 403
	// StatusNotFound http status '404'
	StatusNotFound = 404
	// StatusMethodNotAllowed http status '405'
	StatusMethodNotAllowed = 405
	// StatusNotAcceptable http status '406'
	StatusNotAcceptable = 406
	// StatusProxyAuthRequired http status '407'
	StatusProxyAuthRequired = 407
	// StatusRequestTimeout http status '408'
	StatusRequestTimeout = 408
	// StatusConflict http status '409'
	StatusConflict = 409
	// StatusGone http status '410'
	StatusGone = 410
	// StatusLengthRequired http status '411'
	StatusLengthRequired = 411
	// StatusPreconditionFailed http status '412'
	StatusPreconditionFailed = 412
	// StatusRequestEntityTooLarge http status '413'
	StatusRequestEntityTooLarge = 413
	// StatusRequestURITooLong http status '414'
	StatusRequestURITooLong = 414
	// StatusUnsupportedMediaType http status '415'
	StatusUnsupportedMediaType = 415
	// StatusRequestedRangeNotSatisfiable http status '416'
	StatusRequestedRangeNotSatisfiable = 416
	// StatusExpectationFailed http status '417'
	StatusExpectationFailed = 417
	// StatusTeapot http status '418'
	StatusTeapot = 418
	// StatusPreconditionRequired http status '428'
	StatusPreconditionRequired = 428
	// StatusTooManyRequests http status '429'
	StatusTooManyRequests = 429
	// StatusRequestHeaderFieldsTooLarge http status '431'
	StatusRequestHeaderFieldsTooLarge = 431
	// StatusUnavailableForLegalReasons http status '451'
	StatusUnavailableForLegalReasons = 451
	// StatusInternalServerError http status '500'
	StatusInternalServerError = 500
	// StatusNotImplemented http status '501'
	StatusNotImplemented = 501
	// StatusBadGateway http status '502'
	StatusBadGateway = 502
	// StatusServiceUnavailable http status '503'
	StatusServiceUnavailable = 503
	// StatusGatewayTimeout http status '504'
	StatusGatewayTimeout = 504
	// StatusHTTPVersionNotSupported http status '505'
	StatusHTTPVersionNotSupported = 505
	// StatusNetworkAuthenticationRequired http status '511'
	StatusNetworkAuthenticationRequired = 511
)

var statusText = map[int]string{
	StatusContinue:                      "Continue",
	StatusSwitchingProtocols:            "Switching Protocols",
	StatusOK:                            "OK",
	StatusCreated:                       "Created",
	StatusAccepted:                      "Accepted",
	StatusNonAuthoritativeInfo:          "Non-Authoritative Information",
	StatusNoContent:                     "No Content",
	StatusResetContent:                  "Reset Content",
	StatusPartialContent:                "Partial Content",
	StatusMultipleChoices:               "Multiple Choices",
	StatusMovedPermanently:              "Moved Permanently",
	StatusFound:                         "Found",
	StatusSeeOther:                      "See Other",
	StatusNotModified:                   "Not Modified",
	StatusUseProxy:                      "Use Proxy",
	StatusTemporaryRedirect:             "Temporary Redirect",
	StatusBadRequest:                    "Bad Request",
	StatusUnauthorized:                  "Unauthorized",
	StatusPaymentRequired:               "Payment Required",
	StatusForbidden:                     "Forbidden",
	StatusNotFound:                      "Not Found",
	StatusMethodNotAllowed:              "Method Not Allowed",
	StatusNotAcceptable:                 "Not Acceptable",
	StatusProxyAuthRequired:             "Proxy Authentication Required",
	StatusRequestTimeout:                "Request Timeout",
	StatusConflict:                      "Conflict",
	StatusGone:                          "Gone",
	StatusLengthRequired:                "Length Required",
	StatusPreconditionFailed:            "Precondition Failed",
	StatusRequestEntityTooLarge:         "Request Entity Too Large",
	StatusRequestURITooLong:             "Request URI Too Long",
	StatusUnsupportedMediaType:          "Unsupported Media Type",
	StatusRequestedRangeNotSatisfiable:  "Requested Range Not Satisfiable",
	StatusExpectationFailed:             "Expectation Failed",
	StatusTeapot:                        "I'm a teapot",
	StatusPreconditionRequired:          "Precondition Required",
	StatusTooManyRequests:               "Too Many Requests",
	StatusRequestHeaderFieldsTooLarge:   "Request Header Fields Too Large",
	StatusUnavailableForLegalReasons:    "Unavailable For Legal Reasons",
	StatusInternalServerError:           "Internal Server Error",
	StatusNotImplemented:                "Not Implemented",
	StatusBadGateway:                    "Bad Gateway",
	StatusServiceUnavailable:            "Service Unavailable",
	StatusGatewayTimeout:                "Gateway Timeout",
	StatusHTTPVersionNotSupported:       "HTTP Version Not Supported",
	StatusNetworkAuthenticationRequired: "Network Authentication Required",
}

// StatusText returns a text for the HTTP status code. It returns the empty
// string if the code is unknown.
func StatusText(code int) string {
	return statusText[code]
}

// errHandler returns na error with message: 'Passed argument is not func(*Context) neither an object which implements the iris.Handler with Serve(ctx *Context)
// It seems to be a  +type Points to: +pointer.'
var errHandler = errors.New("Passed argument is not func(*Context) neither an object which implements the iris.Handler with Serve(ctx *Context)\n It seems to be a  %T Points to: %v.")

type (
	// Handler the main Iris Handler interface.
	Handler interface {
		Serve(ctx *Context)
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

// ToHandler converts an http.Handler or http.HandlerFunc to an iris.Handler
func ToHandler(handler interface{}) Handler {
	//this is not the best way to do it, but I dont have any options right now.
	switch handler.(type) {
	case Handler:
		//it's already an iris handler
		return handler.(Handler)
	case http.Handler:
		//it's http.Handler
		h := fasthttpadaptor.NewFastHTTPHandlerFunc(handler.(http.Handler).ServeHTTP)

		return ToHandlerFastHTTP(h)
	case func(http.ResponseWriter, *http.Request):
		//it's http.HandlerFunc
		h := fasthttpadaptor.NewFastHTTPHandlerFunc(handler.(func(http.ResponseWriter, *http.Request)))
		return ToHandlerFastHTTP(h)
	default:
		panic(errHandler.Format(handler, handler))
	}
}

// ToHandlerFunc converts an http.Handler or http.HandlerFunc to an iris.HandlerFunc
func ToHandlerFunc(handler interface{}) HandlerFunc {
	return ToHandler(handler).Serve
}

// ToHandlerFastHTTP converts an fasthttp.RequestHandler to an iris.Handler
func ToHandlerFastHTTP(h fasthttp.RequestHandler) Handler {
	return HandlerFunc((func(ctx *Context) {
		h(ctx.RequestCtx)
	}))
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

	isStatic entryCase = iota
	isRoot
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
	cachedHandler fasthttp.RequestHandler
}

func newCachedMuxEntry(f *Framework, bodyHandler HandlerFunc, expiration time.Duration) *cachedMuxEntry {
	fhandler := func(reqCtx *fasthttp.RequestCtx) {
		ctx := f.AcquireCtx(reqCtx)
		bodyHandler.Serve(ctx)
		f.ReleaseCtx(ctx)
	}

	cachedHandler := httpcache.CacheFasthttpFunc(fhandler, expiration)
	return &cachedMuxEntry{
		cachedHandler: cachedHandler,
	}
}

func (c *cachedMuxEntry) Serve(ctx *Context) {
	c.cachedHandler(ctx.RequestCtx)
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
		// Path returns the path
		Path() string
		// SetPath changes/sets the path for this route
		SetPath(string)
		// Middleware returns the slice of Handler([]Handler) registed to this route
		Middleware() Middleware
		// SetMiddleware changes/sets the middleware(handler(s)) for this route
		SetMiddleware(Middleware)
	}

	route struct {
		// if no name given then it's the subdomain+path
		name           string
		subdomain      string
		method         []byte
		methodStr      string
		path           string
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

func newRoute(method []byte, subdomain string, path string, middleware Middleware) *route {
	r := &route{name: path + subdomain, method: method, methodStr: string(method), subdomain: subdomain, path: path, middleware: middleware}
	r.formatPath()
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

func (r *route) setName(newName string) {
	r.name = newName
}

func (r route) Name() string {
	return r.name
}

func (r route) Subdomain() string {
	return r.subdomain
}

func (r route) Method() string {
	if r.methodStr == "" {
		r.methodStr = string(r.method)
	}
	return r.methodStr
}

func (r route) Path() string {
	return r.path
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
	// subdomainIndicator where './' exists in a registed path then it contains subdomain
	subdomainIndicator = "./"
	// dynamicSubdomainIndicator where a registed path starts with '*.' then it contains a dynamic subdomain, if subdomain == "*." then its dynamic
	dynamicSubdomainIndicator = "*."
)

type (
	muxTree struct {
		method []byte
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
		// if false then searching by unescaped path
		// defaults to true
		escapePath bool
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
		escapePath:           !DefaultDisablePathEscape,
		correctPath:          !DefaultDisablePathCorrection,
		fireMethodNotAllowed: false,
		logger:               logger,
	}

	return mux
}

func (mux *serveMux) setHostname(h string) {
	mux.hostname = h
}

func (mux *serveMux) setEscapePath(b bool) {
	mux.escapePath = b
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
			ctx.ResetBody()
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
			ctx.ResetBody()
			ctx.SetStatusCode(statusCode)
			ctx.SetBodyString(statusText[statusCode])
		})
		mux.errorHandlers[statusCode] = errHandler
	}
	mux.mu.Unlock()

	errHandler.Serve(ctx)
}

func (mux *serveMux) getTree(method []byte, subdomain string) *muxTree {
	for i := range mux.garden {
		t := mux.garden[i]
		if bytes.Equal(t.method, method) && t.subdomain == subdomain {
			return t
		}
	}
	return nil
}

func (mux *serveMux) register(method []byte, subdomain string, path string, middleware Middleware) *route {
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
// this happens once when server is setting the mux's handler.
func (mux *serveMux) build() (getRequestPath func(*fasthttp.RequestCtx) string, methodEqual func([]byte, []byte) bool) {

	sort.Sort(bySubdomain(mux.lookups))

	for i := range mux.lookups {
		r := mux.lookups[i]
		// add to the registry tree
		tree := mux.getTree(r.method, r.subdomain)
		if tree == nil {
			//first time we register a route to this method with this domain
			tree = &muxTree{method: r.method, subdomain: r.subdomain, entry: &muxEntry{}}
			mux.garden = append(mux.garden, tree)
		}
		// I decide that it's better to explicit give subdomain and a path to it than registedPath(mysubdomain./something) now its: subdomain: mysubdomain., path: /something
		// we have different tree for each of subdomains, now you can use everything you can use with the normal paths ( before you couldn't set /any/*path)
		if err := tree.entry.add(r.path, r.middleware); err != nil {
			mux.logger.Panic(err)
		}

		if mp := tree.entry.paramsLen; mp > mux.maxParameters {
			mux.maxParameters = mp
		}
	}

	// optimize this once once, we could do that: context.RequestPath(mux.escapePath), but we lose some nanoseconds on if :)
	getRequestPath = func(reqCtx *fasthttp.RequestCtx) string {
		return string(reqCtx.Path())
	}

	if !mux.escapePath {
		getRequestPath = func(reqCtx *fasthttp.RequestCtx) string { return string(reqCtx.RequestURI()) }
	}

	methodEqual = func(reqMethod []byte, treeMethod []byte) bool {
		return bytes.Equal(reqMethod, treeMethod)
	}
	// check for cors conflicts FIRST in order to put them in OPTIONS tree also
	for i := range mux.lookups {
		r := mux.lookups[i]
		if r.hasCors() {
			// cors middleware is updated also, ref: https://github.com/kataras/iris/issues/461
			methodEqual = func(reqMethod []byte, treeMethod []byte) bool {
				// preflights
				return bytes.Equal(reqMethod, MethodOptionsBytes) || bytes.Equal(reqMethod, treeMethod)
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

// BuildHandler the default Iris router when iris.Handler is nil
func (mux *serveMux) BuildHandler() HandlerFunc {

	// initialize the router once
	getRequestPath, methodEqual := mux.build()

	return func(context *Context) {
		routePath := getRequestPath(context.RequestCtx)
		for i := range mux.garden {
			tree := mux.garden[i]
			if !methodEqual(context.Method(), tree.method) {
				continue
			}

			if mux.hosts && tree.subdomain != "" {
				// context.VirtualHost() is a slow method because it makes string.Replaces but user can understand that if subdomain then server will have some nano/or/milleseconds performance cost
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
			} else if mustRedirect && mux.correctPath && !bytes.Equal(context.Method(), MethodConnectBytes) {

				reqPath := routePath
				pathLen := len(reqPath)

				if pathLen > 1 {
					if reqPath[pathLen-1] == '/' {
						reqPath = reqPath[:pathLen-1] //remove the last /
					} else {
						//it has path prefix, it doesn't ends with / and it hasn't be found, then just add the slash
						reqPath = reqPath + "/"
					}

					context.Request.URI().SetPath(reqPath)
					urlToRedirect := string(context.Request.RequestURI())

					statisForRedirect := StatusMovedPermanently //	StatusMovedPermanently, this document is obselte, clients caches this.
					if bytes.Equal(tree.method, MethodPostBytes) ||
						bytes.Equal(tree.method, MethodPutBytes) ||
						bytes.Equal(tree.method, MethodDeleteBytes) {
						statisForRedirect = StatusTemporaryRedirect //	To maintain POST data
					}

					context.Redirect(urlToRedirect, statisForRedirect)
					// RFC2616 recommends that a short note "SHOULD" be included in the
					// response because older user agents may not understand 301/307.
					// Shouldn't send the response for POST or HEAD; that leaves GET.
					if bytes.Equal(tree.method, MethodGetBytes) {
						note := "<a href=\"" + HTMLEscape(urlToRedirect) + "\">Moved Permanently</a>.\n"
						context.Write(note)
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

// TCP4 returns a new tcp4 Listener
// *tcp6 has some bugs in some operating systems, as reported by Go Community*
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

// Accept implements the listener and sets the keep alive period which is 2minutes
func (ln TCPKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(2 * time.Minute)
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
			a = DefaultServerAddr
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

// ProxyHandler returns a new fasthttp handler which works as 'proxy', maybe doesn't suits you look its code before using that in production
var ProxyHandler = func(proxyAddr string, redirectSchemeAndHost string) fasthttp.RequestHandler {
	return func(reqCtx *fasthttp.RequestCtx) {
		// override the handler and redirect all requests to this addr
		redirectTo := redirectSchemeAndHost
		fakehost := string(reqCtx.Request.Host())
		path := string(reqCtx.Path())
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
					reqCtx.Redirect(redirectTo, StatusMovedPermanently)
					return
				}
			}
		}
		if path != "/" {
			redirectTo += path
		}

		reqCtx.Redirect(redirectTo, StatusMovedPermanently)
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
	h := ProxyHandler(proxyAddr, redirectSchemeAndHost)
	prx := New(OptionDisableBanner(true))
	prx.Router = h

	go prx.Listen(proxyAddr)

	return prx.Close
}
