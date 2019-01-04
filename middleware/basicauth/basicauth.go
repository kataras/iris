// Package basicauth provides http basic authentication via middleware. See _examples/authentication/basicauth
package basicauth

// test file: ../../_examples/authentication/basicauth/main_test.go

import (
	"encoding/base64"
	"strconv"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

type (
	encodedUser struct {
		HeaderValue string
		Username    string
		logged      bool
		expires     time.Time
	}
	encodedUsers []encodedUser

	basicAuthMiddleware struct {
		config Config
		// these are filled from the config.Users map at the startup
		auth             encodedUsers
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

	b := &basicAuthMiddleware{config: config}
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
	b.auth = make(encodedUsers, 0, len(b.config.Users))

	for k, v := range b.config.Users {
		fullUser := k + ":" + v
		header := "Basic " + base64.StdEncoding.EncodeToString([]byte(fullUser))
		b.auth = append(b.auth, encodedUser{HeaderValue: header, Username: k, logged: false, expires: DefaultExpireTime})
	}

	// set the auth realm header's value
	b.realmHeaderValue = "Basic realm=" + strconv.Quote(b.config.Realm)

	b.expireEnabled = b.config.Expires > 0
	b.askHandlerEnabled = b.config.OnAsk != nil
}

func (b *basicAuthMiddleware) findAuth(headerValue string) (auth *encodedUser, found bool) {
	if len(headerValue) == 0 {
		return
	}

	for _, user := range b.auth {
		if user.HeaderValue == headerValue {
			auth = &user
			found = true
			break
		}
	}

	return
}

func (b *basicAuthMiddleware) askForCredentials(ctx context.Context) {
	ctx.Header("WWW-Authenticate", b.realmHeaderValue)
	ctx.StatusCode(iris.StatusUnauthorized)
	if b.askHandlerEnabled {
		b.config.OnAsk(ctx)
	}
}

// Serve the actual middleware
func (b *basicAuthMiddleware) Serve(ctx context.Context) {

	auth, found := b.findAuth(ctx.GetHeader("Authorization"))
	if !found {
		b.askForCredentials(ctx)
		ctx.StopExecution()
		return
		// don't continue to the next handler
	}
	// all ok
	if b.expireEnabled {
		if auth.logged == false {
			auth.expires = time.Now().Add(b.config.Expires)
			auth.logged = true
		}

		if time.Now().After(auth.expires) {
			b.askForCredentials(ctx) // ask for authentication again
			ctx.StopExecution()
			return
		}
	}
	ctx.Next() // continue
}
