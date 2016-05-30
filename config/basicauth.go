package config

import (
	"time"

	"github.com/imdario/mergo"
)

const (
	DefaultBasicAuthRealm      = "Authorization Required"
	DefaultBasicAuthContextKey = "auth"
)

type BasicAuth struct {
	// Users a map of login and the value (username/password)
	Users map[string]string
	// Realm http://tools.ietf.org/html/rfc2617#section-1.2. Default is "Authorization Required"
	Realm string
	// ContextKey the key for ctx.GetString(...). Default is 'auth'
	ContextKey string
	// Expires expiration duration, default is 0 never expires
	Expires time.Duration
}

// DefaultBasicAuth returns the default configs for the BasicAuth middleware
func DefaultBasicAuth() BasicAuth {
	return BasicAuth{make(map[string]string), DefaultBasicAuthRealm, DefaultBasicAuthContextKey, 0}
}

// Merge MergeSingle the default with the given config and returns the result
func (c BasicAuth) MergeSingle(cfg BasicAuth) (config BasicAuth) {

	config = cfg
	mergo.Merge(&config, c)

	return
}
