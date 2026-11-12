package iris

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/buildinfo"
	"github.com/kataras/iris/v14/core/pkg/pagination"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/core/server"
	"github.com/kataras/iris/v14/internal/signalhandler"
)

var (
	// BuildRevision holds the vcs commit id information of the program's build.
	// To display the Iris' version please use the iris.Version constant instead.
	BuildRevision = buildinfo.BuildRevision
	// BuildTime holds the vcs commit time information of the program's build.
	BuildTime = buildinfo.BuildTime

	// RegisterOnInterrupt takes a function and appends it to the list of functions to be called on an interrupt signal (CTRL+C).
	RegisterOnInterrupt = signalhandler.Register
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
	// ServerOptions is a type alias for the `server.ServerOptions`.
	ServerOptions = server.ServerOptions
	// NgrokOptions is a type alias for the `server.NgrokOptions`.
	NgrokOptions = server.NgrokOptions
	// TimeoutOptions is a type alias for the `server.TimeoutOptions`.
	TimeoutOptions = server.TimeoutOptions
	// ContextOptions is a type alias for the `context.ContextOptions`.
	// It carries the per-request settings, among them PostMaxMemory,
	// BodyMaxMemory, RemoteAddrHeaders, RequestValidator and ErrorHandler.
	// Reachable as Options.Context.
	ContextOptions = context.ContextOptions
	// RouterOptions is a type alias for the `router.RouterOptions`.
	RouterOptions = router.RouterOptions
	// UnmarshalerFunc a shortcut, an alias for the `context#UnmarshalerFunc` type
	// which implements the `context#Unmarshaler` interface for reading request's body
	// via custom decoders, most of them already implement the `context#UnmarshalerFunc`
	// like the json.Unmarshal, xml.Unmarshal, yaml.Unmarshal and every library which
	// follows the best practises and is aligned with the Go standards.
	//
	// See 'context#UnmarshalBody` for more.
	UnmarshalerFunc = context.UnmarshalerFunc
	// DecodeFunc is a generic type of decoder function.
	// When the returned error is not nil the decode operation
	// is terminated and the error is received by the ReadJSONStream method,
	// otherwise it continues to read the next available object.
	// Look the `Context.ReadJSONStream` method.
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
	// A Map is an alias of map[string]any.
	Map = context.Map
	// User is a generic view of an authorized client.
	// See `Context.User` and `SetUser` methods for more.
	// An alias for the `context/User` type.
	User = context.User
	// SimpleUser is a simple implementation of the User interface.
	SimpleUser = context.SimpleUser
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
	// XML the optional settings for XML renderer.
	//
	// It is an alias of the `context#XML` type.
	XML = context.XML
	// Markdown the optional settings for Markdown renderer.
	// See `Context.Markdown` for more.
	//
	// It is an alias of the `context#Markdown` type.
	Markdown = context.Markdown

	// Router is just a group joiner of routes which have the same prefix and share same middleware(s) also.
	// Router could also be named as 'Join' or 'Node' or 'Group' , Router chosen because it is fun.
	//
	// Look the `core/router#RouterBuilder` for its implementation.
	//
	// A shortcut for the `core/router#Router`, the type a controller's Routing
	// method and a Router("/path", func(iris.Router)) callback receive.
	Router = router.Router

	// CookieOption is the type of function that is accepted on
	// context's methods like `AddCookie`, `RemoveCookie` and `SetCookie`
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

	// PageOptions is an alias for the `context.PageOptions`:
	// the page/size/details a client sends through the URL query,
	// read by Context.BindPagination and Context.Paginated.
	PageOptions = context.PageOptions
	// Page is an alias for the `context.Page[T]`:
	// the paginated response envelope of Context.Paginated and context.NewPage.
	Page[T any] = context.Page[T]
	// CountType is an alias for the `context.CountType`:
	// the numeric constraint of Context.Count and Context.Paginated totals.
	CountType = context.CountType
	// CountResponse is an alias for the `context.CountResponse[N]`:
	// the {"count": N} body of Context.Count.
	CountResponse[N CountType] = context.CountResponse[N]
	// IDType is an alias for the `context.IDType`:
	// the identifier types Context.Created wraps as an IDPayload.
	IDType = context.IDType
	// IDPayload is an alias for the `context.IDPayload[T]`:
	// the {"id": T} body Context.Created renders for a bare identifier.
	IDPayload[T IDType] = context.IDPayload[T]
	// Validatable is an alias for the `context.Validatable`:
	// implemented by request types that validate themselves after decoding;
	// Context.BindJSON and its variants call Validate and reject the request
	// on a non-nil error.
	Validatable = context.Validatable
	// QueryOption is an alias for the `context.QueryOption`:
	// the option type of the strict query readers Context.QueryDate,
	// Context.QueryInt and Context.QueryBool.
	// See QueryRequired, QueryDefault and QueryZeroValues.
	QueryOption = context.QueryOption
	// ListFilter is an alias for the `context.ListFilter`:
	// the request half every list endpoint shares (term, names, substring
	// and the sort request), meant to be embedded in a service's own filter.
	ListFilter = context.ListFilter
	// PaginationOption is an alias for the `context.PaginationOption`:
	// the option type of Context.BindPagination and Context.Paginated.
	// See AllowSort and DenySort.
	PaginationOption = context.PaginationOption
	// ErrPrivate if provided then the error saved in context
	// should NOT be visible to the client no matter what.
	// An alias for the `context.ErrPrivate`.
	ErrPrivate = context.ErrPrivate
)

var (
	// QueryRequired is a shortcut for the `context.QueryRequired`:
	// an absent query value becomes a 400 Bad Request naming the key.
	QueryRequired = context.QueryRequired
	// QueryDefault is a shortcut for the `context.QueryDefault`:
	// the raw value to parse when the query key is absent.
	QueryDefault = context.QueryDefault
	// QueryZeroValues is a shortcut for the `context.QueryZeroValues`:
	// the raw values that mean "unset", e.g. "0" or "0000-00-00".
	QueryZeroValues = context.QueryZeroValues

	// AllowSort is a shortcut for the `context.AllowSort`:
	// a sort column outside the list becomes a 400 Bad Request.
	AllowSort = context.AllowSort
	// DenySort is a shortcut for the `context.DenySort`:
	// any sort request becomes a 400 Bad Request with the given message.
	DenySort = context.DenySort
	// ErrSortNotAllowed is a shortcut for the `context.ErrSortNotAllowed`:
	// the error ListFilter.SortColumn wraps for a column outside the
	// endpoint's allowed set.
	ErrSortNotAllowed = context.ErrSortNotAllowed
)

// Constants for input argument at `router.RouteRegisterRule`.
// See `Router#SetRegisterRule`.
const (
	// RouteOverride replaces an existing route with the new one, the default rule.
	RouteOverride = router.RouteOverride
	// RouteSkip keeps the original route and skips the new one.
	RouteSkip = router.RouteSkip
	// RouteError log when a route already exists, shown after the `Build` state,
	// server never starts.
	RouteError = router.RouteError
)

// NewPage returns a new paginated response envelope for the given page
// options, filter, items and total count. It forwards to `context.NewPage`;
// a generic function cannot be a type alias, so this is a wrapper, not an
// alias like `iris.Page[T]` itself.
//
// Use it when a handler builds the envelope on its own; Context.Paginated
// already returns one.
func NewPage[T any](opts PageOptions, filter any, items []T, totalCount int64) *Page[T] {
	return context.NewPage(opts, filter, items, totalCount)
}

// Component returns a new Handler which can be registered as a main handler for a route.
// It's a shortcut handler that renders the given component as HTML through Context.RenderComponent.
func Component(component context.Component) Handler {
	return func(ctx Context) {
		ctx.RenderComponent(component)
	}
}

/* START: Controller-specific */

type (
	// Controller is a type alias for the `core/router#Controller`.
	Controller = router.Controller
	// Routing is a type alias for the `core/router#Routing`.
	Routing = router.Routing
)

var (
	// Join is a shortcut for the `router.Join`.
	Join = router.Join
	// Prefix is a shortcut for the `router.Prefix`.
	Prefix = router.Prefix
	// Dependencies is a shortcut for the `router.Dependencies`.
	// Dependencies returns a Controller which can be used to register
	// dependencies to the controllers follows in the same `Router.Router` method call.
	// Useful when you have the same interface declared for different services
	// that can be binded to more than one controller.
	Dependencies = router.Dependencies
	// Middleware is a shortcut for the `router.Middleware`.
	// Middleware is a Controller which can be used
	// as a shortcut to add middlewares on Router.Router("/path", Middleware(handler), new(example.API)).
	Middleware = router.Middleware
)

// ResolveController is a shortcut for the `router.ResolveController`.
func ResolveController[T any]() Controller {
	return router.ResolveController[T]()
}

/* END: Controller-specific */

var (
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
	// Example can be found at: _examples/06-middleware/conditional.
	//
	// A shortcut for the `context#NewConditionalHandler`.
	NewConditionalHandler = context.NewConditionalHandler

	// FromStd converts native http.Handler, http.HandlerFunc & func(w, r, next) to context.Handler.
	//
	// Supported form types:
	// 		 .FromStd(h http.Handler)
	// 		 .FromStd(func(w http.ResponseWriter, r *http.Request))
	// 		 .FromStd(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc))
	//
	// A shortcut for the `context#FromStd`.
	// See `context#TryFromStd` for the non-panicking form.
	FromStd = context.FromStd

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
	// If set it will add the cookie to (on `SetCookie`, `AddCookie`)
	// or remove the cookie from (on `RemoveCookie`) the Request object too.
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
	// CookieSecure sets the cookie's Secure option on set operation if the current request's
	// connection is using TLS. See `CookieHTTPOnly` too.
	//
	// A shortcut for the `context#CookieSecure`.
	CookieSecure = context.CookieSecure
	// CookieSecureForce is a `CookieOption`. It sets the cookie's Secure option to true every time.
	//
	// A shortcut for the `context#CookieSecureForce`.
	CookieSecureForce = context.CookieSecureForce
	// CookieHTTPOnly is a `CookieOption`.
	// Use it to set the cookie's HttpOnly field to false or true.
	// HttpOnly field defaults to true for `RemoveCookie` and `AddCookie`.
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
	// CookiePartitioned is a `CookieOption`.
	// It enables the "Partitioned" cookie attribute, which binds a third-party
	// cookie to the top-level site that caused it to be set (CHIPS).
	//
	// A shortcut for the `context#CookiePartitioned`.
	CookiePartitioned = context.CookiePartitioned
	// CookieEncoding accepts a value which implements `Encode` and `Decode` methods.
	// It calls its `Encode` on `Context.SetCookie` and `AddCookie` methods.
	// And on `Context.Cookie` method it calls its `Decode`.
	//
	// A shortcut for the `context#CookieEncoding`.
	CookieEncoding = context.CookieEncoding

	// IsErrEmptyJSON reports whether the given "err" is caused by a
	// Context.ReadJSON call when the request body
	// didn't start with { or it was totally empty.
	IsErrEmptyJSON = context.IsErrEmptyJSON
	// IsErrPath can be used at `context#ReadForm` and `context#ReadQuery`.
	// It reports whether the incoming error is caused by an unknown field
	// (`schema.UnknownKeyError`), which can be ignored when server allows
	// unknown post values to be sent by the client.
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
	// XMLMap wraps a map[string]any to compatible xml marshaler,
	// in order to be able to render maps as XML on the `Context.XML` method.
	//
	// Example: `Context.XML(XMLMap("Root", map[string]any{...})`.
	//
	// A shortcut for the `context#XMLMap`.
	XMLMap = context.XMLMap
	// ErrHijackNotSupported is returned by the Hijack method to
	// indicate that Hijack feature is not available.
	//
	// A shortcut for the `context#ErrHijackNotSupported`.
	ErrHijackNotSupported = context.ErrHijackNotSupported

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
	// MethodQuery is the RFC 10008 QUERY method.
	MethodQuery = router.MethodQuery
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
	// Timeout returns a new Timeout configuration builder.
	//
	// It's a shortcut of the `server#Timeout`.
	Timeout = server.Timeout

	// StatusText returns a text for the HTTP status code. It returns the empty
	// string if the code is unknown.
	//
	// Shortcut for context#StatusText.
	StatusText = context.StatusText
)

// WithErrorHandler returns a middleware which sets a custom error handler for
// the requests it runs on. It overrides the Options.Context.ErrorHandler value
// for those requests only, and applies on Render, RenderComponent, JSON, JSONP,
// XML, Markdown, YAML, Protobuf and MsgPack write errors.
//
// Register it like any other middleware, on the application, a router or a
// single route:
//
//	app.Use(iris.WithErrorHandler(func(ctx iris.Context, err error) {
//		ctx.StopWithPlainError(iris.StatusInternalServerError, err)
//	}))
//
// To customize panic recovery errors instead, see middleware/recovery.WithErrorHandler.
func WithErrorHandler(handler context.ErrorHandler) Handler {
	return func(ctx Context) {
		ctx.Options().ErrorHandler = handler
		ctx.Next()
	}
}

var globalPatches = &GlobalPatches{
	contextPatches: &ContextPatches{
		writers: &ContextWriterPatches{},
	},
}

// GlobalPatches is a singleton that provides a uniform way to apply global/package-level modifications.
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
// Example: https://github.com/kataras/iris/blob/main/_examples/16-advanced/patches/main.go
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
	server.GetRootDomain = patchFunc
}

// SetDefaultCookieExpiration modifies the lifetime Context.AddCookie gives a cookie
// when no CookieExpires option is passed (context.DefaultCookieExpiration, 365 days by default).
func (cp *ContextPatches) SetDefaultCookieExpiration(patch time.Duration) {
	context.DefaultCookieExpiration = patch
}

// ResolveHTTPFS modifies the default way to resolve a filesystem by any type of value.
// It affects the Application's API Builder's `fileserver.Handler` method.
func (cp *ContextPatches) ResolveHTTPFS(patchFunc func(fsOrDir any) http.FileSystem) {
	context.ResolveHTTPFS = patchFunc
}

// SetMaxPageSize modifies the largest page size PageOptions.GetLimit returns
// (pagination.MaxPageSize, 100000 by default). Zero or less means no bound.
func (cp *ContextPatches) SetMaxPageSize(size int) {
	pagination.MaxPageSize = size
}

// SetDefaultPageSize modifies the page size PageOptions.GetLimit returns when
// the request asks for none (pagination.DefaultPageSize, 100 by default).
func (cp *ContextPatches) SetDefaultPageSize(size int) {
	pagination.DefaultPageSize = size
}

// ResolveFS modifies the default way to resolve a filesystem by any type of value.
// It affects the view engine's filesystem resolver.
func (cp *ContextPatches) ResolveFS(patchFunc func(fsOrDir any) fs.FS) {
	context.ResolveFS = patchFunc
}

// ContextWriterPatches features the context's writers patches.
type ContextWriterPatches struct{}

// JSON sets a custom function which runs and overrides the default behavior of the `Context#JSON` method.
func (cwp *ContextWriterPatches) JSON(patchFunc func(ctx Context, v any, options *JSON) error) {
	context.WriteJSON = patchFunc
}

// JSONP sets a custom function which runs and overrides the default behavior of the `Context#JSONP` method.
func (cwp *ContextWriterPatches) JSONP(patchFunc func(ctx Context, v any, options *JSONP) error) {
	context.WriteJSONP = patchFunc
}

// XML sets a custom function which runs and overrides the default behavior of the `Context#XML` method.
func (cwp *ContextWriterPatches) XML(patchFunc func(ctx Context, v any, options *XML) error) {
	context.WriteXML = patchFunc
}

// Markdown sets a custom function which runs and overrides the default behavior of the `Context#Markdown` method.
func (cwp *ContextWriterPatches) Markdown(patchFunc func(ctx Context, v []byte, options *Markdown) error) {
	context.WriteMarkdown = patchFunc
}

// YAML sets a custom function which runs and overrides the default behavior of the `Context#YAML` method.
func (cwp *ContextWriterPatches) YAML(patchFunc func(ctx Context, v any, indentSpace int) error) {
	context.WriteYAML = patchFunc
}
