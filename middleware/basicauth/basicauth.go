package basicauth

import (
	stdContext "context"
	"strconv"
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/sessions"
)

func init() {
	context.SetHandlerName("iris/middleware/basicauth.*", "iris.basicauth")
}

const (
	DefaultRealm          = "Authorization Required"
	DefaultMaxTriesCookie = "basicmaxtries"
)

const (
	authorizationType           = "Basic Authentication"
	authenticateHeaderKey       = "WWW-Authenticate"
	proxyAuthenticateHeaderKey  = "Proxy-Authenticate"
	authorizationHeaderKey      = "Authorization"
	proxyAuthorizationHeaderKey = "Proxy-Authorization"
)

type AuthFunc func(ctx *context.Context, username, password string) (interface{}, bool)

type Options struct {
	// Realm http://tools.ietf.org/html/rfc2617#section-1.2.
	// E.g. "Authorization Required".
	Realm string
	// In the case of proxies, the challenging status code is 407 (Proxy Authentication Required),
	// the Proxy-Authenticate response header contains at least one challenge applicable to the proxy,
	// and the Proxy-Authorization request header is used for providing the credentials to the proxy server.
	//
	// Proxy should be used to gain access to a resource behind a proxy server.
	// It authenticates the request to the proxy server, allowing it to transmit the request further.
	Proxy bool
	// Usage:
	//  - Allow: AllowUsers(iris.Map{"username": "...", "password": "...", "other_field": ...}, [BCRYPT])
	//  - Allow: AllowUsersFile("users.yml", [BCRYPT])
	Allow AuthFunc
	// If greater than zero then the server will send 403 forbidden status code afer MaxTries
	// of invalid credentials of a specific client consumed (session or cookie based, see MaxTriesCookie).
	// By default the server will re-ask for credentials on any amount of invalid credentials.
	MaxTries int
	// If a session manager is register under the current request,
	// then this value should be the key of the session storage which
	// the current tries will be stored. Otherwise
	// it is the raw cookie name.
	// The cookie is stored up to the configured MaxAge if greater than zero or for 1 year,
	// so a forbidden client can request for authentication again after the MaxAge expired.
	//
	// Note that, the session way is recommended as the current tries
	// cannot be modified by the client (unless the client removes the session cookie).
	// However the raw cookie performs faster. You can always set custom logic
	// on the Allow field as you have access to the current request Context.
	// To set custom cookie options use the `Context.AddCookieOptions(options ...iris.CookieOption)`
	// before the basic auth middleware.
	//
	// If MaxTries > 0 then it defaults to "basicmaxtries".
	// The MaxTries should be set to greater than zero.
	MaxTriesCookie string
	// If not nil runs after 401 (or 407 if proxy is enabled) status code.
	// Can be used to set custom response for unauthenticated clients.
	OnAsk context.Handler
	// If not nil runs after the 403 forbidden status code (when Allow returned false and MaxTries consumed).
	// Can be used to set custom response when client tried to access a resource with invalid credentials.
	OnForbidden context.Handler
	// MaxAge sets expiration duration for the in-memory credentials map.
	// By default an old map entry will be removed when the user visits a page.
	// In order to remove old entries automatically please take a look at the `GC` option too.
	//
	// Usage:
	//  MaxAge: 30*time.Minute
	MaxAge time.Duration
	// GC automatically clears old entries every x duration.
	// Note that, by old entries we mean expired credentials therefore
	// the `MaxAge` option should be already set,
	// if it's not then all entries will be removed on "every" duration.
	// The standard context can be used for the internal ticker cancelation, it can be nil.
	//
	// Usage:
	//  GC: basicauth.GC{Every: 2*time.Hour}
	GC GC
}

type GC struct {
	Context stdContext.Context
	Every   time.Duration
}

// https://tools.ietf.org/html/rfc2617
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication
//
// As the user ID and password are passed over the network as clear text
// (it is base64 encoded, but base64 is a reversible encoding), the basic authentication scheme is not secure.
// HTTPS/TLS should be used with basic authentication. Without these additional security enhancements,
// basic authentication should not be used to protect sensitive or valuable information.
type BasicAuth struct {
	opts Options
	// built based on proxy field
	askCode             int
	authorizationHeader string
	authenticateHeader  string
	// built based on realm field.
	authenticateHeaderValue string

	credentials map[string]*time.Time // key = username:password, value = expiration time (if MaxAge > 0).
	mu          sync.RWMutex          // protects the credentials as they can modified.
}

func New(opts Options) context.Handler {
	var (
		askCode                 = 401
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
		askCode = 407
		authenticateHeader = proxyAuthenticateHeaderKey
		authorizationHeader = proxyAuthorizationHeaderKey
	}

	if opts.MaxTries > 0 && opts.MaxTriesCookie == "" {
		opts.MaxTriesCookie = DefaultMaxTriesCookie
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

// - map[string]string form of: {username:password, ...} form.
// - map[string]interface{} form of: []{"username": "...", "password": "...", "other_field": ...}, ...}.
// - []T which T completes the User interface.
// - []T which T contains at least Username and Password fields.
func Default(users interface{}, userOpts ...UserAuthOption) context.Handler {
	opts := Options{
		Realm: DefaultRealm,
		Allow: AllowUsers(users, userOpts...),
	}
	return New(opts)
}

func Load(jsonOrYamlFilename string, userOpts ...UserAuthOption) context.Handler {
	opts := Options{
		Realm: DefaultRealm,
		Allow: AllowUsersFile(jsonOrYamlFilename, userOpts...),
	}
	return New(opts)
}

// askForCredentials sends a response to the client which client should catch
// and ask for username:password credentials.
func (b *BasicAuth) askForCredentials(ctx *context.Context) {
	ctx.Header(b.authenticateHeader, b.authenticateHeaderValue)
	ctx.StopWithStatus(b.askCode)

	if h := b.opts.OnAsk; h != nil {
		h(ctx)
	}
}

// If a (proxy) server receives valid credentials that are inadequate to access a given resource,
// the server should respond with the 403 Forbidden status code.
// Unlike 401 Unauthorized or 407 Proxy Authentication Required, authentication is impossible for this user.
func (b *BasicAuth) forbidden(ctx *context.Context) {
	ctx.StopWithStatus(403)

	if h := b.opts.OnForbidden; h != nil {
		h(ctx)
	}
}

func (b *BasicAuth) getCurrentTries(ctx *context.Context) (tries int) {
	sess := sessions.Get(ctx)
	if sess != nil {
		tries = sess.GetIntDefault(b.opts.MaxTriesCookie, 0)
	} else {
		if v := ctx.GetCookie(b.opts.MaxTriesCookie); v != "" {
			tries, _ = strconv.Atoi(v)
		}
	}

	return
}

func (b *BasicAuth) setCurrentTries(ctx *context.Context, tries int) {
	sess := sessions.Get(ctx)
	if sess != nil {
		sess.Set(b.opts.MaxTriesCookie, tries)
	} else {
		maxAge := b.opts.MaxAge
		if maxAge == 0 {
			maxAge = context.SetCookieKVExpiration // 1 year.
		}
		ctx.SetCookieKV(b.opts.MaxTriesCookie, strconv.Itoa(tries), context.CookieExpires(maxAge))
	}
}

func (b *BasicAuth) resetCurrentTries(ctx *context.Context) {
	sess := sessions.Get(ctx)
	if sess != nil {
		sess.Delete(b.opts.MaxTriesCookie)
	} else {
		ctx.RemoveCookie(b.opts.MaxTriesCookie)
	}
}

// serveHTTP is the main method of this middleware,
// checks and verifies the auhorization header for basic authentication,
// next handlers will only be executed when the client is allowed to continue.
func (b *BasicAuth) serveHTTP(ctx *context.Context) {
	header := ctx.GetHeader(b.authorizationHeader)
	fullUser, username, password, ok := decodeHeader(header)
	if !ok { // Header is malformed or missing.
		b.askForCredentials(ctx)
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
				b.forbidden(ctx) // a user was forbidden, to reset its status should clear the Authorization header and cookie and request the resource again.
				return
			}
		}

		b.askForCredentials(ctx)
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
				b.askForCredentials(ctx)
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
		// not implement those two, then extract from the request header:
		header := ctx.GetHeader(b.authorizationHeader)
		fullUser, username, password, ok = decodeHeader(header)
	}

	if ok { // If it's authorized then try to lock and delete.
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

// gc removes all entries expired based on the max age or all entries (if max age is missing).
func (b *BasicAuth) gc() int {
	now := time.Now()
	var markedForDeletion []string

	b.mu.RLock()
	for fullUser, expiresAt := range b.credentials {
		if expiresAt == nil {
			markedForDeletion = append(markedForDeletion, fullUser)
		} else if expiresAt.Before(now) {
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
