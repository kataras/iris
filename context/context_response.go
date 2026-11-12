package context

import (
	"context"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/kataras/iris/v14/core/pkg/jsonutil"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/microcosm-cc/bluemonday"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

// ResponseWriter returns an http.ResponseWriter compatible response writer, as expected.
func (ctx *Context) ResponseWriter() ResponseWriter {
	return ctx.writer
}

// SetResponseWriter sets a new ResponseWriter implementation
// to this Context to use as its writer.
// Note, to change the underline http.ResponseWriter use
// ctx.ResponseWriter().SetWriter(http.ResponseWriter) instead.
func (ctx *Context) SetResponseWriter(newResponseWriter ResponseWriter) {
	ctx.writer = newResponseWriter
}

// maxResponseWriterDepth bounds the walk in DisableResponseBuffering. Real
// chains are two or three writers deep; anything past this is a Disable that
// returns a wrapper wrapping it back.
const maxResponseWriterDepth = 16

// DisableResponseBuffering removes every response-writer wrapper that buffers or
// transforms the body, restoring a chain that reaches the client on every flush.
//
// It is what lets a streaming route survive middleware that is correct for an
// ordinary response and wrong for a stream. middleware/compress buffers into a
// compressor; middleware/recorder buffers into a bytes.Buffer, and accesslog
// installs a recorder whenever it reads the response body, which the builder
// enables by default. Neither forwards anything on a flush, so the handler
// streams into a buffer and the client sees one lump when the handler returns,
// or nothing at all.
//
// Call it before the first write. Wrappers cooperate through
// ResponseWriterDisabler, so one that does not implement it - a third-party
// writer, say - stops the walk and is named in the returned error instead of
// silently swallowing the stream.
//
// The sse package calls this for you.
func (ctx *Context) DisableResponseBuffering() error {
	for range maxResponseWriterDepth {
		d, ok := ctx.writer.(ResponseWriterDisabler)
		if !ok {
			return nil // reached a writer that does not buffer.
		}

		underline := d.Disable()
		if underline == nil {
			return fmt.Errorf("iris: disable response buffering: %T returned no underlying writer", d)
		}

		ctx.writer = underline
	}

	return fmt.Errorf("iris: disable response buffering: %T wraps too deeply", ctx.writer)
}

// SetHeader sets a header on the response, replacing any value already
// set for that name. If value is empty it removes the header.
//
// A middleware registered twice (say on UseRouter and again on a router)
// therefore cannot emit a duplicate single-valued header such as
// Access-Control-Allow-Origin, which browsers reject outright.
//
// See `AddHeader` for list-valued headers such as Vary.
func (ctx *Context) SetHeader(name string, value string) {
	if value == "" {
		ctx.writer.Header().Del(name)
		return
	}
	ctx.writer.Header().Set(name, value)
}

// AddHeader adds a header value to the response, keeping any value already
// set for that name: calling it twice with the same name emits the header
// twice. That is what a list-valued header such as Vary or Link needs, and
// what a single-valued one does not; use `SetHeader` for those.
// An empty value is ignored.
func (ctx *Context) AddHeader(name string, value string) {
	if value == "" {
		return
	}
	ctx.writer.Header().Add(name, value)
}

// ResponseHeader returns the response writer's header value of the given
// "name", i.e. a header this handler has already set through SetHeader.
// If more than one header value found, it returns them joined by comma and space.
//
// The request's headers are read through Context.Header instead:
//
//	apiKey := ctx.Header("X-API-Key")         // request  (what the client sent)
//	cached := ctx.ResponseHeader("X-Cache")   // response (what this handler set)
func (ctx *Context) ResponseHeader(name string) string {
	return strings.Join(ctx.writer.Header().Values(name), ", ")
}

// StatusCode sets the status code header to the response.
// Look .GetStatusCode too, and `Application#HandleNoRoute`, `Application#HandleNoMethod`
// for handling the 404 and 405 failure codes.
//
// Note that you must set status code before write response body (except when recorder is used).
func (ctx *Context) StatusCode(statusCode int) {
	ctx.writer.WriteHeader(statusCode)
}

// GetStatusCode returns the current status code of the response.
// Look StatusCode too.
func (ctx *Context) GetStatusCode() int {
	return ctx.writer.StatusCode()
}

const contentTypeContextKey = "iris.content_type"

// SetContentType sets the Content-Type header of the response.
// If a previous call of SetContentType with the Once option was made, it will not be set again.
// If options are provided and the rawContentType is not a valid Content-Type header value,
// an error is returned; without options the value is written as-is, the rawContentType is used
// to determine the charset parameter too and the returned error is always nil.
//
// See `ContentTypeOption` for available options.
func (ctx *Context) SetContentType(rawContentType string, opts ...ContentTypeOption) error {
	if ctx.values.Exists(contentTypeContextKey) {
		return nil
	}

	if len(opts) > 0 { // if we have options means that we need to create a ContentType.
		ct, err := ParseContentType(rawContentType)
		if err != nil {
			return err
		}

		for _, opt := range opts {
			if opt == nil {
				continue
			}

			opt(&ct)
		}

		if ct.Once {
			ctx.values.Set(contentTypeContextKey, struct{}{})
		}

		defaultCharset := ctx.OptionsReadOnly().Charset
		setContentType(ctx.writer, ct, defaultCharset)
		return nil
	}

	cType := ctx.OptionsReadOnly().MimeToContentType(rawContentType)

	writeContentType(ctx.writer, cType)
	return nil
}

// ResponseContentType returns the response writer's header value of "Content-Type".
// The request's content type is read through Context.ContentType instead.
func (ctx *Context) ResponseContentType() ContentType {
	ct, _ := ParseContentType(ctx.ResponseHeader(ContentTypeHeaderKey))
	return ct
}

// ResponseMediaType returns the media type of the response writer's header value of "Content-Type".
// The request's media type is read through Context.MediaType instead.
func (ctx *Context) ResponseMediaType() string {
	return TrimHeaderValue(ctx.ResponseHeader(ContentTypeHeaderKey))
	// faster than: return ctx.ResponseContentType().MediaType
}

// NotFound emits an error 404 to the client, using the specific custom error error handler.
// Note that you may need to call ctx.StopExecution() if you don't want the next handlers
// to be executed. Next handlers are being executed on iris because you can alt the
// error code and change it to a more specific one, i.e
// users := app.Router("/users")
// users.Done(func(ctx iris.Context){ if ctx.GetStatusCode() == 400 { /*  custom error code for /users */ }})
func (ctx *Context) NotFound() {
	ctx.StatusCode(http.StatusNotFound)
}

// Redirect sends a redirect response to the client
// of an absolute or relative target URL.
// It accepts 2 input arguments, a string and an optional integer.
// The first parameter is the target url to redirect.
// The second one is the HTTP status code should be sent
// among redirection response,
// If the second parameter is missing, then it defaults to 302 (StatusFound).
// It can be set to 301 (permanent redirect), StatusTemporaryRedirect(307)
// or 303 (StatusSeeOther) if POST method.
//
// Redirect calls StopExecution, so the handlers after this one in the chain do
// not run.
//
// SECURITY: the target is written to the Location header as given. Redirecting
// to a URL that came from the request, a "?next=" query value being the usual
// case, lets a caller send your users to any site they choose while the link
// still reads as yours. Match untrusted targets against an allowlist, or accept
// only a path:
//
//	next := ctx.Query().Get("next")
//	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
//		next = "/"
//	}
//	ctx.Redirect(next)
func (ctx *Context) Redirect(urlToRedirect string, statusHeader ...int) {
	ctx.StopExecution()
	// get the previous status code given by the end-developer.
	status := ctx.GetStatusCode()
	if status < 300 { // the previous is not a RCF-valid redirect status.
		status = 0
	}

	if len(statusHeader) > 0 {
		// check if status code is passed via receivers.
		if s := statusHeader[0]; s > 0 {
			status = s
		}
	}
	if status == 0 {
		// if status remains zero then default it.
		// a 'temporary-redirect-like' which works better than for our purpose
		status = http.StatusFound
	}

	http.Redirect(ctx.writer, ctx.request, urlToRedirect, status)
}

//  +------------------------------------------------------------+
//  | Cookies                                                    |
//  +------------------------------------------------------------+

// Set of Cookie actions for `CookieOption`.
const (
	OpCookieGet uint8 = iota
	OpCookieSet
	OpCookieDel
)

// CookieOption is the type of function that is accepted on
// context's methods like `AddCookie`, `RemoveCookie` and `SetCookie`
// as their (last) variadic input argument to amend the to-be-sent cookie.
//
// The "op" is the operation code, 0 is GET, 1 is SET and 2 is REMOVE.
type CookieOption func(ctx *Context, c *http.Cookie, op uint8)

// CookieIncluded reports whether the "cookie.Name" is in the list of "cookieNames".
// Notes:
// If "cookieNames" slice is empty then it returns true,
// If "cookie.Name" is empty then it returns false.
func CookieIncluded(cookie *http.Cookie, cookieNames []string) bool {
	if cookie.Name == "" {
		return false
	}

	if len(cookieNames) > 0 {
		return slices.Contains(cookieNames, cookie.Name)
	}

	return true
}

// var cookieNameSanitizer = strings.NewReplacer("\n", "-", "\r", "-")
//
// func sanitizeCookieName(n string) string {
// 	return cookieNameSanitizer.Replace(n)
// }

// CookieOverride is a CookieOption which overrides the cookie explicitly to the given "cookie".
//
// Usage:
// ctx.RemoveCookie("the_cookie_name", iris.CookieOverride(&http.Cookie{Domain: "example.com"}))
func CookieOverride(cookie *http.Cookie) CookieOption { // The "Cookie" word method name is reserved as it's used as an alias.
	return func(_ *Context, c *http.Cookie, op uint8) {
		if op == OpCookieGet {
			return
		}

		*cookie = *c
	}
}

// CookieDomain is a CookieOption which sets the cookie's Domain field.
// If empty then the current domain is used.
//
// Usage:
// ctx.RemoveCookie("the_cookie_name", iris.CookieDomain("example.com"))
func CookieDomain(domain string) CookieOption {
	return func(_ *Context, c *http.Cookie, op uint8) {
		if op == OpCookieGet {
			return
		}

		c.Domain = domain
	}
}

// CookieAllowReclaim accepts the Context itself.
// If set it will add the cookie to (on `SetCookie`, `AddCookie`)
// or remove the cookie from (on `RemoveCookie`) the Request object too.
func CookieAllowReclaim(cookieNames ...string) CookieOption {
	return func(ctx *Context, c *http.Cookie, op uint8) {
		if op == OpCookieGet {
			return
		}

		if !CookieIncluded(c, cookieNames) {
			return
		}

		switch op {
		case OpCookieSet:
			// perform upsert on request cookies or is it too much and not worth the cost?
			ctx.Request().AddCookie(c)
		case OpCookieDel:
			cookies := ctx.Request().Cookies()
			ctx.Request().Header.Del("Cookie")
			for i, v := range cookies {
				if v.Name != c.Name {
					ctx.Request().AddCookie(cookies[i])
				}
			}
		}
	}
}

// CookieAllowSubdomains set to the Cookie Options
// in order to allow subdomains to have access to the cookies.
// It sets the cookie's Domain field (if was empty) and
// it also sets the cookie's SameSite to lax mode too.
func CookieAllowSubdomains(cookieNames ...string) CookieOption {
	return func(ctx *Context, c *http.Cookie, _ uint8) {
		if c.Domain != "" {
			return // already set.
		}

		if !CookieIncluded(c, cookieNames) {
			return
		}

		c.Domain = ctx.Domain()
		c.SameSite = http.SameSiteLaxMode // allow subdomain sharing.
	}
}

// CookieSameSite sets a same-site rule for cookies to set.
// SameSite allows a server to define a cookie attribute making it impossible for
// the browser to send this cookie along with cross-site requests. The main
// goal is to mitigate the risk of cross-origin information leakage, and provide
// some protection against cross-site request forgery attacks.
//
// See https://tools.ietf.org/html/draft-ietf-httpbis-cookie-same-site-00 for details.
func CookieSameSite(sameSite http.SameSite) CookieOption {
	return func(_ *Context, c *http.Cookie, op uint8) {
		if op == OpCookieSet {
			c.SameSite = sameSite
		}
	}
}

// CookieSecure is a `CookieOption`.
// It sets the cookie's Secure option on set operation if the current request's
// connection is using TLS. See `CookieHTTPOnly` too.
func CookieSecure(ctx *Context, c *http.Cookie, op uint8) {
	if op == OpCookieSet {
		if ctx.Request().TLS != nil {
			c.Secure = true
		}
	}
}

// CookieSecureForce is a `CookieOption`. It sets the cookie's Secure option to true every time.
func CookieSecureForce(ctx *Context, c *http.Cookie, op uint8) {
	c.Secure = true
}

// CookieHTTPOnly is a `CookieOption`.
// Use it to set the cookie's HttpOnly field to false or true.
// HttpOnly field defaults to true for `RemoveCookie` and `AddCookie`.
// See `CookieSecure` too.
func CookieHTTPOnly(httpOnly bool) CookieOption {
	return func(_ *Context, c *http.Cookie, op uint8) {
		if op == OpCookieSet {
			c.HttpOnly = httpOnly
		}
	}
}

// CookiePath is a `CookieOption`.
// Use it to change the cookie's Path field.
func CookiePath(path string) CookieOption {
	return func(_ *Context, c *http.Cookie, op uint8) {
		if op > OpCookieGet { // on set and remove.
			c.Path = path
		}
	}
}

// CookieCleanPath is a `CookieOption`.
// Use it to clear the cookie's Path field, exactly the same as `CookiePath("")`.
func CookieCleanPath(_ *Context, c *http.Cookie, op uint8) {
	if op > OpCookieGet {
		c.Path = ""
	}
}

// CookieExpires is a `CookieOption`.
// Use it to change the cookie's Expires and MaxAge fields by passing the lifetime of the cookie.
func CookieExpires(durFromNow time.Duration) CookieOption {
	return func(_ *Context, c *http.Cookie, op uint8) {
		if op == OpCookieSet {
			c.Expires = time.Now().Add(durFromNow)
			c.MaxAge = int(durFromNow.Seconds())
		}
	}
}

// CookiePartitioned is a `CookieOption`.
// It enables the "Partitioned" cookie attribute.
// Read more at: https://github.com/golang/go/issues/62490#issuecomment-2127685955 and
// https://developers.google.com/privacy-sandbox/cookies.
func CookiePartitioned(_ *Context, c *http.Cookie, op uint8) {
	if op == OpCookieSet {
		c.SameSite = http.SameSiteNoneMode
		c.Secure = true
		c.Partitioned = true
	}
}

// SecureCookie should encodes and decodes
// authenticated and optionally encrypted cookie values.
// See `CookieEncoding` package-level function.
type SecureCookie interface {
	// Encode should encode the cookie value.
	// Should accept the cookie's name as its first argument
	// and as second argument the cookie value ptr.
	// Should return an encoded value or an empty one if encode operation failed.
	// Should return an error if encode operation failed.
	//
	// Note: Errors are not printed, so you have to know what you're doing,
	// and remember: if you use AES it only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	//
	// See `Decode` too.
	Encode(cookieName string, cookieValue any) (string, error)
	// Decode should decode the cookie value.
	// Should accept the cookie's name as its first argument,
	// as second argument the encoded cookie value and as third argument the decoded value ptr.
	// Should return a decoded value or an empty one if decode operation failed.
	// Should return an error if decode operation failed.
	//
	// Note: Errors are not printed, so you have to know what you're doing,
	// and remember: if you use AES it only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	//
	// See `Encode` too.
	Decode(cookieName string, cookieValue string, cookieValuePtr any) error
}

// CookieEncoding accepts a value which implements `Encode` and `Decode` methods.
// It calls its `Encode` on `Context.SetCookie` and `AddCookie` methods.
// And on `Context.Cookie` and `Context.CookieValue` methods it calls its `Decode`.
// If "cookieNames" slice is not empty then only cookies
// with that `Name` will be encoded on set and decoded on get, that way you can encrypt
// specific cookie names (like the session id) and let the rest of the cookies "insecure".
func CookieEncoding(encoding SecureCookie, cookieNames ...string) CookieOption {
	if encoding == nil {
		return func(_ *Context, _ *http.Cookie, _ uint8) {}
	}

	return func(ctx *Context, c *http.Cookie, op uint8) {
		if op == OpCookieDel {
			return
		}

		if !CookieIncluded(c, cookieNames) {
			return
		}

		switch op {
		case OpCookieSet:
			// Should encode, it's a write to the client operation.
			newVal, err := encoding.Encode(c.Name, c.Value)
			if err != nil {
				ctx.handleContextError(err)
				// ctx.Application().Logger().Error(err)
				c.Value = ""
			} else {
				c.Value = newVal
			}

			return
		case OpCookieGet:
			// Should decode, it's a read from the client operation.
			if err := encoding.Decode(c.Name, c.Value, &c.Value); err != nil {
				c.Value = ""
			}
		}
	}
}

const cookieOptionsContextKey = "iris.cookie.options"

// AddCookieOptions adds cookie options for `SetCookie`,
// `AddCookie` and `RemoveCookie` methods
// for the current request. It can be called from a middleware before
// cookies sent or received from the next Handler in the chain.
//
// Available builtin Cookie options are:
//   - CookieOverride
//   - CookieDomain
//   - CookieAllowReclaim
//   - CookieAllowSubdomains
//   - CookieSecure
//   - CookieHTTPOnly
//   - CookieSameSite
//   - CookiePath
//   - CookieCleanPath
//   - CookieExpires
//   - CookieEncoding
func (ctx *Context) AddCookieOptions(options ...CookieOption) {
	if len(options) == 0 {
		return
	}

	if v := ctx.values.Get(cookieOptionsContextKey); v != nil {
		if opts, ok := v.([]CookieOption); ok {
			options = append(opts, options...)
		}
	}

	ctx.values.Set(cookieOptionsContextKey, options)
}

func (ctx *Context) applyCookieOptions(c *http.Cookie, op uint8, override []CookieOption) {
	if v := ctx.values.Get(cookieOptionsContextKey); v != nil {
		if options, ok := v.([]CookieOption); ok {
			for _, opt := range options {
				opt(ctx, c, op)
			}
		}
	}

	// The function's ones should be called last, so they can override
	// the stored ones (i.e by a prior middleware).
	for _, opt := range override {
		opt(ctx, c, op)
	}
}

// ClearCookieOptions clears any previously registered cookie options.
// See `AddCookieOptions` too.
func (ctx *Context) ClearCookieOptions() {
	ctx.values.Remove(cookieOptionsContextKey)
}

// SetCookie adds a cookie.
// Use of the "options" is not required, they can be used to amend the "cookie".
// Options registered for the whole request through [Context.AddCookieOptions]
// apply as well, and the ones passed here run last.
//
// SetCookie replaces, like [Context.SetHeader]: if the response already carries
// a Set-Cookie line with the same name, path and domain, that line is replaced
// in place; a cookie with a different path or domain is added as a new line.
//
// The cookie is written as given: unlike [Context.AddCookie], SetCookie sets no
// defaults. Set the security attributes yourself on anything that identifies a
// session:
//
//	ctx.SetCookie(&http.Cookie{
//		Name:     "session",
//		Value:    id,
//		Path:     "/",
//		HttpOnly: true,                  // not readable from JavaScript
//		Secure:   true,                  // HTTPS only
//		SameSite: http.SameSiteLaxMode,  // not sent on cross-site POSTs
//	})
//
// See [Context.RemoveCookie] to delete one, and note that it must be given the
// same Path and Domain or the browser keeps the original.
func (ctx *Context) SetCookie(cookie *http.Cookie, options ...CookieOption) {
	ctx.applyCookieOptions(cookie, OpCookieSet, options)
	ctx.writeSetCookie(cookie)
}

const setCookieHeaderKey = "Set-Cookie"

// writeSetCookie writes the cookie as a Set-Cookie line. A line already in the
// response for the same name, path and domain is replaced in place and any
// further duplicates of it are dropped (RFC 6265 section 4.1.1); otherwise the
// line is appended. Lines that do not parse are left alone.
func (ctx *Context) writeSetCookie(cookie *http.Cookie) {
	line := cookie.String()
	if line == "" { // invalid name, http.SetCookie writes nothing either.
		return
	}

	header := ctx.writer.Header()
	lines := header[setCookieHeaderKey]
	if len(lines) == 0 {
		header[setCookieHeaderKey] = []string{line}
		return
	}

	// Key the new cookie the way it goes out on the wire, so both sides of
	// the comparison carry what http.Cookie.String wrote.
	written, err := http.ParseSetCookie(line)
	if err != nil {
		header[setCookieHeaderKey] = append(lines, line)
		return
	}

	prefix := cookie.Name + "="
	replaced := false
	kept := lines[:0]
	for _, existing := range lines {
		if strings.HasPrefix(existing, prefix) && sameCookieKey(existing, written) {
			if !replaced {
				kept = append(kept, line)
				replaced = true
			}
			continue // drop further duplicates.
		}
		kept = append(kept, existing)
	}

	if !replaced {
		header[setCookieHeaderKey] = append(lines, line)
		return
	}

	clear(lines[len(kept):])
	header[setCookieHeaderKey] = kept
}

// sameCookieKey reports whether the Set-Cookie "line" names the same cookie as
// "c": equal name and path, and equal domain ignoring case and a leading dot.
func sameCookieKey(line string, c *http.Cookie) bool {
	existing, err := http.ParseSetCookie(line)
	if err != nil {
		return false
	}

	return existing.Name == c.Name && existing.Path == c.Path &&
		strings.EqualFold(strings.TrimPrefix(existing.Domain, "."), strings.TrimPrefix(c.Domain, "."))
}

// DefaultCookieExpiration is the lifetime [Context.AddCookie] gives a cookie
// when no [CookieExpires] option is passed: 365 days. Change it once for the
// whole process with iris.Patches().Context().SetDefaultCookieExpiration.
var DefaultCookieExpiration = 8760 * time.Hour

// AddCookie adds a cookie, requires the name(string) and the value(string).
// It goes through [Context.SetCookie], so a second AddCookie with the same name,
// path and domain replaces the first one instead of adding a second line.
//
// By default it expires after 365 days and it is added to the root URL path,
// use the `CookieExpires` and `CookiePath` to modify them.
// Alternatively: ctx.SetCookie(&http.Cookie{...}) or ctx.AddCookieOptions(...)
//
// If you want to set custom the path:
// ctx.AddCookie(name, value, iris.CookiePath("/custom/path/cookie/will/be/stored"))
//
// If you want to be visible only to current request path:
// (note that client should be responsible for that if server sent an empty cookie's path, all browsers are compatible)
// ctx.AddCookie(name, value, iris.CookieCleanPath/iris.CookiePath(""))
// More:
//
//	iris.CookieExpires(time.Duration)
//	iris.CookieHTTPOnly(false)
func (ctx *Context) AddCookie(name, value string, options ...CookieOption) {
	c := &http.Cookie{}
	c.Path = "/"
	c.Name = name
	c.Value = url.QueryEscape(value)
	c.HttpOnly = true
	c.SameSite = http.SameSiteLaxMode // v14: default SameSite=Lax to mitigate CSRF. Adjust with the CookieSameSite option; add the Secure attribute with iris.CookieSecure (TLS-aware) or CookieSecureForce.

	// MaxAge=0 means no 'Max-Age' attribute specified.
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	// MaxAge>0 means Max-Age attribute present and given in seconds
	c.Expires = time.Now().Add(DefaultCookieExpiration)
	c.MaxAge = int(time.Until(c.Expires).Seconds())

	ctx.SetCookie(c, options...)
}

var (
	// CookieExpireDelete may be set on Cookie.Expire for expiring the given cookie.
	CookieExpireDelete = time.Unix(0, 0) // time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	// CookieExpireUnlimited indicates that does expires after 24 years.
	CookieExpireUnlimited = time.Now().AddDate(24, 10, 10)
)

// RemoveCookie deletes a cookie by its name and path = "/".
// Tip: change the cookie's path to the current one by: RemoveCookie("the_cookie_name", iris.CookieCleanPath)
//
// If you intend to remove a cookie with a specific domain and value, please ensure to pass these values explicitly:
//
//	ctx.RemoveCookie("the_cookie_name", iris.CookieDomain("example.com"), iris.CookiePath("/"))
//
// OR use a Cookie value instead:
//
//	ctx.RemoveCookie("the_cookie_name", iris.CookieOverride(&http.Cookie{Domain: "example.com", Path: "/"}))
func (ctx *Context) RemoveCookie(name string, options ...CookieOption) {
	// Note(@kataras): the cookie must be fully populated BEFORE the options run.
	//
	// Options that consult CookieIncluded (CookieAllowSubdomains,
	// CookieAllowReclaim, CookieEncoding) bail out immediately on an empty Name,
	// so applying them first made all of them silent no-ops on removal. The
	// visible effect: a session cookie set with CookieAllowSubdomains carries a
	// Domain, but its deletion was emitted without one, so the browser kept the
	// cookie and the session survived "logout".
	c := &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		// Set the cookie expiration date to a past time
		Expires: CookieExpireDelete,
		MaxAge:  -1, // RFC says 1 second, but let's do it -1  to make sure is working.
	}

	ctx.applyCookieOptions(c, OpCookieDel, options)

	// Send the cookie back to the client, replacing a cookie set earlier in
	// this response with the same name, path and domain.
	ctx.writeSetCookie(c)
}

const (
	// ContentBinaryHeaderValue header value for binary data.
	ContentBinaryHeaderValue = "application/octet-stream"
	// ContentWebassemblyHeaderValue header value for web assembly files.
	ContentWebassemblyHeaderValue = "application/wasm"
	// ContentHTMLHeaderValue is the  string of text/html response header's content type value.
	ContentHTMLHeaderValue = "text/html"
	// ContentJSONHeaderValue header value for JSON data.
	ContentJSONHeaderValue = "application/json"
	// ContentJSONProblemHeaderValue header value for JSON API problem error.
	// Read more at: https://tools.ietf.org/html/rfc7807
	ContentJSONProblemHeaderValue = "application/problem+json"
	// ContentXMLProblemHeaderValue header value for XML API problem error.
	// Read more at: https://tools.ietf.org/html/rfc7807
	ContentXMLProblemHeaderValue = "application/problem+xml"
	// ContentJavascriptHeaderValue header value for JSONP & Javascript data.
	ContentJavascriptHeaderValue = "text/javascript"
	// ContentTextHeaderValue header value for Text data.
	ContentTextHeaderValue = "text/plain"
	// ContentXMLHeaderValue header value for XML data.
	ContentXMLHeaderValue = "text/xml"
	// ContentXMLUnreadableHeaderValue obselete header value for XML.
	ContentXMLUnreadableHeaderValue = "application/xml"
	// ContentMarkdownHeaderValue custom key/content type, the real is the text/html.
	ContentMarkdownHeaderValue = "text/markdown"
	// ContentYAMLHeaderValue header value for YAML data.
	ContentYAMLHeaderValue = "application/x-yaml"
	// ContentYAMLTextHeaderValue header value for YAML plain text.
	ContentYAMLTextHeaderValue = "text/yaml"
	// ContentProtobufHeaderValue header value for Protobuf messages data.
	ContentProtobufHeaderValue = "application/x-protobuf"
	// ContentMsgPackHeaderValue header value for MsgPack data.
	ContentMsgPackHeaderValue = "application/msgpack"
	// ContentMsgPack2HeaderValue alternative header value for MsgPack data.
	ContentMsgPack2HeaderValue = "application/x-msgpack"
	// ContentFormHeaderValue header value for post form data.
	ContentFormHeaderValue = "application/x-www-form-urlencoded"
	// ContentFormMultipartHeaderValue header value for post multipart form data.
	ContentFormMultipartHeaderValue = "multipart/form-data"
	// ContentMultipartRelatedHeaderValue header value for multipart related data.
	ContentMultipartRelatedHeaderValue = "multipart/related"
	// ContentGRPCHeaderValue Content-Type header value for gRPC.
	ContentGRPCHeaderValue = "application/grpc"
)

// SetCompressHeaders sets the necessary headers for the compression.
func SetCompressHeaders(h http.Header, encoding string) {
	h.Set(VaryHeaderKey, AcceptEncodingHeaderKey)
	h.Set(ContentEncodingHeaderKey, encoding)
	h.Del(ContentLengthHeaderKey)
}

type (
	// ErrorHandler describes a context error handler which applies on
	// Render, RenderComponent, JSON, JSONP, XML, Markdown, YAML, Protobuf and MsgPack writes.
	//
	// It does NOT modify or handle the result of Context.GetErr at all,
	// use a custom middleware instead if you want to handle the handler-provided custom errors (Context.SetErr)
	//
	// An ErrorHandler is registered through Options.Context.ErrorHandler
	// (or per request via iris.WithErrorHandler) to override the default
	// behavior, which is to send an internal server error status code
	// without a body back to the client.
	ErrorHandler = func(ctx *Context, err error)
)

// DefaultErrorHandler is the default error handler for Render, RenderComponent, JSON, JSONP, XML, Markdown, YAML, Protobuf and MsgPack write methods.
var DefaultErrorHandler ErrorHandler = func(ctx *Context, err error) {
	if ctx.IsDebug() {
		ctx.app.Logger().Error(err)
	}
	ctx.StopWithPlainError(http.StatusInternalServerError, err)
}

func (ctx *Context) handleContextError(err error) error {
	if err == nil {
		return nil
	}

	ctx.OptionsReadOnly().HandleError(ctx, err)
	return err
}

// StopWithStatus stops the handlers chain and writes the "statusCode".
//
// If the status code is a failure one then
// it will also fire the specified error code handler.
func (ctx *Context) StopWithStatus(statusCode int) {
	ctx.StopExecution()
	ctx.StatusCode(statusCode)
}

// StopWithText stops the handlers chain and
// writes the "statusCode" among with a text.
//
// If the status code is a failure one then
// it will also fire the specified error code handler.
func (ctx *Context) StopWithText(statusCode int, text string) {
	ctx.StopWithStatus(statusCode)
	ctx.WriteString(text)
}

// StopWithError stops the handlers chain and writes the "statusCode"
// among with the error "err".
// It Calls the `SetErr` method so error handlers can access the given error.
//
// If the status code is a failure one then
// it will also fire the specified error code handler.
//
// If the given "err" is private then the
// status code's text is rendered instead (unless a registered error handler overrides it).
// An error is private when it is an ErrPrivate or wraps one anywhere in its
// chain: PrivateError(err), PrivateErrorf(err) (rest/errors.Private) and a
// private error wrapped again with fmt.Errorf("...: %w", err) all count.
func (ctx *Context) StopWithError(statusCode int, err error) {
	if err == nil {
		return
	}

	if hasPrivateError(err) {
		// error is private, we SHOULD not render it,
		// leave the error handler alone to
		// render the code's text instead.
		//
		// Stored through PrivateError, which leaves an error that is private
		// at the top as it is and wraps one that carries the mark deeper in
		// its chain. ErrPublic looks at the stored value alone, and the
		// router's default error handler writes the text of whatever it
		// reports as public. GetErr still returns "err".
		ctx.SetErr(PrivateError(err))
		ctx.StopWithStatus(statusCode)
		return
	}

	ctx.SetErr(err)
	ctx.StopWithText(statusCode, err.Error()) // lint: allow inefficient strings concatenation.
}

// hasPrivateError reports whether err, or an error anywhere in the tree it
// wraps, is an ErrPrivate. StopWithError uses it; ErrPublic does not, it
// looks at the stored value alone.
func hasPrivateError(err error) bool {
	_, ok := errors.AsType[ErrPrivate](err)
	return ok
}

// StopWithPlainError like `StopWithError` but it does NOT
// write anything to the response writer, it stores the error
// so any error handler matching the given "statusCode" can handle it by its own.
func (ctx *Context) StopWithPlainError(statusCode int, err error) {
	if err == nil {
		return
	}

	ctx.SetErr(err)
	ctx.StopWithStatus(statusCode)
}

// StopWithJSON stops the handlers chain, writes the status code
// and sends a JSON response.
//
// If the status code is a failure one then
// it will also fire the specified error code handler.
//
// When a body has already been written the status went out with it, so
// StopWithJSON only stops the chain and returns nil: appending a JSON document
// to a committed body would corrupt the response the client is reading.
func (ctx *Context) StopWithJSON(statusCode int, jsonObject any) error {
	if ctx.writer.Written() > 0 {
		ctx.StopExecution()
		return nil
	}

	ctx.StopWithStatus(statusCode)
	return ctx.writeJSON(jsonObject, &DefaultJSONOptions) // do not modify - see errors.DefaultContextErrorHandler.
}

// Binary writes out the raw bytes as binary data.
// It returns any write error; see `Write` for the byte count.
func (ctx *Context) Binary(data []byte) error {
	ctx.SetContentType(ContentBinaryHeaderValue)
	_, err := ctx.Write(data)
	return err
}

// Text writes out a string as plain text.
// It returns any write error; see `WriteString` for the byte count.
func (ctx *Context) Text(text string) error {
	ctx.SetContentType(ContentTextHeaderValue)
	_, err := ctx.WriteString(text)
	return err
}

// HTML writes out a string as text/html.
//
// The body is written exactly as given. Nothing here escapes it, so
// interpolating a request value into the string is a cross-site scripting hole.
// Render through the view middleware, which escapes by default, or build the
// markup with html/template.
//
// It returns any write error; see `WriteString` for the byte count.
func (ctx *Context) HTML(body string) error {
	ctx.SetContentType(ContentHTMLHeaderValue)
	_, err := ctx.WriteString(body)
	return err
}

// jsonArrayPrefix is written before a top-level JSON array when the
// JSON.ArrayPrefix option is set (anti-hijacking).
var jsonArrayPrefix = []byte("while(1);")

// JSON contains the options for the JSON (Context's) Renderer.
type JSON struct {
	// EscapeHTML escapes <, > and & inside JSON strings as <, >
	// and & (the old encoding/json default). Off by default: JSON
	// parsers decode both forms identically, responses are served as
	// application/json (never inline HTML), and the bytes are smaller.
	// JSONP responses always escape, regardless of this option — there the
	// payload lands in a script body where "</script>" breakout matters.
	EscapeHTML bool
	// Indent is the per-level indentation string; empty writes compact JSON.
	Indent string
	// Prefix is written before the body, e.g. ")]}',\n".
	Prefix string
	// EscapeNonASCII writes every non-ASCII rune as a \uXXXX escape
	// (surrogate pairs above the Basic Multilingual Plane), so the body is
	// pure ASCII.
	EscapeNonASCII bool
	// ArrayPrefix prepends "while(1);" when the top-level value encodes to a
	// JSON array. The prefix makes the body a syntax error for a <script>
	// tag, which is the classic guard against JSON array hijacking; clients
	// strip it before parsing. Objects are never prefixed.
	ArrayPrefix bool
	// MarshalOptions are appended after the framework defaults
	// (deterministic map ordering, invalid UTF-8 allowed on output); later
	// options win, so any encoding/json/v2 or jsontext encode behavior can
	// be overridden per call.
	MarshalOptions []jsonv2.Options
}

// DefaultJSONOptions is the optional settings that are being used
// inside `Context.JSON`.
var DefaultJSONOptions = JSON{}

// IsDefault reports whether this JSON options structure holds the default values.
func (j *JSON) IsDefault() bool {
	return j.EscapeHTML == DefaultJSONOptions.EscapeHTML &&
		j.Indent == DefaultJSONOptions.Indent &&
		j.Prefix == DefaultJSONOptions.Prefix &&
		j.EscapeNonASCII == DefaultJSONOptions.EscapeNonASCII &&
		j.ArrayPrefix == DefaultJSONOptions.ArrayPrefix &&
		len(j.MarshalOptions) == 0
}

// MarshalDurationAsNanos and UnmarshalDurationFromNanos preserve the old
// encoding/json representation of time.Duration — a plain int64 nanosecond
// count; see the jsonutil package, the single source of the framework's
// JSON policy. They are re-exported here so custom encoders can compose them
// without a second import.
var (
	MarshalDurationAsNanos     = jsonutil.MarshalDurationAsNanos
	UnmarshalDurationFromNanos = jsonutil.UnmarshalDurationFromNanos
)

// WriteJSON marshals the given interface object and writes the JSON response to the 'writer'.
//
// The value is marshaled into a pooled buffer first and written as one piece:
// streaming into the live response writer would emit a partial body when the
// marshaler fails halfway, corrupting the error response that follows.
var WriteJSON = func(ctx *Context, v any, options *JSON) error {
	// The all-default path (the overwhelmingly common one) passes the
	// precomputed slice: variadic reuse of an existing slice allocates
	// nothing, where building one per response costs a heap allocation.
	opts := jsonutil.DefaultMarshalOptionsSlice
	if options.EscapeHTML || options.Indent != "" || len(options.MarshalOptions) > 0 {
		extended := make([]jsonv2.Options, 0, 4+len(options.MarshalOptions))
		extended = append(extended, jsonutil.DefaultMarshalOptions)
		if options.EscapeHTML {
			extended = append(extended, jsontext.EscapeForHTML(true))
		}
		if options.Indent != "" {
			extended = append(extended, jsontext.Multiline(true), jsontext.WithIndent(options.Indent))
		}
		opts = append(extended, options.MarshalOptions...)
	}

	buf := jsonutil.AcquireBuffer()
	defer jsonutil.ReleaseBuffer(buf)

	if err := jsonv2.MarshalWrite(buf, v, opts...); err != nil {
		return err // a v2 error carries the JSON pointer of the offending value.
	}

	// Preserve the trailing newline the old Encoder.Encode fast path always
	// produced (and the old ArrayPrefix/EscapeNonASCII/Prefix path inconsistently lacked).
	// Appended here so the common path costs a single ctx.Write.
	buf.WriteByte('\n')
	result := buf.Bytes()

	if options.EscapeNonASCII {
		result = escapeNonASCII(result)
	}

	if prefix := options.Prefix; prefix != "" {
		if _, err := ctx.Write(stringToBytes(prefix)); err != nil {
			return err
		}
	}

	if options.ArrayPrefix && len(result) > 0 && result[0] == '[' {
		if _, err := ctx.Write(jsonArrayPrefix); err != nil {
			return err
		}
	}

	_, err := ctx.Write(result)
	return err
}

// escapeNonASCII rewrites every non-ASCII rune as a \uXXXX escape, using
// UTF-16 surrogate pairs for runes above the Basic Multilingual Plane.
// (The previous implementation printed an astral rune as a single five-hex
// escape — "ὠ0" for U+1F600 — which no JSON parser can decode.)
func escapeNonASCII(src []byte) []byte {
	ascii := true
	for _, b := range src {
		if b >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii {
		return src
	}

	out := make([]byte, 0, len(src)+16)
	for _, r := range string(src) {
		switch {
		case r < 0x80:
			out = append(out, byte(r))
		case r <= 0xFFFF:
			out = fmt.Appendf(out, "\\u%04x", r)
		default:
			hi, lo := utf16.EncodeRune(r)
			out = fmt.Appendf(out, "\\u%04x\\u%04x", hi, lo)
		}
	}

	return out
}

// See https://golang.org/src/strings/builder.go#L45
// func bytesToString(b []byte) string {
// 	return unsafe.String(unsafe.SliceData(b), len(b))
// }

func stringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// JSON marshals the given "v" value to JSON and writes the response to the client.
//
// It reports any JSON parser or write errors back to the caller.
// Set Options.Context.ErrorHandler (or iris.WithErrorHandler for a
// per-request override) to replace the default 500 error response.
//
// The behavior of every `Context.JSON` call can be customized by replacing the
// package-level `WriteJSON` function on program initialization.
func (ctx *Context) JSON(v any, opts ...JSON) error {
	var options *JSON
	if len(opts) > 0 {
		options = &opts[0]
	} else {
		// If no options are given safely read the already-initialized value.
		options = &DefaultJSONOptions
	}

	return ctx.writeJSON(v, options)
}

func (ctx *Context) writeJSON(v any, options *JSON) error {
	ctx.SetContentType(ContentJSONHeaderValue)
	return ctx.handleContextError(WriteJSON(ctx, v, options))
}

// ProtobufJSON marshals the given "m" value of proto Message to JSON and writes the response to the client.
// It reports any JSON parser or write errors back to the caller.
func (ctx *Context) ProtobufJSON(m proto.Message, opts ...protojson.MarshalOptions) error {
	var protoJSON protojson.MarshalOptions
	if len(opts) > 0 {
		protoJSON = opts[0]
	}

	result, err := protoJSON.Marshal(m)
	if err != nil {
		return ctx.handleContextError(err)
	}

	ctx.SetContentType(ContentJSONHeaderValue)
	_, err = ctx.writer.Write(result)
	return ctx.handleContextError(err)
}

var finishCallbackB = []byte(");")

// ErrInvalidJSONPCallback is returned by `WriteJSONP` when the JSONP callback
// is not a valid JavaScript identifier reference.
//
// The callback is written into a "text/javascript" response verbatim, and the
// canonical way to populate it is from the request itself:
//
//	ctx.JSONP(v, iris.JSONP{Callback: ctx.Query().Get("callback")})
//
// so an unvalidated callback is attacker-authored script served from the
// application's own origin.
var ErrInvalidJSONPCallback = errors.New("invalid jsonp callback")

// jsonpCallbackRegexp matches a JavaScript identifier reference, optionally
// dotted or indexed with an integer, e.g. "cb", "window.cb", "cbs[0]".
var jsonpCallbackRegexp = regexp.MustCompile(`^[A-Za-z_$][\w$]*(?:\.[A-Za-z_$][\w$]*|\[\d+\])*$`)

// IsValidJSONPCallback reports whether the given JSONP callback is safe to
// write into a JavaScript response body. See `ErrInvalidJSONPCallback`.
func IsValidJSONPCallback(callback string) bool {
	// The bound keeps a pathological input from reaching the regexp at all;
	// no real callback name is anywhere near this long.
	if callback == "" || len(callback) > 128 {
		return false
	}

	return jsonpCallbackRegexp.MatchString(callback)
}

// WriteJSONP marshals the given interface object and writes the JSONP response to the writer.
//
// The whole response (callback, payload, closing) is assembled in a pooled
// buffer and written as one piece, so a marshal failure leaves nothing on the
// wire; writing "callback(" first left a dangling script prefix in front of
// the error response that followed.
//
// It returns `ErrInvalidJSONPCallback` when the callback is not a valid
// JavaScript identifier reference.
var WriteJSONP = func(ctx *Context, v any, options *JSONP) error {
	callback := options.Callback
	if callback != "" && !IsValidJSONPCallback(callback) {
		return fmt.Errorf("%w: %q", ErrInvalidJSONPCallback, callback)
	}

	// The payload lands in a script body, so it is always escaped for it:
	// an unescaped "</script>" in a string value breaks out of the element
	// context, and a raw U+2028/U+2029 (valid in JSON strings) terminates a
	// JavaScript string literal.
	opts := []jsonv2.Options{jsonutil.DefaultMarshalOptions, jsontext.EscapeForHTML(true), jsontext.EscapeForJS(true)}
	if options.Indent != "" {
		opts = append(opts, jsontext.Multiline(true), jsontext.WithIndent(options.Indent))
	}

	buf := jsonutil.AcquireBuffer()
	defer jsonutil.ReleaseBuffer(buf)

	if callback != "" {
		buf.WriteString(callback)
		buf.WriteByte('(')
	}

	if err := jsonv2.MarshalWrite(buf, v, opts...); err != nil {
		return err
	}

	buf.WriteByte('\n') // parity with the plain JSON writer.
	if callback != "" {
		buf.Write(finishCallbackB)
	}

	_, err := ctx.Write(buf.Bytes())
	return err
}

// JSONP contains the options for the JSONP (Context's) Renderer.
type JSONP struct {
	// content-specific
	Indent   string
	Callback string
}

// DefaultJSONPOptions is the optional settings that are being used
// inside `ctx.JSONP`.
var DefaultJSONPOptions = JSONP{}

// JSONP marshals the given "v" value to JSON and sends the response to the client.
//
// It reports any JSON parser or write errors back to the caller.
// Set Options.Context.ErrorHandler (or iris.WithErrorHandler for a
// per-request override) to replace the default 500 error response.
func (ctx *Context) JSONP(v any, opts ...JSONP) (err error) {
	var options *JSONP
	if len(opts) > 0 {
		options = &opts[0]
	} else {
		options = &DefaultJSONPOptions
	}

	ctx.SetContentType(ContentJavascriptHeaderValue)
	if err = WriteJSONP(ctx, v, options); err != nil {
		ctx.handleContextError(err)
	}

	return
}

type xmlMapEntry struct {
	XMLName xml.Name
	Value   any `xml:",chardata"`
}

// XMLMap wraps a map[string]any to compatible xml marshaler,
// in order to be able to render maps as XML on the `Context.XML` method.
//
// Example: `Context.XML(XMLMap("Root", map[string]any{...})`.
func XMLMap(elementName string, v Map) xml.Marshaler {
	return xmlMap{
		entries:     v,
		elementName: elementName,
	}
}

type xmlMap struct {
	entries     Map
	elementName string
}

// MarshalXML marshals a map to XML.
func (m xmlMap) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(m.entries) == 0 {
		return nil
	}

	start.Name = xml.Name{Local: m.elementName}
	err := e.EncodeToken(start)
	if err != nil {
		return err
	}

	for k, v := range m.entries {
		err = e.Encode(xmlMapEntry{XMLName: xml.Name{Local: k}, Value: v})
		if err != nil {
			return err
		}
	}

	return e.EncodeToken(start.End())
}

// XML contains the options for the XML (Context's) Renderer.
type XML struct {
	// content-specific
	Indent string
	Prefix string
}

// DefaultXMLOptions is the optional settings that are being used
// from `ctx.XML`.
var DefaultXMLOptions = XML{}

// WriteXML marshals the given interface object and writes the XML response to the writer.
var WriteXML = func(ctx *Context, v any, options *XML) error {
	if prefix := options.Prefix; prefix != "" {
		_, err := ctx.Write(stringToBytes(prefix))
		if err != nil {
			return err
		}
	}

	encoder := xml.NewEncoder(ctx.writer)
	encoder.Indent("", options.Indent)
	if err := encoder.Encode(v); err != nil {
		return err
	}

	return encoder.Flush()
}

// XML marshals the given interface object and writes the XML response to the client.
// To render maps as XML see the `XMLMap` package-level function.
//
// It reports any XML parser or write errors back to the caller.
// Set Options.Context.ErrorHandler (or iris.WithErrorHandler for a
// per-request override) to replace the default 500 error response.
func (ctx *Context) XML(v any, opts ...XML) (err error) {
	var options *XML
	if len(opts) > 0 {
		options = &opts[0]
	} else {
		options = &DefaultXMLOptions
	}

	ctx.SetContentType(ContentXMLHeaderValue)
	if err = WriteXML(ctx, v, options); err != nil {
		ctx.handleContextError(err)
	}

	return
}

var sanitizer = bluemonday.UGCPolicy()

// Markdown contains the options for the Markdown (Context's) Renderer.
type Markdown struct {
	// content-specific
	Sanitize bool
	//
	// Library-specific.
	// E.g. Flags: html.CommonFlags | html.HrefTargetBlank
	RenderOptions html.RendererOptions
}

// DefaultMarkdownOptions is the optional settings that are being used
// from `WriteMarkdown` and `ctx.Markdown`.
var DefaultMarkdownOptions = Markdown{}

// WriteMarkdown parses the markdown to html and writes these contents to the writer.
var WriteMarkdown = func(ctx *Context, markdownB []byte, options *Markdown) error {
	out := markdown.NormalizeNewlines(markdownB)

	renderer := html.NewRenderer(options.RenderOptions)
	doc := markdown.Parse(out, nil)
	out = markdown.Render(doc, renderer)
	if options.Sanitize {
		out = sanitizer.SanitizeBytes(out)
	}

	_, err := ctx.Write(out)
	return err
}

// Markdown parses the markdown to html and renders its result to the client.
//
// It reports any Markdown parser or write errors back to the caller.
// Set Options.Context.ErrorHandler (or iris.WithErrorHandler for a
// per-request override) to replace the default 500 error response.
func (ctx *Context) Markdown(markdownB []byte, opts ...Markdown) (err error) {
	var options *Markdown
	if len(opts) > 0 {
		options = &opts[0]
	} else {
		options = &DefaultMarkdownOptions
	}

	ctx.SetContentType(ContentHTMLHeaderValue)
	if err = WriteMarkdown(ctx, markdownB, options); err != nil {
		ctx.handleContextError(err)
	}

	return
}

// WriteYAML sends YAML response to the client.
var WriteYAML = func(ctx *Context, v any, indentSpace int) error {
	encoder := yaml.NewEncoder(ctx.writer)
	encoder.SetIndent(indentSpace)

	if err := encoder.Encode(v); err != nil {
		return err
	}

	return encoder.Close()
}

// YAML marshals the given "v" value using the yaml marshaler and writes the result to the client.
//
// It reports any YAML parser or write errors back to the caller.
// Set Options.Context.ErrorHandler (or iris.WithErrorHandler for a
// per-request override) to replace the default 500 error response.
func (ctx *Context) YAML(v any) error {
	ctx.SetContentType(ContentYAMLHeaderValue)

	err := WriteYAML(ctx, v, 0)
	if err != nil {
		ctx.handleContextError(err)
		return err
	}

	return nil
}

// TextYAML calls the Context.YAML method but with the text/yaml content type instead.
func (ctx *Context) TextYAML(v any) error {
	ctx.SetContentType(ContentYAMLTextHeaderValue)

	err := WriteYAML(ctx, v, 4)
	if err != nil {
		ctx.handleContextError(err)
		return err
	}

	return nil
}

// Protobuf marshals the given "v" value of proto Message and writes its result to the client.
//
// It reports any protobuf parser or write errors back to the caller.
// Set Options.Context.ErrorHandler (or iris.WithErrorHandler for a
// per-request override) to replace the default 500 error response.
func (ctx *Context) Protobuf(v proto.Message) error {
	out, err := proto.Marshal(v)
	if err != nil {
		ctx.handleContextError(err)
		return err
	}

	ctx.SetContentType(ContentProtobufHeaderValue)
	if _, err = ctx.Write(out); err != nil {
		ctx.handleContextError(err)
		return err
	}

	return nil
}

// MsgPack marshals the given "v" value of msgpack format and writes its result to the client.
//
// It reports any message pack or write errors back to the caller.
// Set Options.Context.ErrorHandler (or iris.WithErrorHandler for a
// per-request override) to replace the default 500 error response.
func (ctx *Context) MsgPack(v any) error {
	out, err := msgpack.Marshal(v)
	if err != nil {
		ctx.handleContextError(err)
		return err
	}

	ctx.SetContentType(ContentMsgPackHeaderValue)
	if _, err = ctx.Write(out); err != nil {
		ctx.handleContextError(err)
		return err
	}

	return nil
}

// ResponseRenderer is the interface which all custom response renderers must implement.
type ResponseRenderer interface {
	// Render should push data with custom content type to the client.
	Render(http.ResponseWriter) error
	// WriteContentType writes custom content type to the response.
	WriteContentType(w http.ResponseWriter)
}

// RenderResponse writes the given status code and calls the given renderer "r" to render data.
// On 1xx, 204 and 304 responses only the content type is written, without a body.
// This method can be used while migrating from other frameworks.
//
// See `RenderComponent` and `Render` too.
func (ctx *Context) RenderResponse(statusCode int, r ResponseRenderer) error {
	ctx.StatusCode(statusCode)

	if statusCode >= 100 && statusCode <= 199 || statusCode == http.StatusNoContent || statusCode == http.StatusNotModified {
		r.WriteContentType(ctx.writer)
		return nil
	}

	err := r.Render(ctx.writer)
	if err != nil {
		ctx.handleContextError(err)
	}

	return err
}

// Component is the interface which all components must implement.
// A component is a struct which can be rendered to a writer.
// It's being used by the `Context.RenderComponent` method.
// An example of compatible Component is a templ.Component.
type Component interface {
	Render(context.Context, io.Writer) error
}

// RenderComponent renders a component to the client.
// It sets the "Content-Type" header to "text/html; charset=utf-8".
// It reports any component render errors back to the caller.
// Set Options.Context.ErrorHandler (or iris.WithErrorHandler for a
// per-request override) to replace the default 500 error response.
//
// See `RenderResponse` and `Render` too.
func (ctx *Context) RenderComponent(component Component) error {
	ctx.SetContentType(ContentHTMLHeaderValue)
	err := component.Render(ctx.Request().Context(), ctx.ResponseWriter())
	if err != nil {
		ctx.handleContextError(err)
	}
	return err
}

// Renderer is the interface that wraps the Render method. It is used to render a response to the client
// using the Iris Context as an argument.
type Renderer interface {
	Render(*Context) error
}

// Render writes the response headers and calls the given renderer "r" to render data.
//
// See `RenderResponse` and `RenderComponent` too.
func (ctx *Context) Render(r Renderer) error {
	err := r.Render(ctx)
	if err != nil {
		ctx.handleContextError(err)
	}

	return err
}

//  +-----------------------------------------------------------------------+
//  | Content Νegotiation                                                   |
//  | https://developer.mozilla.org/en-US/docs/Web/HTTP/Content_negotiation |                                       |
//  +-----------------------------------------------------------------------+

// ErrContentNotSupported returns from the `Negotiate` method
// when server responds with 406.
var ErrContentNotSupported = errors.New("unsupported content")

// ContentSelector is the interface which structs can implement
// to manually choose a content based on the negotiated mime (content type).
// It can be passed to the `Context.Negotiate` method.
//
// See the `N` struct too.
type ContentSelector interface {
	SelectContent(mime string) any
}

// ContentNegotiator is the interface which structs can implement
// to override the `Context.Negotiate` default implementation and
// manually respond to the client based on a manuall call of `Context.Negotiation().Build()`
// to get the final negotiated mime and charset.
// It can be passed to the `Context.Negotiate` method.
type ContentNegotiator interface {
	// mime and charset can be retrieved by:
	// mime, charset := Context.Negotiation().Build()
	// Pass this method to `Context.Negotiate` method
	// to write custom content.
	// Overriding the existing behavior of Context.Negotiate for selecting values based on
	// content types, although it can accept any custom mime type with []byte.
	// Content type is already set.
	// Use it with caution, 99.9% you don't need this but it's here for extreme cases.
	Negotiate(ctx *Context) error
}

// N is a struct which can be passed on the `Context.Negotiate` method.
// It contains fields which should be filled based on the `Context.Negotiation()`
// server side values. If no matched mime then its "Other" field will be sent,
// which should be a string or []byte.
// It completes the `ContentSelector` interface.
type N struct {
	Text, HTML string
	Markdown   []byte
	Binary     []byte

	JSON     any
	JSONP    any
	XML      any
	YAML     any
	Protobuf any
	MsgPack  any

	Other []byte // custom content types.
}

var _ ContentSelector = N{}

// SelectContent returns a content based on the matched negotiated "mime".
func (n N) SelectContent(mime string) any {
	switch mime {
	case ContentTextHeaderValue:
		return n.Text
	case ContentHTMLHeaderValue:
		return n.HTML
	case ContentMarkdownHeaderValue:
		return n.Markdown
	case ContentBinaryHeaderValue:
		return n.Binary
	case ContentJSONHeaderValue:
		return n.JSON
	case ContentJavascriptHeaderValue:
		return n.JSONP
	case ContentXMLHeaderValue, ContentXMLUnreadableHeaderValue:
		return n.XML
	case ContentYAMLHeaderValue:
		return n.YAML
	case ContentProtobufHeaderValue:
		return n.Protobuf
	case ContentMsgPackHeaderValue, ContentMsgPack2HeaderValue:
		return n.MsgPack
	default:
		return n.Other
	}
}

const negotiationContextKey = "iris.negotiation_builder"

// Negotiation creates once and returns the negotiation builder
// to build server-side available prioritized content
// for specific content type(s), charset(s) and encoding algorithm(s).
//
// See `Negotiate` method too.
func (ctx *Context) Negotiation() *NegotiationBuilder {
	if n := ctx.values.Get(negotiationContextKey); n != nil {
		return n.(*NegotiationBuilder)
	}

	acceptBuilder := NegotiationAcceptBuilder{}
	acceptBuilder.accept = parseHeader(ctx.Header("Accept"))
	acceptBuilder.charset = parseHeader(ctx.Header("Accept-Charset"))

	n := &NegotiationBuilder{Accept: acceptBuilder}

	ctx.values.Set(negotiationContextKey, n)

	return n
}

func parseHeader(headerValue string) []string {
	in := strings.Split(headerValue, ",")
	out := make([]string, 0, len(in))

	for _, value := range in {
		// remove any spaces and quality values such as ;q=0.8.
		v, _, _ := strings.Cut(value, ";")
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}

	return out
}

// Negotiate used for serving different representations of a resource at the same URI.
//
// The "v" can be a single `N` struct value.
// The "v" can be any value completes the `ContentSelector` interface.
// The "v" can be any value completes the `ContentNegotiator` interface.
// The "v" can be any value of struct(JSON, JSONP, XML, YAML, Protobuf, MsgPack) or
// string(TEXT, HTML) or []byte(Markdown, Binary) or []byte with any matched mime type.
//
// If the "v" is nil, the `Context.Negotiation()` builder's
// content will be used instead, otherwise "v" overrides builder's content
// (server mime types are still retrieved by its registered, supported, mime list)
//
// Set mime type priorities by `Negotiation().JSON().XML().HTML()...`.
// Set charset priorities by `Negotiation().Charset(...)`.
// Set encoding algorithm priorities by `Negotiation().Encoding(...)`.
// Modify the accepted by
// `Negotiation().Accept./Override()/.XML().JSON().Charset(...).Encoding(...)...`.
//
// It returns `ErrContentNotSupported` when not matched mime type(s).
//
// Resources:
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Content_negotiation
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Charset
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Encoding
//
// Supports the above without quality values.
//
// Read more at: https://github.com/kataras/iris/tree/main/_examples/05-response/content-negotiation
func (ctx *Context) Negotiate(v any) error {
	mediaType, charset, _, content := ctx.Negotiation().Build()
	if v == nil {
		v = content
	}

	if mediaType == "" {
		// If the server cannot serve any matching set,
		// it SHOULD send back a 406 (Not Acceptable) error code.
		ctx.StatusCode(http.StatusNotAcceptable)
		return ErrContentNotSupported
	}

	if v == nil {
		// The mime type matched but neither the caller nor the builder provided
		// a value for it, e.g. Negotiation().HTML().JSON(data) answering a
		// "Accept: text/html" request. Without this the type switch below would
		// assert on a nil interface and panic the handler.
		ctx.StatusCode(http.StatusNotAcceptable)
		return ErrContentNotSupported
	}

	// if encoding != "" {
	// 	ctx.CompressWriter(true)
	// }

	ctx.SetContentType(mediaType, ContentTypeCharset(charset) /* note: if "charset" is empty then it will set it to empty by force*/, ContentTypeOnce)

	if n, ok := v.(ContentNegotiator); ok {
		return n.Negotiate(ctx)
	}

	if s, ok := v.(ContentSelector); ok {
		v = s.SelectContent(mediaType)
	}

	// switch v := value.(type) {
	// case []byte:
	// 	if contentType == ContentMarkdownHeaderValue {
	// 		return ctx.Markdown(v)
	// 	}

	// 	return ctx.Write(v)
	// case string:
	// 	return ctx.WriteString(v)
	// default:
	// make it switch by content-type only, but we lose custom mime types capability that way:
	//                                                 ^ solved with []byte on default case and
	//                                                 ^ N.Other and
	//                                                 ^ ContentSelector and ContentNegotiator interfaces.

	switch mediaType {
	case ContentTextHeaderValue, ContentHTMLHeaderValue:
		// Note(@kataras): comma-ok, never a bare assertion: "v" comes from the
		// caller or from the negotiation builder's registered contents, so a
		// mismatch here is a programming error to report, not a panic to take
		// on a request goroutine.
		s, ok := v.(string)
		if !ok {
			ctx.StatusCode(http.StatusNotAcceptable)
			return fmt.Errorf("%w: %s: expected string, got %T", ErrContentNotSupported, mediaType, v)
		}

		_, err := ctx.WriteString(s)
		return err
	case ContentMarkdownHeaderValue:
		b, ok := v.([]byte)
		if !ok {
			ctx.StatusCode(http.StatusNotAcceptable)
			return fmt.Errorf("%w: %s: expected []byte, got %T", ErrContentNotSupported, mediaType, v)
		}

		return ctx.Markdown(b)
	case ContentJSONHeaderValue:
		if msg, ok := v.(proto.Message); ok {
			return ctx.ProtobufJSON(msg)
		}

		// If not specific Accept-Charset, there is no utf-8 default charset need for JSON since this is redundant by RFC 8259.
		// Types implementing encoding/json/v2's MarshalerTo stream
		// straight through the encoder here — no special case needed.
		return ctx.JSON(v)
	case ContentJavascriptHeaderValue:
		return ctx.JSONP(v)
	case ContentXMLHeaderValue, ContentXMLUnreadableHeaderValue:
		return ctx.XML(v)
	case ContentYAMLHeaderValue:
		return ctx.YAML(v)
	case ContentYAMLTextHeaderValue:
		return ctx.TextYAML(v)
	case ContentProtobufHeaderValue:
		msg, ok := v.(proto.Message)
		if !ok {
			return ErrContentNotSupported
		}

		return ctx.Protobuf(msg)
	case ContentMsgPackHeaderValue, ContentMsgPack2HeaderValue:
		return ctx.MsgPack(v)
	default:
		// maybe "Other" or v is []byte or string but not a built-in framework mime,
		// for custom content types,
		// panic if not correct usage.
		switch vv := v.(type) {
		case []byte:
			_, err := ctx.Write(vv)
			return err
		case string:
			_, err := ctx.WriteString(vv)
			return err
		default:
			ctx.StatusCode(http.StatusNotAcceptable)
			return ErrContentNotSupported
		}
	}
}

// NegotiationBuilder returns from the `Context.Negotiation`
// and can be used inside chain of handlers to build server-side
// mime type(s), charset(s) and encoding algorithm(s)
// that should match with the client's
// Accept, Accept-Charset and Accept-Encoding headers (by-default).
// To modify the client's accept use its "Accept" field
// which it's the `NegotiationAcceptBuilder`.
//
// See the `Negotiate` method too.
type NegotiationBuilder struct {
	Accept NegotiationAcceptBuilder

	mime     []string       // we need order.
	contents map[string]any // map to the "mime" and content should be rendered if that mime requested.
	charset  []string
	encoding []string
}

// MIME registers a mime type and optionally the value that should be rendered
// through `Context.Negotiate` when this mime type is accepted by client.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) MIME(mime string, content any) *NegotiationBuilder {
	mimes := parseHeader(mime) // if contains more than one sep by commas ",".
	if content == nil {
		n.mime = append(n.mime, mimes...)
		return n
	}

	if n.contents == nil {
		n.contents = make(map[string]any)
	}

	for _, m := range mimes {
		n.mime = append(n.mime, m)
		n.contents[m] = content
	}

	return n
}

// Text registers the "text/plain" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "text/plain" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Text(v ...string) *NegotiationBuilder {
	var content any
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentTextHeaderValue, content)
}

// HTML registers the "text/html" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "text/html" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) HTML(v ...string) *NegotiationBuilder {
	var content any
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentHTMLHeaderValue, content)
}

// Markdown registers the "text/markdown" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "text/markdown" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Markdown(v ...[]byte) *NegotiationBuilder {
	var content any
	if len(v) > 0 {
		content = v
	}
	return n.MIME(ContentMarkdownHeaderValue, content)
}

// Binary registers the "application/octet-stream" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/octet-stream" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Binary(v ...[]byte) *NegotiationBuilder {
	var content any
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentBinaryHeaderValue, content)
}

// JSON registers the "application/json" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/json" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) JSON(v ...any) *NegotiationBuilder {
	var content any
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentJSONHeaderValue, content)
}

// JSONP registers the "text/javascript" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "text/javascript" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) JSONP(v ...any) *NegotiationBuilder {
	var content any
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentJavascriptHeaderValue, content)
}

// XML registers the "text/xml" and "application/xml" content types and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts one of the "text/xml" or "application/xml" content types.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) XML(v ...any) *NegotiationBuilder {
	var content any
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentXMLHeaderValue+","+ContentXMLUnreadableHeaderValue, content)
}

// YAML registers the "application/x-yaml" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/x-yaml" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) YAML(v ...any) *NegotiationBuilder {
	var content any
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentYAMLHeaderValue, content)
}

// TextYAML registers the "text/yaml" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "text/yaml" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) TextYAML(v ...any) *NegotiationBuilder {
	var content any
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentYAMLTextHeaderValue, content)
}

// Protobuf registers the "application/x-protobuf" content type and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts the "application/x-protobuf" content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Protobuf(v ...any) *NegotiationBuilder {
	var content any
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentProtobufHeaderValue, content)
}

// MsgPack registers the "application/x-msgpack" and "application/msgpack" content types and, optionally,
// a value that `Context.Negotiate` will render
// when a client accepts one of the "application/x-msgpack" or "application/msgpack" content types.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) MsgPack(v ...any) *NegotiationBuilder {
	var content any
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME(ContentMsgPackHeaderValue+","+ContentMsgPack2HeaderValue, content)
}

// Any registers a wildcard that can match any client's accept content type.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Any(v ...any) *NegotiationBuilder {
	var content any
	if len(v) > 0 {
		content = v[0]
	}
	return n.MIME("*", content)
}

// Charset overrides the application's config's charset (which defaults to "utf-8")
// that a client should match for
// (through Accept-Charset header or custom through `NegotiationBuilder.Accept.Override().Charset(...)` call).
// Do not set it if you don't know what you're doing.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Charset(charset ...string) *NegotiationBuilder {
	n.charset = append(n.charset, charset...)
	return n
}

// Encoding registers one or more encoding algorithms by name, i.e gzip, deflate, br, snappy, s2, zstd.
// that a client should match for (through Accept-Encoding header).
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) Encoding(encoding ...string) *NegotiationBuilder {
	n.encoding = append(n.encoding, encoding...)
	return n
}

// EncodingGzip registers the "gzip" encoding algorithm
// that a client should match for (through Accept-Encoding header or call of Accept.Encoding(enc)).
//
// It will make resources to served by "gzip" if Accept-Encoding contains the "gzip" as well.
//
// Returns itself for recursive calls.
func (n *NegotiationBuilder) EncodingGzip() *NegotiationBuilder {
	return n.Encoding(GzipHeaderValue)
}

// Build calculates the client's and server's mime type(s), charset(s) and encoding
// and returns the final content type, charset and encoding that server should render
// to the client. It does not clear the fields, use the `Clear` method if neeeded.
//
// The returned "content" can be nil if the matched "contentType" does not provide any value,
// in that case the `Context.Negotiate(v)` must be called with a non-nil value.
func (n *NegotiationBuilder) Build() (mediaType, charset, encoding string, content any) {
	mediaType = negotiationMatch(n.Accept.accept, n.mime)
	charset = negotiationMatch(n.Accept.charset, n.charset)
	encoding = negotiationMatch(n.Accept.encoding, n.encoding)

	if n.contents != nil {
		if data, ok := n.contents[mediaType]; ok {
			content = data
		}
	}

	return
}

// Clear clears the prioritized mime type(s), charset(s) and any contents
// relative to those mime type(s).
// The "Accept" field is stay as it is, use its `Override` method
// to clear out the client's accepted mime type(s) and charset(s).
func (n *NegotiationBuilder) Clear() *NegotiationBuilder {
	n.mime = n.mime[0:0]
	n.contents = nil
	n.charset = n.charset[0:0]
	return n
}

// NegotiationAcceptBuilder builds the accepted mime types and charset
//
// and "Accept-Charset" headers respectfully.
// The default values are set by the client side, server can append or override those.
// The end result will be challenged with runtime preffered set of content types and charsets.
//
// See the `Negotiate` method too.
type NegotiationAcceptBuilder struct {
	// initialized with "Accept" request header values.
	accept []string
	// initialized with "Accept-Charset" request header. and if was empty then the
	// application's default (which defaults to utf-8).
	charset []string
	// initialized with "Accept-Encoding" request header values.
	encoding []string

	// To support override in request life cycle.
	// We need slice when data is the same format
	// for one or more mime types,
	// i.e text/xml and obselete application/xml.
	lastAccept   []string
	lastCharset  []string
	lastEncoding []string
}

// Override clears the default values for accept and accept charset.
// Returns itself.
func (n *NegotiationAcceptBuilder) Override() *NegotiationAcceptBuilder {
	// when called first.
	n.accept = n.accept[0:0]
	n.charset = n.charset[0:0]
	n.encoding = n.encoding[0:0]

	// when called after.
	if len(n.lastAccept) > 0 {
		n.accept = append(n.accept, n.lastAccept...)
		n.lastAccept = n.lastAccept[0:0]
	}

	if len(n.lastCharset) > 0 {
		n.charset = append(n.charset, n.lastCharset...)
		n.lastCharset = n.lastCharset[0:0]
	}

	if len(n.lastEncoding) > 0 {
		n.encoding = append(n.encoding, n.lastEncoding...)
		n.lastEncoding = n.lastEncoding[0:0]
	}

	return n
}

// MIME adds accepted client's mime type(s).
// Returns itself.
func (n *NegotiationAcceptBuilder) MIME(mimeType ...string) *NegotiationAcceptBuilder {
	n.lastAccept = mimeType
	n.accept = append(n.accept, mimeType...)
	return n
}

// Text adds the "text/plain" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) Text() *NegotiationAcceptBuilder {
	return n.MIME(ContentTextHeaderValue)
}

// HTML adds the "text/html" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) HTML() *NegotiationAcceptBuilder {
	return n.MIME(ContentHTMLHeaderValue)
}

// Markdown adds the "text/markdown" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) Markdown() *NegotiationAcceptBuilder {
	return n.MIME(ContentMarkdownHeaderValue)
}

// Binary adds the "application/octet-stream" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) Binary() *NegotiationAcceptBuilder {
	return n.MIME(ContentBinaryHeaderValue)
}

// JSON adds the "application/json" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) JSON() *NegotiationAcceptBuilder {
	return n.MIME(ContentJSONHeaderValue)
}

// JSONP adds the "text/javascript" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) JSONP() *NegotiationAcceptBuilder {
	return n.MIME(ContentJavascriptHeaderValue)
}

// XML adds the "text/xml" and "application/xml" as accepted client content types.
// Returns itself.
func (n *NegotiationAcceptBuilder) XML() *NegotiationAcceptBuilder {
	return n.MIME(ContentXMLHeaderValue, ContentXMLUnreadableHeaderValue)
}

// YAML adds the "application/x-yaml" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) YAML() *NegotiationAcceptBuilder {
	return n.MIME(ContentYAMLHeaderValue)
}

// TextYAML adds the "text/yaml" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) TextYAML() *NegotiationAcceptBuilder {
	return n.MIME(ContentYAMLTextHeaderValue)
}

// Protobuf adds the "application/x-protobuf" as accepted client content type.
// Returns itself.
func (n *NegotiationAcceptBuilder) Protobuf() *NegotiationAcceptBuilder {
	return n.MIME(ContentProtobufHeaderValue)
}

// MsgPack adds the "application/msgpack" and "application/x-msgpack" as accepted client content types.
// Returns itself.
func (n *NegotiationAcceptBuilder) MsgPack() *NegotiationAcceptBuilder {
	return n.MIME(ContentMsgPackHeaderValue, ContentMsgPack2HeaderValue)
}

// Charset adds one or more client accepted charsets.
// Returns itself.
func (n *NegotiationAcceptBuilder) Charset(charset ...string) *NegotiationAcceptBuilder {
	n.lastCharset = charset
	n.charset = append(n.charset, charset...)

	return n
}

// Encoding adds one or more client accepted encoding algorithms.
// Returns itself.
func (n *NegotiationAcceptBuilder) Encoding(encoding ...string) *NegotiationAcceptBuilder {
	n.lastEncoding = encoding
	n.encoding = append(n.encoding, encoding...)

	return n
}

// EncodingGzip adds the "gzip" as accepted encoding.
// Returns itself.
func (n *NegotiationAcceptBuilder) EncodingGzip() *NegotiationAcceptBuilder {
	return n.Encoding(GzipHeaderValue)
}

//  +------------------------------------------------------------+
//  | Serve files                                                |
//  +------------------------------------------------------------+

// ServeContent replies to the request using the content in the
// provided ReadSeeker. The main benefit of ServeContent over io.Copy
// is that it handles Range requests properly, sets the MIME type, and
// handles If-Match, If-Unmodified-Since, If-None-Match, If-Modified-Since,
// and If-Range requests.
//
// If the response's Content-Type header is not set, ServeContent
// first tries to deduce the type from name's file extension.
//
// The name is otherwise unused; in particular it can be empty and is
// never sent in the response.
//
// If modtime is not the zero time or Unix epoch, ServeContent
// includes it in a Last-Modified header in the response. If the
// request includes an If-Modified-Since header, ServeContent uses
// modtime to decide whether the content needs to be sent at all.
//
// The content's Seek method must work: ServeContent uses
// a seek to the end of the content to determine its size.
//
// If the caller has set w's ETag header formatted per RFC 7232, section 2.3,
// ServeContent uses it to handle requests using If-Match, If-None-Match, or If-Range.
func (ctx *Context) ServeContent(content io.ReadSeeker, filename string, modtime time.Time) {
	ctx.ServeContentWithRate(content, filename, modtime, 0, 0)
}

// rateReadSeeker is a io.ReadSeeker that is rate limited by
// the given token bucket. Each token in the bucket
// represents one byte. See "golang.org/x/time/rate" package.
type rateReadSeeker struct {
	io.ReadSeeker
	ctx     context.Context
	limiter *rate.Limiter
}

func (rs *rateReadSeeker) Read(buf []byte) (int, error) {
	n, err := rs.ReadSeeker.Read(buf)
	if n <= 0 {
		return n, err
	}
	err = rs.limiter.WaitN(rs.ctx, n)
	return n, err
}

// ServeContentWithRate same as `ServeContent` but it can throttle the speed of reading
// and though writing the "content" to the client.
func (ctx *Context) ServeContentWithRate(content io.ReadSeeker, filename string, modtime time.Time, limit float64, burst int) {
	if limit > 0 {
		content = &rateReadSeeker{
			ReadSeeker: content,
			ctx:        ctx.request.Context(),
			limiter:    rate.NewLimiter(rate.Limit(limit), burst),
		}
	}

	http.ServeContent(ctx.writer, ctx.request, filename, modtime, content)
}

// ServeFile replies to the request with the contents of the named
// file or directory.
//
// If the provided file or directory name is a relative path, it is
// interpreted relative to the current directory and may ascend to
// parent directories. If the provided name is constructed from user
// input, it should be sanitized before calling `ServeFile`.
//
// Use it when you want to serve assets like css and javascript files.
// If client should confirm and save the file use the `SendFile` instead.
func (ctx *Context) ServeFile(filename string) error {
	return ctx.ServeFileWithRate(filename, 0, 0)
}

// ServeFileWithRate same as `ServeFile` but it can throttle the speed of reading
// and though writing the file to the client.
func (ctx *Context) ServeFileWithRate(filename string, limit float64, burst int) error {
	f, err := os.Open(filename)
	if err != nil {
		ctx.StatusCode(http.StatusNotFound)
		return err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		code := http.StatusInternalServerError
		if os.IsNotExist(err) {
			code = http.StatusNotFound
		}

		if os.IsPermission(err) {
			code = http.StatusForbidden
		}

		ctx.StatusCode(code)
		return err
	}

	if st.IsDir() {
		return ctx.ServeFile(path.Join(filename, "index.html"))
	}

	ctx.ServeContentWithRate(f, st.Name(), st.ModTime(), limit, burst)
	return nil
}

// SendFile sends a file as an attachment, that is downloaded and saved locally from client.
//
// "src" is the path on the server, "destName" the name the client is told to
// save it as; an empty destName falls back to filepath.Base(src). The name goes
// through [MakeDisposition], which encodes it per RFC 6266 so that a non-ASCII
// or quote-bearing name cannot break out of the Content-Disposition header.
//
// SECURITY: "src" is not sanitized. Joining a request value into it serves any
// file the process can read. Use [SafeFilename], or serve through the
// fileserver controller, which confines reads to one root.
//
// See `SendFileWithRate` to throttle the transfer.
func (ctx *Context) SendFile(src string, destName string) error {
	return ctx.SendFileWithRate(src, destName, 0, 0)
}

// SendFileWithRate same as `SendFile` but it can throttle the speed of reading
// and though writing the file to the client.
func (ctx *Context) SendFileWithRate(src, destName string, limit float64, burst int) error {
	if destName == "" {
		destName = filepath.Base(src)
	}

	ctx.writer.Header().Set(ContentDispositionHeaderKey, MakeDisposition(destName))
	return ctx.ServeFileWithRate(src, limit, burst)
}

// MakeDisposition generates an HTTP Content-Disposition field-value.
// Similar solution followed by: Spring(Java), Symfony(PHP) and Ruby on Rails frameworks too.
//
// Fixes CVE-2020-5398. Reported by motoyasu-saburi.
func MakeDisposition(filename string) string {
	return `attachment; filename*=UTF-8''` + url.QueryEscape(filename)
} /*
// Found at: https://stackoverflow.com/questions/53069040/checking-a-string-contains-only-ascii-characters
// A faster (better, more idiomatic) version, which avoids unnecessary rune conversions.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}
func isRFC5987AttrChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		c == '!' || c == '#' || c == '$' || c == '&' || c == '+' || c == '-' ||
		c == '.' || c == '^' || c == '_' || c == '`' || c == '|' || c == '~'
}
*/

// Write writes the data to the connection as part of an HTTP reply.
//
// If WriteHeader has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. If the Header
// does not contain a Content-Type line, Write adds a Content-Type set
// to the result of passing the initial 512 bytes of written data to
// DetectContentType.
//
// Depending on the HTTP protocol version and the client, calling
// Write or WriteHeader may prevent future reads on the
// Request.Body. For HTTP/1.x requests, handlers should read any
// needed request body data before writing the response. Once the
// headers have been flushed (due to either an explicit Flusher.Flush
// call or writing enough data to trigger a flush), the request body
// may be unavailable. For HTTP/2 requests, the Go HTTP server permits
// handlers to continue to read the request body while concurrently
// writing the response. However, such behavior may not be supported
// by all HTTP/2 clients. Handlers should read before writing if
// possible to maximize compatibility.
//
// It reports any write errors back to the caller, Application.SetContentErrorHandler does NOT apply here
// as this is a lower-level method which must be remain as it is.
func (ctx *Context) Write(rawBody []byte) (int, error) {
	return ctx.writer.Write(rawBody)
}

// Writef lives in before_generic_methods_writef.go: its declaration must come
// before every generic *Context method for go vet to key its printf fact
// correctly. That file's header explains why.

// WriteString writes a simple string to the response.
//
// Returns the number of bytes written and any write error encountered.
func (ctx *Context) WriteString(body string) (n int, err error) {
	return io.WriteString(ctx.writer, body)
}

const (
	// ContentTypeHeaderKey is the header key of "Content-Type".
	ContentTypeHeaderKey = "Content-Type"

	// LastModifiedHeaderKey is the header key of "Last-Modified".
	LastModifiedHeaderKey = "Last-Modified"
	// IfModifiedSinceHeaderKey is the header key of "If-Modified-Since".
	IfModifiedSinceHeaderKey = "If-Modified-Since"
	// CacheControlHeaderKey is the header key of "Cache-Control".
	CacheControlHeaderKey = "Cache-Control"
	// ETagHeaderKey is the header key of "ETag".
	ETagHeaderKey = "ETag"

	// ContentDispositionHeaderKey is the header key of "Content-Disposition".
	ContentDispositionHeaderKey = "Content-Disposition"
	// ContentLengthHeaderKey is the header key of "Content-Length"
	ContentLengthHeaderKey = "Content-Length"
	// ContentEncodingHeaderKey is the header key of "Content-Encoding".
	ContentEncodingHeaderKey = "Content-Encoding"
	// GzipHeaderValue is the header value of "gzip".
	GzipHeaderValue = "gzip"
	// AcceptEncodingHeaderKey is the header key of "Accept-Encoding".
	AcceptEncodingHeaderKey = "Accept-Encoding"
	// VaryHeaderKey is the header key of "Vary".
	VaryHeaderKey = "Vary"
)

var unixEpochTime = time.Unix(0, 0)

// IsZeroTime reports whether t is obviously unspecified (either zero or Unix()=0).
func IsZeroTime(t time.Time) bool {
	return t.IsZero() || t.Equal(unixEpochTime)
}

// ParseTime parses a time header (such as the Date: header),
// trying each forth formats (or three if Application's configuration's TimeFormat is defaulted)
// that are allowed by HTTP/1.1:
// Application's configuration's TimeFormat or/and http.TimeFormat,
// time.RFC850, and time.ANSIC.
//
// Look `context#FormatTime` for the opossite operation (Time to string).
var ParseTime = func(ctx *Context, text string) (t time.Time, err error) {
	t, err = time.Parse(ctx.OptionsReadOnly().TimeFormat, text)
	if err != nil {
		return http.ParseTime(text)
	}

	return
}

// FormatTime returns a textual representation of the time value formatted
// according to the Application's configuration's TimeFormat field
// which defines the format.
//
// Look `context#ParseTime` for the opossite operation (string to Time).
var FormatTime = func(ctx *Context, t time.Time) string {
	return t.Format(ctx.OptionsReadOnly().TimeFormat)
}

// SetLastModified sets the "Last-Modified" based on the "modtime" input.
// If "modtime" is zero then it does nothing.
//
// It's mostly internally on core/router and context packages.
func (ctx *Context) SetLastModified(modtime time.Time) {
	if !IsZeroTime(modtime) {
		ctx.SetHeader(LastModifiedHeaderKey, FormatTime(ctx, modtime.UTC())) // or modtime.UTC()?
	}
}

// ErrPreconditionFailed may be returned from `Context` methods
// that has to perform one or more client side preconditions before the actual check, e.g. `CheckIfModifiedSince`.
// Usage:
// ok, err := context.CheckIfModifiedSince(modTime)
//
//	if err != nil {
//	   if errors.Is(err, context.ErrPreconditionFailed) {
//	        [handle missing client conditions,such as not valid request method...]
//	    }else {
//	        [the error is probably a time parse error...]
//	   }
//	}
var ErrPreconditionFailed = errors.New("precondition failed")

// methodQuery mirrors router.MethodQuery (RFC 10008); the router package imports this one.
const methodQuery = "QUERY"

// CheckIfModifiedSince checks if the response is modified since the "modtime".
// Note that it has nothing to do with server-side caching.
// It does those checks by checking if the "If-Modified-Since" request header
// sent by client or a previous server response header
// (e.g with WriteWithExpiration)
// is a valid one and it's before the "modtime".
//
// A check for !modtime && err == nil is necessary to make sure that
// it's not modified since, because it may return false but without even
// had the chance to check the client-side (request) header due to some errors,
// like the HTTP Method is not "GET", "HEAD" or "QUERY" or if the "modtime" is zero
// or if parsing time from the header failed. See `ErrPreconditionFailed` too.
//
// It's mostly used internally, e.g. `context#WriteWithExpiration`.
func (ctx *Context) CheckIfModifiedSince(modtime time.Time) (bool, error) {
	if method := ctx.Method(); method != http.MethodGet && method != http.MethodHead && method != methodQuery {
		return false, fmt.Errorf("method: %s: %w", method, ErrPreconditionFailed)
	}
	ims := ctx.Header(IfModifiedSinceHeaderKey)
	if ims == "" || IsZeroTime(modtime) {
		return false, fmt.Errorf("zero time: %w", ErrPreconditionFailed)
	}
	t, err := ParseTime(ctx, ims)
	if err != nil {
		return false, err
	}
	// sub-second precision, so
	// use mtime < t+1s instead of mtime <= t to check for unmodified.
	if modtime.UTC().Before(t.Add(1 * time.Second)) {
		return false, nil
	}
	return true, nil
}

// WriteNotModified sends a 304 "Not Modified" status code to the client,
// it makes sure that the content type and the content length headers are removed
// before the response sent, and that the "Last-Modified" one is removed as well
// when an "ETag" is present (RFC 7232 section 4.1). The "ETag" itself is kept.
//
// It's mostly used internally on core/router/fs.go and context methods.
func (ctx *Context) WriteNotModified() {
	// RFC 7232 section 4.1:
	// a sender SHOULD NOT generate representation metadata other than the
	// above listed fields unless said metadata exists for the purpose of
	// guiding cache updates (e.g.," Last-Modified" might be useful if the
	// response does not have an ETag field).
	h := ctx.ResponseWriter().Header()
	delete(h, ContentTypeHeaderKey)
	delete(h, ContentLengthHeaderKey)
	if h.Get(ETagHeaderKey) != "" {
		delete(h, LastModifiedHeaderKey)
	}
	ctx.StatusCode(http.StatusNotModified)
}

// WriteWithExpiration works like `Write` but it will check if a resource is modified,
// based on the "modtime" input argument,
// otherwise sends a 304 status code in order to let the client-side render the cached content.
func (ctx *Context) WriteWithExpiration(body []byte, modtime time.Time) (int, error) {
	if modified, err := ctx.CheckIfModifiedSince(modtime); !modified && err == nil {
		ctx.WriteNotModified()
		return 0, nil
	}

	ctx.SetLastModified(modtime)
	return ctx.writer.Write(body)
}

// StreamWriter registers the given stream writer for populating
// response body.
//
// Access to context's and/or its' members is forbidden from writer.
//
// The writer is called repeatedly until it returns an error. io.EOF is the
// sentinel for normal completion: there is no other way to end the loop, so a
// writer that has nothing more to send returns it and StreamWriter passes it
// back unchanged. Any other error ends the stream and is returned as-is, and a
// client that disconnects ends it with the request context's error.
//
// This function may be used in the following cases:
//
//   - if response body is too big (more than bodylimit.New(if set)).
//   - if response body is streamed from slow external sources.
//   - if response body must be streamed to the client in chunks.
//     (aka `http server push`).
//
// For Server-Sent Events prefer the sse package. This is a poll loop, which
// suits a writer that blocks until it has something (a ticker, a queue read),
// and fits badly when events arrive from a producer goroutine: sse.Stream is a
// push API, and it also handles the response-writer buffering, write deadlines,
// heartbeats and framing that an event stream needs.
func (ctx *Context) StreamWriter(writer func(w io.Writer) error) error {
	cancelCtx := ctx.Request().Context()
	notifyClosed := cancelCtx.Done()

	for {
		select {
		// response writer forced to close, exit.
		case <-notifyClosed:
			return cancelCtx.Err()
		default:
		}

		if err := writer(ctx.writer); err != nil {
			return err
		}

		// A failed flush is the earliest sign the client has gone: nothing else
		// reports it until the kernel gives up retransmitting, which on Linux
		// takes about fifteen minutes by default. ErrNotSupported is not a
		// failure, it means this response cannot flush at all - which is what
		// happens under Timeout.HandlerTimeout - so the loop carries on as it
		// did before.
		if err := FlushResponseWriter(ctx.writer); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return err
		}

		// Check again after the write, not only before it. A writer that returns
		// without blocking would otherwise run the whole generation before
		// noticing that nobody is reading it.
		select {
		case <-notifyClosed:
			return cancelCtx.Err()
		default:
		}
	}
}

//  +------------------------------------------------------------+
//  | Body Writers with compression                              |
//  +------------------------------------------------------------+

// ClientSupportsEncoding reports whether the
// client expects one of the given "encodings" compression.
//
// Note, this method just reports back the first valid encoding it sees,
// meaning that request accept-encoding offers don't matter here.
// See the `middleware/compress` middleware to actually compress the response.
func (ctx *Context) ClientSupportsEncoding(encodings ...string) bool {
	if len(encodings) == 0 {
		return false
	}

	if h := ctx.Header(AcceptEncodingHeaderKey); h != "" {
		for v := range strings.SplitSeq(h, ",") {
			for _, encoding := range encodings {
				if strings.Contains(v, encoding) {
					return true
				}
			}
		}
	}

	return false
}
