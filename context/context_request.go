package context

import (
	"context"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kataras/iris/v14/context/encoding"
	"github.com/kataras/iris/v14/core/pkg/jsonutil"
	"github.com/kataras/iris/v14/core/pkg/netutil"
	"github.com/kataras/iris/v14/core/router/macro"
	"github.com/kataras/iris/v14/core/server"

	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"

	"github.com/iris-contrib/schema"
)

// IsErrCanceled reports whether the "err" is caused by a cancellation or timeout.
func IsErrCanceled(err error) bool {
	if err == nil {
		return false
	}

	netErr, isNetErr := errors.AsType[net.Error](err)
	return (isNetErr && netErr.Timeout()) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, http.ErrHandlerTimeout)
}

// ShuttingDown returns a channel that is closed when the server begins a
// graceful shutdown.
//
// It is deliberately a separate signal from cancellation. A draining stream
// still needs to write its terminal frame, so the request context stays live;
// select on both.
//
//	for {
//		select {
//		case <-ctx.ShuttingDown():
//			return // the server is going away: finish cleanly.
//		case <-ctx.Done():
//			return // the client is gone.
//		case ev := <-events:
//			// ...
//		}
//	}
//
// Without it a handler blocked in a stream loop is never told that the process
// is stopping: net/http cancels a request context on client disconnect, on an
// HTTP/2 stream reset and when ServeHTTP returns, and shutdown is none of those.
// http.Server.Shutdown then waits for a handler that cannot know it should stop.
//
// It returns a nil channel, which blocks forever, when the request did not come
// from a server that Iris started. That is the correct answer for a handler
// running under httptest or under another host: there is no shutdown to observe.
func (ctx *Context) ShuttingDown() <-chan struct{} {
	if ctx.request == nil {
		return nil
	}

	return server.ShutdownSignal(ctx.request.Context())
}

// IsCanceled reports whether the client canceled the request
// or the underlying connection has gone.
// Note that it will always return true
// when called from a goroutine after the request-response lifecycle.
func (ctx *Context) IsCanceled() bool {
	var err error
	if reqCtx := ctx.request.Context(); reqCtx != nil {
		err = reqCtx.Err()
	} else {
		err = ctx.GetErr()
	}

	return IsErrCanceled(err)
}

// OnConnectionClose registers the "cb" Handler
// which will be fired on its on goroutine on a cloned Context
// when the underlying connection has gone away.
//
// The code inside the given callback is running on its own routine,
// as explained above, therefore the callback should NOT
// try to access to handler's Context response writer.
//
// This mechanism can be used to cancel long operations on the server
// if the client has disconnected before the response is ready.
//
// It depends on the Request's Context.Done() channel.
//
// The Context handed to "cb" is a snapshot taken at the moment
// OnConnectionClose is called, not at the moment the connection goes away.
// Values and params set by handlers that run later in the chain are therefore
// not visible to the callback; read what the callback needs before registering
// it, or pass it through the closure.
//
// Finally, it reports whether the protocol supports pipelines (HTTP/1.1 with pipelines disabled is not supported).
// The "cb" will not fire for sure if the output value is false.
//
// Note that you can register only one callback per route.
func (ctx *Context) OnConnectionClose(cb Handler) bool {
	if cb == nil {
		return false
	}

	reqCtx := ctx.Request().Context()
	if reqCtx == nil {
		return false
	}

	notifyClose := reqCtx.Done()
	if notifyClose == nil {
		return false
	}

	// Note(@kataras): the clone MUST happen here, before the goroutine starts.
	//
	// notifyClose fires at the end of every request, not only when the client
	// disconnects: net/http cancels the request's context right after ServeHTTP
	// returns. The router releases the Context back to the pool synchronously at
	// that same point (see core/router.buildMainHandler -> contextPool.Release),
	// which nils out its request and writer. Cloning inside the goroutine races
	// that release: either it reads a nil request and panics in a goroutine
	// net/http cannot recover from, or it reads a Context that another in-flight
	// request has already acquired and hands the callback someone else's data.
	cloned := ctx.Clone()

	go func() {
		<-notifyClose
		cb(cloned)
	}()

	return true
}

// SetRequest sets the Context's Request,
// It is useful to store the new request created by a std *http.Request#WithContext() into Iris' Context.
// Use `SetRequest` when for some reason you want to make a full
// override of the *http.Request.
// Note that: when you just want to change one of each fields you can use the Request() which returns a pointer to Request,
// so the changes will have affect without a full override.
// Usage: you use a native http handler which uses the standard "context" package
// to get values instead of the Iris' Context#Values():
// r := ctx.Request()
// stdCtx := context.WithValue(r.Context(), key, val)
// ctx.SetRequest(r.WithContext(stdCtx)).
func (ctx *Context) SetRequest(r *http.Request) {
	ctx.request = r
}

// Request returns the original *http.Request, as expected.
func (ctx *Context) Request() *http.Request {
	return ctx.request
}

// DumpRequest returns the given request in its HTTP/1.x wire
// representation. It should only be used by servers to debug client
// requests. If "includeBody" is true the request body is drained,
// if it's already drained it cannot be captured (unless the middleware/recorder
// package's RecordRequestBody(ctx) was called previously),
// so store the dump request in a middleware before the main handler.
//
// It just calls the standard httputil.DumpRequest function.
func (ctx *Context) DumpRequest(includeBody bool) ([]byte, error) {
	return httputil.DumpRequest(ctx.request, includeBody)
}

// Values returns the current "user" storage.
// Named path parameters and any optional data can be saved here.
// This storage, as the whole context, is per-request lifetime.
//
// You can use this function to Set and Get local values
// that can be used to share information between handlers and middleware.
func (ctx *Context) Values() *encoding.ContextValues {
	return &ctx.values
}

// ValueAs returns any cached value of the given key from the Iris Context,
// type-asserted to T. It reports whether the value was found and of type T.
func (ctx *Context) ValueAs[T any](key string) (value T, gotValue bool) {
	if contextRequestValue := ctx.values.Get(key); contextRequestValue != nil {
		value, gotValue = contextRequestValue.(T)
	}

	return
}

//  +------------------------------------------------------------+
//  | Path, Host, Subdomain, IP, Headers etc...                  |
//  +------------------------------------------------------------+

// Method returns the request.Method, the client's http method to the server.
func (ctx *Context) Method() string {
	return ctx.request.Method
}

// FindClosest returns a list of "n" paths close to
// this request based on subdomain and request path.
//
// Order may change.
func (ctx *Context) FindClosest(n int) []string {
	return ctx.app.FindClosestPaths(ctx.Subdomain(), ctx.Path(), n)
}

// Addr returns the remote address of the client.
// Its different from `RemoteAddr` because it doesn't try to parse the real client's request IP as a public one,
// so this can return a valid value even if the client is a local one.
//
// SECURITY WARNING: never use this for access control, rate limiting or any
// other authorization decision.
//
// When Options.RemoteAddrHeaders is configured, this returns the FIRST entry of
// the first matching header - the left-most hop, which is entirely written by
// the client. Any caller can send "X-Forwarded-For: 10.0.0.1" and be seen as
// that address. Unlike `RemoteAddr`, no private-subnet filtering is applied and
// RemoteAddrHeadersForce is not consulted.
//
// Use `RemoteAddr` for anything that makes a decision about the request, and
// only enable RemoteAddrHeaders when running behind a trusted proxy that
// overwrites those headers.
//
// Addr exists for logging and diagnostics, where seeing the client's own claim
// (including a private-network address) is the point.
func (ctx *Context) Addr() string {
	reqOpts := ctx.OptionsReadOnly()

	for _, headerName := range reqOpts.RemoteAddrHeaders {
		// return the first valid IP,
		//  even if it's a part of a private network.
		ipAddresses := strings.SplitSeq(strings.TrimSpace(ctx.Header(headerName)), ",")
		for addr := range ipAddresses {
			if ip := getIP(addr); ip != "" {
				return ip
			}
		}
	}

	addr := strings.TrimSpace(ctx.request.RemoteAddr)
	if addr != "" {
		if ip := getIP(addr); ip != "" {
			return ip
		}
	}

	return addr
}

// RemoteAddr returns the client's IP address.
//
// SECURITY WARNING: This method trusts headers configured in RemoteAddrHeaders.
// Only enable RemoteAddrHeaders when running behind a trusted proxy.
//
// Configuration:
//   - RemoteAddrHeaders: Headers to check for client IP (default: empty)
//   - RemoteAddrPrivateSubnets: Private IP ranges to skip
//   - RemoteAddrHeadersForce: Force using headers even for private IPs
//
// Best Practices:
//  1. Never trust client IPs for security decisions without additional validation
//  2. Use rate limiting by IP with caution (can be bypassed)
//  3. Log IPs for audit purposes but don't rely solely on them.
func (ctx *Context) RemoteAddr() string {
	reqOpts := ctx.OptionsReadOnly()

	if remoteHeaders := reqOpts.RemoteAddrHeaders; len(remoteHeaders) > 0 {
		privateSubnets := reqOpts.RemoteAddrPrivateSubnets

		for _, headerName := range remoteHeaders {
			ipAddresses := strings.Split(strings.TrimSpace(ctx.Header(headerName)), ",")
			if ip, ok := netutil.GetIPAddress(ipAddresses, privateSubnets); ok {
				return ip
			}
		}

		if reqOpts.RemoteAddrHeadersForce {
			for _, headerName := range remoteHeaders {
				// return the first valid IP,
				//  even if it's a part of a private network.
				ipAddresses := strings.SplitSeq(ctx.Header(headerName), ",")
				for addr := range ipAddresses {
					if ip := getIP(addr); ip != "" {
						return ip
					}
				}
			}
		}
	}

	addr := strings.TrimSpace(ctx.request.RemoteAddr)
	return getIP(addr)
}

// getIP normalises an address that may or may not carry a port.
//
// Note(@kataras): a bare address is tried FIRST.
//
// The previous order went through net.SplitHostPort and only fell back when the
// error said "missing port in address". A bare IPv6 address fails with "too many
// colons in address" instead, so every IPv6 entry of an X-Forwarded-For chain
// was dropped. And the fallback did net.ParseIP(addr).String() without checking
// for nil, which renders as the literal string "<nil>" - so an unparseable value
// was returned as if it were an address.
func getIP(addr string) string {
	if ip := net.ParseIP(addr); ip != nil {
		return ip.String()
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return ""
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}

	return ""
}

// TrimHeaderValue returns the "v[0:first space or semicolon]".
func TrimHeaderValue(v string) string {
	for i, char := range v {
		if char == ' ' || char == ';' {
			return v[:i]
		}
	}
	return v
}

// Header returns the REQUEST header's value based on its name,
// i.e. a header the client sent. Headers this handler set on the
// response are read back through Context.ResponseHeader.
//
//	apiKey := ctx.Header("X-API-Key")         // request  (what the client sent)
//	cached := ctx.ResponseHeader("X-Cache")   // response (what this handler set)
//
// When the client repeats a header, its values are returned joined by comma
// and space, unlike http.Header.Get which returns only the first. Code that
// compares the result against a secret therefore rejects a request carrying
// two copies of the header rather than honoring the first.
func (ctx *Context) Header(name string) string {
	// return ctx.request.Header.Get(name)
	return strings.Join(ctx.request.Header.Values(name), ", ")
}

// ReadHeader binds request header to "ptr". The struct field tag is "header".
func (ctx *Context) ReadHeader(ptr any) error {
	err := schema.DecodeHeaders(ctx.request.Header, ptr)
	if err != nil {
		return err
	}

	return ctx.OptionsReadOnly().Validate(ptr)
}

// IsAjax returns true if this request is an 'ajax request'( XMLHttpRequest)
//
// There is no a 100% way of knowing that a request was made via Ajax.
// You should never trust data coming from the client, they can be easily overcome by spoofing.
//
// Note that "X-Requested-With" Header can be modified by any client(because of "X-"),
// so don't rely on IsAjax for really serious stuff,
// try to find another way of detecting the type(i.e, content type),
// there are many blogs that describe these problems and provide different kind of solutions,
// it's always depending on the application you're building,
// this is the reason why this `IsAjax` is simple enough for general purpose use.
//
// Read more at: https://developer.mozilla.org/en-US/docs/AJAX
// and https://xhr.spec.whatwg.org/
func (ctx *Context) IsAjax() bool {
	return ctx.Header("X-Requested-With") == "XMLHttpRequest"
}

var isMobileRegex = regexp.MustCompile("(?:hpw|i|web)os|alamofire|alcatel|amoi|android|avantgo|blackberry|blazer|cell|cfnetwork|darwin|dolfin|dolphin|fennec|htc|ip(?:hone|od|ad)|ipaq|j2me|kindle|midp|minimo|mobi|motorola|nec-|netfront|nokia|opera m(ob|in)i|palm|phone|pocket|portable|psp|silk-accelerated|skyfire|sony|ucbrowser|up.browser|up.link|windows ce|xda|zte|zune")

// IsMobile checks if client is using a mobile device(phone or tablet) to communicate with this server.
// If the return value is true that means that the http client using a mobile
// device to communicate with the server, otherwise false.
//
// Keep note that this checks the "User-Agent" request header.
func (ctx *Context) IsMobile() bool {
	s := strings.ToLower(ctx.Header("User-Agent"))
	return isMobileRegex.MatchString(s)
}

var isScriptRegex = regexp.MustCompile("curl|wget|collectd|python|urllib|java|jakarta|httpclient|phpcrawl|libwww|perl|go-http|okhttp|lua-resty|winhttp|awesomium")

// IsScript reports whether a client is a script.
func (ctx *Context) IsScript() bool {
	s := strings.ToLower(ctx.Header("User-Agent"))
	return isScriptRegex.MatchString(s)
}

// IsSSL reports whether the client is running under HTTPS SSL.
//
// See `IsHTTP2` too.
func (ctx *Context) IsSSL() bool {
	ssl := strings.EqualFold(ctx.request.URL.Scheme, "https") || ctx.request.TLS != nil
	if !ssl {
		for k, v := range ctx.OptionsReadOnly().SSLProxyHeaders {
			if ctx.Header(k) == v {
				ssl = true
				break
			}
		}
	}
	return ssl
}

// IsHTTP2 reports whether the protocol version for incoming request was HTTP/2.
// The client code always uses either HTTP/1.1 or HTTP/2.
//
// See `IsSSL` too.
func (ctx *Context) IsHTTP2() bool {
	return ctx.request.ProtoMajor == 2
}

// IsGRPC reports whether the request came from a gRPC client.
func (ctx *Context) IsGRPC() bool {
	return ctx.IsHTTP2() && ctx.IsMediaType(ContentGRPCHeaderValue)
}

// IsLocal reports whether the request is from a local client.
func (ctx *Context) IsLocal() bool {
	remoteAddr := ctx.Request().RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If parsing fails, fallback to direct comparison
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsUnspecified())
}

// ContentType returns the request's Content-Type header.
func (ctx *Context) ContentType() ContentType {
	ct, _ := ParseContentType(ctx.Header(ContentTypeHeaderKey))
	return ct
}

// MediaType returns the media type of the request's Content-Type header.
// If the Content-Type header does not contain a mime type, returns an empty string.
func (ctx *Context) MediaType() string {
	return TrimHeaderValue(ctx.Header(ContentTypeHeaderKey))
	// faster than: return ctx.ContentType().MediaType
}

// IsMediaType reports whether the request's or response's Content-Type header contains at least one provided media type.
func (ctx *Context) IsMediaType(mediaTypes ...string) bool {
	requestMediaType := ctx.MediaType()
	responseMediaType := ctx.ResponseMediaType()
	for _, mediaType := range mediaTypes {
		if requestMediaType == mediaType || responseMediaType == mediaType {
			return true
		}
	}

	return false
}

// Charset returns the charset of the request's Content-Type header.
// If the Content-Type header parsing failed or provided but charset parameter is empty it returns an empty string.
func (ctx *Context) Charset() string {
	return ctx.ContentType().Charset
}

// ContentLength returns the request's body length in bytes, or -1 when it is
// unknown (a chunked request, for example).
//
// Note(@kataras): this reads Request.ContentLength rather than re-parsing the
// header. ctx.Header joins duplicate headers with ", ", so a client sending two
// Content-Length headers produced a value like "10, 10" that ParseInt rejected,
// and the method silently answered 0 - the exact case a size guard needs to
// catch. net/http has already validated and reconciled the framing by the time a
// handler runs.
func (ctx *Context) ContentLength() int64 {
	return ctx.request.ContentLength
}

// Params returns the current url's named parameters key-value storage.
// Named path parameters are being saved here.
// This storage, as the whole context, is per-request lifetime.
func (ctx *Context) Params() *encoding.PathParams {
	return &ctx.params
}

// LastPathParameter returns the last path parameter of the current route.
func (ctx *Context) LastPathParameter() (lastParameter macro.TemplateParam) {
	if route := ctx.CurrentRoute(); route != nil {
		tmpl := route.Tmpl()
		if n := len(tmpl.Params); n > 0 {
			lastParameter = tmpl.Params[n-1]
		}
	}

	return
}

// ReadParams binds URL Dynamic Path Parameters to "ptr". The struct field tag is "param".
func (ctx *Context) ReadParams(ptr any) error {
	n := ctx.params.Len()
	if n == 0 {
		return nil
	}

	values := ctx.params.ToMap()

	err := schema.DecodeParams(values, ptr)
	if err != nil {
		return err
	}

	return ctx.OptionsReadOnly().Validate(ptr)
}

// Path returns the request path in its decoded form (URL.Path): "/a%20b"
// reads "/a b". See `EscapedPath` for the percent-encoded form the client sent.
func (ctx *Context) Path() string {
	return ctx.request.URL.Path
}

// ReadURL is a shortcut of ReadParams and ReadQuery.
// It binds dynamic path parameters and URL query parameters
// to the "ptr" pointer struct value.
// The struct fields may contain "url" or "param" binding tags.
// If a validator exists then it validates the result too.
func (ctx *Context) ReadURL(ptr any) error {
	values := make(map[string][]string, ctx.params.Len())
	ctx.params.ForEachString(func(key string, value string) {
		values[key] = strings.Split(value, "/")
	})

	for k, v := range ctx.getQuery() {
		values[k] = append(values[k], v...)
	}

	// Decode using all available binding tags (url, header, param).
	err := schema.Decode(values, ptr)
	if err != nil {
		return err
	}

	return ctx.OptionsReadOnly().Validate(ptr)
}

// ReadQuery binds URL Query to "ptr". The struct field tag is "url".
// If tag is missing it tries to bind using the field's name.
// To ignore a specific field from binding use tag value "-".
func (ctx *Context) ReadQuery(ptr any) error {
	values := ctx.getQuery()
	if len(values) == 0 {
		if ctx.OptionsReadOnly().FireEmptyFormError {
			return ErrEmptyForm
		}
		return nil
	}

	err := schema.DecodeQuery(values, ptr)
	if err != nil {
		return err
	}

	return ctx.OptionsReadOnly().Validate(ptr)
}

func (ctx *Context) getQuery() url.Values {
	// if ctx.query == nil {
	// 	ctx.query = ctx.request.URL.Query()
	// }
	//
	// return ctx.query

	// Do not cache it. Users may modify the underline map.
	return ctx.request.URL.Query()
	// Note that .Query doesn't check for error for the underline url.ParseQuery query,
	// an error may be: invalid semicolon separator in query,
	// if we ever want to handle this:
	//
	// ErrURLQuerySemicolon is logged by the standard net/http server when
	// the request contains a semicolon (;) wihch, after go1.17 it's not used as a key-value separator character.
	//
	// This is automatically ignored by the Server's logger.
	// var ErrURLQuerySemicolon = errors.New("http: URL query contains semicolon, which is no longer a supported separator; parts of the query may be stripped when parsed; see golang.org/issue/25192")
}

const queryEntriesContextKey = "iris.url.query.entries" // cache the url.Values to QueryEntries.

func (ctx *Context) getQueryEntries() encoding.StringSliceEntries {
	if entries, ok := ctx.Values().Lookup(queryEntriesContextKey); ok {
		//	return *(*encoding.QueryEntries)(unsafe.Pointer(&entries))
		return entries.(encoding.StringSliceEntries)
	}

	// Create new, store it and return it.
	sep := ctx.OptionsReadOnly().URLParamSeparator
	entries := encoding.NewStringSliceEntries(ctx.getQuery(), sep)
	ctx.Values().Add(queryEntriesContextKey, entries)
	return entries
}

// Query returns the request's URL query as typed accessors: Get, GetInt,
// GetBool, GetSlice and the rest of [encoding.StringSliceEntries]. Values are
// split on ContextOptions.URLParamSeparator and parsed once per request; the
// entries are cached on the Context for its lifetime.
//
// For the raw url.Values use ctx.Request().URL.Query().
func (ctx *Context) Query() encoding.StringSliceEntries {
	return ctx.getQueryEntries()
}

// ErrNotFound is a generic error to use when something was not found.
var ErrNotFound = errors.New("not found")

// IsWWW returns true if the current subdomain (if any) is www.
func (ctx *Context) IsWWW() bool {
	host := ctx.Host()
	return server.GetSubdomain(host) == "www"
}

// FullRequestURI returns the full URI,
// including the scheme, the host and the relative requested path/resource.
func (ctx *Context) FullRequestURI() string {
	return ctx.AbsoluteURI(ctx.Path())
}

// AbsoluteURI parses the "s" and returns its absolute URI form.
func (ctx *Context) AbsoluteURI(s string) string {
	if s == "" {
		return ""
	}

	userInfo := ""
	if s[0] == '@' {
		endUserInfoIdx := strings.IndexByte(s, '/')
		if endUserInfoIdx > 0 && len(s) > endUserInfoIdx {
			userInfo = s[1:endUserInfoIdx] + "@"
			s = s[endUserInfoIdx:]
		}
	}

	if s[0] == '/' {
		scheme := ctx.request.URL.Scheme
		if scheme == "" {
			if ctx.request.TLS != nil {
				scheme = "https:"
			} else {
				scheme = "http:"
			}
		}

		host := ctx.Host()

		return scheme + "//" + userInfo + host + path.Clean(s)
	}

	if u, err := url.Parse(s); err == nil {
		r := ctx.request

		if u.Scheme == "" && u.Host == "" {
			oldpath := r.URL.Path
			if oldpath == "" {
				oldpath = "/"
			}

			if s == "" || s[0] != '/' {
				olddir, _ := path.Split(oldpath)
				s = olddir + s
			}

			var query string
			if i := strings.Index(s, "?"); i != -1 {
				s, query = s[:i], s[i:]
			}

			// clean up but preserve trailing slash
			trailing := strings.HasSuffix(s, "/")
			s = path.Clean(s)
			if trailing && !strings.HasSuffix(s, "/") {
				s += "/"
			}
			s += query
		}
	}

	return s
}

// TrimParamFilePart is a middleware which replaces all route dynamic path parameters
// with values that do not contain any part after the last dot (.) character.
//
// Example Code:
//
//	package main
//
//	import (
//		"github.com/kataras/iris/v14"
//	)
//
//	func main() {
//		app := iris.New()
//		app.Get("/{uid:string regexp(^[0-9]{1,20}.html$)}", iris.TrimParamFilePart, handler)
//		// TrimParamFilePart can be registered as a middleware to a Router (group of routes) as well.
//		app.Listen(":8080")
//	}
//
//	func handler(ctx iris.Context) {
//		//
//		// The above line is useless now that we've registered the TrimParamFilePart middleware:
//		// uid := ctx.Params().GetTrimFileInt("uid")
//		//
//
//		uid := ctx.Params().GetUint64("uid")
//		ctx.Writef("Param value: %d\n", uid)
//	}
func TrimParamFilePart(ctx *Context) { // See #2024.
	params := ctx.params

	for i, param := range params {
		value := param.String()
		if idx := strings.LastIndexByte(value, '.'); idx > 1 {
			value = value[0:idx]
			param.Value = value
		}

		// ctx.params.SetByIndex(i, param)
		ctx.params[i] = param
	}

	ctx.Next()
}

// DecodeQuery returns the uri parameter as url (string)
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
// it uses just the url.QueryUnescape
func DecodeQuery(path string) string {
	if path == "" {
		return ""
	}
	encodedPath, err := url.QueryUnescape(path)
	if err != nil {
		return path
	}
	return encodedPath
}

// DecodeURL returns the decoded uri
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
// it uses just the url.Parse
func DecodeURL(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	return u.String()
}

// EscapedPath returns the request path in its percent-encoded form, as the
// client sent it: "/a%20b" stays "/a%20b". See `Path` for the decoded form.
func (ctx *Context) EscapedPath() string {
	return ctx.request.URL.EscapedPath()
}

const sufscheme = "://"

// GetScheme returns the full scheme of the request URL (https://, http:// or ws:// and e.t.c.).
func GetScheme(r *http.Request) string {
	scheme := r.URL.Scheme

	if scheme == "" {
		if r.TLS != nil {
			scheme = netutil.SchemeHTTPS
		} else {
			scheme = netutil.SchemeHTTP
		}
	}

	return scheme + sufscheme
}

// Scheme returns the full scheme of the request (including :// suffix).
func (ctx *Context) Scheme() string {
	return GetScheme(ctx.Request())
}

// PathPrefixMap accepts a map of string and a handler.
// The key of "m" is the key, which is the prefix, regular expressions are not valid.
// The value of "m" is the handler that will be executed if HasPrefix(context.Path).
// func (ctx *Context) PathPrefixMap(m map[string]context.Handler) bool {
// 	path := ctx.Path()
// 	for k, v := range m {
// 		if strings.HasPrefix(path, k) {
// 			v(ctx)
// 			return true
// 		}
// 	}
// 	return false
// } no, it will not work because map is a random peek data structure.

// GetHost returns the host part of the current URI.
func GetHost(r *http.Request) string {
	// contains subdomain.
	if host := r.URL.Host; host != "" {
		return host
	}
	return r.Host
}

// GetHostname returns the hostname part of the current URI.
func GetHostname(r *http.Request) string {
	hostport := GetHost(r)

	if host, _, err := net.SplitHostPort(hostport); err == nil {
		return host
	}

	return hostport
}

// Host returns the host:port part of the request URI, calls the `Request().Host`.
// To get the subdomain part as well use the `Request().URL.Host` method instead.
// To get the subdomain only use the `Subdomain` method instead.
// This method respects the `HostProxyHeaders` setting.
func (ctx *Context) Host() string {
	for header, ok := range ctx.OptionsReadOnly().HostProxyHeaders {
		if !ok {
			continue
		}

		if host := ctx.Header(header); host != "" {
			return host
		}
	}

	return GetHost(ctx.request)
}

// Hostname returns the hostname part of the request URI.
// This method respects the `HostProxyHeaders` setting.
func (ctx *Context) Hostname() string {
	hostport := ctx.Host()
	if host, _, err := net.SplitHostPort(hostport); err == nil {
		return host
	}

	return hostport
}

// Domain returns the root level domain.
func (ctx *Context) Domain() string {
	// if host := ctx.Application().GetHost(); host != "" {
	// 	return host
	// }

	return server.GetRootDomain(ctx.Host())
}

// Subdomain returns the subdomain of this request, if any.
func (ctx *Context) Subdomain() (subdomain string) {
	host := ctx.Host()
	return server.GetSubdomain(host)
}

const formEntriesContextKey = "iris.form.entries" // cache the url.Values to QueryEntries.

func (ctx *Context) getFormEntries() encoding.StringSliceEntries {
	if entries, ok := ctx.Values().Lookup(formEntriesContextKey); ok {
		return entries.(encoding.StringSliceEntries)
	}

	// Create new, store it and return it.
	sep := ctx.OptionsReadOnly().URLParamSeparator

	formAndPostValues, _ := ctx.form() // parse form once.

	entries := encoding.NewStringSliceEntries(formAndPostValues, sep)
	ctx.Values().Add(formEntriesContextKey, entries)
	return entries
}

// Form returns the parsed form data, including both the URL
// field's query parameters and the PATCH, POST, or PUT form (and multipart) data.
func (ctx *Context) Form() encoding.StringSliceEntries {
	return ctx.getFormEntries()
}

const postFormEntriesContextKey = "iris.form.post.entries" // cache the url.Values to QueryEntries.

func (ctx *Context) getPostFormEntries() encoding.StringSliceEntries {
	if entries, ok := ctx.Values().Lookup(postFormEntriesContextKey); ok {
		return entries.(encoding.StringSliceEntries)
	}

	// Create new, store it and return it.
	sep := ctx.OptionsReadOnly().URLParamSeparator

	_, _ = ctx.form() // parse form once.
	postValuesOnly := ctx.request.PostForm
	entries := encoding.NewStringSliceEntries(postValuesOnly, sep)
	ctx.Values().Add(postFormEntriesContextKey, entries)
	return entries
}

// PostForm returns the parsed form data from PATCH, POST
// or PUT body parameters.
func (ctx *Context) PostForm() encoding.StringSliceEntries {
	return ctx.getPostFormEntries()
}

// FormValueDefault retruns a single parsed form value.
func FormValueDefault(r *http.Request, name string, def string, postMaxMemory int64) string {
	if form, has := GetForm(r, postMaxMemory); has {
		if v := form[name]; len(v) > 0 {
			return v[0]
		}
	}
	return def
}

// CSRFTokenFormKey the CSRF token key of the form data.
//
// See ReadForm method for more.
const CSRFTokenFormKey = "csrf.token"

// ReadForm binds the request body of a form to the "formObject".
// If tag is missing it tries to bind using the field's name.
// To ignore a specific field from binding use tag value "-".
// It supports any kind of type, including custom structs.
// It will return nothing if request data are empty.
// The struct field tag is "form".
// Note that it will return nil error on empty form data if `Configuration.FireEmptyFormError`
// is false (as defaulted) in this case the caller should check the pointer to
// see if something was actually binded.
//
// If a client sent an unknown field, this method will return an error,
// in order to ignore that error use the `err != nil && !iris.IsErrPath(err)`.
//
// ReadForm drops only the unknown key error for "csrf.token" ([CSRFTokenFormKey]),
// the hidden field CSRF protection usually adds to a form, and returns every other
// decode error as a schema.MultiError without that entry. The decoder splits keys
// on dots, so to bind the token give the struct a field tagged `form:"csrf"` whose
// type has a field tagged `form:"token"`.
//
// Example: https://github.com/kataras/iris/blob/main/_examples/04-request/query-and-forms/main.go
func (ctx *Context) ReadForm(formObject any) error {
	values := ctx.Form().ToMap()
	if len(values) == 0 {
		if ctx.OptionsReadOnly().FireEmptyFormError {
			return ErrEmptyForm
		}
		return nil
	}

	if err := withoutCSRFTokenPathError(schema.DecodeForm(values, formObject)); err != nil {
		return err
	}

	return ctx.OptionsReadOnly().Validate(formObject)
}

// withoutCSRFTokenPathError returns err without its unknown key entry for
// "csrf.token". It returns nil when that entry was the only one, and a new
// schema.MultiError with the other entries when there were more, so that
// IsErrPath still sees a plain schema.MultiError. Any other error is returned
// as it is.
func withoutCSRFTokenPathError(err error) error {
	m, ok := err.(schema.MultiError)
	if !ok {
		return err
	}

	if _, isUnknownKey := m[CSRFTokenFormKey].(schema.UnknownKeyError); !isUnknownKey {
		return err
	}

	if len(m) == 1 {
		return nil
	}

	rest := make(schema.MultiError, len(m)-1)
	for key, keyErr := range m {
		if key != CSRFTokenFormKey {
			rest[key] = keyErr
		}
	}

	return rest
}

type (
	// MultipartRelated is the result of the context.ReadMultipartRelated method.
	MultipartRelated struct {
		// ContentIDs keeps an ordered list of all the
		// content-ids of the multipart related request.
		ContentIDs []string
		// Contents keeps each part's information by Content-ID.
		// Contents holds each of the multipart/related part's data.
		Contents map[string]MultipartRelatedContent
	}

	// MultipartRelatedContent holds a multipart/related part's id, header and body.
	MultipartRelatedContent struct {
		// ID holds the Content-ID.
		ID string
		// Headers holds the part's request headers.
		Headers map[string][]string
		// Body holds the part's body.
		Body []byte
	}
)

// ReadMultipartRelated returns a structure which contain
// information about each part (id, headers, body).
//
// Read more at: https://www.ietf.org/rfc/rfc2387.txt.
//
// Example request (2387/5.2 Text/X-Okie):
// Content-Type: Multipart/Related; boundary=example-2;
// start="<950118.AEBH@XIson.com>"
// type="Text/x-Okie"
//
// --example-2
// Content-Type: Text/x-Okie; charset=iso-8859-1;
// declaration="<950118.AEB0@XIson.com>"
// Content-ID: <950118.AEBH@XIson.com>
// Content-Description: Document
//
// {doc}
// This picture was taken by an automatic camera mounted ...
// {image file=cid:950118.AECB@XIson.com}
// {para}
// Now this is an enlargement of the area ...
// {image file=cid:950118:AFDH@XIson.com}
// {/doc}
// --example-2
// Content-Type: image/jpeg
// Content-ID: <950118.AFDH@XIson.com>
// Content-Transfer-Encoding: BASE64
// Content-Description: Picture A
//
// [encoded jpeg image]
// --example-2
// Content-Type: image/jpeg
// Content-ID: <950118.AECB@XIson.com>
// Content-Transfer-Encoding: BASE64
// Content-Description: Picture B
//
// [encoded jpeg image]
// --example-2--
func (ctx *Context) ReadMultipartRelated() (MultipartRelated, error) {
	contentType, params, err := mime.ParseMediaType(ctx.Header(ContentTypeHeaderKey))
	if err != nil {
		return MultipartRelated{}, err
	}

	if !strings.HasPrefix(contentType, ContentMultipartRelatedHeaderValue) {
		return MultipartRelated{}, ErrEmptyForm
	}

	var (
		contentIDs []string
		contents   = make(map[string]MultipartRelatedContent)
	)

	multipartReader := multipart.NewReader(ctx.limitBody(ctx.request.Body), params["boundary"])
	for {
		part, err := multipartReader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}

			return MultipartRelated{}, fmt.Errorf("multipart related: next part: %w", err)
		}
		defer part.Close()

		b, err := io.ReadAll(part)
		if err != nil {
			return MultipartRelated{}, fmt.Errorf("multipart related: next part: read: %w", err)
		}

		contentID := part.Header.Get("Content-ID")
		contentIDs = append(contentIDs, contentID)
		contents[contentID] = MultipartRelatedContent{ // replace if same Content-ID appears, which it shouldn't.
			ID:      contentID,
			Headers: http.Header(part.Header),
			Body:    b,
		}
	}

	if len(contents) != len(contentIDs) {
		contentIDs = distinctStrings(contentIDs)
	}

	result := MultipartRelated{
		ContentIDs: contentIDs,
		Contents:   contents,
	}
	return result, nil
}

func distinctStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, val := range values {
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}

	return result
}

var (
	// ValidFilenameRegexp is used to validate the user input by using a regular expression.
	// See `Context.UploadFormFiles` method.
	ValidFilenameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)
	// ValidExtensionRegexp acts as an allowlist of valid extensions. It's optional. Defaults to nil (all file extensions are allowed to be uploaded).
	// See `Context.UploadFormFiles` method.
	ValidExtensionRegexp *regexp.Regexp
)

// reservedWindowsNames are the legacy DOS device names. On Windows they resolve
// to a device rather than a file no matter which directory they are "in", and
// with or without an extension - so os.Create("uploads/NUL") writes to the null
// device and os.Create("uploads/COM1.txt") talks to a serial port.
//
// They pass ValidFilenameRegexp and the canonical-prefix check, so SafeFilename
// rejects them explicitly. The check is unconditional rather than
// runtime.GOOS-gated: a name that is unsafe on one target should not be accepted
// into an upload directory that may be read or synchronised on another.
var reservedWindowsNames = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
	"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {},
	"COM6": {}, "COM7": {}, "COM8": {}, "COM9": {}, "COM0": {},
	"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {},
	"LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {}, "LPT0": {},
}

// isReservedFilename reports whether the given base filename is a reserved
// Windows device name, with or without an extension.
func isReservedFilename(filename string) bool {
	base := filename
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}

	_, reserved := reservedWindowsNames[strings.ToUpper(base)]
	return reserved
}

// SafeFilename returns a safe filename based on the given name.
//   - Using filepath.Base and filepath.ToSlash: This ensures that only the base file name is used, without any directory components,
//     and converts all separators to slashes. This is a good practice to prevent directory traversal.
//   - Regular Expression for Filenames: The ValidFilenameRegexp ensures that filenames are restricted to a safe character set.
//     This helps prevent the use of special characters that could lead to path traversal or other types of injection attacks.
//   - Extension Validation: If you have a ValidExtensionRegexp, it would further ensure that the file has an expected and safe extension, which is another good practice.
//   - Canonical Path Check: By evaluating symlinks and ensuring that the destination path starts with the canonical destination directory, you’re adding.
//
// It returns the destination path (the canonical prefix directory joined with the safe filename,
// or an empty string when prefixDir is empty), the safe filename, a boolean indicating whether the
// filename is safe, and an error if any. When the name is rejected - or the prefix directory cannot
// be resolved - the first two results are the unmodified prefixDir and name.
func SafeFilename(prefixDir string, name string) (string, string, bool, error) {
	// Security fix for go < 1.17.5:
	// Reported by Kirill Efimov (snyk.io) through security reports.
	filename := filepath.Base(filepath.ToSlash(name))

	// CWE-99.

	// Sanitize the user input by using a regular expression
	// and an allowlist of valid extensions
	isValidFilename := ValidFilenameRegexp.MatchString(filename)
	if !isValidFilename {
		// Reject the input as it is invalid or unsafe.
		return prefixDir, name, false, nil
	}

	// "." and ".." are made of allowlisted characters, so the regexp lets them
	// through. filepath.Base("") is "." and Join(dir, ".") is the directory
	// itself, which is not a file to write to.
	if strings.Trim(filename, ".") == "" {
		return prefixDir, name, false, nil
	}

	if ValidExtensionRegexp != nil && !ValidExtensionRegexp.MatchString(filename) {
		// Reject the input as it is invalid or unsafe.
		return prefixDir, name, false, nil
	}

	if isReservedFilename(filename) {
		// A reserved Windows device name ("NUL", "COM1", "CON.txt") passes the
		// filename regexp and the prefix check, but os.Create on it opens a
		// device instead of a file.
		return prefixDir, name, false, nil
	}

	var destPath string
	if prefixDir != "" {
		// Get the canonical path of the destination directory.
		canonicalDestDir, err := filepath.EvalSymlinks(prefixDir) // the prefix dir should exists.
		if err != nil {
			return prefixDir, name, false, fmt.Errorf("dest directory: %s: eval symlinks: %w", prefixDir, err)
		}

		// Note(@kataras): join onto the CANONICAL directory, not the raw one.
		//
		// Building destPath from prefixDir and then comparing it against the
		// canonical directory fails closed whenever prefixDir is itself a symlink
		// (or differs from its resolved form in any way, e.g. an 8.3 short path
		// on Windows): the prefix never matched, so every single upload was
		// silently skipped with no error. Joining onto the canonical directory
		// keeps the traversal guard exact and removes that failure mode.
		destPath = filepath.Join(canonicalDestDir, filename)

		// Check if the destination path is within the destination directory.
		if !strings.HasPrefix(destPath, canonicalDestDir) {
			// Reject the input as it is a path traversal attempt.
			return prefixDir, name, false, nil
		}
	}

	return destPath, filename, true, nil
}

// UploadFormFiles uploads files to the specified directory.
//
// Every filename goes through [SafeFilename] first. A name that does not
// survive it is SKIPPED, not saved, and no error is reported for it: compare
// the returned "uploaded" slice against the files you expected when that
// matters. The rules are:
//   - The name must match ValidFilenameRegexp (alphanumeric plus ._-), so a
//     name carrying a path separator, a space or a colon is out.
//   - Directory traversal is stripped, and the destination directory is
//     resolved through its symlinks before the join, so neither "../" nor a
//     symlinked upload directory escapes it.
//   - A name that is only dots (".", "..") is rejected.
//   - Reserved Windows device names are rejected (CON, PRN, AUX, NUL, COM0-9,
//     LPT0-9, with or without an extension). This applies on every platform,
//     so an upload set behaves the same whatever the server runs on.
//
// What it does NOT do, and you may still want:
//   - Cap the upload size. PostMaxMemory bounds multipart parsing (32MB by
//     default), BodyMaxMemory bounds the whole body, and the bodylimit
//     middleware bounds it per route.
//   - Rate-limit the endpoint, scan the content, or check that the bytes match
//     the extension. A name ending in .jpg proves nothing about the file.
//
// The second optional argument "before" gives caller the chance to
// modify or cancel the *miltipart.FileHeader before saving to the disk,
// it can be used to change a file's name based on the current request,
// all FileHeader's options can be changed. You can ignore it if
// you don't need to use this capability before saving a file to the disk.
//
// Note that it doesn't check if request body streamed.
//
// Returns the copied length as int64 and
// a not nil error if at least one new file
// can't be created due to the operating system's permissions or
// http.ErrMissingFile if no file received.
//
// If you want to receive & accept files and manage them manually you can use the `context#FormFile`
// instead and create a copy function that suits your needs or use the `SaveFormFile` method,
// the below is for generic usage.
//
// The default form's memory maximum size is 32MB, it can be changed by
// the `PostMaxMemory` setting or by the `bodylimit` middleware (depends the use case).
//
// See `FormFile` and `FormFiles` to a more controlled way to receive a file.
//
// Example: https://github.com/kataras/iris/tree/main/_examples/04-request/uploads
func (ctx *Context) UploadFormFiles(destDirectory string, before ...func(*Context, *multipart.FileHeader) bool) (uploaded []*multipart.FileHeader, n int64, err error) {
	err = ctx.request.ParseMultipartForm(ctx.OptionsReadOnly().PostMaxMemory)
	if err != nil {
		return nil, 0, err
	}

	if ctx.request.MultipartForm != nil {
		if fhs := ctx.request.MultipartForm.File; fhs != nil {
			for _, files := range fhs {
			innerLoop:
				for _, file := range files {
					for _, b := range before {
						if !b(ctx, file) {
							continue innerLoop
						}
					}

					destPath, filename, ok, err := SafeFilename(destDirectory, file.Filename)
					if err != nil {
						return nil, 0, err
					}
					if !ok {
						continue
					}

					file.Filename = filename

					n0, err0 := ctx.SaveFormFile(file, destPath)
					if err0 != nil {
						return nil, 0, err0
					}
					n += n0

					uploaded = append(uploaded, file)
				}
			}
			return uploaded, n, nil
		}
	}

	return nil, 0, http.ErrMissingFile
}

// SaveFormFile saves a result of `FormFile` to the "dest" disk full path (directory + filename).
//
// SECURITY: "dest" is passed to os.Create as given. Building it from the
// client's FileHeader.Filename writes wherever that name points, so run the
// name through [SafeFilename] first:
//
//	destPath, _, ok, err := context.SafeFilename(uploadDir, fh.Filename)
//	if err != nil || !ok {
//		return // the name was rejected.
//	}
//	n, err := ctx.SaveFormFile(fh, destPath)
//
// [Context.UploadFormFiles] does this for you.
//
// See `FormFile` and `UploadFormFiles` too.
func (ctx *Context) SaveFormFile(fh *multipart.FileHeader, dest string) (int64, error) {
	src, err := fh.Open()
	if err != nil {
		return 0, err
	}
	defer src.Close()

	out, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	return io.Copy(out, src)
}

// Form contains the parsed form data, including both the URL
// field's query parameters and the POST or PUT form data.
func (ctx *Context) form() (form map[string][]string, found bool) {
	return GetForm(ctx.request, ctx.OptionsReadOnly().PostMaxMemory)
}

// GetForm returns the request form (url queries, post or multipart) values. It parses the form once.
//
// A url-encoded body is read only for POST, PUT and PATCH, because that is what net/http's ParseForm does.
// On Go 1.27 the url-encoded body of a QUERY request (RFC 10008) is therefore not part of the form;
// send QUERY bodies as JSON and read them with ReadJSON or ReadBody. Multipart bodies are parsed for any method.
func GetForm(r *http.Request, postMaxMemory int64) (form map[string][]string, found bool) {
	/*
		net/http/request.go#1219
		for k, v := range f.Value {
			r.Form[k] = append(r.Form[k], v...)
			// r.PostForm should also be populated. See Issue 9305.
			r.PostForm[k] = append(r.PostForm[k], v...)
		}
	*/

	if form := r.PostForm; len(form) > 0 {
		return form, true
	}

	if m := r.MultipartForm; m != nil {
		if len(m.Value) > 0 {
			return m.Value, true
		}
	}

	// includes both url parameters and post form.
	if form := r.Form; len(form) > 0 {
		return form, true
	}

	// ParseMultipartForm calls `request.ParseForm` automatically
	// therefore we don't need to call it here, although it doesn't hurt.
	// After one call to ParseMultipartForm or ParseForm,
	// subsequent calls have no effect, are idempotent.
	err := r.ParseMultipartForm(postMaxMemory)
	// if resetBody {
	// 	r.Body = io.NopCloser(bytes.NewBuffer(bodyCopy))
	// }
	if err != nil && err != http.ErrNotMultipart {
		return nil, false
	}

	if form := r.PostForm; len(form) > 0 {
		return form, true
	}

	if m := r.MultipartForm; m != nil {
		if len(m.Value) > 0 {
			return m.Value, true
		}
	}

	if form := r.Form; len(form) > 0 {
		return form, true
	}

	return nil, false
}

// FormFile returns the first uploaded file that received from the client.
//
// The form's memory maximum size defaults to 32MB. Change it through
// Options.Context.PostMaxMemory, or bound the whole body with
// Options.Context.BodyMaxMemory.
//
// The returned header's Filename is the client's, unsanitized. Run it through
// [SafeFilename] before it reaches the filesystem, or use [Context.UploadFormFiles],
// which does that for you.
//
// Example: https://github.com/kataras/iris/tree/main/_examples/04-request/uploads
func (ctx *Context) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	// we don't have access to see if the request is body stream
	// and then the ParseMultipartForm can be useless
	// here but do it in order to apply the post limit,
	// the internal request.FormFile will not do it if that's filled
	// and it's not a stream body.
	if err := ctx.request.ParseMultipartForm(ctx.OptionsReadOnly().PostMaxMemory); err != nil {
		return nil, nil, err
	}

	return ctx.request.FormFile(key)
}

// FormFiles same as FormFile but may return multiple file inputs based on a key, e.g. "files[]".
func (ctx *Context) FormFiles(key string, before ...func(*Context, *multipart.FileHeader) bool) (files []multipart.File, headers []*multipart.FileHeader, err error) {
	err = ctx.request.ParseMultipartForm(ctx.OptionsReadOnly().PostMaxMemory)
	if err != nil {
		return
	}

	if ctx.request.MultipartForm != nil {
		fhs := ctx.request.MultipartForm.File
		if n := len(fhs); n > 0 {
			files = make([]multipart.File, 0, n)
			headers = make([]*multipart.FileHeader, 0, n)

		innerLoop:
			for _, header := range fhs[key] {
				header.Filename = filepath.Base(header.Filename)

				for _, b := range before {
					if !b(ctx, header) {
						continue innerLoop
					}
				}

				file, fErr := header.Open()
				if fErr != nil { // exit on first error but return the succeed.
					return files, headers, fErr
				}

				files = append(files, file)
				headers = append(headers, header)
			}
		}

		return
	}

	return nil, nil, http.ErrMissingFile
}

// Cookie returns the request cookie including any context's cookie options (stored or given by this method).
func (ctx *Context) Cookie(name string, options ...CookieOption) (*http.Cookie, error) {
	c, err := ctx.request.Cookie(name)
	if err != nil {
		return nil, err
	}

	ctx.applyCookieOptions(c, OpCookieGet, options)
	return c, nil
}

// CookieValue returns cookie's value by its name
// returns empty string if nothing was found.
//
// If you want more than the value then:
// cookie, err := ctx.Cookie("name")
func (ctx *Context) CookieValue(name string, options ...CookieOption) string {
	c, err := ctx.Cookie(name, options...)
	if err != nil {
		return ""
	}

	value, _ := url.QueryUnescape(c.Value)
	return value
}

//  +------------------------------------------------------------+
//  | Body Readers                                               |
//  +------------------------------------------------------------+

type (
	// BodyDecoder is an interface which any struct can implement in order to customize the decode action
	// from ReadJSON and ReadXML
	//
	// Trivial example of this could be:
	// type User struct { Username string }
	//
	// func (u *User) Decode(data []byte) error {
	//	  return json.Unmarshal(data, u)
	// }
	//
	// the 'Context.ReadJSON/ReadXML(&User{})' will call the User's
	// Decode option to decode the request body
	//
	// Note: This is totally optionally, the default decoders
	// for ReadJSON is the encoding/json and for ReadXML is the encoding/xml.
	BodyDecoder interface {
		Decode(data []byte) error
	}

	// BodyDecoderWithContext same as BodyDecoder but it can accept a standard context,
	// which is binded to the HTTP request's context.
	BodyDecoderWithContext interface {
		DecodeContext(ctx context.Context, data []byte) error
	}

	// Unmarshaler is the interface implemented by types that can unmarshal any raw data.
	// TIP INFO: Any pointer to a value which implements the BodyDecoder can be override the unmarshaler.
	Unmarshaler interface {
		Unmarshal(data []byte, outPtr any) error
	}

	// UnmarshalerFunc a shortcut for the Unmarshaler interface
	//
	// See 'Unmarshaler' and 'BodyDecoder' for more.
	UnmarshalerFunc func(data []byte, outPtr any) error

	// DecodeFunc is a generic type of decoder function.
	// When the returned error is not nil the decode operation
	// is terminated and the error is received by the ReadJSONStream method,
	// otherwise it continues to read the next available object.
	//
	// A decode error ends the stream even when the callback returns nil:
	// the decoder is not positioned at the next value after a failure, so
	// ReadJSONStream returns that error itself instead of reading on.
	// Look the `Context.ReadJSONStream` method.
	DecodeFunc func(outPtr any) error
)

// Unmarshal parses the X-encoded data and stores the result in the value pointed to by v.
// Unmarshal uses the inverse of the encodings that Marshal uses, allocating maps,
// slices, and pointers as necessary.
func (u UnmarshalerFunc) Unmarshal(data []byte, v any) error {
	return u(data, v)
}

// ErrReadRequestBody is returned when the request body cannot be read.
var ErrReadRequestBody = errors.New("failed to read request body")

// ErrRequestEntityTooLarge is returned when the request body size is bigger than the limit.
var ErrRequestEntityTooLarge = errors.New("request entity too large")

// limitedBodyReader wraps a request body and fails with ErrRequestEntityTooLarge
// once more than "limit" bytes have been read in total. It mirrors the bodylimit
// middleware's reader so the limit flows through the same error handling.
type limitedBodyReader struct {
	io.ReadCloser
	read  int64
	limit int64
}

func (r *limitedBodyReader) Read(b []byte) (int, error) {
	if r.read > r.limit {
		return 0, ErrRequestEntityTooLarge
	}

	// Cap each read at one byte past the limit, like http.MaxBytesReader:
	// a reader that returned the whole oversized body TOGETHER with the
	// error let buffering decoders (encoding/json/v2 fills large buffers)
	// decode the complete value without ever observing the failing read,
	// which turned the 413 into a silent success.
	if maxRead := r.limit + 1 - r.read; int64(len(b)) > maxRead {
		b = b[:maxRead]
	}

	n, err := r.ReadCloser.Read(b)
	r.read += int64(n)
	if r.read > r.limit {
		return n, ErrRequestEntityTooLarge
	}
	return n, err
}

// limitBody wraps body with the ContextOptions.BodyMaxMemory limit when one is
// configured (> 0). With no limit it returns body unchanged, so the default
// behavior is unaffected.
func (ctx *Context) limitBody(body io.ReadCloser) io.ReadCloser {
	if maxMemory := ctx.OptionsReadOnly().BodyMaxMemory; maxMemory > 0 {
		return &limitedBodyReader{ReadCloser: body, limit: maxMemory}
	}
	return body
}

// Body reads the request body and returns its content as a byte slice.
//
// When ContextOptions.BodyMaxMemory is set, reading a body larger than the limit
// returns ErrRequestEntityTooLarge. A request without a body (nil, which the
// net/http server never produces but hand-built requests do) returns
// ErrReadRequestBody.
func (ctx *Context) Body() ([]byte, error) {
	if ctx.request.Body == nil {
		return nil, fmt.Errorf("%w: (content-length: %d, content-type: %s)", ErrReadRequestBody,
			ctx.ContentLength(), ctx.ContentType().String())
	}

	bodyBytes, err := io.ReadAll(ctx.limitBody(ctx.request.Body))
	if err != nil {
		return nil, fmt.Errorf("%w: %w (content-length: %d, content-type: %s)", ErrReadRequestBody,
			err, ctx.ContentLength(), ctx.ContentType().String())
	}
	_ = ctx.request.Body.Close() // Close the original body.
	return bodyBytes, nil
}

// BodyLength returns the request body length in bytes.
// When the content-length is known it is reported as-is, without touching the body
// (if the content-length is missing http sets the body to http.NoBody, so it returns 0).
// When the content-length is unknown (-1, e.g. a chunked request) the body is DRAINED
// in order to measure it, therefore a later `Body`/`ReadJSON` call reads nothing unless
// the request body was recorded first (see the middleware/recorder package).
//
// Returns ErrReadRequestBody if the body is nil or if an error occurs while reading it.
func (ctx *Context) BodyLength() (int64, error) {
	if ctx.request.Body == nil {
		return 0, fmt.Errorf("%w: (content-length: %d, content-type: %s)", ErrReadRequestBody,
			ctx.ContentLength(), ctx.ContentType().String())
	}

	switch ctx.request.ContentLength {
	case 0:
		return 0, nil
	case -1:
		// It's -1 which means unknown, fallback to read the body to determine its length.
		bodyBytes, err := io.Copy(io.Discard, ctx.request.Body)
		if err != nil {

			return 0, fmt.Errorf("%w: %w (content-length: %d, content-type: %s)", ErrReadRequestBody,
				err, ctx.ContentLength(), ctx.ContentType().String())
		}
		return bodyBytes, nil
	default:
		return ctx.request.ContentLength, nil
	}
}

// Validator is the validator for request body on Context methods such as
// ReadJSON, ReadMsgPack, ReadXML, ReadYAML, ReadForm, ReadQuery, ReadBody and e.t.c.
type Validator interface {
	Struct(any) error
	// If community asks for more than a struct validation on JSON, XML, MsgPack, Form, Query and e.t.c
	// then we should add more methods here, alternative approach would be to have a
	// `Validator:Validate(any) error` and a map[reflect.Kind]Validator instead.
}

// UnmarshalBody reads the request's body and binds it to a value or pointer of any type
// Examples of usage: context.ReadJSON, context.ReadXML.
func (ctx *Context) UnmarshalBody(outPtr any, unmarshaler Unmarshaler) error {
	rawData, err := ctx.Body() // guards a nil body itself.
	if err != nil {
		return err
	}

	if decoderWithCtx, ok := outPtr.(BodyDecoderWithContext); ok {
		return decoderWithCtx.DecodeContext(ctx.request.Context(), rawData)
	}

	// check if the v contains its own decode
	// in this case the v should be a pointer also,
	// but this is up to the user's custom Decode implementation*
	//
	// See 'BodyDecoder' for more.
	if decoder, isDecoder := outPtr.(BodyDecoder); isDecoder {
		return decoder.Decode(rawData)
	}

	// // check if v is already a pointer, if yes then pass as it's
	// if reflect.TypeOf(v).Kind() == reflect.Pointer {
	// 	return unmarshaler.Unmarshal(rawData, v)
	// } <- no need for that, ReadJSON is documented enough to receive a pointer,
	// we don't need to reduce the performance here by using the reflect.TypeOf method.

	// f the v doesn't contains a self-body decoder use the custom unmarshaler to bind the body.
	err = unmarshaler.Unmarshal(rawData, outPtr)
	if err != nil {
		return err
	}

	return ctx.OptionsReadOnly().Validate(outPtr)
}

// JSONReader holds the JSON decode options of the `Context.ReadJSON, ReadBody` methods.
type JSONReader struct { // Note(@kataras): struct instead of optional funcs to keep consistently with the encoder options.
	// DisallowUnknownFields causes the json decoder to return an error when the destination
	// is a struct and the input contains object keys which do not match any
	// non-ignored, exported fields in the destination.
	DisallowUnknownFields bool
	// CaseSensitive disables the default case-insensitive matching of JSON
	// object member names to struct field names for this call, restoring
	// encoding/json/v2's strict default. The lenient default exists because
	// deployed clients bind under the old encoding/json matching rules.
	CaseSensitive bool
	// This field only applies to the ReadJSONStream.
	// If set to true the request body stream MUST start with a `[`
	// and end with `]` literals, example:
	//  [
	//   {"username":"john"},
	//   {"username": "makis"},
	//   {"username": "george"}
	//  ]
	// Defaults to false: decodes a json object one by one, example:
	//  {"username":"john"}
	//  {"username": "makis"}
	//  {"username": "george"}
	ArrayStream bool
	// UnmarshalOptions are appended after the defaults built from the fields
	// above; later options win, so any encoding/json/v2 or jsontext decode
	// behavior can be overridden per call, e.g.
	// jsontext.AllowDuplicateNames(true) to accept duplicate object members.
	UnmarshalOptions []jsonv2.Options
}

// unmarshalOptions builds the effective json/v2 option set of this reader.
// Callers pass the result variadically and never append to it.
func (options *JSONReader) unmarshalOptions() []jsonv2.Options {
	if !options.CaseSensitive && !options.DisallowUnknownFields && len(options.UnmarshalOptions) == 0 {
		// The default reader (the overwhelmingly common one): reusing the
		// precomputed slice avoids a heap allocation per decode.
		return jsonutil.DefaultUnmarshalOptionsSlice
	}

	opts := make([]jsonv2.Options, 0, 4+len(options.UnmarshalOptions))
	opts = append(opts, UnmarshalDurationFromNanos) // v1 parity, see its doc.
	if !options.CaseSensitive {
		// v1 EqualFold parity: without the delimiter option v2 also folds
		// '_' and '-', see jsonutil.DefaultUnmarshalOptions.
		opts = append(opts, jsonv2.MatchCaseInsensitiveNames(true), json.MatchCaseSensitiveDelimiter(true))
	}
	if options.DisallowUnknownFields {
		opts = append(opts, jsonv2.RejectUnknownMembers(true))
	}

	return append(opts, options.UnmarshalOptions...)
}

// DefaultJSONReader is the default JSONReader configuration for the `Context.ReadJSON, ReadBody` methods.
var DefaultJSONReader = JSONReader{}

// ErrEmptyJSON is returned by the JSON body readers when the request carries
// no JSON value at all (an empty or whitespace-only body, or no body at all).
// It wraps io.EOF, so pre-existing errors.Is(err, io.EOF) checks keep matching.
var ErrEmptyJSON = fmt.Errorf("iris: read json: empty request body: %w", io.EOF)

// ErrJSONArrayExpected is returned by ReadJSONStream with JSONReader.ArrayStream
// set when the body does not start with a JSON array.
var ErrJSONArrayExpected = errors.New("iris: read json stream: expected a JSON array")

// ReadJSON reads JSON from the request's body and binds it to a value of any json-valid type.
// Can be modified in order to change the default behavior of the `Context.ReadJSON, ReadBody` methods.
var ReadJSON = func(ctx *Context, outPtr any, options *JSONReader) error {
	if ctx.request.Body == nil {
		// A hand-built request carries a nil Body, which the net/http server
		// never produces. It is no JSON value at all, the same answer an empty
		// body gets - and the decoder would panic reading from it.
		return ErrEmptyJSON
	}

	dec := jsontext.NewDecoder(ctx.limitBody(ctx.request.Body))
	if dec.PeekKind() == 0 {
		// No first token: empty body, an immediate syntax error, or a
		// failing reader. ReadValue surfaces which one.
		_, err := dec.ReadValue()
		if err == nil { // unreachable: PeekKind was 0.
			err = io.ErrUnexpectedEOF
		}
		if errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return ErrEmptyJSON // structural: EOF before any token.
		}

		return err
	}

	// UnmarshalDecode, not UnmarshalRead: it decodes exactly one top-level
	// value and tolerates trailing data, like the old json.Decoder.Decode.
	return jsonv2.UnmarshalDecode(dec, outPtr, options.unmarshalOptions()...)
}

// ReadJSON reads JSON from request's body and binds it to a value of any json-valid type.
//
// The read is bounded by Options.Context.BodyMaxMemory when you have set it,
// and a body past that limit fails with [ErrRequestEntityTooLarge], which the
// error map renders as 413. Without it the body is read in full, so set it or
// register the bodylimit middleware before accepting untrusted input.
//
// After a successful decode, Options.Context.RequestValidator runs if one is
// registered. Pass a [JSONReader] to change decoder behavior for this call,
// for example DisallowUnknownFields.
//
// Example: https://github.com/kataras/iris/blob/main/_examples/04-request/read-json-validate/main.go
func (ctx *Context) ReadJSON(outPtr any, opts ...JSONReader) error {
	var options *JSONReader
	if len(opts) > 0 {
		options = &opts[0]
	} else {
		// If no options are given safely read the already-initialized value.
		options = &DefaultJSONReader
	}

	if err := ReadJSON(ctx, outPtr, options); err != nil {
		return err
	}

	return ctx.OptionsReadOnly().Validate(outPtr)
}

// ReadJSONStream is an alternative of ReadJSON which can reduce the memory load
// by reading only one json object every time.
// It buffers just the content required for a single json object instead of the entire string,
// and discards that once it reaches an end of value that can be decoded into the provided struct
// inside the onDecode's DecodeFunc.
//
// It accepts a function which accepts the json Decode function and returns an error.
// The second variadic argument is optional and can be used to customize the decoder even further.
func (ctx *Context) ReadJSONStream(onDecode func(DecodeFunc) error, opts ...JSONReader) error {
	options := &DefaultJSONReader
	if len(opts) > 0 {
		options = &opts[0]
	}

	if ctx.request.Body == nil {
		return ErrEmptyJSON // see ReadJSON.
	}

	unmarshalOptions := options.unmarshalOptions()
	dec := jsontext.NewDecoder(ctx.limitBody(ctx.request.Body))

	// Progress is the loop invariant. The callback's contract lets it
	// swallow a decode error, but the decoder is not at the next value after
	// one (a semantic error leaves it inside the failed object, where
	// PeekKind never reports the array end or 0), so the loop would spin.
	// Each iteration therefore records the decode error and returns it when
	// the callback did not, and a callback that never decodes at all is
	// stopped with io.ErrNoProgress.
	var decodeErr error
	decode := func(outPtr any) error {
		decodeErr = jsonv2.UnmarshalDecode(dec, outPtr, unmarshalOptions...)
		return decodeErr
	}
	step := func() error {
		decodeErr = nil
		before := dec.InputOffset()
		if err := onDecode(decode); err != nil {
			return err
		}
		if decodeErr != nil {
			return decodeErr
		}
		if dec.InputOffset() == before {
			return io.ErrNoProgress
		}
		return nil
	}

	if options.ArrayStream {
		tok, err := dec.ReadToken() // the opening bracket.
		if err != nil {
			if errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				return ErrEmptyJSON
			}
			return err
		}
		if kind := tok.Kind(); kind != '[' {
			return fmt.Errorf("%w, got %s", ErrJSONArrayExpected, kind)
		}

		for {
			switch dec.PeekKind() {
			case ']':
				_, err := dec.ReadToken() // the closing bracket.
				return err
			case 0:
				// A decoder or reader error, or a truncated stream. The
				// callback may have swallowed the decode error (its
				// contract allows that), so surface it from here: without
				// this the offset never advances and the loop spins.
				if _, err := dec.ReadToken(); err != nil {
					return err
				}
				return io.ErrUnexpectedEOF
			}

			if err := step(); err != nil {
				return err
			}
		}
	}

	// One JSON value after another (NDJSON-style stream).
	for dec.PeekKind() > 0 {
		if err := step(); err != nil {
			return err
		}
	}

	// The loop ends at end-of-stream or on a decoder/reader error —
	// PeekKind reports both as 0, so distinguish them here instead of
	// silently swallowing a failing body reader.
	if _, err := dec.ReadToken(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	return nil
}

// ReadXML reads XML from request's body and binds it to a value of any xml-valid type.
func (ctx *Context) ReadXML(outPtr any) error {
	return ctx.UnmarshalBody(outPtr, UnmarshalerFunc(xml.Unmarshal))
}

// ReadYAML reads YAML from request's body and binds it to the "outPtr" value.
func (ctx *Context) ReadYAML(outPtr any) error {
	return ctx.UnmarshalBody(outPtr, UnmarshalerFunc(yaml.Unmarshal))
}

var (
	// IsErrEmptyJSON reports whether the given "err" says the request or
	// response body carried no JSON value at all (empty or truncated-empty).
	// The checks are structural — sentinel and typed errors, never error
	// text, which Go no longer guarantees to be stable.
	IsErrEmptyJSON = func(err error) bool {
		if err == nil {
			return false
		}

		if errors.Is(err, ErrEmptyJSON) {
			return true
		}

		// encoding/json/v2 on an empty input: a syntactic error at offset 0
		// wrapping io.ErrUnexpectedEOF.
		if syn, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
			return syn.ByteOffset == 0 && errors.Is(err, io.ErrUnexpectedEOF)
		}

		if errors.Is(err, io.EOF) {
			return true // bare EOF from v1 decoders and user code.
		}

		// encoding/json (v1) in user code: only the offset is checked; the
		// message text is not part of the compatibility promise anymore.
		if v, ok := errors.AsType[*json.SyntaxError](err); ok {
			return v.Offset == 0
		}

		return false
	}

	// IsErrPath can be used at `context#ReadForm` and `context#ReadQuery`.
	// It reports whether the incoming error
	// can be ignored when server allows unknown post values to be sent by the client.
	//
	// A shortcut for the `schema#IsErrPath`.
	IsErrPath = schema.IsErrPath

	// IsErrPathCSRFToken reports whether "err", or an error it wraps, is a
	// schema.MultiError that carries an unknown key error on "csrf.token".
	// See [Context.ReadForm] for more.
	IsErrPathCSRFToken = func(err error) bool {
		if err == nil {
			return false
		}

		if m, ok := errors.AsType[schema.MultiError](err); ok {
			if csrfErr, hasCSRFToken := m[CSRFTokenFormKey]; hasCSRFToken {
				_, is := csrfErr.(schema.UnknownKeyError)
				return is
			}
		}

		return false
	}

	// ErrEmptyForm is returned by
	// - `context#ReadForm`
	// - `context#ReadQuery`
	// - `context#ReadBody`
	// when the request data (form, query and body respectfully) is empty.
	ErrEmptyForm = errors.New("empty form")

	// ErrEmptyFormField reports whether a specific field exists but it's empty.
	// It is kept for backwards compatibility only: no method of this package
	// returns or wraps it anymore, so an errors.Is check against it never reports true.
	ErrEmptyFormField = errors.New("empty form field")

	// ConnectionCloseErrorSubstr if at least one of the given
	// substrings are found in a net.OpError:os.SyscallError error type
	// on `IsErrConnectionClosed` then the function will report true.
	ConnectionCloseErrorSubstr = []string{
		"broken pipe",
		"connection reset by peer",
	}

	// IsErrConnectionClosed reports whether the given "err"
	// is caused because of a broken connection.
	IsErrConnectionClosed = func(err error) bool {
		if err == nil {
			return false
		}

		if opErr, ok := err.(*net.OpError); ok {
			if syscallErr, ok := opErr.Err.(*os.SyscallError); ok {
				errStr := strings.ToLower(syscallErr.Error())
				for _, s := range ConnectionCloseErrorSubstr {
					if strings.Contains(errStr, s) {
						return true
					}
				}
			}
		}

		return false
	}
)

// ReadProtobuf binds the body to the "ptr" of a proto Message and returns any error.
// Look `ReadProtobufJSON` too.
func (ctx *Context) ReadProtobuf(ptr proto.Message) error {
	rawData, err := ctx.Body()
	if err != nil {
		return err
	}

	return proto.Unmarshal(rawData, ptr)
}

// ReadProtobufJSON reads a JSON body request into the given "ptr" proto.Message.
// Look `ReadProtobuf` too.
func (ctx *Context) ReadProtobufJSON(ptr proto.Message, opts ...protojson.UnmarshalOptions) error {
	rawData, err := ctx.Body()
	if err != nil {
		return err
	}

	opt := protojson.UnmarshalOptions{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	return opt.Unmarshal(rawData, ptr)
}

// ReadMsgPack binds the request body of msgpack format to the "ptr" and returns any error.
func (ctx *Context) ReadMsgPack(ptr any) error {
	rawData, err := ctx.Body()
	if err != nil {
		return err
	}

	err = msgpack.Unmarshal(rawData, ptr)
	if err != nil {
		return err
	}

	return ctx.OptionsReadOnly().Validate(ptr)
}

// ReadBody binds the request body to the "ptr" depending on the HTTP Method and the Request's Content-Type.
// If a GET method request then it reads from a form (or URL Query), otherwise
// it tries to match (depending on the request content-type) the data format e.g.
// JSON, Protobuf, MsgPack, XML, YAML, MultipartForm and binds the result to the "ptr".
// As a special case if the "ptr" was a pointer to string or []byte
// then it will bind it to the request body as it is.
func (ctx *Context) ReadBody(ptr any) error {
	// If the ptr is string or byte, read the body as it is
	// and return: raw targets have no decoder to dispatch to.
	switch v := ptr.(type) {
	case *string:
		b, err := ctx.Body()
		if err != nil {
			return err
		}

		*v = string(b)
		return nil
	case *[]byte:
		b, err := ctx.Body()
		if err != nil {
			return err
		}

		*v = b
		return nil
	}

	if ctx.Method() == http.MethodGet {
		if ctx.Request().URL.RawQuery != "" {
			// try read from query.
			return ctx.ReadQuery(ptr)
		}

		// otherwise use the ReadForm,
		// it's actually the same except
		// ReadQuery will not fire errors on:
		// 1. unknown or empty url query parameters
		// 2. empty query or form (if FireEmptyFormError is enabled).
		return ctx.ReadForm(ptr)
	}

	switch ctx.MediaType() {
	case ContentXMLHeaderValue, ContentXMLUnreadableHeaderValue:
		return ctx.ReadXML(ptr)
		// "%v reflect.Indirect(reflect.ValueOf(ptr)).Interface())
	case ContentYAMLHeaderValue, ContentYAMLTextHeaderValue:
		return ctx.ReadYAML(ptr)
	case ContentFormHeaderValue, ContentFormMultipartHeaderValue:
		return ctx.ReadForm(ptr)
	case ContentMultipartRelatedHeaderValue:
		return fmt.Errorf("context: read body: cannot bind multipart/related: use ReadMultipartRelated instead")
	case ContentJSONHeaderValue:
		return ctx.ReadJSON(ptr)
	case ContentProtobufHeaderValue:
		msg, ok := ptr.(proto.Message)
		if !ok {
			return ErrContentNotSupported
		}

		return ctx.ReadProtobuf(msg)
	case ContentMsgPackHeaderValue, ContentMsgPack2HeaderValue:
		return ctx.ReadMsgPack(ptr)
	default:
		if ctx.Request().URL.RawQuery != "" {
			// try read from query.
			return ctx.ReadQuery(ptr)
		}

		// otherwise default to JSON.
		return ctx.ReadJSON(ptr)
	}
}

var maxAgeExp = regexp.MustCompile(`maxage=(\d+)`)

// MaxAge returns the "cache-control" request header's value
// seconds as int64
// if header not found or parse failed then it returns -1.
func (ctx *Context) MaxAge() int64 {
	header := ctx.Header(CacheControlHeaderKey)
	if header == "" {
		return -1
	}
	m := maxAgeExp.FindStringSubmatch(header)
	if len(m) == 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			return int64(v)
		}
	}
	return -1
}
