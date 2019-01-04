package basicauth

import (
	"time"

	"github.com/kataras/iris/context"
)

const (
	// DefaultBasicAuthRealm is "Authorization Required"
	DefaultBasicAuthRealm = "Authorization Required"
)

// DefaultExpireTime zero time
var DefaultExpireTime time.Time // 0001-01-01 00:00:00 +0000 UTC

// Config the configs for the basicauth middleware
type Config struct {
	// Users a map of login and the value (username/password)
	Users map[string]string
	// Realm http://tools.ietf.org/html/rfc2617#section-1.2. Default is "Authorization Required"
	Realm string
	// Expires expiration duration, default is 0 never expires
	Expires time.Duration

	// OnAsk fires each time the server asks to the client for credentials in order to gain access and continue to the next handler.
	//
	// You could also ignore this option and
	// - just add a listener for unauthorized status codes with:
	// `app.OnErrorCode(iris.StatusUnauthorized, unauthorizedWantsAccessHandler)`
	// - or register a middleware which will force `ctx.Next/or direct call`
	// the basicauth middleware and check its `ctx.GetStatusCode()`.
	//
	// However, this option is very useful when you want the framework to fire a handler
	// ONLY when the Basic Authentication sends an `iris.StatusUnauthorized`,
	// and free the error code listener to catch other types of unauthorized access, i.e Kerberos.
	// Also with this one, not recommended at all but, you are able to "force-allow" other users by calling the `ctx.StatusCode` inside this handler;
	// i.e when it is possible to create authorized users dynamically but
	// if that is the case then you should go with something like sessions instead of basic authentication.
	//
	// Usage: basicauth.New(basicauth.Config{..., OnAsk: unauthorizedWantsAccessViaBasicAuthHandler})
	//
	// Defaults to nil.
	OnAsk context.Handler
}

// DefaultConfig returns the default configs for the BasicAuth middleware
func DefaultConfig() Config {
	return Config{make(map[string]string), DefaultBasicAuthRealm, 0, nil}
}

// User returns the user from context key same as  ctx.Request().BasicAuth().
func (c Config) User(ctx context.Context) (string, string, bool) {
	return ctx.Request().BasicAuth()
}
