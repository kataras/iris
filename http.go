package iris

import (
	"bytes"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iris-contrib/errors"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/logger"
	"github.com/kataras/iris/utils"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
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
	// methodGetBytes "GET"
	methodGetBytes = []byte(MethodGet)
	// methodPostBytes "POST"
	methodPostBytes = []byte(MethodPost)
	// methodPutBytes "PUT"
	methodPutBytes = []byte(MethodPut)
	// methodDeleteBytes "DELETE"
	methodDeleteBytes = []byte(MethodDelete)
	// methodConnectBytes "CONNECT"
	methodConnectBytes = []byte(MethodConnect)
	// methodHeadBytes "HEAD"
	methodHeadBytes = []byte(MethodHead)
	// methodPatchBytes "PATCH"
	methodPatchBytes = []byte(MethodPatch)
	// methodOptionsBytes "OPTIONS"
	methodOptionsBytes = []byte(MethodOptions)
	// methodTraceBytes "TRACE"
	methodTraceBytes = []byte(MethodTrace)
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
	StatusContinue:           "Continue",
	StatusSwitchingProtocols: "Switching Protocols",

	StatusOK:                   "OK",
	StatusCreated:              "Created",
	StatusAccepted:             "Accepted",
	StatusNonAuthoritativeInfo: "Non-Authoritative Information",
	StatusNoContent:            "No Content",
	StatusResetContent:         "Reset Content",
	StatusPartialContent:       "Partial Content",

	StatusMultipleChoices:   "Multiple Choices",
	StatusMovedPermanently:  "Moved Permanently",
	StatusFound:             "Found",
	StatusSeeOther:          "See Other",
	StatusNotModified:       "Not Modified",
	StatusUseProxy:          "Use Proxy",
	StatusTemporaryRedirect: "Temporary Redirect",

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
	StatusNetworkAuthenticationRequired: "Network Authentication Required",
}

// StatusText returns a text for the HTTP status code. It returns the empty
// string if the code is unknown.
func StatusText(code int) string {
	return statusText[code]
}

// Errors introduced by server.
var (
	errServerPortAlreadyUsed = errors.New("Server can't run, port is already used")
	errServerAlreadyStarted  = errors.New("Server is already started and listening")
	errServerHandlerMissing  = errors.New("Handler is missing from server, can't start without handler")
	errServerIsClosed        = errors.New("Can't close the server, propably is already closed or never started")
	errServerRemoveUnix      = errors.New("Unexpected error when trying to remove unix socket file. Addr: %s | Trace: %s")
	errServerChmod           = errors.New("Cannot chmod %#o for %q: %s")
)

type (
	// Server the http server
	Server struct {
		*fasthttp.Server
		listener net.Listener
		Config   config.Server
		tls      bool
		mu       sync.Mutex
	}
	// ServerList contains the servers connected to the Iris station
	ServerList struct {
		mux     *serveMux
		servers []*Server
	}
)

// newServer returns a pointer to a Server object, and set it's options if any,  nothing more
func newServer(cfg config.Server) *Server {
	s := &Server{Server: &fasthttp.Server{Name: config.ServerName}, Config: cfg}
	s.prepare()
	return s
}

// prepare just prepares the listening addr
func (s *Server) prepare() {
	s.Config.ListeningAddr = config.ServerParseAddr(s.Config.ListeningAddr)
}

// IsListening returns true if server is listening/started, otherwise false
func (s *Server) IsListening() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listener != nil && s.listener.Addr().String() != ""
}

// IsOpened checks if handler is not nil and returns true if not, otherwise false
// this is used to see if a server has opened, use IsListening if you want to see if the server is actually ready to serve connections
func (s *Server) IsOpened() bool {
	if s == nil {
		return false
	}
	return s.Server != nil && s.Server.Handler != nil
}

// IsSecure returns true if server uses TLS, otherwise false
func (s *Server) IsSecure() bool {
	return s.tls
}

// Listener returns the net.Listener which this server (is) listening to
func (s *Server) Listener() net.Listener {
	return s.listener
}

// Host returns the registered host for the server
func (s *Server) Host() (host string) {
	return s.Config.ListeningAddr
}

// Port returns the port which server listening for
// if no port given with the ListeningAddr, it returns 80
func (s *Server) Port() (port int) {
	a := s.Config.ListeningAddr
	if portIdx := strings.IndexByte(a, ':'); portIdx != -1 {
		p, err := strconv.Atoi(a[portIdx+1:])
		if err != nil {
			port = 80
		} else {
			port = p
		}
	} else {
		port = 80
	}
	return
}

// FullHost returns the scheme+host
func (s *Server) FullHost() string {
	scheme := "http://"
	// we need to be able to take that before(for testing &debugging) and after server's listen
	if s.IsSecure() || (s.Config.CertFile != "" && s.Config.KeyFile != "") {
		scheme = "https://"
	}
	return scheme + s.Host()
}

// Hostname returns the hostname part of the host (host expect port)
func (s *Server) Hostname() string {
	return s.Host()[0:strings.IndexByte(s.Host(), ':')] // no the port
}

func (s *Server) listen() error {
	if s.IsListening() {
		return errServerAlreadyStarted.Return()
	}
	listener, err := net.Listen("tcp4", s.Config.ListeningAddr)

	if err != nil {
		return err
	}

	go s.serve(listener) // we don't catch underline errors, we catched all already
	return nil

}

func (s *Server) listenUNIX() error {

	mode := s.Config.Mode
	addr := s.Config.ListeningAddr

	if errOs := os.Remove(addr); errOs != nil && !os.IsNotExist(errOs) {
		return errServerRemoveUnix.Format(s.Config.ListeningAddr, errOs.Error())
	}

	listener, err := net.Listen("unix", addr)

	if err != nil {
		return errServerPortAlreadyUsed.Return()
	}

	if err = os.Chmod(addr, mode); err != nil {
		return errServerChmod.Format(mode, addr, err.Error())
	}

	go s.serve(listener) // we don't catch underline errors, we catched all already
	return nil

}

//Serve just serves a listener, it is a blocking action, plugin.PostListen is not fired here.
func (s *Server) serve(l net.Listener) error {
	s.mu.Lock()
	s.listener = l
	s.mu.Unlock()
	if s.Config.CertFile != "" && s.Config.KeyFile != "" {
		s.tls = true
		return s.Server.ServeTLS(s.listener, s.Config.CertFile, s.Config.KeyFile)
	}
	s.tls = false
	return s.Server.Serve(s.listener)
}

// Open opens/starts/runs/listens (to) the server, listen tls if Cert && Key is registed, listenUNIX if Mode is registed, otherwise listen
func (s *Server) Open(h fasthttp.RequestHandler) error {
	if h == nil {
		return errServerHandlerMissing.Return()
	}

	if s.IsListening() {
		return errServerAlreadyStarted.Return()
	}

	s.prepare() // do it again for any case

	if s.Config.MaxRequestBodySize > config.DefaultMaxRequestBodySize {
		s.Server.MaxRequestBodySize = int(s.Config.MaxRequestBodySize)
	}

	if s.Config.RedirectTo != "" {
		// override the handler and redirect all requests to this addr
		s.Server.Handler = func(reqCtx *fasthttp.RequestCtx) {
			path := string(reqCtx.Path())
			redirectTo := s.Config.RedirectTo
			if path != "/" {
				redirectTo += "/" + path
			}
			reqCtx.Redirect(redirectTo, StatusMovedPermanently)
		}
	} else {
		s.Server.Handler = h
	}

	if s.Config.Virtual {
		return nil
	}

	if s.Config.Mode > 0 {
		return s.listenUNIX()
	}
	return s.listen()

}

// Close terminates the server
func (s *Server) Close() (err error) {
	if !s.IsListening() {
		return errServerIsClosed.Return()
	}
	err = s.listener.Close()

	return
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// --------------------------------ServerList implementation-----------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// Add adds a server to the list by its config
// returns the new server
func (s *ServerList) Add(cfg config.Server) *Server {
	srv := newServer(cfg)
	s.servers = append(s.servers, srv)
	return srv
}

// Len returns the size of the server list
func (s *ServerList) Len() int {
	return len(s.servers)
}

// Main returns the main server,
// the last added server is the main server, even if's Virtual
func (s *ServerList) Main() (srv *Server) {
	l := len(s.servers) - 1
	for i := range s.servers {
		if i == l {
			return s.servers[i]
		}
	}
	return nil
}

// Get returns the server by it's registered Address
func (s *ServerList) Get(addr string) (srv *Server) {
	for i := range s.servers {
		srv = s.servers[i]
		if srv.Config.ListeningAddr == addr {
			return
		}
	}
	return
}

// GetAll returns all registered servers
func (s *ServerList) GetAll() []*Server {
	return s.servers
}

// GetByIndex returns a server from the list by it's index
func (s *ServerList) GetByIndex(i int) *Server {
	if len(s.servers) >= i+1 {
		return s.servers[i]
	}
	return nil
}

// Remove deletes a server by it's registered Address
// returns true if something was removed, otherwise returns false
func (s *ServerList) Remove(addr string) bool {
	servers := s.servers
	for i := range servers {
		srv := servers[i]
		if srv.Config.ListeningAddr == addr {
			copy(servers[i:], servers[i+1:])
			servers[len(servers)-1] = nil
			s.servers = servers[:len(servers)-1]
			return true
		}
	}
	return false
}

// CloseAll terminates all listening servers
// returns the first error, if erro happens it continues to closes the rest of the servers
func (s *ServerList) CloseAll() (err error) {
	for i := range s.servers {
		if err == nil {
			err = s.servers[i].Close()
		}
	}
	return
}

// OpenAll starts all servers
// returns the first error happens to one of these servers
// if one server gets error it closes the previous servers and exits from this process
func (s *ServerList) OpenAll() error {
	l := len(s.servers) - 1
	h := s.mux.ServeRequest()
	for i := range s.servers {

		if err := s.servers[i].Open(h); err != nil {
			time.Sleep(2 * time.Second)
			// for any case,
			// we don't care about performance on initialization,
			// we must make sure that the previous servers are running before closing them
			s.CloseAll()
			break
		}
		if i == l {
			s.mux.setHostname(s.servers[i].Hostname())
		}

	}
	return nil
}

// GetAllOpened returns all opened/started servers
func (s *ServerList) GetAllOpened() (servers []*Server) {
	for i := range s.servers {
		if s.servers[i].IsOpened() {
			servers = append(servers, s.servers[i])
		}
	}
	return
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

// ToHandler converts an httapi.Handler or http.HandlerFunc to an iris.Handler
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

func profileMiddleware(debugPath string) Middleware {
	htmlMiddleware := HandlerFunc(func(ctx *Context) {
		ctx.SetContentType(contentHTML + "; charset=" + config.Charset)
		ctx.Next()
	})
	indexHandler := ToHandlerFunc(pprof.Index)
	cmdlineHandler := ToHandlerFunc(pprof.Cmdline)
	profileHandler := ToHandlerFunc(pprof.Profile)
	symbolHandler := ToHandlerFunc(pprof.Symbol)
	goroutineHandler := ToHandlerFunc(pprof.Handler("goroutine"))
	heapHandler := ToHandlerFunc(pprof.Handler("heap"))
	threadcreateHandler := ToHandlerFunc(pprof.Handler("threadcreate"))
	debugBlockHandler := ToHandlerFunc(pprof.Handler("block"))

	return Middleware{htmlMiddleware, HandlerFunc(func(ctx *Context) {
		action := ctx.Param("action")
		if len(action) > 1 {
			if strings.Contains(action, "cmdline") {
				cmdlineHandler.Serve((ctx))
			} else if strings.Contains(action, "profile") {
				profileHandler.Serve(ctx)
			} else if strings.Contains(action, "symbol") {
				symbolHandler.Serve(ctx)
			} else if strings.Contains(action, "goroutine") {
				goroutineHandler.Serve(ctx)
			} else if strings.Contains(action, "heap") {
				heapHandler.Serve(ctx)
			} else if strings.Contains(action, "threadcreate") {
				threadcreateHandler.Serve(ctx)
			} else if strings.Contains(action, "debug/block") {
				debugBlockHandler.Serve(ctx)
			}
		} else {
			indexHandler.Serve(ctx)
		}
	})}
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
	// PathParameter is a struct which contains Key and Value, used for named path parameters
	PathParameter struct {
		Key   string
		Value string
	}

	// PathParameters type for a slice of PathParameter
	// Tt's a slice of PathParameter type, because it's faster than map
	PathParameters []PathParameter

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

// Get returns a value from a key inside this Parameters
// If no parameter with this key given then it returns an empty string
func (params PathParameters) Get(key string) string {
	for _, p := range params {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

// String returns a string implementation of all parameters that this PathParameters object keeps
// hasthe form of key1=value1,key2=value2...
func (params PathParameters) String() string {
	var buff bytes.Buffer
	for i := range params {
		buff.WriteString(params[i].Key)
		buff.WriteString("=")
		buff.WriteString(params[i].Value)
		if i < len(params)-1 {
			buff.WriteString(",")
		}

	}
	return buff.String()
}

// ParseParams receives a string and returns PathParameters (slice of PathParameter)
// received string must have this form:  key1=value1,key2=value2...
func ParseParams(str string) PathParameters {
	_paramsstr := strings.Split(str, ",")
	if len(_paramsstr) == 0 {
		return nil
	}

	params := make(PathParameters, 0) // PathParameters{}

	//	for i := 0; i < len(_paramsstr); i++ {
	for i := range _paramsstr {
		idxOfEq := strings.IndexRune(_paramsstr[i], '=')
		if idxOfEq == -1 {
			//error
			return nil
		}

		key := _paramsstr[i][:idxOfEq]
		val := _paramsstr[i][idxOfEq+1:]
		params = append(params, PathParameter{key, val})
	}
	return params
}

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
			max := utils.FindLower(len(path), len(e.part))
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

						if len(e.part) >= len(path) || path[len(e.part)] == '/' {
							continue loop
						}
					}
					return errMuxEntryConflictsWildcard.Format(path, e.part, fullPath)
				}

				c := path[0]

				if e.entryCase == hasParams && c == '/' && len(e.nodes) == 1 {
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
func (e *muxEntry) get(path string, _params PathParameters) (middleware Middleware, params PathParameters, mustRedirect bool) {
	params = _params
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

					if cap(params) < int(e.paramsLen) {
						params = make(PathParameters, 0, e.paramsLen)
					}
					i := len(params)
					params = params[:i+1]
					params[i].Key = e.part[1:]
					params[i].Value = path[:end]

					if end < len(path) {
						if len(e.nodes) > 0 {
							path = path[end:]
							e = e.nodes[0]
							continue loop
						}

						mustRedirect = (len(path) == end+1)
						return
					}

					if middleware = e.middleware; middleware != nil {
						return
					} else if len(e.nodes) == 1 {
						e = e.nodes[0]
						mustRedirect = (e.part == slash && e.middleware != nil)
					}

					return

				case matchEverything:
					if cap(params) < int(e.paramsLen) {
						params = make(PathParameters, 0, e.paramsLen)
					}
					i := len(params)
					params = params[:i+1]
					params[i].Key = e.part[2:]
					params[i].Value = path

					middleware = e.middleware
					return

				default:
					return
				}
			}
		} else if path == e.part {
			if middleware = e.middleware; middleware != nil {
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

//
//
//

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
		// Middleware returns the slice of Handler([]Handler) registed to this route
		Middleware() Middleware
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
	r := &route{name: path + subdomain, method: method, subdomain: subdomain, path: path, middleware: middleware}
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

func (r route) Middleware() Middleware {
	return r.middleware
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
		next      *muxTree
	}

	serveMux struct {
		cPool   *sync.Pool
		tree    *muxTree
		lookups []*route

		api           *muxAPI
		errorHandlers map[int]Handler
		logger        *logger.Logger
		// the main server host's name, ex:  localhost, 127.0.0.1, iris-go.com
		hostname string
		// if any of the trees contains not empty subdomain
		hosts bool
		// if false then searching by unescaped path
		// defaults to true
		escapePath bool
		// if false then the /something it's not the same as /something/
		// defaults to true
		correctPath bool
		mu          sync.Mutex
	}
)

func newServeMux(contextPool sync.Pool, logger *logger.Logger) *serveMux {
	mux := &serveMux{
		cPool:         &contextPool,
		lookups:       make([]*route, 0),
		errorHandlers: make(map[int]Handler, 0),
		hostname:      "127.0.0.1",
		escapePath:    !config.DefaultDisablePathEscape,
		correctPath:   !config.DefaultDisablePathCorrection,
		logger:        logger,
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

func (mux *serveMux) getTree(method []byte, subdomain string) (tree *muxTree) {
	tree = mux.tree
	for tree != nil {
		if bytes.Equal(tree.method, method) && tree.subdomain == subdomain {
			return
		}
		tree = tree.next
	}
	// tree is nil here, return that.
	return
}

func (mux *serveMux) register(method []byte, subdomain string, path string, middleware Middleware) *route {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if subdomain != "" {
		mux.hosts = true
	}

	// add to the lookups, it's just a collection of routes information
	lookup := newRoute(method, subdomain, path, middleware)
	mux.lookups = append(mux.lookups, lookup)

	return lookup

}

// build collects all routes info and adds them to the registry in order to be served from the request handler
// this happens once when server is setting the mux's handler.
func (mux *serveMux) build() {

	sort.Sort(bySubdomain(mux.lookups))
	for _, r := range mux.lookups {
		// add to the registry tree
		tree := mux.getTree(r.method, r.subdomain)
		if tree == nil {
			//first time we register a route to this method with this domain
			tree = &muxTree{method: r.method, subdomain: r.subdomain, entry: &muxEntry{}, next: nil}
			if mux.tree == nil {
				// it's the first entry
				mux.tree = tree
			} else {
				// find the last tree and make the .next to the tree we created before
				lastTree := mux.tree
				for lastTree != nil {
					if lastTree.next == nil {
						lastTree.next = tree
						break
					}
					lastTree = lastTree.next
				}
			}
		}
		// I decide that it's better to explicit give subdomain and a path to it than registedPath(mysubdomain./something) now its: subdomain: mysubdomain., path: /something
		// we have different tree for each of subdomains, now you can use everything you can use with the normal paths ( before you couldn't set /any/*path)
		if err := tree.entry.add(r.path, r.middleware); err != nil {
			mux.logger.Panic(err.Error())
		}
	}
}

func (mux *serveMux) lookup(routeName string) *route {
	for i := range mux.lookups {
		if r := mux.lookups[i]; r.name == routeName {
			return r
		}
	}
	return nil
}

func (mux *serveMux) ServeRequest() fasthttp.RequestHandler {

	// initialize the router once
	mux.build()
	// optimize this once once, we could do that: context.RequestPath(mux.escapePath), but we lose some nanoseconds on if :)
	getRequestPath := func(reqCtx *fasthttp.RequestCtx) string {
		return utils.BytesToString(reqCtx.Path())
	}
	if !mux.escapePath {
		getRequestPath = func(reqCtx *fasthttp.RequestCtx) string { return utils.BytesToString(reqCtx.RequestURI()) }
	}

	return func(reqCtx *fasthttp.RequestCtx) {
		context := mux.cPool.Get().(*Context)
		context.Reset(reqCtx)

		routePath := getRequestPath(reqCtx)
		tree := mux.tree
		for tree != nil {
			if !bytes.Equal(tree.method, reqCtx.Method()) {
				// we break any CORS OPTIONS method
				// but for performance reasons if user wants http method OPTIONS to be served
				// then must register it with .Options(...)
				tree = tree.next
				continue
			}
			// we have at least one subdomain on the root
			if mux.hosts && tree.subdomain != "" {
				// context.VirtualHost() is a slow method because it makes string.Replaces but user can understand that if subdomain then server will have some nano/or/milleseconds performance cost
				requestHost := context.VirtualHostname()
				if requestHost != mux.hostname {
					// we have a subdomain
					if strings.Index(tree.subdomain, dynamicSubdomainIndicator) != -1 {
					} else {
						// mux.host = iris-go.com:8080, the subdomain for example is api.,
						// so the host must be api.iris-go.com:8080
						if tree.subdomain+mux.hostname != requestHost {
							// go to the next tree, we have a subdomain but it is not the correct
							tree = tree.next
							continue
						}

					}
				} else {
					//("it's subdomain but the request is the same as the listening addr mux.host == requestHost =>" + mux.host + "=" + requestHost + " ____ and tree's subdomain was: " + tree.subdomain)
					tree = tree.next
					continue
				}
			}
			middleware, params, mustRedirect := tree.entry.get(routePath, context.Params) // pass the parameters here for 0 allocation
			if middleware != nil {
				// ok we found the correct route, serve it and exit entirely from here
				context.Params = params
				context.middleware = middleware
				//ctx.Request.Header.SetUserAgentBytes(DefaultUserAgent)
				context.Do()
				mux.cPool.Put(context)
				return
			} else if mustRedirect && mux.correctPath && !bytes.Equal(reqCtx.Method(), methodConnectBytes) {

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
					urlToRedirect := utils.BytesToString(context.Request.RequestURI())

					context.Redirect(urlToRedirect, StatusMovedPermanently) //	StatusMovedPermanently
					// RFC2616 recommends that a short note "SHOULD" be included in the
					// response because older user agents may not understand 301/307.
					// Shouldn't send the response for POST or HEAD; that leaves GET.
					if bytes.Equal(tree.method, methodGetBytes) {
						note := "<a href=\"" + utils.HTMLEscape(urlToRedirect) + "\">Moved Permanently</a>.\n"
						context.Write(note)
					}
					mux.cPool.Put(context)
					return
				}
			}
			// not found
			break
		}
		mux.fireError(StatusNotFound, context)
		mux.cPool.Put(context)
	}
}
