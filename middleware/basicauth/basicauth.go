// Package basicauth provides http basic authentication via middleware. See _examples/auth/basicauth
package basicauth

// test file: ../../_examples/auth/basicauth/main_test.go

import (
	"encoding/base64"
	"strconv"
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"
)

func init() {
	context.SetHandlerName("iris/middleware/basicauth.*", "iris.basicauth")
}

type (
	encodedUser struct {
		HeaderValue string
		Username    string
		logged      bool
		forceLogout bool // in order to be able to invalidate and use a redirect response.
		expires     time.Time
		mu          sync.RWMutex
	}

	basicAuthMiddleware struct {
		config *Config
		// these are filled from the config.Users map at the startup
		auth             []*encodedUser
		realmHeaderValue string

		// The below can be removed but they are here because on the future we may add dynamic options for those two fields,
		// it is a bit faster to check the b.$bool as well.
		expireEnabled     bool // if the config.Expires is a valid date, default is disabled.
		askHandlerEnabled bool // if the config.OnAsk is not nil, defaults to false.
	}
)

//

// New accepts basicauth.Config and returns a new Handler
// which will ask the client for basic auth (username, password),
// validate that and if valid continues to the next handler, otherwise
// throws a StatusUnauthorized http error code.
func New(c Config) context.Handler {
	config := DefaultConfig()
	if c.Realm != "" {
		config.Realm = c.Realm
	}
	config.Users = c.Users
	config.Expires = c.Expires
	config.OnAsk = c.OnAsk

	b := &basicAuthMiddleware{config: &config}
	b.init()
	return b.Serve
}

// Default accepts only the users and returns a new Handler
// which will ask the client for basic auth (username, password),
// validate that and if valid continues to the next handler, otherwise
// throws a StatusUnauthorized http error code.
func Default(users map[string]string) context.Handler {
	c := DefaultConfig()
	c.Users = users
	return New(c)
}

func (b *basicAuthMiddleware) init() {
	// pass the encoded users from the user's config's Users value
	b.auth = make([]*encodedUser, 0, len(b.config.Users))

	for k, v := range b.config.Users {
		fullUser := k + ":" + v
		header := "Basic " + base64.StdEncoding.EncodeToString([]byte(fullUser))
		b.auth = append(b.auth, &encodedUser{HeaderValue: header, Username: k, logged: false, expires: DefaultExpireTime})
	}

	// set the auth realm header's value
	b.realmHeaderValue = "Basic realm=" + strconv.Quote(b.config.Realm)

	b.expireEnabled = b.config.Expires > 0
	b.askHandlerEnabled = b.config.OnAsk != nil
}

func (b *basicAuthMiddleware) findAuth(headerValue string) (*encodedUser, bool) {
	if headerValue != "" {
		for _, user := range b.auth {
			if user.HeaderValue == headerValue {
				return user, true
			}
		}
	}

	return nil, false
}

func (b *basicAuthMiddleware) askForCredentials(ctx *context.Context) {
	ctx.Header("WWW-Authenticate", b.realmHeaderValue)
	ctx.StatusCode(401)
	if b.askHandlerEnabled {
		b.config.OnAsk(ctx)
	}
}

// Serve the actual middleware
func (b *basicAuthMiddleware) Serve(ctx *context.Context) {
	auth, found := b.findAuth(ctx.GetHeader("Authorization"))
	if !found || auth.forceLogout {
		if auth != nil {
			auth.mu.Lock()
			auth.forceLogout = false
			auth.mu.Unlock()
		}

		b.askForCredentials(ctx)
		ctx.StopExecution()
		return
		// don't continue to the next handler
	}

	// all ok
	if b.expireEnabled {
		if !auth.logged {
			auth.mu.Lock()
			auth.expires = time.Now().Add(b.config.Expires)
			auth.logged = true
			auth.mu.Unlock()
		}

		auth.mu.RLock()
		expired := time.Now().After(auth.expires)
		auth.mu.RUnlock()
		if expired {
			auth.mu.Lock()
			auth.logged = false
			auth.mu.Unlock()
			b.askForCredentials(ctx) // ask for authentication again
			ctx.StopExecution()
			return
		}
	}

	if !b.config.DisableLogoutFunc {
		ctx.SetLogoutFunc(b.Logout)
	}

	ctx.Next() // continue
}

// Logout sends a 401 so the browser/client can invalidate the
// Basic Authentication and also sets the underline user's logged field to false,
// so its expiration resets when re-ask for credentials.
//
// End-developers should call the `Context.Logout()` method
// to fire this method as this structure is hidden.
func (b *basicAuthMiddleware) Logout(ctx *context.Context) {
	ctx.StatusCode(401)
	if auth, found := b.findAuth(ctx.GetHeader("Authorization")); found {
		auth.mu.Lock()
		auth.logged = false
		auth.forceLogout = true
		auth.mu.Unlock()
	}
}
