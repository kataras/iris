package basicauth

import (
	"encoding/base64"
	"strconv"

	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
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
		config config.BasicAuth
		// these are filled from the config.Users map at the startup
		auth             encodedUsers
		realmHeaderValue string
		expireEnabled    bool // if the config.Expires is a valid date, default disabled
	}
)

//

// New takes one parameter, the config.BasicAuth returns a HandlerFunc
// use: iris.UseFunc(New(...)), iris.Get(...,New(...),...)
func New(c config.BasicAuth) iris.HandlerFunc {
	return NewHandler(c).Serve
}

// NewHandler takes one parameter, the config.BasicAuth returns a Handler
// use: iris.Use(NewHandler(...)), iris.Get(...,iris.HandlerFunc(NewHandler(...)),...)
func NewHandler(c config.BasicAuth) iris.Handler {
	b := &basicAuthMiddleware{config: config.DefaultBasicAuth().MergeSingle(c)}
	b.init()
	return b
}

// Default takes one parameter, the users returns a HandlerFunc
// use: iris.UseFunc(Default(...)), iris.Get(...,Default(...),...)
func Default(users map[string]string) iris.HandlerFunc {
	return DefaultHandler(users).Serve
}

// DefaultHandler takes one parameter, the users returns a Handler
// use: iris.Use(DefaultHandler(...)), iris.Get(...,iris.HandlerFunc(Default(...)),...)
func DefaultHandler(users map[string]string) iris.Handler {
	c := config.DefaultBasicAuth()
	c.Users = users
	return NewHandler(c)
}

//

func (b *basicAuthMiddleware) init() {
	// pass the encoded users from the user's config's Users value
	b.auth = make(encodedUsers, 0, len(b.config.Users))

	for k, v := range b.config.Users {
		fullUser := k + ":" + v
		header := "Basic " + base64.StdEncoding.EncodeToString([]byte(fullUser))
		b.auth = append(b.auth, encodedUser{HeaderValue: header, Username: k, logged: false, expires: config.CookieExpireNever})
	}

	// set the auth realm header's value
	b.realmHeaderValue = "Basic realm=" + strconv.Quote(b.config.Realm)

	if b.config.Expires > 0 {
		b.expireEnabled = true
	}
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

func (b *basicAuthMiddleware) askForCredentials(ctx *iris.Context) {
	ctx.SetHeader("WWW-Authenticate", b.realmHeaderValue)
	ctx.SetStatusCode(iris.StatusUnauthorized)
}

// Serve the actual middleware
func (b *basicAuthMiddleware) Serve(ctx *iris.Context) {

	if auth, found := b.findAuth(ctx.RequestHeader("Authorization")); !found {
		/* I spent time for nothing
		if b.banEnabled && auth != nil { // this propably never work

			if auth.tries == b.config.MaxTries {
				auth.bannedTime = time.Now()
				auth.unbanTime = time.Now().Add(b.config.BanDuration) // set the unban time
				auth.tries++                                          // we plus them in order to check if already banned later
				// client is banned send a forbidden status and don't continue
				ctx.SetStatusCode(iris.StatusForbidden)
				return
			} else if auth.tries > b.config.MaxTries { // it's already banned, so check the ban duration with the bannedTime
				if time.Now().After(auth.unbanTime) { // here we unban the client
					auth.tries = 0
					auth.bannedTime = config.CookieExpireNever
					auth.unbanTime = config.CookieExpireNever
					// continue and askCredentials as normal
				} else {
					// client is banned send a forbidden status and don't continue
					ctx.SetStatusCode(iris.StatusForbidden)
					return
				}

			}
		}
		if auth != nil {
			auth.tries++
		}*/

		b.askForCredentials(ctx)
		// don't continue to the next handler
	} else {
		// all ok set the context's value in order to be getable from the next handler
		ctx.Set(b.config.ContextKey, auth.Username)
		if b.expireEnabled {

			if auth.logged == false {
				auth.expires = time.Now().Add(b.config.Expires)
				auth.logged = true
			}

			if time.Now().Before(auth.expires) {
				b.askForCredentials(ctx) // ask for authentication again
				return
			}

		}

		//auth.tries = 0
		ctx.Next() // continue
	}

}
