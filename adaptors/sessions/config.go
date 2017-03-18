package sessions

import (
	"time"
)

const (
	// DefaultCookieName the secret cookie's name for sessions
	DefaultCookieName = "irissessionid"
	// DefaultCookieLength is the default Session Manager's CookieLength, which is 32
	DefaultCookieLength = 32
)

type (
	// Config is the configuration for sessions
	// has 5 fields
	// first is the cookieName, the session's name (string) ["mysessionsecretcookieid"]
	// second enable if you want to decode the cookie's key also
	// third is the time which the client's cookie expires
	// forth is the cookie length (sessionid) int, defaults to 32, do not change if you don't have any reason to do
	// fifth is the DisableSubdomainPersistence which you can set it to true in order dissallow your q subdomains to have access to the session cook
	Config struct {
		// Cookie string, the session's client cookie name, for example: "mysessionid"
		//
		// Defaults to "irissessionid"
		Cookie string

		// Encode the cookie value if not nil.
		// Should accept as first argument the cookie name (config.Name)
		//         as second argument the server's generated session id.
		// Should return the new session id, if error the session id setted to empty which is invalid.
		//
		// Note: Errors are not printed, so you have to know what you're doing,
		// and remember: if you use AES it only supports key sizes of 16, 24 or 32 bytes.
		// You either need to provide exactly that amount or you derive the key from what you type in.
		//
		// Defaults to nil
		Encode func(cookieName string, value interface{}) (string, error)
		// Decode the cookie value if not nil.
		// Should accept as first argument the cookie name (config.Name)
		//               as second second accepts the client's cookie value (the encoded session id).
		// Should return an error if decode operation failed.
		//
		// Note: Errors are not printed, so you have to know what you're doing,
		// and remember: if you use AES it only supports key sizes of 16, 24 or 32 bytes.
		// You either need to provide exactly that amount or you derive the key from what you type in.
		//
		// Defaults to nil
		Decode func(cookieName string, cookieValue string, v interface{}) error

		// Expires the duration of which the cookie must expires (created_time.Add(Expires)).
		// If you want to delete the cookie when the browser closes, set it to -1.
		//
		// 0 means no expire, (24 years)
		// -1 means when browser closes
		// > 0 is the time.Duration which the session cookies should expire.
		//
		// Defaults to infinitive/unlimited life duration(0)
		Expires time.Duration

		// CookieLength the length of the sessionid's cookie's value, let it to 0 if you don't want to change it
		//
		// Defaults to 32
		CookieLength int

		// DisableSubdomainPersistence set it to true in order dissallow your q subdomains to have access to the session cookie
		//
		// Defaults to false
		DisableSubdomainPersistence bool
	}
)

// Validate corrects missing fields configuration fields and returns the right configuration
func (c Config) Validate() Config {

	if c.Cookie == "" {
		c.Cookie = DefaultCookieName
	}

	if c.CookieLength <= 0 {
		c.CookieLength = DefaultCookieLength
	}

	return c
}
