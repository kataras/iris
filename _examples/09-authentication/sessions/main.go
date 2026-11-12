// Typed sessions with the v14 secure defaults.
//
// sessions.Handler[T] loads (or creates) one session per request and commits it
// automatically afterwards, unless a handler already committed or the response
// is a 4xx/5xx. Handlers read and write it through sessions.Get[T](ctx), so the
// payload stays a plain Go struct instead of a bag of any values.
//
// sessions.NewOptions sets Path "/", MaxAge 24h, SameSite=Lax and Secure=true;
// session cookies are always HttpOnly. Secure means the browser will NOT send
// the cookie over plain HTTP, including http://localhost, so this example
// sets opts.Secure = false to stay usable with curl, exactly as MIGRATION.md
// describes. Remove that line before deploying.
//
// Learn: keep one typed struct per visitor in an encrypted cookie, guard routes
// on it, and rotate the session id after sign-in.
// Before: ../jwt-middleware
//
// Run:
//
//	go run main.go
//
// Try (curl's -c/-b flags keep the cookie jar between the calls):
//
//	curl -i -c jar.txt -b jar.txt -X POST -H "Content-Type: application/json" \
//	  http://localhost:8080/signin -d '{"username":"maria","password":"maria_password"}'   # 200, Set-Cookie: app_session
//	curl -i -c jar.txt -b jar.txt http://localhost:8080/profile                   # 200, the UserSession data
//	curl -s -c jar.txt -b jar.txt http://localhost:8080/count                     # 200, views 1, then 2, then 3...
//	curl -i -c jar.txt -b jar.txt -X POST http://localhost:8080/signout           # 200, the cookie expires
//	curl -i -c jar.txt -b jar.txt http://localhost:8080/profile                   # 401, signed out
//
// Next: ../auth-sdk
package main

import (
	"crypto/subtle"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/sessions"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	// Persist these keys (environment, secret manager): regenerating them
	// invalidates every existing session. Hash key: 32 or 64 bytes, authenticates
	// the cookie. Block key: 16, 24 or 32 bytes, encrypts its contents.
	hashKey := []byte("hash-key-32-characters-long-str!")
	blockKey := []byte("block-key-24-characters!")

	opts := sessions.NewOptions(hashKey, blockKey)
	opts.MaxAge = 8 * time.Hour
	// Secure=true, SameSite=Lax and HttpOnly are the v14 defaults. This next
	// line relaxes Secure so the cookie also travels over plain HTTP, which is
	// what makes the curl commands above work on localhost. Delete it before
	// you deploy: a Secure cookie still works behind a TLS-terminating proxy,
	// because the browser connection is HTTPS even when the app sees HTTP.
	opts.Secure = false

	// The cookie database keeps the whole payload in the encrypted, authenticated
	// cookie: zero infrastructure, but nothing to revoke server-side. For
	// revocable sessions swap in sessions.NewRedisDatabase[UserSession] or
	// sessions.NewPostgresDatabase[UserSession]. Only an opaque ID then travels
	// in the cookie and every handler below stays exactly the same.
	db := sessions.NewCookieDatabase[UserSession](opts)

	app := iris.New()

	// One registration covers the routes of this router.
	site := app.Router("/", sessions.Handler(db, "app_session"))

	site.Post("/signin", signin)
	site.Post("/signout", signout)
	site.Get("/profile", requireUser, profile)
	site.Get("/count", requireUser, count)

	app.Listen(":8080")
}

// UserSession is the typed session payload. It is gob-encoded into the cookie,
// so keep it small: cookies cap out at about 4KB.
type UserSession struct {
	UserID   string
	Username string
	Roles    []string
	SigninAt time.Time
	Views    int
}

// SigninRequest is the /signin payload. ctx.BindJSON needs a named type to
// instantiate, which is also where a Validate method would go.
type SigninRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func signin(ctx iris.Context) {
	req, ok := ctx.BindJSON[SigninRequest]()
	if !ok {
		return // 400 already written for a malformed body.
	}

	// Stands in for a database lookup plus a bcrypt comparison.
	if req.Username != "maria" ||
		subtle.ConstantTimeCompare([]byte(req.Password), []byte("maria_password")) != 1 {
		errors.Unauthenticated.Message(ctx, "invalid credentials")
		return
	}

	session := sessions.Get[UserSession](ctx)
	session.Set(UserSession{
		UserID:   "1",
		Username: req.Username,
		Roles:    []string{"member"},
		SigninAt: time.Now(),
	})

	// Session fixation defense: rotate the identifier now that the session is
	// privileged, so an identifier an attacker planted in the victim's browser
	// before sign-in cannot be reused. Set the authenticated data first:
	// RegenerateID persists the session itself, so no extra Commit is needed.
	//
	// With the cookie database there is no server-side identifier, so this is a
	// no-op; with a Redis or Postgres store it deletes the record kept under the
	// old ID and re-issues the cookie under a fresh one.
	err := session.RegenerateID(ctx)

	// ctx.OK carries the error branch: the mapped error response on failure,
	// the body on success.
	ctx.OK(iris.Map{"message": "signed in", "username": req.Username}, err)
}

func signout(ctx iris.Context) {
	session := sessions.Get[UserSession](ctx)
	// Delete expires the cookie and removes the record from a server-side store.
	err := session.Delete(ctx)

	ctx.OK(iris.Map{"message": "signed out"}, err)
}

// requireUser guards the routes below it: no session data, no access.
func requireUser(ctx iris.Context) {
	session := sessions.Get[UserSession](ctx)
	if session == nil || session.Data.UserID == "" {
		errors.Unauthenticated.Message(ctx, "sign in required")
		return
	}

	ctx.Next()
}

func profile(ctx iris.Context) {
	session := sessions.Get[UserSession](ctx)
	ctx.JSON(session.Data)
}

// count shows a write: mutate a copy of the data and Set it back. The Handler
// commits the session after this handler returns, so the updated cookie goes
// out with the response.
func count(ctx iris.Context) {
	session := sessions.Get[UserSession](ctx)

	data := session.Data
	data.Views++
	session.Set(data)

	ctx.JSON(iris.Map{"views": data.Views})
}
