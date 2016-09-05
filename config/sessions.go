package config

import (
	"github.com/imdario/mergo"
	"github.com/kataras/go-sessions"
	"time"
)

var (
	universe time.Time // 0001-01-01 00:00:00 +0000 UTC
	// CookieExpireNever the default cookie's life for sessions, unlimited (23 years)
	CookieExpireNever = time.Now().AddDate(23, 0, 0)
)

const (
	// DefaultCookieName the secret cookie's name for sessions
	DefaultCookieName = "irissessionid"
	// DefaultSessionGcDuration  is the default Session Manager's GCDuration , which is 2 hours
	DefaultSessionGcDuration = time.Duration(2) * time.Hour
	// DefaultCookieLength is the default Session Manager's CookieLength, which is 32
	DefaultCookieLength = 32
)

// Sessions the configuration for sessions
// has 6 fields
// first is the cookieName, the session's name (string) ["mysessionsecretcookieid"]
// second enable if you want to decode the cookie's key also
// third is the time which the client's cookie expires
// forth is the cookie length (sessionid) int, defaults to 32, do not change if you don't have any reason to do
// fifth is the gcDuration (time.Duration) when this time passes it removes the unused sessions from the memory until the user come back
// sixth is the DisableSubdomainPersistence which you can set it to true in order dissallow your q subdomains to have access to the session cook
//
type Sessions sessions.Config

// DefaultSessions the default configs for Sessions
func DefaultSessions() Sessions {
	return Sessions{
		Cookie:                      DefaultCookieName,
		CookieLength:                DefaultCookieLength,
		DecodeCookie:                false,
		Expires:                     0,
		GcDuration:                  DefaultSessionGcDuration,
		DisableSubdomainPersistence: false,
	}
}

// Merge merges the default with the given config and returns the result
func (c Sessions) Merge(cfg []Sessions) (config Sessions) {

	if len(cfg) > 0 {
		config = cfg[0]
		mergo.Merge(&config, c)
	} else {
		_default := c
		config = _default
	}

	return
}

// MergeSingle merges the default with the given config and returns the result
func (c Sessions) MergeSingle(cfg Sessions) (config Sessions) {

	config = cfg
	mergo.Merge(&config, c)

	return
}
