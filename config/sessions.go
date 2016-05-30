package config

import (
	"time"

	"github.com/imdario/mergo"
)

var (
	universe time.Time // 0001-01-01 00:00:00 +0000 UTC
	// CookieExpireNever the default cookie's life for sessions, unlimited
	CookieExpireNever = universe
)

const (
	// DefaultCookieName the secret cookie's name for sessions
	DefaultCookieName        = "irissessionid"
	DefaultSessionGcDuration = time.Duration(2) * time.Hour
	// DefaultRedisNetwork the redis network option, "tcp"
	DefaultRedisNetwork = "tcp"
	// DefaultRedisAddr the redis address option, "127.0.0.1:6379"
	DefaultRedisAddr = "127.0.0.1:6379"
	// DefaultRedisIdleTimeout the redis idle timeout option, time.Duration(5) * time.Minute
	DefaultRedisIdleTimeout = time.Duration(5) * time.Minute
	// DefaultRedisMaxAgeSeconds the redis storage last parameter (SETEX), 31556926.0 (1 year)
	DefaultRedisMaxAgeSeconds = 31556926.0 //1 year

)

type (

	// Redis the redis configuration used inside sessions
	Redis struct {
		// Network "tcp"
		Network string
		// Addr "127.0.01:6379"
		Addr string
		// Password string .If no password then no 'AUTH'. Default ""
		Password string
		// If Database is empty "" then no 'SELECT'. Default ""
		Database string
		// MaxIdle 0 no limit
		MaxIdle int
		// MaxActive 0 no limit
		MaxActive int
		// IdleTimeout  time.Duration(5) * time.Minute
		IdleTimeout time.Duration
		// Prefix "myprefix-for-this-website". Default ""
		Prefix string
		// MaxAgeSeconds how much long the redis should keep the session in seconds. Default 31556926.0 (1 year)
		MaxAgeSeconds int
	}

	// Sessions the configuration for sessions
	// has 4 fields
	// first is the providerName (string) ["memory","redis"]
	// second is the cookieName, the session's name (string) ["mysessionsecretcookieid"]
	// third is the time which the client's cookie expires
	// forth is the gcDuration (time.Duration) when this time passes it removes the unused sessions from the memory until the user come back
	Sessions struct {
		// Provider string, usage iris.Config().Provider = "memory" or "redis". If you wan to customize redis then import the package, and change it's config
		Provider string
		// Cookie string, the session's client cookie name, for example: "irissessionid"
		Cookie string
		//Expires the date which the cookie must expires. Default infinitive/unlimited life
		Expires time.Time
		// GcDuration every how much duration(GcDuration) the memory should be clear for unused cookies (GcDuration)
		// for example: time.Duration(2)*time.Hour. it will check every 2 hours if cookie hasn't be used for 2 hours,
		// deletes it from memory until the user comes back, then the session continue to work as it was
		//
		// Default 2 hours
		GcDuration time.Duration
	}
)

// DefaultSessions the default configs for Sessions
func DefaultSessions() Sessions {
	return Sessions{
		Provider:   "memory", // the default provider is "memory", if you set it to ""  means that sessions are disabled.
		Cookie:     DefaultCookieName,
		Expires:    CookieExpireNever,
		GcDuration: DefaultSessionGcDuration,
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

// Merge MergeSingle the default with the given config and returns the result
func (c Sessions) MergeSingle(cfg Sessions) (config Sessions) {

	config = cfg
	mergo.Merge(&config, c)

	return
}

// DefaultRedis returns the default configuration for Redis service
func DefaultRedis() Redis {
	return Redis{
		Network:       DefaultRedisNetwork,
		Addr:          DefaultRedisAddr,
		Password:      "",
		Database:      "",
		MaxIdle:       0,
		MaxActive:     0,
		IdleTimeout:   DefaultRedisIdleTimeout,
		Prefix:        "",
		MaxAgeSeconds: DefaultRedisMaxAgeSeconds,
	}
}

// Merge merges the default with the given config and returns the result
func (c Redis) Merge(cfg []Redis) (config Redis) {

	if len(cfg) > 0 {
		config = cfg[0]
		mergo.Merge(&config, c)
	} else {
		_default := c
		config = _default
	}

	return
}

// Merge MergeSingle the default with the given config and returns the result
func (c Redis) MergeSingle(cfg Redis) (config Redis) {

	config = cfg
	mergo.Merge(&config, c)

	return
}
