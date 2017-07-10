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
}

// DefaultConfig returns the default configs for the BasicAuth middleware
func DefaultConfig() Config {
	return Config{make(map[string]string), DefaultBasicAuthRealm, 0}
}

// User returns the user from context key same as  ctx.Request().BasicAuth().
func (c Config) User(ctx context.Context) (string, string, bool) {
	return ctx.Request().BasicAuth()
}
