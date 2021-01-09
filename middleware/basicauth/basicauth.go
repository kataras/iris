package basicauth

import (
	stdContext "context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/sessions"
)

func init() {
	context.SetHandlerName("iris/middleware/basicauth.*", "iris.basicauth")
}

const (
	// DefaultRealm is the default realm directive value on Default and Load functions.
	DefaultRealm = "Authorization Required"
	// DefaultMaxTriesCookie is the default cookie name to store the
	// current amount of login failures when MaxTries > 0.
	DefaultMaxTriesCookie = "basicmaxtries"
	// DefaultCookieMaxAge is the default cookie max age on MaxTries,
	// when the Options.MaxAge is zero.
	DefaultCookieMaxAge = time.Hour
)

const (
	authorizationType           = "Basic Authentication"
	authenticateHeaderKey       = "WWW-Authenticate"
	proxyAuthenticateHeaderKey  = "Proxy-Authenticate"
	authorizationHeaderKey      = "Authorization"
	proxyAuthorizationHeaderKey = "Proxy-Authorization"
)

// AuthFunc accepts the current request and the username and password user inputs
// and it should optionally return a user value and report whether the login succeed or not.
// Look the Options.Allow field.
//
// Default implementations are:
// AllowUsers and AllowUsersFile functions.
type AuthFunc func(ctx *context.Context, username, password string) (interface{}, bool)

// ErrorHandler should handle the given request credentials failure.
// See Options.ErrorHandler and DefaultErrorHandler for details.
type ErrorHandler func(ctx *context.Context, err error)

// Options holds the necessary information that the BasicAuth instance needs to perform.
// The only required value is the Allow field.
//
// Usage:
//  opts := Options { ... }
//  auth := New(opts)
type Options struct {
	// Realm directive, read http://tools.ietf.org/html/rfc2617#section-1.2 for details.
	// E.g. "Authorization Required".
	Realm string
	// In the case of proxies, the challenging status code is 407 (Proxy Authentication Required),
	// the Proxy-Authenticate response header contains at least one challenge applicable to the proxy,
	// and the Proxy-Authorization request header is used for providing the credentials to the proxy server.
	//
	// Proxy should be used to gain access to a resource behind a proxy server.
	// It authenticates the request to the proxy server, allowing it to transmit the request further.
	Proxy bool
	// If set to true then any non-https request will immediately
	// dropped with a 505 status code (StatusHTTPVersionNotSupported) response.
	//
	// Defaults to false.
	HTTPSOnly bool
	// Allow is the only one required field for the Options type.
	// Can be customized to validate a username and password combination
	// and return a user object, e.g. fetch from database.
	//
	// There are two available builtin values, the AllowUsers and AllowUsersFile,
	// both of them decode a static list of users and compares with the user input (see BCRYPT function too).
	// Usage:
	//  - Allow: AllowUsers(iris.Map{"username": "...", "password": "...", "other_field": ...}, [BCRYPT])
	//  - Allow: AllowUsersFile("users.yml", [BCRYPT])
	// Look the user.go source file for details.
	Allow AuthFunc
	// MaxAge sets expiration duration for the in-memory credentials map.
	// By default an old map entry will be removed when the user visits a page.
	// In order to remove old entries automatically please take a look at the `GC` option too.
	//
	// Usage:
	//  MaxAge: 30 * time.Minute
	MaxAge time.Duration
	// If greater than zero then the server will send 403 forbidden status code afer
	// MaxTries amount of sign in failures (see MaxTriesCookie).
	// Note that the client can modify the cookie and its value,
	// do NOT depend for any type of custom domain logic based on this field.
	// By default the server will re-ask for credentials on invalid credentials, each time.
	MaxTries int
	// MaxTriesCookie is the cookie name the middleware uses to
	// store the failures amount on the client side.
	// The lifetime of the cookie is the same as the configured MaxAge or one hour,
	// therefore a forbidden client can request for authentication again after expiration.
	//
	// You can always set custom logic on the Allow field as you have access to the current request instance.
	//
	// Defaults to "basicmaxtries".
	// The MaxTries should be set to greater than zero.
	MaxTriesCookie string
	// If not empty then this session key will be used to store
	// the current tries of login failures. If not a session manager
	// was registered then the application will log an error.
	// Note that this field has a priority over the MaxTriesCookie.
	MaxTriesSession string
	// ErrorHandler handles the given request credentials failure.
	// E.g  when the client tried to access a protected resource
	// with empty or invalid or expired credentials or
	// when Allow returned false and MaxTries consumed.
	//
	// Defaults to the DefaultErrorHandler, do not modify if you don't need to.
	ErrorHandler ErrorHandler
	// GC automatically clears old entries every x duration.
	// Note that, by old entries we mean expired credentials therefore
	// the `MaxAge` option should be already set,
	// if it's not then all entries will be removed on "every" duration.
	// The standard context can be used for the internal ticker cancelation, it can be nil.
	//
	// Usage:
	//  GC: basicauth.GC{Every: 2 * time.Hour}
	GC GC
}

// GC holds the context and the tick duration to clear expired stored credentials.
// See the Options.GC field.
type GC struct {
	Context stdContext.Context
	Every   time.Duration
}

// BasicAuth implements the basic access authentication.
// It is a method for an HTTP client (e.g. a web browser)
// to provide a user name and password when making a request.
// Basic authentication implementation is the simplest technique
// for enforcing access controls to web resources because it does not require
// cookies, session identifiers, or login pages; rather,
// HTTP Basic authentication uses standard fields in the HTTP header.
//
// As the username and password are passed over the network as clear text
// the basic authentication scheme is not secure on plain HTTP communication.
// It is base64 encoded, but base64 is a reversible encoding.
// HTTPS/TLS should be used with basic authentication.
// Without these additional security enhancements,
// basic authentication should NOT be used to protect sensitive or valuable information.
//
// Read https://tools.ietf.org/html/rfc2617 and
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication for details.
type BasicAuth struct {
	opts Options
	// built based on proxy field
	askCode             int
	authorizationHeader string
	authenticateHeader  string
	// built based on realm field.
	authenticateHeaderValue string

	// credentials stores the user expiration,
	// key = username:password, value = expiration time (if MaxAge > 0).
	credentials map[string]*time.Time // TODO: think of just a uint64 here (unix seconds).
	// protects the credentials concurrent access.
	mu sync.RWMutex
}

// New returns a new basic authentication middleware.
// The result should be used to wrap an existing handler or the HTTP application's root router.
//
// Example Code:
//  opts := basicauth.Options{
//  	Realm: basicauth.DefaultRealm,
//      ErrorHandler: basicauth.DefaultErrorHandler,
//  	MaxAge: 2 * time.Hour,
//  	GC: basicauth.GC{
//  		Every: 3 * time.Hour,
//  	},
//  	Allow: basicauth.AllowUsers(users),
//  }
//  auth := basicauth.New(opts)
//  app.Use(auth)
//
// Access the user in the route handler with: ctx.User().GetRaw().(*myCustomType).
//
// Look the BasicAuth type docs for more information.
func New(opts Options) context.Handler {
	var (
		askCode                 = http.StatusUnauthorized
		authorizationHeader     = authorizationHeaderKey
		authenticateHeader      = authenticateHeaderKey
		authenticateHeaderValue = "Basic"
	)

	if opts.Allow == nil {
		panic("BasicAuth: Allow field is required")
	}

	if opts.Realm != "" {
		authenticateHeaderValue += " realm=" + strconv.Quote(opts.Realm)
	}

	if opts.Proxy {
		askCode = http.StatusProxyAuthRequired
		authenticateHeader = proxyAuthenticateHeaderKey
		authorizationHeader = proxyAuthorizationHeaderKey
	}

	if opts.MaxTries > 0 && opts.MaxTriesCookie == "" {
		opts.MaxTriesCookie = DefaultMaxTriesCookie
	}

	if opts.ErrorHandler == nil {
		opts.ErrorHandler = DefaultErrorHandler
	}

	b := &BasicAuth{
		opts:                    opts,
		askCode:                 askCode,
		authorizationHeader:     authorizationHeader,
		authenticateHeader:      authenticateHeader,
		authenticateHeaderValue: authenticateHeaderValue,
		credentials:             make(map[string]*time.Time),
	}

	if opts.GC.Every > 0 {
		go b.runGC(opts.GC.Context, opts.GC.Every)
	}

	return b.serveHTTP
}

// Default returns a new basic authentication middleware
// based on pre-defined user list.
// A user can hold any custom fields but the username and password
// are required as they are compared against the user input
// when access to protected resource is requested.
// A user list can defined with one of the following values:
//  map[string]string form of: {username:password, ...}
//  map[string]interface{} form of: {"username": {"password": "...", "other_field": ...}, ...}
//  []T which T completes the User interface, where T is a struct value
//  []T which T contains at least Username and Password fields.
//
// Usage:
//  auth := Default(map[string]string{
//    "admin": "admin",
//    "john": "p@ss",
//  })
func Default(users interface{}, userOpts ...UserAuthOption) context.Handler {
	opts := Options{
		Realm: DefaultRealm,
		Allow: AllowUsers(users, userOpts...),
	}
	return New(opts)
}

// Load same as Default but instead of a hard-coded user list it accepts
// a filename to load the users from.
//
// Usage:
//  auth := Load("users.yml")
func Load(jsonOrYamlFilename string, userOpts ...UserAuthOption) context.Handler {
	opts := Options{
		Realm: DefaultRealm,
		Allow: AllowUsersFile(jsonOrYamlFilename, userOpts...),
	}
	return New(opts)
}

func (b *BasicAuth) getCurrentTries(ctx *context.Context) (tries int) {
	if key := b.opts.MaxTriesSession; key != "" {
		if sess := sessions.Get(ctx); sess != nil {
			tries = sess.GetIntDefault(key, 0)
		} else {
			ctx.Application().Logger().Error("basicauth: getCurrentTries: session key: %s but no session manager is registered", key)
			return
		}
	} else {
		cookie := ctx.GetCookie(b.opts.MaxTriesCookie)
		if cookie != "" {
			tries, _ = strconv.Atoi(cookie)
		}
	}

	return
}

func (b *BasicAuth) setCurrentTries(ctx *context.Context, tries int) {
	if key := b.opts.MaxTriesSession; key != "" {
		if sess := sessions.Get(ctx); sess != nil {
			sess.Set(key, tries)
		} else {
			ctx.Application().Logger().Error("basicauth: setCurrentTries: session key: %s but no session manager is registered", key)
			return
		}
	} else {
		maxAge := b.opts.MaxAge
		if maxAge == 0 {
			maxAge = DefaultCookieMaxAge // 1 hour.
		}

		c := &http.Cookie{
			Name:     b.opts.MaxTriesCookie,
			Path:     "/",
			Value:    url.QueryEscape(strconv.Itoa(tries)),
			HttpOnly: true,
			Expires:  time.Now().Add(maxAge),
			MaxAge:   int(maxAge.Seconds()),
		}

		ctx.SetCookie(c)
	}
}

func (b *BasicAuth) resetCurrentTries(ctx *context.Context) {
	if key := b.opts.MaxTriesSession; key != "" {
		if sess := sessions.Get(ctx); sess != nil {
			sess.Delete(key)
		} else {
			ctx.Application().Logger().Error("basicauth: resetCurrentTries: session key: %s but no session manager is registered", key)
			return
		}
	} else {
		ctx.RemoveCookie(b.opts.MaxTriesCookie)
	}
}

func isHTTPS(r *http.Request) bool {
	return (strings.EqualFold(r.URL.Scheme, "https") || r.TLS != nil) && r.ProtoMajor == 2
}

func (b *BasicAuth) handleError(ctx *context.Context, err error) {
	ctx.Application().Logger().Debug(err)

	// should not be nil as it's defaulted on New.
	b.opts.ErrorHandler(ctx, err)
}

// serveHTTP is the main method of this middleware,
// checks and verifies the auhorization header for basic authentication,
// next handlers will only be executed when the client is allowed to continue.
func (b *BasicAuth) serveHTTP(ctx *context.Context) {
	if b.opts.HTTPSOnly && !isHTTPS(ctx.Request()) {
		b.handleError(ctx, ErrHTTPVersion{})
		return
	}

	header := ctx.GetHeader(b.authorizationHeader)
	fullUser, username, password, ok := decodeHeader(header)
	if !ok { // Header is malformed or missing (e.g. browser cancel button on user prompt).
		b.handleError(ctx, ErrCredentialsMissing{
			Header:                  header,
			AuthenticateHeader:      b.authenticateHeader,
			AuthenticateHeaderValue: b.authenticateHeaderValue,
			Code:                    b.askCode,
		})
		return
	}

	var (
		maxTries = b.opts.MaxTries
		tries    int
	)

	if maxTries > 0 {
		tries = b.getCurrentTries(ctx)
	}

	user, ok := b.opts.Allow(ctx, username, password)
	if !ok { // This username:password combination was not allowed.
		if maxTries > 0 {
			tries++
			b.setCurrentTries(ctx, tries)
			if tries >= maxTries { // e.g. if MaxTries == 1 then it should be allowed only once, so we must send forbidden now.
				b.handleError(ctx, ErrCredentialsForbidden{
					Username: username,
					Password: password,
					Tries:    tries,
					Age:      b.opts.MaxAge,
				})
				return
			}
		}

		b.handleError(ctx, ErrCredentialsInvalid{
			Username:                username,
			Password:                password,
			CurrentTries:            tries,
			AuthenticateHeader:      b.authenticateHeader,
			AuthenticateHeaderValue: b.authenticateHeaderValue,
			Code:                    b.askCode,
		})
		return
	}

	if tries > 0 {
		// had failures but it's ok, reset the tries on success.
		b.resetCurrentTries(ctx)
	}

	b.mu.RLock()
	expiresAt, ok := b.credentials[fullUser]
	b.mu.RUnlock()
	var authorizedAt time.Time
	if ok {
		if expiresAt != nil { // Has expiration.
			if expiresAt.Before(time.Now()) { // Has been expired.
				b.mu.Lock() // Delete the entry.
				delete(b.credentials, fullUser)
				b.mu.Unlock()

				// Re-ask for new credentials.
				b.handleError(ctx, ErrCredentialsExpired{
					Username:                username,
					Password:                password,
					AuthenticateHeader:      b.authenticateHeader,
					AuthenticateHeaderValue: b.authenticateHeaderValue,
					Code:                    b.askCode,
				})
				return
			}

			// It's ok, find the time authorized to fill the user below, if necessary.
			authorizedAt = expiresAt.Add(-b.opts.MaxAge)
		}
	} else {
		// Saved credential not found, first login.
		if b.opts.MaxAge > 0 { // Expiration is enabled, set the value.
			authorizedAt = time.Now()
			t := authorizedAt.Add(b.opts.MaxAge)
			expiresAt = &t
		}
		b.mu.Lock()
		b.credentials[fullUser] = expiresAt
		b.mu.Unlock()
	}

	if user == nil {
		// No custom uset was set by the auth func,
		// it is passed though, set a simple user here:
		user = &context.SimpleUser{
			Authorization: authorizationType,
			AuthorizedAt:  authorizedAt,
			Username:      username,
			Password:      password,
		}
	}

	// Store user instance and logout function.
	// Note that the end-developer has always have access
	// to the Request.BasicAuth, however, we support any user struct,
	// so we must store it on this request instance so it can be retrieved later on.
	ctx.SetUser(user)
	ctx.SetLogoutFunc(b.logout)

	ctx.Next()
}

// logout clears the current user's credentials.
func (b *BasicAuth) logout(ctx *context.Context) {
	var (
		fullUser, username, password string
		ok                           bool
	)

	if u := ctx.User(); u != nil { // Get the saved ones, if any.
		username, _ = u.GetUsername()
		password, _ = u.GetPassword()
		fullUser = username + colonLiteral + password
		ok = username != "" && password != ""
	}

	if !ok {
		// If the custom user does
		// not implement the User interface, then extract from the request header (most common scenario):
		header := ctx.GetHeader(b.authorizationHeader)
		fullUser, _, _, ok = decodeHeader(header)
	}

	if ok { // If it's authorized then try to lock and delete.
		ctx.SetUser(nil)
		ctx.SetLogoutFunc(nil)

		if b.opts.Proxy {
			ctx.Request().Header.Del(proxyAuthorizationHeaderKey)
		}
		// delete the request header so future Request().BasicAuth are empty.
		ctx.Request().Header.Del(authorizationHeaderKey)

		b.mu.Lock()
		delete(b.credentials, fullUser)
		b.mu.Unlock()
	}
}

// runGC runs a function in a separate go routine
// every x duration to clear in-memory expired credential entries.
func (b *BasicAuth) runGC(ctx stdContext.Context, every time.Duration) {
	if ctx == nil {
		ctx = stdContext.Background()
	}

	t := time.NewTicker(every)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.gc()
		}
	}
}

// gc removes all entries expired based on the max age or all entries (if max age is missing),
// note that this does not mean that the server will send 401/407 to the next request,
// when the request header credentials are still valid (Allow passed).
func (b *BasicAuth) gc() int {
	now := time.Now()
	var markedForDeletion []string

	b.mu.RLock()
	for fullUser, expiresAt := range b.credentials {
		if expiresAt == nil || expiresAt.Before(now) {
			markedForDeletion = append(markedForDeletion, fullUser)
		}
	}
	b.mu.RUnlock()

	n := len(markedForDeletion)
	if n > 0 {
		for _, fullUser := range markedForDeletion {
			b.mu.Lock()
			delete(b.credentials, fullUser)
			b.mu.Unlock()
		}
	}

	return n
}
