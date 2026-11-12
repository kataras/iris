# Basic Authentication Middleware for Iris

This middleware puts HTTP Basic Authentication ([RFC 7617](https://tools.ietf.org/html/rfc7617)) in front of a route, a router group, or a whole [Iris](https://www.iris-go.com) application. Users can come from an in-memory map, a JSON or YAML file, or your own database lookup. The middleware is generic over the user type: whatever `Allow` returns is what `basicauth.GetUser[U](ctx)` gives back to the handlers, and `ctx.User()` sees the same value as a `context.User`. Passwords can be stored as bcrypt hashes, credentials expire after `MaxAge`, repeated failures can be capped with `MaxTries`, and you can replace the challenge response with your own error handler.

## Installation

```go
import "github.com/kataras/iris/v14/middleware/basicauth"
```

## Getting started

`basicauth.Default` builds the middleware from a fixed `username: password` map, which is what you want for an admin group, an internal dashboard or a test. It returns a plain `iris.Handler`, so it protects exactly the router it is mounted on.

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/basicauth"
)

func main() {
	app := iris.New()

	auth := basicauth.Default(map[string]string{
		"admin":  "admin",
		"makis":  "makis_pass",
		"george": "george_pass",
	})

	admin := app.Router("/admin", auth)
	admin.Get("/", func(ctx iris.Context) {
		username, _ := ctx.User().GetUsername()
		ctx.Writef("Hello, %s!", username)
	})

	app.Listen(":8080")
}
```

A request without credentials gets a 401 with `WWW-Authenticate: Basic realm="Authorization Required"`, which is what makes browsers show the login prompt. `app.Use(auth)` or `app.UseRelevant(auth)` guards the whole application instead of one group.

The user `Default` stores is a `*iris.SimpleUser` with `ID` and `Username` set to the username, `Authorization` set to `"Basic Authentication"` and `Password` set to what the client sent. Read it with `ctx.User()` or, typed, with `basicauth.GetUser[*iris.SimpleUser](ctx)`. Never write that value to the response as it is; pick the fields you mean to expose.

## Your own user type

`basicauth.New` takes an `Options[U]` where `U` is your user type. The only required field is `Allow`, the function that checks the credentials and returns the user.

```go
type Operator struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

var operators = []Operator{
	{"admin", "admin", []string{"admin"}},
	{"makis", "makis_pass", []string{"manager", "author"}},
}

auth := basicauth.New(basicauth.Options[Operator]{
	Realm:  basicauth.DefaultRealm,
	MaxAge: 10 * time.Minute,
	Allow:  basicauth.AllowUsers(operators),
})

app.Get("/admin", auth, func(ctx iris.Context) {
	op, _ := basicauth.GetUser[Operator](ctx) // op is an Operator, no type assertion.
	ctx.JSON(iris.Map{"username": op.Username, "roles": op.Roles})
})
```

`AllowUsers` reads the username and password out of each element once, at startup: from the `basicauth.User` interface (`GetUsername() string` and `GetPassword() string`) when the type implements it, otherwise from the `Username` and `Password` struct fields (json tags count, embedded structs are searched too), or from `"username"` and `"password"` keys when the element is a map. For anything else pass the `basicauth.Credentials` option and tell it where to look:

```go
basicauth.AllowUsers(members, basicauth.Credentials(func(m Member) (string, string) {
	return m.Email, m.PasswordHash
}), basicauth.BCRYPT)
```

A request then costs one map lookup and one password comparison.

For a database, write `Allow` yourself. It receives the request context, so the lookup can carry the request's deadline:

```go
auth := basicauth.New(basicauth.Options[*User]{
	Realm: basicauth.DefaultRealm,
	Allow: func(ctx iris.Context, username, password string) (*User, bool) {
		user, err := db.getUserByUsernameAndPassword(ctx, username, password)
		return user, err == nil
	},
})
```

`ctx.User()` keeps working for any `U`: a value that does not implement `context.User` is wrapped, so `GetRaw()` returns your value and the getters your type does not provide answer `context.ErrNotSupported`.

## Users from a file

`basicauth.Load` reads the list from a JSON or YAML file and stores each entry as a `basicauth.Map` (`map[string]any`), so extra fields such as a role survive. `AllowUsersFile[U]` decodes the same file straight into your own type.

```go
auth := basicauth.Load("users.yml", basicauth.BCRYPT)

auth := basicauth.New(basicauth.Options[Operator]{
	Allow: basicauth.AllowUsersFile[Operator]("users.yml", basicauth.BCRYPT),
})
```

The file holds either a list of users or the short `username: password` form:

```yaml
- username: kataras
  password: $2a$10$...
  role: admin
- username: makis
  password: $2a$10$...
```

```yaml
kataras: $2a$10$...
makis: $2a$10$...
```

Set the `basicauth.ReadFile` variable to read the file from somewhere other than the disk, an `embed.FS` for example. `Load` and `AllowUsersFile` panic when the file cannot be read or decoded, since a misconfigured user list should stop the application at startup.

## BCRYPT

Pass `basicauth.BCRYPT` to any of `Default`, `Load`, `AllowUsers`, `AllowUsersMap` and `AllowUsersFile` to compare the client's password against bcrypt hashes instead of plaintext. Bcrypt is deliberately slow to compute, which is what makes a stolen hash expensive to brute-force. Generate the hashes with `golang.org/x/crypto/bcrypt`.

## Options

| Field | What it does |
|-------|--------------|
| `Realm` | The `realm` of the `WWW-Authenticate: Basic` challenge. `Default` and `Load` use `basicauth.DefaultRealm`. |
| `Proxy` | Challenge with 407 and `Proxy-Authenticate`, read the credentials from `Proxy-Authorization`. |
| `HTTPSOnly` | Reject any request that did not arrive over TLS with a 505. HTTPS over HTTP/1.1 counts, and so does a plaintext hop behind a proxy when the application's SSL proxy headers are configured (`ctx.IsSSL()`). |
| `MaxAge` | How long a successful login stays valid. Counted from the first login, not refreshed by later requests. After that the client is challenged again. |
| `MaxTries` | Answer 403 after this many failed logins. The counter is kept on the server per client address (`ctx.RemoteAddr()`, which honours the configured trusted proxies) and username. A cookie mirrors it for browsers but is not trusted. |
| `MaxTriesCookie` | Name of that cookie. Defaults to `basicmaxtries`. |
| `ErrorHandler` | Receives every failure as a typed error (`ErrCredentialsMissing`, `ErrCredentialsInvalid`, `ErrCredentialsExpired`, `ErrCredentialsForbidden`, `ErrHTTPVersion`). The default writes the matching status and, for the first three, the `WWW-Authenticate` challenge. |
| `GC` | A ticker that drops expired entries from memory every `GC.Every`, cancellable through `GC.Context`. Without `MaxAge` every entry counts as expired. |

## Custom error handling

Render the failure through `github.com/kataras/iris/v14/rest/errors`: the error code carries the 401, so the handler writes no status code of its own and the body is the same envelope the rest of the application sends. Use `ErrPrivate`, not `Err`: the error the middleware hands you carries the attempted credentials in its fields (see `ErrCredentialsInvalid`), and `Err` would marshal them into the response. `ErrPrivate` keeps the reason in the server log and sends only the generic message. A custom handler also replaces `DefaultErrorHandler`, which is what sets the `WWW-Authenticate` challenge, so set it yourself if browsers should still prompt.

```go
auth := basicauth.New(basicauth.Options[Operator]{
	Allow: basicauth.AllowUsers(operators),
	ErrorHandler: func(ctx iris.Context, err error) {
		ctx.SetHeader("WWW-Authenticate", `Basic realm="Authorization Required"`)
		errors.Unauthenticated.ErrPrivate(ctx, err)
	},
})
```

## Logout and authorization time

`ctx.Logout()` forgets the stored login, removes the user from the request and strips the `Authorization` header, so `ctx.User()` is nil for the rest of the handler and the client is challenged on its next request. The response status becomes 401.

`basicauth.AuthorizedAt(ctx)` returns when the current credentials were first accepted. It reports false without `MaxAge`, because the middleware keeps no per-login time when nothing expires. The `*iris.SimpleUser` that `Default` builds carries the same value in `AuthorizedAt`.

## Security

Passwords are compared in constant time, and an unknown username is compared against a decoy so the response takes the same time whether or not the user exists. With `BCRYPT` the decoy is a real bcrypt hash. Logged-in credentials are kept in memory as a SHA-256 digest of the `username:password` pair, never as cleartext. The `MaxTries` lockout is enforced on the server, so a client that drops the cookie is still counted, and the cookie itself is `HttpOnly`, `SameSite=Lax` and `Secure` over TLS. Error messages name the user but never contain the password or the raw header.

Basic authentication itself sends the password with every request, base64-encoded but not encrypted. Serve it over TLS. `HTTPSOnly` enforces that at the middleware. `basicauth.Default(map[string]string{"admin": "pass"})` with plaintext credentials in source is for local development only.

## Testing

Handlers behind the middleware are tested through the `httptest` package with `WithBasicAuth`:

```go
package main_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/basicauth"
)

func TestIndexHandler(t *testing.T) {
	app := iris.New()
	app.Get("/", basicauth.New(basicauth.Options[Operator]{
		Realm: basicauth.DefaultRealm,
		Allow: basicauth.AllowUsers(operators),
	}), index)

	e := httptest.New(t, app)

	e.GET("/").WithBasicAuth("admin", "admin").Expect().Status(httptest.StatusOK).JSON().Object().Value("username").IsEqual("admin")
	e.GET("/").WithBasicAuth("admin", "wrongpassword").Expect().Status(httptest.StatusUnauthorized)
}
```

The full example lives in [_examples/09-authentication/basicauth](../../_examples/09-authentication/basicauth/main.go).
