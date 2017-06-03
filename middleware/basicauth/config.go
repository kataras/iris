// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package basicauth

import (
	"time"

	"github.com/kataras/iris/context"
)

const (
	// DefaultBasicAuthRealm is "Authorization Required"
	DefaultBasicAuthRealm = "Authorization Required"
	// DefaultBasicAuthContextKey is the "auth"
	// this key is used to do context.Set("user", theUsernameFromBasicAuth)
	DefaultBasicAuthContextKey = "user"
)

// DefaultExpireTime zero time
var DefaultExpireTime time.Time // 0001-01-01 00:00:00 +0000 UTC

// Config the configs for the basicauth middleware
type Config struct {
	// Users a map of login and the value (username/password)
	Users map[string]string
	// Realm http://tools.ietf.org/html/rfc2617#section-1.2. Default is "Authorization Required"
	Realm string
	// ContextKey the key for ctx.GetString(...). Default is 'user'
	ContextKey string
	// Expires expiration duration, default is 0 never expires
	Expires time.Duration
}

// DefaultConfig returns the default configs for the BasicAuth middleware
func DefaultConfig() Config {
	return Config{make(map[string]string), DefaultBasicAuthRealm, DefaultBasicAuthContextKey, 0}
}

// User returns the user from context key same as 'ctx.GetString("user")' but cannot be used by the developer, this is only here in order to understand how you can get the authenticated username
func (c Config) User(ctx context.Context) string {
	return ctx.Values().GetString(c.ContextKey)
}
