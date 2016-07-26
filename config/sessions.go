package config

import (
	"time"

	"github.com/imdario/mergo"
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
)

type (

	// Sessions the configuration for sessions
	// has 5 fields
	// first is the cookieName, the session's name (string) ["mysessionsecretcookieid"]
	// second enable if you want to decode the cookie's key also
	// third is the time which the client's cookie expires
	// forth is the gcDuration (time.Duration) when this time passes it removes the unused sessions from the memory until the user come back
	// fifth is the DisableSubdomainPersistence which you can set it to true in order dissallow your iris subdomains to have access to the session cook
	Sessions struct {
		// Cookie string, the session's client cookie name, for example: "irissessionid"
		Cookie string
		// DecodeCookie set it to true to decode the cookie key with base64 URLEncoding
		// Defaults to false
		DecodeCookie bool
		// Expires the duration of which the cookie must expires (created_time.Add(Expires)).
		// If you want to delete the cookie when the browser closes, set it to -1 but in this case, the server side's session duration is up to GcDuration
		//
		// Default infinitive/unlimited life duration(0)

		Expires time.Duration
		// GcDuration every how much duration(GcDuration) the memory should be clear for unused cookies (GcDuration)
		// for example: time.Duration(2)*time.Hour. it will check every 2 hours if cookie hasn't be used for 2 hours,
		// deletes it from backend memory until the user comes back, then the session continue to work as it was
		//
		// Default 2 hours
		GcDuration time.Duration

		// DisableSubdomainPersistence set it to true in order dissallow your iris subdomains to have access to the session cookie
		// defaults to false
		DisableSubdomainPersistence bool
	}
)

// DefaultSessions the default configs for Sessions
func DefaultSessions() Sessions {
	return Sessions{
		Cookie:                      DefaultCookieName,
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
