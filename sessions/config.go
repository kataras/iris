package sessions

import (
	"time"

	"github.com/kataras/iris/v12/context"

	"github.com/google/uuid"
	"github.com/kataras/golog"
)

const (
	// DefaultCookieName the secret cookie's name for sessions
	DefaultCookieName = "irissessionid"
)

type (
	// Config is the configuration for sessions. Please read it before using sessions.
	Config struct {
		// Logger instance for sessions usage, e.g. { Logger: app.Logger() }.
		// Defaults to a child of "sessions" of the latest Iris Application's main Logger.
		Logger *golog.Logger
		// Cookie string, the session's client cookie name, for example: "mysessionid"
		//
		// Defaults to "irissessionid".
		Cookie string

		// CookieSecureTLS set to true if server is running over TLS
		// and you need the session's cookie "Secure" field to be set true.
		// Defaults to false.
		CookieSecureTLS bool

		// AllowReclaim will allow to
		// Destroy and Start a session in the same request handler.
		// All it does is that it removes the cookie for both `Request` and `ResponseWriter` while `Destroy`
		// or add a new cookie to `Request` while `Start`.
		//
		// Defaults to false.
		AllowReclaim bool

		// Encoding should encodes and decodes
		// authenticated and optionally encrypted cookie values.
		//
		// Defaults to nil.
		Encoding context.SecureCookie

		// Expires the duration of which the cookie must expires (created_time.Add(Expires)).
		// If you want to delete the cookie when the browser closes, set it to -1.
		// However, if you use a database storage setting this value to -1 may
		// cause you problems because of the fact that the database
		// may has its own expiration mechanism and value will be expired and removed immediately.
		//
		// 0 means no expire, (24 years)
		// -1 means when browser closes
		// > 0 is the time.Duration which the session cookies should expire.
		//
		// Defaults to infinitive/unlimited life duration(0).
		Expires time.Duration

		// SessionIDGenerator can be set to a function which
		// return a unique session id.
		// By default we will use a uuid impl package to generate
		// that, but developers can change that with simple assignment.
		SessionIDGenerator func(ctx *context.Context) string

		// DisableSubdomainPersistence set it to true in order dissallow your subdomains to have access to the session cookie
		//
		// Defaults to false.
		DisableSubdomainPersistence bool
	}
)

// Validate corrects missing fields configuration fields and returns the right configuration
func (c Config) Validate() Config {
	if c.Logger == nil {
		c.Logger = context.DefaultLogger("sessions")
	}

	if c.Cookie == "" {
		c.Cookie = DefaultCookieName
	}

	if c.SessionIDGenerator == nil {
		c.SessionIDGenerator = func(ctx *context.Context) string {
			id, err := uuid.NewRandom()
			if err != nil {
				ctx.StopWithError(400, err)
				return ""
			}

			return id.String()
		}
	}

	return c
}
