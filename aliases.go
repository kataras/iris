package iris

import (
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/kataras/iris/v12/cache"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/handlerconv"
	"github.com/kataras/iris/v12/core/host"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/hero"
	"github.com/kataras/iris/v12/view"
)

var (
	// BuildRevision holds the vcs commit id information of the program's build.
	// To display the Iris' version please use the iris.Version constant instead.
	// Available at go version 1.18+
	BuildRevision = context.BuildRevision
	// BuildTime holds the vcs commit time information of the program's build.
	// Available at go version 1.18+
	BuildTime = context.BuildTime
)

// SameSite attributes.
const (
	SameSiteDefaultMode = http.SameSiteDefaultMode
	SameSiteLaxMode     = http.SameSiteLaxMode
	SameSiteStrictMode  = http.SameSiteStrictMode
	SameSiteNoneMode    = http.SameSiteNoneMode
)

type (
	// Context is the middle-man server's "object" for the clients.
	//
	// A New context is being acquired from a sync.Pool on each connection.
	// The Context is the most important thing on the iris's http flow.
	//
	// Developers send responses to the client's request through a Context.
	// Developers get request information from the client's request by a Context.
	Context = *context.Context
	// ViewEngine is an alias of `context.ViewEngine`.
	// See HTML, Blocks, Django, Jet, Pug, Ace, Handlebars and e.t.c.
	ViewEngine = context.ViewEngine
	// UnmarshalerFunc a shortcut, an alias for the `context#UnmarshalerFunc` type
	// which implements the `context#Unmarshaler` interface for reading request's body
	// via custom decoders, most of them already implement the `context#UnmarshalerFunc`
	// like the json.Unmarshal, xml.Unmarshal, yaml.Unmarshal and every library which
	// follows the best practises and is aligned with the Go standards.
	//
	// See 'context#UnmarshalBody` for more.
	//
	// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-custom-via-unmarshaler/main.go
	UnmarshalerFunc = context.UnmarshalerFunc
	// DecodeFunc is a generic type of decoder function.
	// When the returned error is not nil the decode operation
	// is terminated and the error is received by the ReadJSONStream method,
	// otherwise it continues to read the next available object.
	// Look the `Context.ReadJSONStream` method.
	//
	// Example: https://github.com/kataras/iris/blob/main/_examples/request-body/read-json-stream.
	DecodeFunc = context.DecodeFunc
	// A Handler responds to an HTTP request.
	// It writes reply headers and data to the Context.ResponseWriter() and then return.
	// Returning signals that the request is finished;
	// it is not valid to use the Context after or concurrently with the completion of the Handler call.
	//
	// Depending on the HTTP client software, HTTP protocol version,
	// and any intermediaries between the client and the iris server,
	// it may not be possible to read from the Context.Request().Body after writing to the context.ResponseWriter().
	// Cautious handlers should read the Context.Request().Body first, and then reply.
	//
	// Except for reading the body, handlers should not modify the provided Context.
	//
	// If Handler panics, the server (the caller of Handler) assumes that the effect of the panic was isolated to the active request.
	// It recovers the panic, logs a stack trace to the server error log, and hangs up the connection.
	Handler = context.Handler
	// Filter is just a type of func(Context) bool which reports whether an action must be performed
	// based on the incoming request.
	//
	// See `NewConditionalHandler` for more.
	// An alias for the `context/Filter`.
	Filter = context.Filter
	// A Map is an alias of map[string]interface{}.
	Map = context.Map
	// User is a generic view of an authorized client.
	// See `Context.User` and `SetUser` methods for more.
	// An alias for the `context/User` type.
	User = context.User
	// SimpleUser is a simple implementation of the User interface.
	SimpleUser = context.SimpleUser
	// Problem Details for HTTP APIs.
	// Pass a Problem value to `context.Problem` to
	// write an "application/problem+json" response.
	//
	// Read more at: https://github.com/kataras/iris/blob/main/_examples/routing/http-errors.
	//
	// It is an alias of the `context#Problem` type.
	Problem = context.Problem
	// ProblemOptions the optional settings when server replies with a Problem.
	// See `Context.Problem` method and `Problem` type for more details.
	//
	// It is an alias of the `context#ProblemOptions` type.
	ProblemOptions = context.ProblemOptions
	// JSON the optional settings for JSON renderer.
	//
	// It is an alias of the `context#JSON` type.
	JSON = context.JSON
	// JSONReader holds the JSON decode options of the `Context.ReadJSON, ReadBody` methods.
	//
	// It is an alias of the `context#JSONReader` type.
	JSONReader = context.JSONReader
	// JSONP the optional settings for JSONP renderer.
	//
	// It is an alias of the `context#JSONP` type.
	JSONP = context.JSONP
	// ProtoMarshalOptions is a type alias for protojson.MarshalOptions.
	ProtoMarshalOptions = context.ProtoMarshalOptions
	// ProtoUnmarshalOptions is a type alias for protojson.UnmarshalOptions.
	ProtoUnmarshalOptions = context.ProtoUnmarshalOptions
	// XML the optional settings for XML renderer.
	//
	// It is an alias of the `context#XML` type.
	XML = context.XML
	// Markdown the optional settings for Markdown renderer.
	// See `Context.Markdown` for more.
	//
	// It is an alias of the `context#Markdown` type.
	Markdown = context.Markdown
	// Supervisor is a shortcut of the `host#Supervisor`.
	// Used to add supervisor configurators on common Runners
	// without the need of importing the `core/host` package.
	Supervisor = host.Supervisor

	// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
	// Party could also be named as 'Join' or 'Node' or 'Group' , Party chosen because it is fun.
	//
	// Look the `core/router#APIBuilder` for its implementation.
	//
	// A shortcut for the `core/router#Party`, useful when `PartyFunc` is being used.
	Party = router.Party
	// APIContainer is a wrapper of a common `Party` featured by Dependency Injection.
	// See `Party.ConfigureContainer` for more.
	//
	// A shortcut for the `core/router#APIContainer`.
	APIContainer = router.APIContainer
	// ResultHandler describes the function type which should serve the "v" struct value.
	// See `APIContainer.UseResultHandler`.
	ResultHandler = hero.ResultHandler

	// DirOptions contains the optional settings that
	// `FileServer` and `Party#HandleDir` can use to serve files and assets.
	// A shortcut for the `router.DirOptions`, useful when `FileServer` or `HandleDir` is being used.
	DirOptions = router.DirOptions
	// DirCacheOptions holds the options for the cached file system.
	// See `DirOptions`.
	DirCacheOptions = router.DirCacheOptions
	// DirListRichOptions the options for the `DirListRich` helper function.
	// A shortcut for the `router.DirListRichOptions`.
	// Useful when `DirListRich` function is passed to `DirOptions.DirList` field.
	DirListRichOptions = router.DirListRichOptions
	// Attachments options for files to be downloaded and saved locally by the client.
	// See `DirOptions`.
	Attachments = router.Attachments
	// Dir implements FileSystem using the native file system restricted to a
	// specific directory tree, can be passed to the `FileServer` function
	// and `HandleDir` method. It's an alias of `http.Dir`.
	Dir = http.Dir

	// ExecutionRules gives control to the execution of the route handlers outside of the handlers themselves.
	// Usage:
	// Party#SetExecutionRules(ExecutionRules {
	//   Done: ExecutionOptions{Force: true},
	// })
	//
	// See `core/router/Party#SetExecutionRules` for more.
	// Example: https://github.com/kataras/iris/tree/main/_examples/mvc/middleware/without-ctx-next
	ExecutionRules = router.ExecutionRules
	// ExecutionOptions is a set of default behaviors that can be changed in order to customize the execution flow of the routes' handlers with ease.
	//
	// See `ExecutionRules` and `core/router/Party#SetExecutionRules` for more.
	ExecutionOptions = router.ExecutionOptions

	// CookieOption is the type of function that is accepted on
	// context's methods like `SetCookieKV`, `RemoveCookie` and `SetCookie`
	// as their (last) variadic input argument to amend the end cookie's form.
	//
	// Any custom or builtin `CookieOption` is valid,
	// see `CookiePath`, `CookieCleanPath`, `CookieExpires` and `CookieHTTPOnly` for more.
	//
	// An alias for the `context.CookieOption`.
	CookieOption = context.CookieOption
	// Cookie is a type alias for the standard net/http Cookie struct type.
	// See `Context.SetCookie`.
	Cookie = http.Cookie
	// N is a struct which can be passed on the `Context.Negotiate` method.
	// It contains fields which should be filled based on the `Context.Negotiation()`
	// server side values. If no matched mime then its "Other" field will be sent,
	// which should be a string or []byte.
	// It completes the `context/context.ContentSelector` interface.
	//
	// An alias for the `context.N`.
	N = context.N
	// Locale describes the i18n locale.
	// An alias for the `context.Locale`.
	Locale = context.Locale
	// ErrPrivate if provided then the error saved in context
	// should NOT be visible to the client no matter what.
	// An alias for the `context.ErrPrivate`.
	ErrPrivate = context.ErrPrivate
)

// Constants for input argument at `router.RouteRegisterRule`.
// See `Party#SetRegisterRule`.
const (
	// RouteOverride replaces an existing route with the new one, the default rule.
	RouteOverride = router.RouteOverride
	// RouteSkip keeps the original route and skips the new one.
	RouteSkip = router.RouteSkip
	// RouteError log when a route already exists, shown after the `Build` state,
	// server never starts.
	RouteError = router.RouteError
	// RouteOverlap will overlap the new route to the previous one.
	// If the route stopped and its response can be reset then the new route will be execute.
	RouteOverlap = router.RouteOverlap
)

// Contains the enum values of the `Context.GetReferrer()` method,
// shortcuts of the context subpackage.
const (
	ReferrerInvalid  = context.ReferrerInvalid
	ReferrerIndirect = context.ReferrerIndirect
	ReferrerDirect   = context.ReferrerDirect
	ReferrerEmail    = context.ReferrerEmail
	ReferrerSearch   = context.ReferrerSearch
	ReferrerSocial   = context.ReferrerSocial

	ReferrerNotGoogleSearch     = context.ReferrerNotGoogleSearch
	ReferrerGoogleOrganicSearch = context.ReferrerGoogleOrganicSearch
	ReferrerGoogleAdwords       = context.ReferrerGoogleAdwords
)

// NoLayout to disable layout for a particular template file
// A shortcut for the `view#NoLayout`.
const NoLayout = view.NoLayout

var (
	// HTML view engine.
	// Shortcut of the view.HTML.
	HTML = view.HTML
	// Blocks view engine.
	// Can be used as a faster alternative of the HTML engine.
	// Shortcut of the view.Blocks.
	Blocks = view.Blocks
	// Django view engine.
	// Shortcut of the view.Django.
	Django = view.Django
	// Handlebars view engine.
	// Shortcut of the view.Handlebars.
	Handlebars = view.Handlebars
	// Pug view engine.
	// Shortcut of the view.Pug.
	Pug = view.Pug
	// Jet view engine.
	// Shortcut of the view.Jet.
	Jet = view.Jet
	// Ace view engine.
	// Shortcut of the view.Ace.
	Ace = view.Ace
)

type (
	// ErrViewNotExist reports whether a template was not found in the parsed templates tree.
	ErrViewNotExist = context.ErrViewNotExist
	// FallbackViewFunc is a function that can be registered
	// to handle view fallbacks. It accepts the Context and
	// a special error which contains information about the previous template error.
	// It implements the FallbackViewProvider interface.
	//
	// See `Context.View` method.
	FallbackViewFunc = context.FallbackViewFunc
	// FallbackView is a helper to register a single template filename as a fallback
	// when the provided tempate filename was not found.
	FallbackView = context.FallbackView
	// FallbackViewLayout is a helper to register a single template filename as a fallback
	// layout when the provided layout filename was not found.
	FallbackViewLayout = context.FallbackViewLayout
)

// Component returns a new Handler which can be registered as a main handler for a route.
// It's a shortcut handler that renders the given component as HTML through Context.RenderComponent.
func Component(component context.Component) Handler {
	return func(ctx Context) {
		ctx.RenderComponent(component)
	}
}

// PrefixDir returns a new FileSystem that opens files
// by adding the given "prefix" to the directory tree of "fs".
//
// Useful when having templates and static files in the same
// bindata AssetFile method. This way you can select
// which one to serve as static files and what for templates.
// All view engines have a `RootDir` method for that reason too
// but alternatively, you can wrap the given file system with this `PrefixDir`.
//
// Example: https://github.com/kataras/iris/blob/main/_examples/file-server/single-page-application/embedded-single-page-application/main.go
func PrefixDir(prefix string, fs http.FileSystem) http.FileSystem {
	return &prefixedDir{prefix, fs}
}

// PrefixFS same as "PrefixDir" but for `fs.FS` type.
func PrefixFS(fileSystem fs.FS, dir string) (fs.FS, error) {
	return fs.Sub(fileSystem, dir)
}

type prefixedDir struct {
	prefix string
	fs     http.FileSystem
}

func (p *prefixedDir) Open(name string) (http.File, error) {
	// Don't do this: as this is responsibility of the underline fs.
	// _, filename, ok, err := context.SafeFilename("", name)

	destPath := path.Join(p.prefix, name)
	return p.fs.Open(destPath)
}

type partyConfiguratorMiddleware struct {
	handlers []Handler
}

func (p *partyConfiguratorMiddleware) Configure(r Party) {
	r.Use(p.handlers...)
}

// ConfigureMiddleware is a PartyConfigurator which can be used
// as a shortcut to add middlewares on Party.PartyConfigure("/path", WithMiddleware(handler), new(example.API)).
func ConfigureMiddleware(handlers ...Handler) router.PartyConfigurator {
	return &partyConfiguratorMiddleware{handlers: handlers}
}

// Compression is a middleware which enables
// writing and reading using the best offered compression.
// Usage:
// app.Use (for matched routes)
// app.UseRouter (for both matched and 404s or other HTTP errors).
func Compression(ctx Context) {
	ctx.CompressWriter(true)
	ctx.CompressReader(true)
	ctx.Next()
}

var (
	// AllowQuerySemicolons returns a middleware that serves requests by converting any
	// unescaped semicolons(;) in the URL query to ampersands(&).
	//
	// This restores the pre-Go 1.17 behavior of splitting query parameters on both
	// semicolons and ampersands.
	// (See golang.org/issue/25192 and https://github.com/kataras/iris/issues/1875).
	// Note that this behavior doesn't match that of many proxies,
	// and the mismatch can lead to security issues.
	//
	// AllowQuerySemicolons should be invoked before any Context read query or
	// form methods are called.
	//
	// To skip HTTP Server logging for this type of warning:
	// app.Listen/Run(..., iris.WithoutServerError(iris.ErrURLQuerySemicolon)).
	AllowQuerySemicolons = func(ctx Context) {
		// clopy of net/http.AllowQuerySemicolons.
		r := ctx.Request()
		if s := r.URL.RawQuery; strings.Contains(s, ";") {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.RawQuery = strings.ReplaceAll(s, ";", "&")
			ctx.ResetRequest(r2)
		}

		ctx.Next()
	}

	// MatchImagesAssets is a simple regex expression
	// that can be passed to the DirOptions.Cache.CompressIgnore field
	// in order to skip compression on already-compressed file types
	// such as images and pdf.
	MatchImagesAssets = regexp.MustCompile("((.*).pdf|(.*).jpg|(.*).jpeg|(.*).gif|(.*).tif|(.*).tiff)$")
	// MatchCommonAssets is a simple regex expression which
	// can be used on `DirOptions.PushTargetsRegexp`.
	// It will match and Push
	// all available js, css, font and media files.
	// Ideal for Single Page Applications.
	MatchCommonAssets = regexp.MustCompile("((.*).js|(.*).css|(.*).ico|(.*).png|(.*).ttf|(.*).svg|(.*).webp|(.*).gif)$")
)

var (
	// RegisterOnInterrupt registers a global function to call when CTRL+C/CMD+C pressed or a unix kill command received.
	//
	// A shortcut for the `host#RegisterOnInterrupt`.
	RegisterOnInterrupt = host.RegisterOnInterrupt

	// LimitRequestBodySize is a middleware which sets a request body size limit
	// for all next handlers in the chain.
	//
	// A shortcut for the `context#LimitRequestBodySize`.
	LimitRequestBodySize = context.LimitRequestBodySize
	// NewConditionalHandler returns a single Handler which can be registered
	// as a middleware.
	// Filter is just a type of Handler which returns a boolean.
	// Handlers here should act like middleware, they should contain `ctx.Next` to proceed
	// to the next handler of the chain. Those "handlers" are registered to the per-request context.
	//
	//
	// It checks the "filter" and if passed then
	// it, correctly, executes the "handlers".
	//
	// If passed, this function makes sure that the Context's information
	// about its per-request handler chain based on the new "handlers" is always updated.
	//
	// If not passed, then simply the Next handler(if any) is executed and "handlers" are ignored.
	// Example can be found at: _examples/routing/conditional-chain.
	//
	// A shortcut for the `context#NewConditionalHandler`.
	NewConditionalHandler = context.NewConditionalHandler
	// FileServer returns a Handler which serves files from a specific system, phyisical, directory
	// or an embedded one.
	// The first parameter is the directory, relative to the executable program.
	// The second optional parameter is any optional settings that the caller can use.
	//
	// See `Party#HandleDir` too.
	// Examples can be found at: https://github.com/kataras/iris/tree/main/_examples/file-server
	// A shortcut for the `router.FileServer`.
	FileServer = router.FileServer
	// DirList is the default `DirOptions.DirList` field.
	// Read more at: `core/router.DirList`.
	DirList = router.DirList
	// DirListRich can be passed to `DirOptions.DirList` field
	// to override the default file listing appearance.
	// Read more at: `core/router.DirListRich`.
	DirListRich = router.DirListRich
	// StripPrefix returns a handler that serves HTTP requests
	// by removing the given prefix from the request URL's Path
	// and invoking the handler h. StripPrefix handles a
	// request for a path that doesn't begin with prefix by
	// replying with an HTTP 404 not found error.
	//
	// Usage:
	// fileserver := iris.FileServer("./static_files", DirOptions {...})
	// h := iris.StripPrefix("/static", fileserver)
	// app.Get("/static/{file:path}", h)
	// app.Head("/static/{file:path}", h)
	StripPrefix = router.StripPrefix
	// FromStd converts native http.Handler, http.HandlerFunc & func(w, r, next) to context.Handler.
	//
	// Supported form types:
	// 		 .FromStd(h http.Handler)
	// 		 .FromStd(func(w http.ResponseWriter, r *http.Request))
	// 		 .FromStd(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc))
	//
	// A shortcut for the `handlerconv#FromStd`.
	FromStd = handlerconv.FromStd
	// Cache is a middleware providing server-side cache functionalities
	// to the next handlers, can be used as: `app.Get("/", iris.Cache, aboutHandler)`.
	// It should be used after Static methods.
	// See `iris#Cache304` for an alternative, faster way.
	//
	// Examples can be found at: https://github.com/kataras/iris/tree/main/_examples/#caching
	Cache = cache.Handler
	// NoCache is a middleware which overrides the Cache-Control, Pragma and Expires headers
	// in order to disable the cache during the browser's back and forward feature.
	//
	// A good use of this middleware is on HTML routes; to refresh the page even on "back" and "forward" browser's arrow buttons.
	//
	// See `iris#StaticCache` for the opposite behavior.
	//
	// A shortcut of the `cache#NoCache`
	NoCache = cache.NoCache
	// StaticCache middleware for caching static files by sending the "Cache-Control" and "Expires" headers to the client.
	// It accepts a single input parameter, the "cacheDur", a time.Duration that it's used to calculate the expiration.
	//
	// If "cacheDur" <=0 then it returns the `NoCache` middleware instaed to disable the caching between browser's "back" and "forward" actions.
	//
	// Usage: `app.Use(iris.StaticCache(24 * time.Hour))` or `app.Use(iris.StaticCache(-1))`.
	// A middleware, which is a simple Handler can be called inside another handler as well, example:
	// cacheMiddleware := iris.StaticCache(...)
	// func(ctx iris.Context){
	//  cacheMiddleware(ctx)
	//  [...]
	// }
	//
	// A shortcut of the `cache#StaticCache`
	StaticCache = cache.StaticCache
	// Cache304 sends a `StatusNotModified` (304) whenever
	// the "If-Modified-Since" request header (time) is before the
	// time.Now() + expiresEvery (always compared to their UTC values).
	// Use this, which is a shortcut of the, `chache#Cache304` instead of the "github.com/kataras/iris/v12/cache" or iris.Cache
	// for better performance.
	// Clients that are compatible with the http RCF (all browsers are and tools like postman)
	// will handle the caching.
	// The only disadvantage of using that instead of server-side caching
	// is that this method will send a 304 status code instead of 200,
	// So, if you use it side by side with other micro services
	// you have to check for that status code as well for a valid response.
	//
	// Developers are free to extend this method's behavior
	// by watching system directories changes manually and use of the `ctx.WriteWithExpiration`
	// with a "modtime" based on the file modified date,
	// similar to the `HandleDir`(which sends status OK(200) and browser disk caching instead of 304).
	//
	// A shortcut of the `cache#Cache304`.
	Cache304 = cache.Cache304

	// CookieOverride is a CookieOption which overrides the cookie explicitly to the given "cookie".
	//
	// A shortcut for the `context#CookieOverride`.
	CookieOverride = context.CookieOverride
	// CookieDomain is a CookieOption which sets the cookie's Domain field.
	// If empty then the current domain is used.
	//
	// A shortcut for the `context#CookieDomain`.
	CookieDomain = context.CookieDomain
	// CookieAllowReclaim accepts the Context itself.
	// If set it will add the cookie to (on `CookieSet`, `CookieSetKV`, `CookieUpsert`)
	// or remove the cookie from (on `CookieRemove`) the Request object too.
	//
	// A shortcut for the `context#CookieAllowReclaim`.
	CookieAllowReclaim = context.CookieAllowReclaim
	// CookieAllowSubdomains set to the Cookie Options
	// in order to allow subdomains to have access to the cookies.
	// It sets the cookie's Domain field (if was empty) and
	// it also sets the cookie's SameSite to lax mode too.
	//
	// A shortcut for the `context#CookieAllowSubdomains`.
	CookieAllowSubdomains = context.CookieAllowSubdomains
	// CookieSameSite sets a same-site rule for cookies to set.
	// SameSite allows a server to define a cookie attribute making it impossible for
	// the browser to send this cookie along with cross-site requests. The main
	// goal is to mitigate the risk of cross-origin information leakage, and provide
	// some protection against cross-site request forgery attacks.
	//
	// See https://tools.ietf.org/html/draft-ietf-httpbis-cookie-same-site-00 for details.
	//
	// A shortcut for the `context#CookieSameSite`.
	CookieSameSite = context.CookieSameSite
	// CookieSecure sets the cookie's Secure option if the current request's
	// connection is using TLS. See `CookieHTTPOnly` too.
	//
	// A shortcut for the `context#CookieSecure`.
	CookieSecure = context.CookieSecure
	// CookieHTTPOnly is a `CookieOption`.
	// Use it to set the cookie's HttpOnly field to false or true.
	// HttpOnly field defaults to true for `RemoveCookie` and `SetCookieKV`.
	//
	// A shortcut for the `context#CookieHTTPOnly`.
	CookieHTTPOnly = context.CookieHTTPOnly
	// CookiePath is a `CookieOption`.
	// Use it to change the cookie's Path field.
	//
	// A shortcut for the `context#CookiePath`.
	CookiePath = context.CookiePath
	// CookieCleanPath is a `CookieOption`.
	// Use it to clear the cookie's Path field, exactly the same as `CookiePath("")`.
	//
	// A shortcut for the `context#CookieCleanPath`.
	CookieCleanPath = context.CookieCleanPath
	// CookieExpires is a `CookieOption`.
	// Use it to change the cookie's Expires and MaxAge fields by passing the lifetime of the cookie.
	//
	// A shortcut for the `context#CookieExpires`.
	CookieExpires = context.CookieExpires
	// CookieEncoding accepts a value which implements `Encode` and `Decode` methods.
	// It calls its `Encode` on `Context.SetCookie, UpsertCookie, and SetCookieKV` methods.
	// And on `Context.GetCookie` method it calls its `Decode`.
	//
	// A shortcut for the `context#CookieEncoding`.
	CookieEncoding = context.CookieEncoding

	// IsErrEmptyJSON reports whether the given "err" is caused by a
	// Context.ReadJSON call when the request body
	// didn't start with { or it was totally empty.
	IsErrEmptyJSON = context.IsErrEmptyJSON
	// IsErrPath can be used at `context#ReadForm` and `context#ReadQuery`.
	// It reports whether the incoming error is type of `schema.ErrPath`,
	// which can be ignored when server allows unknown post values to be sent by the client.
	//
	// A shortcut for the `context#IsErrPath`.
	IsErrPath = context.IsErrPath
	// IsErrCanceled reports whether the "err" is caused by a cancellation or timeout.
	//
	// A shortcut for the `context#IsErrCanceled`.
	IsErrCanceled = context.IsErrCanceled
	// ErrEmptyForm is the type error which API users can make use of
	// to check if a form was empty on `Context.ReadForm`.
	//
	// A shortcut for the `context#ErrEmptyForm`.
	ErrEmptyForm = context.ErrEmptyForm
	// ErrEmptyFormField reports whether if form value is empty.
	// An alias of `context.ErrEmptyFormField`.
	ErrEmptyFormField = context.ErrEmptyFormField
	// ErrNotFound reports whether a key was not found, useful
	// on post data, versioning feature and others.
	// An alias of `context.ErrNotFound`.
	ErrNotFound = context.ErrNotFound
	// NewProblem returns a new Problem.
	// Head over to the `Problem` type godoc for more.
	//
	// A shortcut for the `context#NewProblem`.
	NewProblem = context.NewProblem
	// XMLMap wraps a map[string]interface{} to compatible xml marshaler,
	// in order to be able to render maps as XML on the `Context.XML` method.
	//
	// Example: `Context.XML(XMLMap("Root", map[string]interface{}{...})`.
	//
	// A shortcut for the `context#XMLMap`.
	XMLMap = context.XMLMap
	// ErrStopExecution if returned from a hero middleware or a request-scope dependency
	// stops the handler's execution, see _examples/dependency-injection/basic/middleware.
	ErrStopExecution = hero.ErrStopExecution
	// ErrHijackNotSupported is returned by the Hijack method to
	// indicate that Hijack feature is not available.
	//
	// A shortcut for the `context#ErrHijackNotSupported`.
	ErrHijackNotSupported = context.ErrHijackNotSupported
	// ErrPushNotSupported is returned by the Push method to
	// indicate that HTTP/2 Push support is not available.
	//
	// A shortcut for the `context#ErrPushNotSupported`.
	ErrPushNotSupported = context.ErrPushNotSupported
	// PrivateError accepts an error and returns a wrapped private one.
	// A shortcut for the `context#PrivateError` function.
	PrivateError = context.PrivateError

	// TrimParamFilePart is a middleware which trims any last part after a dot (.) character
	// of the current route's dynamic path parameters.
	// A shortcut for the `context#TrimParamFilePart` function.
	TrimParamFilePart Handler = context.TrimParamFilePart
)

// HTTP Methods copied from `net/http`.
const (
	MethodGet     = http.MethodGet
	MethodPost    = http.MethodPost
	MethodPut     = http.MethodPut
	MethodDelete  = http.MethodDelete
	MethodConnect = http.MethodConnect
	MethodHead    = http.MethodHead
	MethodPatch   = http.MethodPatch
	MethodOptions = http.MethodOptions
	MethodTrace   = http.MethodTrace
	// MethodNone is an iris-specific "virtual" method
	// to store the "offline" routes.
	MethodNone = router.MethodNone
)

// HTTP status codes as registered with IANA.
// See: http://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml.
// Raw Copy from the future(tip) net/http std package in order to recude the import path of "net/http" for the users.
const (
	StatusContinue           = http.StatusContinue           // RFC 7231, 6.2.1
	StatusSwitchingProtocols = http.StatusSwitchingProtocols // RFC 7231, 6.2.2
	StatusProcessing         = http.StatusProcessing         // RFC 2518, 10.1
	StatusEarlyHints         = http.StatusEarlyHints         // RFC 8297

	StatusOK                   = http.StatusOK                   // RFC 7231, 6.3.1
	StatusCreated              = http.StatusCreated              // RFC 7231, 6.3.2
	StatusAccepted             = http.StatusAccepted             // RFC 7231, 6.3.3
	StatusNonAuthoritativeInfo = http.StatusNonAuthoritativeInfo // RFC 7231, 6.3.4
	StatusNoContent            = http.StatusNoContent            // RFC 7231, 6.3.5
	StatusResetContent         = http.StatusResetContent         // RFC 7231, 6.3.6
	StatusPartialContent       = http.StatusPartialContent       // RFC 7233, 4.1
	StatusMultiStatus          = http.StatusMultiStatus          // RFC 4918, 11.1
	StatusAlreadyReported      = http.StatusAlreadyReported      // RFC 5842, 7.1
	StatusIMUsed               = http.StatusIMUsed               // RFC 3229, 10.4.1

	StatusMultipleChoices   = http.StatusMultipleChoices   // RFC 7231, 6.4.1
	StatusMovedPermanently  = http.StatusMovedPermanently  // RFC 7231, 6.4.2
	StatusFound             = http.StatusFound             // RFC 7231, 6.4.3
	StatusSeeOther          = http.StatusSeeOther          // RFC 7231, 6.4.4
	StatusNotModified       = http.StatusNotModified       // RFC 7232, 4.1
	StatusUseProxy          = http.StatusUseProxy          // RFC 7231, 6.4.5
	_                       = 306                          // RFC 7231, 6.4.6 (Unused)
	StatusTemporaryRedirect = http.StatusTemporaryRedirect // RFC 7231, 6.4.7
	StatusPermanentRedirect = http.StatusPermanentRedirect // RFC 7538, 3

	StatusBadRequest                   = http.StatusBadRequest                   // RFC 7231, 6.5.1
	StatusUnauthorized                 = http.StatusUnauthorized                 // RFC 7235, 3.1
	StatusPaymentRequired              = http.StatusPaymentRequired              // RFC 7231, 6.5.2
	StatusForbidden                    = http.StatusForbidden                    // RFC 7231, 6.5.3
	StatusNotFound                     = http.StatusNotFound                     // RFC 7231, 6.5.4
	StatusMethodNotAllowed             = http.StatusMethodNotAllowed             // RFC 7231, 6.5.5
	StatusNotAcceptable                = http.StatusNotAcceptable                // RFC 7231, 6.5.6
	StatusProxyAuthRequired            = http.StatusProxyAuthRequired            // RFC 7235, 3.2
	StatusRequestTimeout               = http.StatusRequestTimeout               // RFC 7231, 6.5.7
	StatusConflict                     = http.StatusConflict                     // RFC 7231, 6.5.8
	StatusGone                         = http.StatusGone                         // RFC 7231, 6.5.9
	StatusLengthRequired               = http.StatusLengthRequired               // RFC 7231, 6.5.10
	StatusPreconditionFailed           = http.StatusPreconditionFailed           // RFC 7232, 4.2
	StatusRequestEntityTooLarge        = http.StatusRequestEntityTooLarge        // RFC 7231, 6.5.11
	StatusRequestURITooLong            = http.StatusRequestURITooLong            // RFC 7231, 6.5.12
	StatusUnsupportedMediaType         = http.StatusUnsupportedMediaType         // RFC 7231, 6.5.13
	StatusRequestedRangeNotSatisfiable = http.StatusRequestedRangeNotSatisfiable // RFC 7233, 4.4
	StatusExpectationFailed            = http.StatusExpectationFailed            // RFC 7231, 6.5.14
	StatusTeapot                       = http.StatusTeapot                       // RFC 7168, 2.3.3
	StatusMisdirectedRequest           = http.StatusMisdirectedRequest           // RFC 7540, 9.1.2
	StatusUnprocessableEntity          = http.StatusUnprocessableEntity          // RFC 4918, 11.2
	StatusLocked                       = http.StatusLocked                       // RFC 4918, 11.3
	StatusFailedDependency             = http.StatusFailedDependency             // RFC 4918, 11.4
	StatusTooEarly                     = http.StatusTooEarly                     // RFC 8470, 5.2.
	StatusUpgradeRequired              = http.StatusUpgradeRequired              // RFC 7231, 6.5.15
	StatusPreconditionRequired         = http.StatusPreconditionRequired         // RFC 6585, 3
	StatusTooManyRequests              = http.StatusTooManyRequests              // RFC 6585, 4
	StatusRequestHeaderFieldsTooLarge  = http.StatusRequestHeaderFieldsTooLarge  // RFC 6585, 5
	StatusUnavailableForLegalReasons   = http.StatusUnavailableForLegalReasons   // RFC 7725, 3
	// Unofficial Client Errors.
	StatusPageExpired                      = context.StatusPageExpired
	StatusBlockedByWindowsParentalControls = context.StatusBlockedByWindowsParentalControls
	StatusInvalidToken                     = context.StatusInvalidToken
	StatusTokenRequired                    = context.StatusTokenRequired
	//
	StatusInternalServerError           = http.StatusInternalServerError           // RFC 7231, 6.6.1
	StatusNotImplemented                = http.StatusNotImplemented                // RFC 7231, 6.6.2
	StatusBadGateway                    = http.StatusBadGateway                    // RFC 7231, 6.6.3
	StatusServiceUnavailable            = http.StatusServiceUnavailable            // RFC 7231, 6.6.4
	StatusGatewayTimeout                = http.StatusGatewayTimeout                // RFC 7231, 6.6.5
	StatusHTTPVersionNotSupported       = http.StatusHTTPVersionNotSupported       // RFC 7231, 6.6.6
	StatusVariantAlsoNegotiates         = http.StatusVariantAlsoNegotiates         // RFC 2295, 8.1
	StatusInsufficientStorage           = http.StatusInsufficientStorage           // RFC 4918, 11.5
	StatusLoopDetected                  = http.StatusLoopDetected                  // RFC 5842, 7.2
	StatusNotExtended                   = http.StatusNotExtended                   // RFC 2774, 7
	StatusNetworkAuthenticationRequired = http.StatusNetworkAuthenticationRequired // RFC 6585, 6
	// Unofficial Server Errors.
	StatusBandwidthLimitExceeded = context.StatusBandwidthLimitExceeded
	StatusInvalidSSLCertificate  = context.StatusInvalidSSLCertificate
	StatusSiteOverloaded         = context.StatusSiteOverloaded
	StatusSiteFrozen             = context.StatusSiteFrozen
	StatusNetworkReadTimeout     = context.StatusNetworkReadTimeout
)

var (
	// StatusText returns a text for the HTTP status code. It returns the empty
	// string if the code is unknown.
	//
	// Shortcut for core/router#StatusText.
	StatusText = context.StatusText
	// RegisterMethods adds custom http methods to the "AllMethods" list.
	// Use it on initialization of your program.
	//
	// Shortcut for core/router#RegisterMethods.
	RegisterMethods = router.RegisterMethods

	// WebDAVMethods contains a list of WebDAV HTTP Verbs.
	// Register using RegiterMethods package-level function or
	// through HandleMany party-level method.
	WebDAVMethods = []string{
		MethodGet,
		MethodHead,
		MethodPatch,
		MethodPut,
		MethodPost,
		MethodDelete,
		MethodOptions,
		MethodConnect,
		MethodTrace,
		"MKCOL",
		"COPY",
		"MOVE",
		"LOCK",
		"UNLOCK",
		"PROPFIND",
		"PROPPATCH",
		"LINK",
		"UNLINK",
		"PURGE",
		"VIEW",
	}
)

var globalPatches = &GlobalPatches{
	contextPatches: &ContextPatches{
		writers: &ContextWriterPatches{},
	},
}

// GlobalPatches is a singleton features a uniform way to apply global/package-level modifications.
//
// See the `Patches` package-level function.
type GlobalPatches struct {
	contextPatches *ContextPatches
}

// Patches returns the singleton of GlobalPatches, an easy way to modify
// global(package-level) configuration for Iris applications.
//
// See its `Context` method.
//
// Example: https://github.com/kataras/iris/blob/main/_examples/response-writer/json-third-party/main.go
func Patches() *GlobalPatches { // singleton.
	return globalPatches
}

// Context returns the available context patches.
func (p *GlobalPatches) Context() *ContextPatches {
	return p.contextPatches
}

// ContextPatches contains the available global Iris context modifications.
type ContextPatches struct {
	writers *ContextWriterPatches
}

// Writers returns the available global Iris context modifications for REST writers.
func (cp *ContextPatches) Writers() *ContextWriterPatches {
	return cp.writers
}

// GetDomain modifies the way a domain is fetched from `Context#Domain` method,
// which is used on subdomain redirect feature, i18n's language cookie for subdomain sharing
// and the rewrite middleware.
func (cp *ContextPatches) GetDomain(patchFunc func(hostport string) string) {
	context.GetDomain = patchFunc
}

// SetCookieKVExpiration modifies the default cookie expiration time on `Context#SetCookieKV` method.
func (cp *ContextPatches) SetCookieKVExpiration(patch time.Duration) {
	context.SetCookieKVExpiration = patch
}

// ResolveHTTPFS modifies the default way to resolve a filesystem by any type of value.
// It affects the Application's API Builder's `HandleDir` method.
func (cp *ContextPatches) ResolveHTTPFS(patchFunc func(fsOrDir interface{}) http.FileSystem) {
	context.ResolveHTTPFS = patchFunc
}

// ResolveHTTPFS modifies the default way to resolve a filesystem by any type of value.
// It affects the view engine's filesystem resolver.
func (cp *ContextPatches) ResolveFS(patchFunc func(fsOrDir interface{}) fs.FS) {
	context.ResolveFS = patchFunc
}

// ContextWriterPatches features the context's writers patches.
type ContextWriterPatches struct{}

// JSON sets a custom function which runs and overrides the default behavior of the `Context#JSON` method.
func (cwp *ContextWriterPatches) JSON(patchFunc func(ctx Context, v interface{}, options *JSON) error) {
	context.WriteJSON = patchFunc
}

// JSONP sets a custom function which runs and overrides the default behavior of the `Context#JSONP` method.
func (cwp *ContextWriterPatches) JSONP(patchFunc func(ctx Context, v interface{}, options *JSONP) error) {
	context.WriteJSONP = patchFunc
}

// XML sets a custom function which runs and overrides the default behavior of the `Context#XML` method.
func (cwp *ContextWriterPatches) XML(patchFunc func(ctx Context, v interface{}, options *XML) error) {
	context.WriteXML = patchFunc
}

// Markdown sets a custom function which runs and overrides the default behavior of the `Context#Markdown` method.
func (cwp *ContextWriterPatches) Markdown(patchFunc func(ctx Context, v []byte, options *Markdown) error) {
	context.WriteMarkdown = patchFunc
}

// YAML sets a custom function which runs and overrides the default behavior of the `Context#YAML` method.
func (cwp *ContextWriterPatches) YAML(patchFunc func(ctx Context, v interface{}, indentSpace int) error) {
	context.WriteYAML = patchFunc
}

// Singleton is a structure which can be used as an embedded field on
// struct/controllers that should be marked as singletons on `PartyConfigure` or `MVC` Applications.
type Singleton struct{}

// Singleton returns true as this controller is a singleton.
func (c Singleton) Singleton() bool { return true }
