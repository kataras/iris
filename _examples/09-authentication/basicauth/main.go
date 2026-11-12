// HTTP Basic Authentication for an admin group.
//
// basicauth.Default takes a static user list and returns a plain iris.Handler,
// so it protects exactly the router it is mounted on: here /admin, while the
// root stays public. On success the authenticated user is stored on the request
// and read back with ctx.User(); a missing or wrong credential gets a 401 with
// the WWW-Authenticate challenge that makes browsers show the login prompt.
//
// basicauth.New is generic over your own user type: Options[Operator] with a
// []Operator list, and basicauth.GetUser[Operator](ctx) gives the struct back
// to the handler with no type assertion. That form guards /ops below.
//
// Basic auth sends username:password with EVERY request, base64-encoded but not
// encrypted, so serve it over HTTPS only. It fits internal dashboards and admin
// APIs; for user-facing sign-in use the auth SDK or sessions instead.
//
// Learn: protect one router with a static user list and read the authenticated
// user back with ctx.User() or the typed basicauth.GetUser[T].
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/admin/stats                          # 401 + WWW-Authenticate: Basic realm="Authorization Required"
//	curl -i -u admin:admin_password http://localhost:8080/admin/stats   # 200
//	curl -i -u ops:wrong-password http://localhost:8080/admin/stats     # 401
//	curl -i -u admin:admin_password http://localhost:8080/admin/whoami  # 200, the identity behind the credentials
//	curl -i -u makis:makis_password http://localhost:8080/ops/me        # 200, the typed Operator
//	curl -i http://localhost:8080/                                      # 200, public, no credentials needed
//
// Next: ../api-key
package main

import (
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/basicauth"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"message": "public page, try /admin/stats"})
	})

	// A map[string]string user list is the quickest form. Never commit real
	// passwords: basicauth.Load("users.yml", basicauth.BCRYPT) reads the same
	// list from a JSON/YAML file and compares against bcrypt hashes, and
	// basicauth.New(basicauth.Options[User]{Allow: ...}) plugs in your own
	// database lookup, credential expiration (MaxAge) and a failure limit (MaxTries).
	guard := basicauth.Default(map[string]string{
		"admin": "admin_password",
		"ops":   "ops_password",
	})

	// The middleware is passed to the router, so every route registered under
	// /admin requires credentials and nothing else does.
	admin := app.Router("/admin", guard)

	admin.Get("/stats", func(ctx iris.Context) {
		// ctx.User() is non-nil for any request that got past the middleware.
		username, err := ctx.User().GetUsername()
		if err != nil {
			// HandleError classifies an unmapped error as a 500 and logs it,
			// so the client sees the generic body and not the cause.
			errors.HandleError(ctx, err)
			return
		}

		ctx.OK(iris.Map{"user": username, "goroutines": 12, "uptime": "3h20m"}, nil)
	})

	admin.Get("/whoami", func(ctx iris.Context) {
		user := ctx.User()

		id, _ := user.GetID()
		method, _ := user.GetAuthorization()
		// user.GetAuthorizedAt() is filled only when Options.MaxAge is set,
		// which is what makes credentials expire and re-prompt.

		// For the map form above the user is a *iris.SimpleUser, which carries
		// the plaintext Password, so never serialize it straight into a response.
		ctx.JSON(iris.Map{"id": id, "method": method})
	})

	// The typed form: Options[Operator] means GetUser[Operator] returns the
	// matching list element. MaxAge makes a login expire ten minutes after it
	// was first accepted, so AuthorizedAt is available too.
	typed := basicauth.New(basicauth.Options[Operator]{
		Realm:  "Operators",
		MaxAge: 10 * time.Minute,
		Allow:  basicauth.AllowUsers(operators),
	})

	ops := app.Router("/ops", typed)
	ops.Get("/me", func(ctx iris.Context) {
		op, ok := basicauth.GetUser[Operator](ctx)
		if !ok {
			errors.Unauthenticated.Message(ctx, "no operator in context")
			return
		}

		since, _ := basicauth.AuthorizedAt(ctx)
		ctx.JSON(iris.Map{"username": op.Username, "roles": op.Roles, "authorized_at": since})
	})

	app.Listen(":8080")
}

// Operator is an application's own user type. AllowUsers reads the credentials
// from the Username and Password fields (json tags count too); a type that
// stores them elsewhere passes basicauth.Credentials to say where.
type Operator struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

var operators = []Operator{
	{"makis", "makis_password", []string{"admin", "oncall"}},
	{"george", "george_password", []string{"oncall"}},
}
