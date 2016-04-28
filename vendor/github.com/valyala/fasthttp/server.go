package fasthttp

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ServeConn serves HTTP requests from the given connection
// using the given handler.
//
// ServeConn returns nil if all requests from the c are successfully served.
// It returns non-nil error otherwise.
//
// Connection c must immediately propagate all the data passed to Write()
// to the client. Otherwise requests' processing may hang.
//
// ServeConn closes c before returning.
func ServeConn(c net.Conn, handler RequestHandler) error {
	v := serverPool.Get()
	if v == nil {
		v = &Server{}
	}
	s := v.(*Server)
	s.Handler = handler
	err := s.ServeConn(c)
	s.Handler = nil
	serverPool.Put(v)
	return err
}

var serverPool sync.Pool

// Serve serves incoming connections from the given listener
// using the given handler.
//
// Serve blocks until the given listener returns permanent error.
func Serve(ln net.Listener, handler RequestHandler) error {
	s := &Server{
		Handler: handler,
	}
	return s.Serve(ln)
}

// ServeTLS serves HTTPS requests from the given net.Listener
// using the given handler.
//
// certFile and keyFile are paths to TLS certificate and key files.
func ServeTLS(ln net.Listener, certFile, keyFile string, handler RequestHandler) error {
	s := &Server{
		Handler: handler,
	}
	return s.ServeTLS(ln, certFile, keyFile)
}

// ServeTLSEmbed serves HTTPS requests from the given net.Listener
// using the given handler.
//
// certData and keyData must contain valid TLS certificate and key data.
func ServeTLSEmbed(ln net.Listener, certData, keyData []byte, handler RequestHandler) error {
	s := &Server{
		Handler: handler,
	}
	return s.ServeTLSEmbed(ln, certData, keyData)
}

// ListenAndServe serves HTTP requests from the given TCP addr
// using the given handler.
func ListenAndServe(addr string, handler RequestHandler) error {
	s := &Server{
		Handler: handler,
	}
	return s.ListenAndServe(addr)
}

// ListenAndServeUNIX serves HTTP requests from the given UNIX addr
// using the given handler.
//
// The function deletes existing file at addr before starting serving.
//
// The server sets the given file mode for the UNIX addr.
func ListenAndServeUNIX(addr string, mode os.FileMode, handler RequestHandler) error {
	s := &Server{
		Handler: handler,
	}
	return s.ListenAndServeUNIX(addr, mode)
}

// ListenAndServeTLS serves HTTPS requests from the given TCP addr
// using the given handler.
//
// certFile and keyFile are paths to TLS certificate and key files.
func ListenAndServeTLS(addr, certFile, keyFile string, handler RequestHandler) error {
	s := &Server{
		Handler: handler,
	}
	return s.ListenAndServeTLS(addr, certFile, keyFile)
}

// ListenAndServeTLSEmbed serves HTTPS requests from the given TCP addr
// using the given handler.
//
// certData and keyData must contain valid TLS certificate and key data.
func ListenAndServeTLSEmbed(addr string, certData, keyData []byte, handler RequestHandler) error {
	s := &Server{
		Handler: handler,
	}
	return s.ListenAndServeTLSEmbed(addr, certData, keyData)
}

// RequestHandler must process incoming requests.
//
// RequestHandler must call ctx.TimeoutError() before returning
// if it keeps references to ctx and/or its' members after the return.
// Consider wrapping RequestHandler into TimeoutHandler if response time
// must be limited.
type RequestHandler func(ctx *RequestCtx)

// Server implements HTTP server.
//
// Default Server settings should satisfy the majority of Server users.
// Adjust Server settings only if you really understand the consequences.
//
// It is forbidden copying Server instances. Create new Server instances
// instead.
//
// It is safe to call Server methods from concurrently running goroutines.
type Server struct {
	noCopy noCopy

	// Handler for processing incoming requests.
	Handler RequestHandler

	// Server name for sending in response headers.
	//
	// Default server name is used if left blank.
	Name string

	// The maximum number of concurrent connections the server may serve.
	//
	// DefaultConcurrency is used if not set.
	Concurrency int

	// Whether to disable keep-alive connections.
	//
	// The server will close all the incoming connections after sending
	// the first response to client if this option is set to true.
	//
	// By default keep-alive connections are enabled.
	DisableKeepalive bool

	// Per-connection buffer size for requests' reading.
	// This also limits the maximum header size.
	//
	// Increase this buffer if your clients send multi-KB RequestURIs
	// and/or multi-KB headers (for example, BIG cookies).
	//
	// Default buffer size is used if not set.
	ReadBufferSize int

	// Per-connection buffer size for responses' writing.
	//
	// Default buffer size is used if not set.
	WriteBufferSize int

	// Maximum duration for reading the full request (including body).
	//
	// This also limits the maximum duration for idle keep-alive
	// connections.
	//
	// By default request read timeout is unlimited.
	ReadTimeout time.Duration

	// Maximum duration for writing the full response (including body).
	//
	// By default response write timeout is unlimited.
	WriteTimeout time.Duration

	// Maximum number of concurrent client connections allowed per IP.
	//
	// By default unlimited number of concurrent connections
	// may be established to the server from a single IP address.
	MaxConnsPerIP int

	// Maximum number of requests served per connection.
	//
	// The server closes connection after the last request.
	// 'Connection: close' header is added to the last response.
	//
	// By default unlimited number of requests may be served per connection.
	MaxRequestsPerConn int

	// Maximum keep-alive connection lifetime.
	//
	// The server closes keep-alive connection after its' lifetime
	// expiration.
	//
	// See also ReadTimeout for limiting the duration of idle keep-alive
	// connections.
	//
	// By default keep-alive connection lifetime is unlimited.
	MaxKeepaliveDuration time.Duration

	// Maximum request body size.
	//
	// The server rejects requests with bodies exceeding this limit.
	//
	// By default request body size is unlimited.
	MaxRequestBodySize int

	// Aggressively reduces memory usage at the cost of higher CPU usage
	// if set to true.
	//
	// Try enabling this option only if the server consumes too much memory
	// serving mostly idle keep-alive connections (more than 1M concurrent
	// connections). This may reduce memory usage by up to 50%.
	//
	// Aggressive memory usage reduction is disabled by default.
	ReduceMemoryUsage bool

	// Rejects all non-GET requests if set to true.
	//
	// This option is useful as anti-DoS protection for servers
	// accepting only GET requests. The request size is limited
	// by ReadBufferSize if GetOnly is set.
	//
	// Server accepts all the requests by default.
	GetOnly bool

	// Logs all errors, including the most frequent
	// 'connection reset by peer', 'broken pipe' and 'connection timeout'
	// errors. Such errors are common in production serving real-world
	// clients.
	//
	// By default the most frequent errors such as
	// 'connection reset by peer', 'broken pipe' and 'connection timeout'
	// are suppressed in order to limit output log traffic.
	LogAllErrors bool

	// Header names are passed as-is without normalization
	// if this option is set.
	//
	// Disabled header names' normalization may be useful only for proxying
	// incoming requests to other servers expecting case-sensitive
	// header names. See https://github.com/valyala/fasthttp/issues/57
	// for details.
	//
	// By default request and response header names are normalized, i.e.
	// The first letter and the first letters following dashes
	// are uppercased, while all the other letters are lowercased.
	// Examples:
	//
	//     * HOST -> Host
	//     * content-type -> Content-Type
	//     * cONTENT-lenGTH -> Content-Length
	DisableHeaderNamesNormalizing bool

	// Logger, which is used by RequestCtx.Logger().
	//
	// By default standard logger from log package is used.
	Logger Logger

	concurrency      uint32
	perIPConnCounter perIPConnCounter
	serverName       atomic.Value

	ctxPool        sync.Pool
	readerPool     sync.Pool
	writerPool     sync.Pool
	hijackConnPool sync.Pool
	bytePool       sync.Pool
}

// TimeoutHandler creates RequestHandler, which returns StatusRequestTimeout
// error with the given msg to the client if h didn't return during
// the given duration.
func TimeoutHandler(h RequestHandler, timeout time.Duration, msg string) RequestHandler {
	if timeout <= 0 {
		return h
	}

	return func(ctx *RequestCtx) {
		ch := ctx.timeoutCh
		if ch == nil {
			ch = make(chan struct{}, 1)
			ctx.timeoutCh = ch
		}
		go func() {
			h(ctx)
			ch <- struct{}{}
		}()
		ctx.timeoutTimer = initTimer(ctx.timeoutTimer, timeout)
		select {
		case <-ch:
		case <-ctx.timeoutTimer.C:
			ctx.TimeoutError(msg)
		}
		stopTimer(ctx.timeoutTimer)
	}
}

// CompressHandler returns RequestHandler that transparently compresses
// response body generated by h if the request contains 'gzip' or 'deflate'
// 'Accept-Encoding' header.
func CompressHandler(h RequestHandler) RequestHandler {
	return CompressHandlerLevel(h, CompressDefaultCompression)
}

// CompressHandlerLevel returns RequestHandler that transparently compresses
// response body generated by h if the request contains 'gzip' or 'deflate'
// 'Accept-Encoding' header.
//
// Level is the desired compression level:
//
//     * CompressNoCompression
//     * CompressBestSpeed
//     * CompressBestCompression
//     * CompressDefaultCompression
func CompressHandlerLevel(h RequestHandler, level int) RequestHandler {
	return func(ctx *RequestCtx) {
		h(ctx)
		if ctx.Request.Header.HasAcceptEncodingBytes(strGzip) {
			ctx.Response.gzipBody(level)
		} else if ctx.Request.Header.HasAcceptEncodingBytes(strDeflate) {
			ctx.Response.deflateBody(level)
		}
	}
}

// RequestCtx contains incoming request and manages outgoing response.
//
// It is forbidden copying RequestCtx instances.
//
// RequestHandler should avoid holding references to incoming RequestCtx and/or
// its' members after the return.
// If holding RequestCtx references after the return is unavoidable
// (for instance, ctx is passed to a separate goroutine and ctx lifetime cannot
// be controlled), then the RequestHandler MUST call ctx.TimeoutError()
// before return.
//
// It is unsafe modifying/reading RequestCtx instance from concurrently
// running goroutines. The only exception is TimeoutError*, which may be called
// while other goroutines accessing RequestCtx.
type RequestCtx struct {
	noCopy noCopy

	// Incoming request.
	//
	// Copying Request by value is forbidden. Use pointer to Request instead.
	Request Request

	// Outgoing response.
	//
	// Copying Response by value is forbidden. Use pointer to Response instead.
	Response Response

	userValues userData

	lastReadDuration time.Duration

	connID         uint64
	connRequestNum uint64
	connTime       time.Time

	time time.Time

	logger ctxLogger
	s      *Server
	c      net.Conn
	fbr    firstByteReader

	timeoutResponse *Response
	timeoutCh       chan struct{}
	timeoutTimer    *time.Timer

	hijackHandler HijackHandler
}

// HijackHandler must process the hijacked connection c.
//
// The connection c is automatically closed after returning from HijackHandler.
//
// The connection c must not be used after returning from the handler.
type HijackHandler func(c net.Conn)

// Hijack registers the given handler for connection hijacking.
//
// The handler is called after returning from RequestHandler
// and sending http response. The current connection is passed
// to the handler. The connection is automatically closed after
// returning from the handler.
//
// The server skips calling the handler in the following cases:
//
//     * 'Connection: close' header exists in either request or response.
//     * Unexpected error during response writing to the connection.
//
// The server stops processing requests from hijacked connections.
// Server limits such as Concurrency, ReadTimeout, WriteTimeout, etc.
// aren't applied to hijacked connections.
//
// The handler must not retain references to ctx members.
//
// Arbitrary 'Connection: Upgrade' protocols may be implemented
// with HijackHandler. For instance,
//
//     * WebSocket ( https://en.wikipedia.org/wiki/WebSocket )
//     * HTTP/2.0 ( https://en.wikipedia.org/wiki/HTTP/2 )
//
func (ctx *RequestCtx) Hijack(handler HijackHandler) {
	ctx.hijackHandler = handler
}

// SetUserValue stores the given value (arbitrary object)
// under the given key in ctx.
//
// The value stored in ctx may be obtained by UserValue*.
//
// This functionality may be useful for passing arbitrary values between
// functions involved in request processing.
//
// All the values are removed from ctx after returning from the top
// RequestHandler. Additionally, Close method is called on each value
// implementing io.Closer before removing the value from ctx.
func (ctx *RequestCtx) SetUserValue(key string, value interface{}) {
	ctx.userValues.Set(key, value)
}

// SetUserValueBytes stores the given value (arbitrary object)
// under the given key in ctx.
//
// The value stored in ctx may be obtained by UserValue*.
//
// This functionality may be useful for passing arbitrary values between
// functions involved in request processing.
//
// All the values stored in ctx are deleted after returning from RequestHandler.
func (ctx *RequestCtx) SetUserValueBytes(key []byte, value interface{}) {
	ctx.userValues.SetBytes(key, value)
}

// UserValue returns the value stored via SetUserValue* under the given key.
func (ctx *RequestCtx) UserValue(key string) interface{} {
	return ctx.userValues.Get(key)
}

// UserValueBytes returns the value stored via SetUserValue*
// under the given key.
func (ctx *RequestCtx) UserValueBytes(key []byte) interface{} {
	return ctx.userValues.GetBytes(key)
}

// IsTLS returns true if the underlying connection is tls.Conn.
//
// tls.Conn is an encrypted connection (aka SSL, HTTPS).
func (ctx *RequestCtx) IsTLS() bool {
	_, ok := ctx.c.(*tls.Conn)
	return ok
}

// TLSConnectionState returns TLS connection state.
//
// The function returns nil if the underlying connection isn't tls.Conn.
//
// The returned state may be used for verifying TLS version, client certificates,
// etc.
func (ctx *RequestCtx) TLSConnectionState() *tls.ConnectionState {
	tlsConn, ok := ctx.c.(*tls.Conn)
	if !ok {
		return nil
	}
	state := tlsConn.ConnectionState()
	return &state
}

type firstByteReader struct {
	c        net.Conn
	ch       byte
	byteRead bool
}

func (r *firstByteReader) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	nn := 0
	if !r.byteRead {
		b[0] = r.ch
		b = b[1:]
		r.byteRead = true
		nn = 1
	}
	n, err := r.c.Read(b)
	return n + nn, err
}

// Logger is used for logging formatted messages.
type Logger interface {
	// Printf must have the same semantics as log.Printf.
	Printf(format string, args ...interface{})
}

var ctxLoggerLock sync.Mutex

type ctxLogger struct {
	ctx    *RequestCtx
	logger Logger
}

func (cl *ctxLogger) Printf(format string, args ...interface{}) {
	ctxLoggerLock.Lock()
	msg := fmt.Sprintf(format, args...)
	ctx := cl.ctx
	req := &ctx.Request
	cl.logger.Printf("%.3f #%016X - %s<->%s - %s %s - %s",
		time.Since(ctx.Time()).Seconds(), ctx.ID(), ctx.LocalAddr(), ctx.RemoteAddr(), req.Header.Method(), ctx.URI().FullURI(), msg)
	ctxLoggerLock.Unlock()
}

var zeroTCPAddr = &net.TCPAddr{
	IP: net.IPv4zero,
}

// ID returns unique ID of the request.
func (ctx *RequestCtx) ID() uint64 {
	return (ctx.connID << 32) | ctx.connRequestNum
}

// ConnID returns unique connection ID.
//
// This ID may be used to match distinct requests to the same incoming
// connection.
func (ctx *RequestCtx) ConnID() uint64 {
	return ctx.connID
}

// Time returns RequestHandler call time.
func (ctx *RequestCtx) Time() time.Time {
	return ctx.time
}

// ConnTime returns the time server starts serving the connection
// the current request came from.
func (ctx *RequestCtx) ConnTime() time.Time {
	return ctx.connTime
}

// ConnRequestNum returns request sequence number
// for the current connection.
func (ctx *RequestCtx) ConnRequestNum() uint64 {
	return ctx.connRequestNum
}

// SetConnectionClose sets 'Connection: close' response header and closes
// connection after the RequestHandler returns.
func (ctx *RequestCtx) SetConnectionClose() {
	ctx.Response.SetConnectionClose()
}

// SetStatusCode sets response status code.
func (ctx *RequestCtx) SetStatusCode(statusCode int) {
	ctx.Response.SetStatusCode(statusCode)
}

// SetContentType sets response Content-Type.
func (ctx *RequestCtx) SetContentType(contentType string) {
	ctx.Response.Header.SetContentType(contentType)
}

// SetContentTypeBytes sets response Content-Type.
//
// It is safe modifying contentType buffer after function return.
func (ctx *RequestCtx) SetContentTypeBytes(contentType []byte) {
	ctx.Response.Header.SetContentTypeBytes(contentType)
}

// RequestURI returns RequestURI.
//
// This uri is valid until returning from RequestHandler.
func (ctx *RequestCtx) RequestURI() []byte {
	return ctx.Request.Header.RequestURI()
}

// URI returns requested uri.
//
// The uri is valid until returning from RequestHandler.
func (ctx *RequestCtx) URI() *URI {
	return ctx.Request.URI()
}

// Referer returns request referer.
//
// The referer is valid until returning from RequestHandler.
func (ctx *RequestCtx) Referer() []byte {
	return ctx.Request.Header.Referer()
}

// UserAgent returns User-Agent header value from the request.
func (ctx *RequestCtx) UserAgent() []byte {
	return ctx.Request.Header.UserAgent()
}

// Path returns requested path.
//
// The path is valid until returning from RequestHandler.
func (ctx *RequestCtx) Path() []byte {
	return ctx.URI().Path()
}

// Host returns requested host.
//
// The host is valid until returning from RequestHandler.
func (ctx *RequestCtx) Host() []byte {
	return ctx.URI().Host()
}

// QueryArgs returns query arguments from RequestURI.
//
// It doesn't return POST'ed arguments - use PostArge() for this.
//
// Returned arguments are valid until returning from RequestHandler.
//
// See also PostArgs, FormValue and FormFile.
func (ctx *RequestCtx) QueryArgs() *Args {
	return ctx.URI().QueryArgs()
}

// PostArgs returns POST arguments.
//
// It doesn't return query arguments from RequestURI - use QueryArgs for this.
//
// Returned arguments are valid until returning from RequestHandler.
//
// See also QueryArgs, FormValue and FormFile.
func (ctx *RequestCtx) PostArgs() *Args {
	return ctx.Request.PostArgs()
}

// MultipartForm returns requests's multipart form.
//
// Returns ErrNoMultipartForm if request's content-type
// isn't 'multipart/form-data'.
//
// All uploaded temporary files are automatically deleted after
// returning from RequestHandler. Either move or copy uploaded files
// into new place if you want retaining them.
//
// Use SaveMultipartFile function for permanently saving uploaded file.
//
// The returned form is valid until returning from RequestHandler.
//
// See also FormFile and FormValue.
func (ctx *RequestCtx) MultipartForm() (*multipart.Form, error) {
	return ctx.Request.MultipartForm()
}

// FormFile returns uploaded file associated with the given multipart form key.
//
// The file is automatically deleted after returning from RequestHandler,
// so either move or copy uploaded file into new place if you want retaining it.
//
// Use SaveMultipartFile function for permanently saving uploaded file.
//
// The returned file header is valid until returning from RequestHandler.
func (ctx *RequestCtx) FormFile(key string) (*multipart.FileHeader, error) {
	mf, err := ctx.MultipartForm()
	if err != nil {
		return nil, err
	}
	if mf.File == nil {
		return nil, err
	}
	fhh := mf.File[key]
	if fhh == nil {
		return nil, ErrMissingFile
	}
	return fhh[0], nil
}

// ErrMissingFile may be returned from FormFile when the is no uploaded file
// associated with the given multipart form key.
var ErrMissingFile = errors.New("there is no uploaded file associated with the given key")

// SaveMultipartFile saves multipart file fh under the given filename path.
func SaveMultipartFile(fh *multipart.FileHeader, path string) error {
	f, err := fh.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	if ff, ok := f.(*os.File); ok {
		return os.Rename(ff.Name(), path)
	}

	ff, err := os.Create(path)
	if err != nil {
		return err
	}
	defer ff.Close()
	_, err = copyZeroAlloc(ff, f)
	return err
}

// FormValue returns form value associated with the given key.
//
// The value is searched in the following places:
//
//   * Query string.
//   * POST or PUT body.
//
// There are more fine-grained methods for obtaining form values:
//
//   * QueryArgs for obtaining values from query string.
//   * PostArgs for obtaining values from POST or PUT body.
//   * MultipartForm for obtaining values from multipart form.
//   * FormFile for obtaining uploaded files.
//
// The returned value is valid until returning from RequestHandler.
func (ctx *RequestCtx) FormValue(key string) []byte {
	v := ctx.QueryArgs().Peek(key)
	if len(v) > 0 {
		return v
	}
	v = ctx.PostArgs().Peek(key)
	if len(v) > 0 {
		return v
	}
	mf, err := ctx.MultipartForm()
	if err == nil && mf.Value != nil {
		vv := mf.Value[key]
		if len(vv) > 0 {
			return []byte(vv[0])
		}
	}
	return nil
}

// IsGet returns true if request method is GET.
func (ctx *RequestCtx) IsGet() bool {
	return ctx.Request.Header.IsGet()
}

// IsPost returns true if request method is POST.
func (ctx *RequestCtx) IsPost() bool {
	return ctx.Request.Header.IsPost()
}

// IsPut returns true if request method is PUT.
func (ctx *RequestCtx) IsPut() bool {
	return ctx.Request.Header.IsPut()
}

// Method return request method.
//
// Returned value is valid until returning from RequestHandler.
func (ctx *RequestCtx) Method() []byte {
	return ctx.Request.Header.Method()
}

// IsHead returns true if request method is HEAD.
func (ctx *RequestCtx) IsHead() bool {
	return ctx.Request.Header.IsHead()
}

// RemoteAddr returns client address for the given request.
//
// Always returns non-nil result.
func (ctx *RequestCtx) RemoteAddr() net.Addr {
	addr := ctx.c.RemoteAddr()
	if addr == nil {
		return zeroTCPAddr
	}
	return addr
}

// LocalAddr returns server address for the given request.
//
// Always returns non-nil result.
func (ctx *RequestCtx) LocalAddr() net.Addr {
	addr := ctx.c.LocalAddr()
	if addr == nil {
		return zeroTCPAddr
	}
	return addr
}

// RemoteIP returns client ip for the given request.
//
// Always returns non-nil result.
func (ctx *RequestCtx) RemoteIP() net.IP {
	x, ok := ctx.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return net.IPv4zero
	}
	return x.IP
}

// Error sets response status code to the given value and sets response body
// to the given message.
func (ctx *RequestCtx) Error(msg string, statusCode int) {
	ctx.Response.Reset()
	ctx.SetStatusCode(statusCode)
	ctx.SetContentTypeBytes(defaultContentType)
	ctx.SetBodyString(msg)
}

// Success sets response Content-Type and body to the given values.
func (ctx *RequestCtx) Success(contentType string, body []byte) {
	ctx.SetContentType(contentType)
	ctx.SetBody(body)
}

// SuccessString sets response Content-Type and body to the given values.
func (ctx *RequestCtx) SuccessString(contentType, body string) {
	ctx.SetContentType(contentType)
	ctx.SetBodyString(body)
}

// Redirect sets 'Location: uri' response header and sets the given statusCode.
//
// statusCode must have one of the following values:
//
//    * StatusMovedPermanently (301)
//    * StatusFound (302)
//    * StatusSeeOther (303)
//    * StatusTemporaryRedirect (307)
//
// All other statusCode values are replaced by StatusFound (302).
//
// The redirect uri may be either absolute or relative to the current
// request uri.
func (ctx *RequestCtx) Redirect(uri string, statusCode int) {
	u := AcquireURI()
	ctx.URI().CopyTo(u)
	u.Update(uri)
	ctx.redirect(u.FullURI(), statusCode)
	ReleaseURI(u)
}

// RedirectBytes sets 'Location: uri' response header and sets
// the given statusCode.
//
// statusCode must have one of the following values:
//
//    * StatusMovedPermanently (301)
//    * StatusFound (302)
//    * StatusSeeOther (303)
//    * StatusTemporaryRedirect (307)
//
// All other statusCode values are replaced by StatusFound (302).
//
// The redirect uri may be either absolute or relative to the current
// request uri.
func (ctx *RequestCtx) RedirectBytes(uri []byte, statusCode int) {
	s := b2s(uri)
	ctx.Redirect(s, statusCode)
}

func (ctx *RequestCtx) redirect(uri []byte, statusCode int) {
	ctx.Response.Header.SetCanonical(strLocation, uri)
	statusCode = getRedirectStatusCode(statusCode)
	ctx.Response.SetStatusCode(statusCode)
}

func getRedirectStatusCode(statusCode int) int {
	if statusCode == StatusMovedPermanently || statusCode == StatusFound ||
		statusCode == StatusSeeOther || statusCode == StatusTemporaryRedirect {
		return statusCode
	}
	return StatusFound
}

// SetBody sets response body to the given value.
//
// It is safe re-using body argument after the function returns.
func (ctx *RequestCtx) SetBody(body []byte) {
	ctx.Response.SetBody(body)
}

// SetBodyString sets response body to the given value.
func (ctx *RequestCtx) SetBodyString(body string) {
	ctx.Response.SetBodyString(body)
}

// ResetBody resets response body contents.
func (ctx *RequestCtx) ResetBody() {
	ctx.Response.ResetBody()
}

// SendFile sends local file contents from the given path as response body.
//
// This is a shortcut to ServeFile(ctx, path).
//
// SendFile logs all the errors via ctx.Logger.
//
// See also ServeFile, FSHandler and FS.
func (ctx *RequestCtx) SendFile(path string) {
	ServeFile(ctx, path)
}

// SendFileBytes sends local file contents from the given path as response body.
//
// This is a shortcut to ServeFileBytes(ctx, path).
//
// SendFileBytes logs all the errors via ctx.Logger.
//
// See also ServeFileBytes, FSHandler and FS.
func (ctx *RequestCtx) SendFileBytes(path []byte) {
	ServeFileBytes(ctx, path)
}

// IfModifiedSince returns true if lastModified exceeds 'If-Modified-Since'
// value from the request header.
//
// The function returns true also 'If-Modified-Since' request header is missing.
func (ctx *RequestCtx) IfModifiedSince(lastModified time.Time) bool {
	ifModStr := ctx.Request.Header.peek(strIfModifiedSince)
	if len(ifModStr) == 0 {
		return true
	}
	ifMod, err := ParseHTTPDate(ifModStr)
	if err != nil {
		return true
	}
	lastModified = lastModified.Truncate(time.Second)
	return ifMod.Before(lastModified)
}

// NotModified resets response and sets '304 Not Modified' response status code.
func (ctx *RequestCtx) NotModified() {
	ctx.Response.Reset()
	ctx.SetStatusCode(StatusNotModified)
}

// NotFound resets response and sets '404 Not Found' response status code.
func (ctx *RequestCtx) NotFound() {
	ctx.Response.Reset()
	ctx.SetStatusCode(StatusNotFound)
	ctx.SetBodyString("404 Page not found")
}

// Write writes p into response body.
func (ctx *RequestCtx) Write(p []byte) (int, error) {
	ctx.Response.AppendBody(p)
	return len(p), nil
}

// WriteString appends s to response body.
func (ctx *RequestCtx) WriteString(s string) (int, error) {
	ctx.Response.AppendBodyString(s)
	return len(s), nil
}

// PostBody returns POST request body.
//
// The returned value is valid until RequestHandler return.
func (ctx *RequestCtx) PostBody() []byte {
	return ctx.Request.Body()
}

// SetBodyStream sets response body stream and, optionally body size.
//
// bodyStream.Close() is called after finishing reading all body data
// if it implements io.Closer.
//
// If bodySize is >= 0, then bodySize bytes must be provided by bodyStream
// before returning io.EOF.
//
// If bodySize < 0, then bodyStream is read until io.EOF.
//
// See also SetBodyStreamWriter.
func (ctx *RequestCtx) SetBodyStream(bodyStream io.Reader, bodySize int) {
	ctx.Response.SetBodyStream(bodyStream, bodySize)
}

// SetBodyStreamWriter registers the given stream writer for populating
// response body.
//
// Access to RequestCtx and/or its' members is forbidden from sw.
//
// This function may be used in the following cases:
//
//     * if response body is too big (more than 10MB).
//     * if response body is streamed from slow external sources.
//     * if response body must be streamed to the client in chunks.
//     (aka `http server push`).
func (ctx *RequestCtx) SetBodyStreamWriter(sw StreamWriter) {
	ctx.Response.SetBodyStreamWriter(sw)
}

// Logger returns logger, which may be used for logging arbitrary
// request-specific messages inside RequestHandler.
//
// Each message logged via returned logger contains request-specific information
// such as request id, request duration, local address, remote address,
// request method and request url.
//
// It is safe re-using returned logger for logging multiple messages
// for the current request.
//
// The returned logger is valid until returning from RequestHandler.
func (ctx *RequestCtx) Logger() Logger {
	if ctx.logger.ctx == nil {
		ctx.logger.ctx = ctx
	}
	if ctx.logger.logger == nil {
		ctx.logger.logger = ctx.s.logger()
	}
	return &ctx.logger
}

// TimeoutError sets response status code to StatusRequestTimeout and sets
// body to the given msg.
//
// All response modifications after TimeoutError call are ignored.
//
// TimeoutError MUST be called before returning from RequestHandler if there are
// references to ctx and/or its members in other goroutines remain.
//
// Usage of this function is discouraged. Prefer eliminating ctx references
// from pending goroutines instead of using this function.
func (ctx *RequestCtx) TimeoutError(msg string) {
	ctx.TimeoutErrorWithCode(msg, StatusRequestTimeout)
}

// TimeoutErrorWithCode sets response body to msg and response status
// code to statusCode.
//
// All response modifications after TimeoutErrorWithCode call are ignored.
//
// TimeoutErrorWithCode MUST be called before returning from RequestHandler
// if there are references to ctx and/or its members in other goroutines remain.
//
// Usage of this function is discouraged. Prefer eliminating ctx references
// from pending goroutines instead of using this function.
func (ctx *RequestCtx) TimeoutErrorWithCode(msg string, statusCode int) {
	var resp Response
	resp.SetStatusCode(statusCode)
	resp.SetBodyString(msg)
	ctx.TimeoutErrorWithResponse(&resp)
}

// TimeoutErrorWithResponse marks the ctx as timed out and sends the given
// response to the client.
//
// All ctx modifications after TimeoutErrorWithResponse call are ignored.
//
// TimeoutErrorWithResponse MUST be called before returning from RequestHandler
// if there are references to ctx and/or its members in other goroutines remain.
//
// Usage of this function is discouraged. Prefer eliminating ctx references
// from pending goroutines instead of using this function.
func (ctx *RequestCtx) TimeoutErrorWithResponse(resp *Response) {
	respCopy := &Response{}
	resp.CopyTo(respCopy)
	ctx.timeoutResponse = respCopy
}

// ListenAndServe serves HTTP requests from the given TCP addr.
func (s *Server) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

// ListenAndServeUNIX serves HTTP requests from the given UNIX addr.
//
// The function deletes existing file at addr before starting serving.
//
// The server sets the given file mode for the UNIX addr.
func (s *Server) ListenAndServeUNIX(addr string, mode os.FileMode) error {
	if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unexpected error when trying to remove unix socket file %q: %s", addr, err)
	}
	ln, err := net.Listen("unix", addr)
	if err != nil {
		return err
	}
	if err = os.Chmod(addr, mode); err != nil {
		return fmt.Errorf("cannot chmod %#o for %q: %s", mode, addr, err)
	}
	return s.Serve(ln)
}

// ListenAndServeTLS serves HTTPS requests from the given TCP addr.
//
// certFile and keyFile are paths to TLS certificate and key files.
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.ServeTLS(ln, certFile, keyFile)
}

// ListenAndServeTLSEmbed serves HTTPS requests from the given TCP addr.
//
// certData and keyData must contain valid TLS certificate and key data.
func (s *Server) ListenAndServeTLSEmbed(addr string, certData, keyData []byte) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.ServeTLSEmbed(ln, certData, keyData)
}

// ServeTLS serves HTTPS requests from the given listener.
//
// certFile and keyFile are paths to TLS certificate and key files.
func (s *Server) ServeTLS(ln net.Listener, certFile, keyFile string) error {
	lnTLS, err := newTLSListener(ln, certFile, keyFile)
	if err != nil {
		return err
	}
	return s.Serve(lnTLS)
}

// ServeTLSEmbed serves HTTPS requests from the given listener.
//
// certData and keyData must contain valid TLS certificate and key data.
func (s *Server) ServeTLSEmbed(ln net.Listener, certData, keyData []byte) error {
	lnTLS, err := newTLSListenerEmbed(ln, certData, keyData)
	if err != nil {
		return err
	}
	return s.Serve(lnTLS)
}

func newTLSListener(ln net.Listener, certFile, keyFile string) (net.Listener, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("cannot load TLS key pair from certFile=%q and keyFile=%q: %s", certFile, keyFile, err)
	}
	return newCertListener(ln, &cert), nil
}

func newTLSListenerEmbed(ln net.Listener, certData, keyData []byte) (net.Listener, error) {
	cert, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return nil, fmt.Errorf("cannot load TLS key pair from the provided certData(%d) and keyData(%d): %s",
			len(certData), len(keyData), err)
	}
	return newCertListener(ln, &cert), nil
}

func newCertListener(ln net.Listener, cert *tls.Certificate) net.Listener {
	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{*cert},
		PreferServerCipherSuites: true,
	}
	return tls.NewListener(ln, tlsConfig)
}

// DefaultConcurrency is the maximum number of concurrent connections
// the Server may serve by default (i.e. if Server.Concurrency isn't set).
const DefaultConcurrency = 256 * 1024

// Serve serves incoming connections from the given listener.
//
// Serve blocks until the given listener returns permanent error.
func (s *Server) Serve(ln net.Listener) error {
	var lastOverflowErrorTime time.Time
	var lastPerIPErrorTime time.Time
	var c net.Conn
	var err error

	maxWorkersCount := s.getConcurrency()
	wp := &workerPool{
		WorkerFunc:      s.serveConn,
		MaxWorkersCount: maxWorkersCount,
		LogAllErrors:    s.LogAllErrors,
		Logger:          s.logger(),
	}
	wp.Start()

	for {
		if c, err = acceptConn(s, ln, &lastPerIPErrorTime); err != nil {
			wp.Stop()
			if err == io.EOF {
				return nil
			}
			return err
		}
		if !wp.Serve(c) {
			s.writeFastError(c, StatusServiceUnavailable,
				"The connection cannot be served because Server.Concurrency limit exceeded")
			c.Close()
			if time.Since(lastOverflowErrorTime) > time.Minute {
				s.logger().Printf("The incoming connection cannot be served, because %d concurrent connections are served. "+
					"Try increasing Server.Concurrency", maxWorkersCount)
				lastOverflowErrorTime = time.Now()
			}

			// The current server reached concurrency limit,
			// so give other concurrently running servers a chance
			// accepting incoming connections on the same address.
			//
			// There is a hope other servers didn't reach their
			// concurrency limits yet :)
			time.Sleep(100 * time.Millisecond)
		}
		c = nil
	}
}

func acceptConn(s *Server, ln net.Listener, lastPerIPErrorTime *time.Time) (net.Conn, error) {
	for {
		c, err := ln.Accept()
		if err != nil {
			if c != nil {
				panic("BUG: net.Listener returned non-nil conn and non-nil error")
			}
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				s.logger().Printf("Temporary error when accepting new connections: %s", netErr)
				time.Sleep(time.Second)
				continue
			}
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				s.logger().Printf("Permanent error when accepting new connections: %s", err)
				return nil, err
			}
			return nil, io.EOF
		}
		if c == nil {
			panic("BUG: net.Listener returned (nil, nil)")
		}
		if s.MaxConnsPerIP > 0 {
			pic := wrapPerIPConn(s, c)
			if pic == nil {
				if time.Since(*lastPerIPErrorTime) > time.Minute {
					s.logger().Printf("The number of connections from %s exceeds MaxConnsPerIP=%d",
						getConnIP4(c), s.MaxConnsPerIP)
					*lastPerIPErrorTime = time.Now()
				}
				continue
			}
			c = pic
		}
		return c, nil
	}
}

func wrapPerIPConn(s *Server, c net.Conn) net.Conn {
	ip := getUint32IP(c)
	if ip == 0 {
		return c
	}
	n := s.perIPConnCounter.Register(ip)
	if n > s.MaxConnsPerIP {
		s.perIPConnCounter.Unregister(ip)
		s.writeFastError(c, StatusTooManyRequests, "The number of connections from your ip exceeds MaxConnsPerIP")
		c.Close()
		return nil
	}
	return acquirePerIPConn(c, ip, &s.perIPConnCounter)
}

var defaultLogger = Logger(log.New(os.Stderr, "", log.LstdFlags))

func (s *Server) logger() Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return defaultLogger
}

var (
	// ErrPerIPConnLimit may be returned from ServeConn if the number of connections
	// per ip exceeds Server.MaxConnsPerIP.
	ErrPerIPConnLimit = errors.New("too many connections per ip")

	// ErrConcurrencyLimit may be returned from ServeConn if the number
	// of concurrenty served connections exceeds Server.Concurrency.
	ErrConcurrencyLimit = errors.New("canot serve the connection because Server.Concurrency concurrent connections are served")

	// ErrKeepaliveTimeout is returned from ServeConn
	// if the connection lifetime exceeds MaxKeepaliveDuration.
	ErrKeepaliveTimeout = errors.New("exceeded MaxKeepaliveDuration")
)

// ServeConn serves HTTP requests from the given connection.
//
// ServeConn returns nil if all requests from the c are successfully served.
// It returns non-nil error otherwise.
//
// Connection c must immediately propagate all the data passed to Write()
// to the client. Otherwise requests' processing may hang.
//
// ServeConn closes c before returning.
func (s *Server) ServeConn(c net.Conn) error {
	if s.MaxConnsPerIP > 0 {
		pic := wrapPerIPConn(s, c)
		if pic == nil {
			return ErrPerIPConnLimit
		}
		c = pic
	}

	n := atomic.AddUint32(&s.concurrency, 1)
	if n > uint32(s.getConcurrency()) {
		atomic.AddUint32(&s.concurrency, ^uint32(0))
		s.writeFastError(c, StatusServiceUnavailable, "The connection cannot be served because Server.Concurrency limit exceeded")
		c.Close()
		return ErrConcurrencyLimit
	}

	err := s.serveConn(c)

	atomic.AddUint32(&s.concurrency, ^uint32(0))

	if err != errHijacked {
		err1 := c.Close()
		if err == nil {
			err = err1
		}
	} else {
		err = nil
	}
	return err
}

var errHijacked = errors.New("connection has been hijacked")

func (s *Server) getConcurrency() int {
	n := s.Concurrency
	if n <= 0 {
		n = DefaultConcurrency
	}
	return n
}

var globalConnID uint64

func nextConnID() uint64 {
	return atomic.AddUint64(&globalConnID, 1)
}

func (s *Server) serveConn(c net.Conn) error {
	connRequestNum := uint64(0)
	connID := nextConnID()
	currentTime := time.Now()
	connTime := currentTime

	ctx := s.acquireCtx(c)
	ctx.connTime = connTime
	var (
		br *bufio.Reader
		bw *bufio.Writer

		err             error
		timeoutResponse *Response
		hijackHandler   HijackHandler

		lastReadDeadlineTime  time.Time
		lastWriteDeadlineTime time.Time

		connectionClose bool
		isHTTP11        bool
	)
	for {
		connRequestNum++
		ctx.time = currentTime

		if s.ReadTimeout > 0 || s.MaxKeepaliveDuration > 0 {
			lastReadDeadlineTime = s.updateReadDeadline(c, ctx, lastReadDeadlineTime)
			if lastReadDeadlineTime.IsZero() {
				err = ErrKeepaliveTimeout
				break
			}
		}

		if !(s.ReduceMemoryUsage || ctx.lastReadDuration > time.Second) || br != nil {
			if br == nil {
				br = acquireReader(ctx)
			}
		} else {
			br, err = acquireByteReader(&ctx)
		}

		if err == nil {
			if s.DisableHeaderNamesNormalizing {
				ctx.Request.Header.DisableNormalizing()
				ctx.Response.Header.DisableNormalizing()
			}
			err = ctx.Request.readLimitBody(br, s.MaxRequestBodySize, s.GetOnly)
			if br.Buffered() == 0 || err != nil {
				releaseReader(s, br)
				br = nil
			}
		}

		currentTime = time.Now()
		ctx.lastReadDuration = currentTime.Sub(ctx.time)

		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}

		// 'Expect: 100-continue' request handling.
		// See http://www.w3.org/Protocols/rfc2616/rfc2616-sec8.html for details.
		if !ctx.Request.Header.noBody() && ctx.Request.MayContinue() {
			// Send 'HTTP/1.1 100 Continue' response.
			if bw == nil {
				bw = acquireWriter(ctx)
			}
			bw.Write(strResponseContinue)
			err = bw.Flush()
			releaseWriter(s, bw)
			bw = nil
			if err != nil {
				break
			}

			// Read request body.
			if br == nil {
				br = acquireReader(ctx)
			}
			err = ctx.Request.ContinueReadBody(br, s.MaxRequestBodySize)
			if br.Buffered() == 0 || err != nil {
				releaseReader(s, br)
				br = nil
			}
			if err != nil {
				break
			}
		}

		connectionClose = s.DisableKeepalive || ctx.Request.Header.connectionCloseFast()
		isHTTP11 = ctx.Request.Header.IsHTTP11()

		ctx.connID = connID
		ctx.connRequestNum = connRequestNum
		ctx.connTime = connTime
		ctx.time = currentTime
		s.Handler(ctx)

		timeoutResponse = ctx.timeoutResponse
		if timeoutResponse != nil {
			ctx = s.acquireCtx(c)
			timeoutResponse.CopyTo(&ctx.Response)
			if br != nil {
				// Close connection, since br may be attached to the old ctx via ctx.fbr.
				ctx.SetConnectionClose()
			}
		}

		if !ctx.IsGet() && ctx.IsHead() {
			ctx.Response.SkipBody = true
		}
		ctx.Request.Reset()

		hijackHandler = ctx.hijackHandler
		ctx.hijackHandler = nil

		ctx.userValues.Reset()

		if s.MaxRequestsPerConn > 0 && connRequestNum >= uint64(s.MaxRequestsPerConn) {
			ctx.SetConnectionClose()
		}

		if s.WriteTimeout > 0 || s.MaxKeepaliveDuration > 0 {
			lastWriteDeadlineTime = s.updateWriteDeadline(c, ctx, lastWriteDeadlineTime)
		}

		// Verify Request.Header.connectionCloseFast() again,
		// since request handler might trigger full headers' parsing.
		connectionClose = connectionClose || ctx.Request.Header.connectionCloseFast() || ctx.Response.ConnectionClose()
		if connectionClose {
			ctx.Response.Header.SetCanonical(strConnection, strClose)
		} else if !isHTTP11 {
			// Set 'Connection: keep-alive' response header for non-HTTP/1.1 request.
			// There is no need in setting this header for http/1.1, since in http/1.1
			// connections are keep-alive by default.
			ctx.Response.Header.SetCanonical(strConnection, strKeepAlive)
		}

		if bw == nil {
			bw = acquireWriter(ctx)
		}
		if err = writeResponse(ctx, bw); err != nil {
			break
		}

		if br == nil || connectionClose {
			err = bw.Flush()
			releaseWriter(s, bw)
			bw = nil
			if err != nil {
				break
			}
			if connectionClose {
				break
			}
		}

		if hijackHandler != nil {
			var hjr io.Reader
			hjr = c
			if br != nil {
				hjr = br
				br = nil

				// br may point to ctx.fbr, so do not return ctx into pool.
				ctx = s.acquireCtx(c)
			}
			if bw != nil {
				err = bw.Flush()
				releaseWriter(s, bw)
				bw = nil
				if err != nil {
					break
				}
			}
			c.SetReadDeadline(zeroTime)
			c.SetWriteDeadline(zeroTime)
			go hijackConnHandler(hjr, c, s, hijackHandler)
			hijackHandler = nil
			err = errHijacked
			break
		}

		currentTime = time.Now()
	}

	if br != nil {
		releaseReader(s, br)
	}
	if bw != nil {
		releaseWriter(s, bw)
	}
	s.releaseCtx(ctx)
	return err
}

func (s *Server) updateReadDeadline(c net.Conn, ctx *RequestCtx, lastDeadlineTime time.Time) time.Time {
	readTimeout := s.ReadTimeout
	currentTime := ctx.time
	if s.MaxKeepaliveDuration > 0 {
		connTimeout := s.MaxKeepaliveDuration - currentTime.Sub(ctx.connTime)
		if connTimeout <= 0 {
			return zeroTime
		}
		if connTimeout < readTimeout {
			readTimeout = connTimeout
		}
	}

	// Optimization: update read deadline only if more than 25%
	// of the last read deadline exceeded.
	// See https://github.com/golang/go/issues/15133 for details.
	if currentTime.Sub(lastDeadlineTime) > (readTimeout >> 2) {
		if err := c.SetReadDeadline(currentTime.Add(readTimeout)); err != nil {
			panic(fmt.Sprintf("BUG: error in SetReadDeadline(%s): %s", readTimeout, err))
		}
		lastDeadlineTime = currentTime
	}
	return lastDeadlineTime
}

func (s *Server) updateWriteDeadline(c net.Conn, ctx *RequestCtx, lastDeadlineTime time.Time) time.Time {
	writeTimeout := s.WriteTimeout
	if s.MaxKeepaliveDuration > 0 {
		connTimeout := s.MaxKeepaliveDuration - time.Since(ctx.connTime)
		if connTimeout <= 0 {
			// MaxKeepAliveDuration exceeded, but let's try sending response anyway
			// in 100ms with 'Connection: close' header.
			ctx.SetConnectionClose()
			connTimeout = 100 * time.Millisecond
		}
		if connTimeout < writeTimeout {
			writeTimeout = connTimeout
		}
	}

	// Optimization: update write deadline only if more than 25%
	// of the last write deadline exceeded.
	// See https://github.com/golang/go/issues/15133 for details.
	currentTime := time.Now()
	if currentTime.Sub(lastDeadlineTime) > (writeTimeout >> 2) {
		if err := c.SetWriteDeadline(currentTime.Add(writeTimeout)); err != nil {
			panic(fmt.Sprintf("BUG: error in SetWriteDeadline(%s): %s", writeTimeout, err))
		}
		lastDeadlineTime = currentTime
	}
	return lastDeadlineTime
}

func hijackConnHandler(r io.Reader, c net.Conn, s *Server, h HijackHandler) {
	hjc := s.acquireHijackConn(r, c)

	defer func() {
		if r := recover(); r != nil {
			s.logger().Printf("panic on hijacked conn: %s\nStack trace:\n%s", r, debug.Stack())
		}

		if br, ok := r.(*bufio.Reader); ok {
			releaseReader(s, br)
		}
		c.Close()
		s.releaseHijackConn(hjc)
	}()

	h(hjc)
}

func (s *Server) acquireHijackConn(r io.Reader, c net.Conn) *hijackConn {
	v := s.hijackConnPool.Get()
	if v == nil {
		hjc := &hijackConn{
			Conn: c,
			r:    r,
		}
		return hjc
	}
	hjc := v.(*hijackConn)
	hjc.Conn = c
	hjc.r = r
	return hjc
}

func (s *Server) releaseHijackConn(hjc *hijackConn) {
	hjc.Conn = nil
	hjc.r = nil
	s.hijackConnPool.Put(hjc)
}

type hijackConn struct {
	net.Conn
	r io.Reader
}

func (c hijackConn) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func (c hijackConn) Close() error {
	// hijacked conn is closed in hijackConnHandler.
	return nil
}

// LastTimeoutErrorResponse returns the last timeout response set
// via TimeoutError* call.
//
// This function is intended for custom server implementations.
func (ctx *RequestCtx) LastTimeoutErrorResponse() *Response {
	return ctx.timeoutResponse
}

func writeResponse(ctx *RequestCtx, w *bufio.Writer) error {
	if ctx.timeoutResponse != nil {
		panic("BUG: cannot write timed out response")
	}
	h := &ctx.Response.Header
	serverOld := h.Server()
	if len(serverOld) == 0 {
		h.server = ctx.s.getServerName()
	}
	err := ctx.Response.Write(w)
	if len(serverOld) == 0 {
		h.server = serverOld
	}
	ctx.Response.Reset()
	return err
}

const (
	defaultReadBufferSize  = 4096
	defaultWriteBufferSize = 4096
)

func acquireByteReader(ctxP **RequestCtx) (*bufio.Reader, error) {
	ctx := *ctxP
	s := ctx.s
	c := ctx.c
	t := ctx.time
	s.releaseCtx(ctx)

	// Make GC happy, so it could garbage collect ctx
	// while we waiting for the next request.
	ctx = nil
	*ctxP = nil

	v := s.bytePool.Get()
	if v == nil {
		v = make([]byte, 1)
	}
	b := v.([]byte)
	n, err := c.Read(b)
	ch := b[0]
	s.bytePool.Put(v)
	ctx = s.acquireCtx(c)
	ctx.time = t
	*ctxP = ctx
	if err != nil {
		// Treat all errors as EOF on unsuccessful read
		// of the first request byte.
		return nil, io.EOF
	}
	if n != 1 {
		panic("BUG: Reader must return at least one byte")
	}

	ctx.fbr.c = c
	ctx.fbr.ch = ch
	ctx.fbr.byteRead = false
	r := acquireReader(ctx)
	r.Reset(&ctx.fbr)
	return r, nil
}

func acquireReader(ctx *RequestCtx) *bufio.Reader {
	v := ctx.s.readerPool.Get()
	if v == nil {
		n := ctx.s.ReadBufferSize
		if n <= 0 {
			n = defaultReadBufferSize
		}
		return bufio.NewReaderSize(ctx.c, n)
	}
	r := v.(*bufio.Reader)
	r.Reset(ctx.c)
	return r
}

func releaseReader(s *Server, r *bufio.Reader) {
	s.readerPool.Put(r)
}

func acquireWriter(ctx *RequestCtx) *bufio.Writer {
	v := ctx.s.writerPool.Get()
	if v == nil {
		n := ctx.s.WriteBufferSize
		if n <= 0 {
			n = defaultWriteBufferSize
		}
		return bufio.NewWriterSize(ctx.c, n)
	}
	w := v.(*bufio.Writer)
	w.Reset(ctx.c)
	return w
}

func releaseWriter(s *Server, w *bufio.Writer) {
	s.writerPool.Put(w)
}

func (s *Server) acquireCtx(c net.Conn) *RequestCtx {
	v := s.ctxPool.Get()
	var ctx *RequestCtx
	if v == nil {
		v = &RequestCtx{
			s: s,
		}
	}
	ctx = v.(*RequestCtx)
	ctx.c = c
	return ctx
}

// Init prepares ctx for passing to RequestHandler.
//
// remoteAddr and logger are optional. They are used by RequestCtx.Logger().
//
// This function is intended for custom Server implementations.
func (ctx *RequestCtx) Init(req *Request, remoteAddr net.Addr, logger Logger) {
	if remoteAddr == nil {
		remoteAddr = zeroTCPAddr
	}
	ctx.c = &fakeAddrer{
		addr: remoteAddr,
	}
	if logger == nil {
		logger = defaultLogger
	}
	ctx.connID = nextConnID()
	ctx.logger.logger = logger
	ctx.s = &fakeServer
	req.CopyTo(&ctx.Request)
	ctx.Response.Reset()
	ctx.connRequestNum = 0
	ctx.connTime = time.Now()
	ctx.time = ctx.connTime
}

var fakeServer Server

type fakeAddrer struct {
	net.Conn
	addr net.Addr
}

func (fa *fakeAddrer) RemoteAddr() net.Addr {
	return fa.addr
}

func (fa *fakeAddrer) LocalAddr() net.Addr {
	return fa.addr
}

func (fa *fakeAddrer) Read(p []byte) (int, error) {
	panic("BUG: unexpected Read call")
}

func (fa *fakeAddrer) Write(p []byte) (int, error) {
	panic("BUG: unexpected Write call")
}

func (fa *fakeAddrer) Close() error {
	panic("BUG: unexpected Close call")
}

func (s *Server) releaseCtx(ctx *RequestCtx) {
	if ctx.timeoutResponse != nil {
		panic("BUG: cannot release timed out RequestCtx")
	}
	ctx.c = nil
	ctx.fbr.c = nil
	s.ctxPool.Put(ctx)
}

func (s *Server) getServerName() []byte {
	v := s.serverName.Load()
	var serverName []byte
	if v == nil {
		serverName = []byte(s.Name)
		if len(serverName) == 0 {
			serverName = defaultServerName
		}
		s.serverName.Store(serverName)
	} else {
		serverName = v.([]byte)
	}
	return serverName
}

func (s *Server) writeFastError(w io.Writer, statusCode int, msg string) {
	w.Write(statusLine(statusCode))
	fmt.Fprintf(w, "Connection: close\r\n"+
		"Server: %s\r\n"+
		"Date: %s\r\n"+
		"Content-Type: text/plain\r\n"+
		"Content-Length: %d\r\n"+
		"\r\n"+
		"%s",
		s.getServerName(), serverDate.Load(), len(msg), msg)
}
