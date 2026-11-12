package auth

import (
	"github.com/kataras/iris/v14/context"

	"github.com/kataras/jwt"
)

type (
	// StandardClaims is an alias of jwt.Claims, it holds the standard JWT claims.
	StandardClaims = jwt.Claims
	// User is an alias of an empty interface, it's here to declare the typeof T,
	// which can be any custom struct type.
	//
	// Example can be found at: https://github.com/kataras/iris/tree/main/_examples/09-authentication/auth-sdk/main.go (Customer).
	User = any
)

const authCntextKey = "iris.auth"

// Get accepts the iris Context and returns the current Auth[T] instance.
// It's only available after Auth.VerifyHandler is executed.
func Get[T User](ctx *context.Context) (*Auth[T], bool) {
	return ctx.ValueAs[*Auth[T]](authCntextKey)
}

const authTokenContextKey = "iris.auth.context.token"

// GetToken accepts the iris Context and returns the
// verified JWT token instance, which contains the raw access token
// and its claims.
// It's only available after Auth.VerifyHandler is executed.
func GetToken(ctx *context.Context) (*jwt.VerifiedToken, bool) {
	return ctx.ValueAs[*jwt.VerifiedToken](authTokenContextKey)
}

const userContextKey = "iris.auth.context.user"

// GetUser is the package-level function of the Auth.GetUser method.
// It returns the T user value after Auth.VerifyHandler is executed.
func GetUser[T User](ctx *context.Context) T {
	t, _ := ctx.ValueAs[T](userContextKey)
	return t
}

// UserAuthorized accepts the iris Context and returns true if the user is authorized.
func UserAuthorized(ctx *context.Context) bool {
	return ctx.Values().Get(userContextKey) != nil
}

// GetUser accepts the iris Context and returns the T custom user/claims struct value.
// It's only available after Auth.VerifyHandler is executed.
func (s *Auth[T]) GetUser(ctx *context.Context) T {
	return GetUser[T](ctx)
}
