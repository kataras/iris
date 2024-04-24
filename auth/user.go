//go:build go1.18
// +build go1.18

package auth

import (
	"github.com/kataras/iris/v12/context"

	"github.com/kataras/jwt"
)

type (
	// StandardClaims is an alias of jwt.Claims, it holds the standard JWT claims.
	StandardClaims = jwt.Claims
	// User is an alias of an empty interface, it's here to declare the typeof T,
	// which can be any custom struct type.
	//
	// Example can be found at: https://github.com/kataras/iris/tree/main/_examples/auth/auth/user.go.
	User = any
)

const accessTokenContextKey = "iris.auth.context.access_token"

// GetAccessToken accepts the iris Context and returns the raw access token value.
// It's only available after Auth.VerifyHandler is executed.
func GetAccessToken(ctx *context.Context) string {
	return ctx.Values().GetString(accessTokenContextKey)
}

const standardClaimsContextKey = "iris.auth.context.standard_claims"

// GetStandardClaims accepts the iris Context and returns the standard token's claims.
// It's only available after Auth.VerifyHandler is executed.
func GetStandardClaims(ctx *context.Context) StandardClaims {
	if v := ctx.Values().Get(standardClaimsContextKey); v != nil {
		if c, ok := v.(StandardClaims); ok {
			return c
		}
	}

	return StandardClaims{}
}

const userContextKey = "iris.auth.context.user"

// GetUser is the package-level function of the Auth.GetUser method.
// It returns the T user value after Auth.VerifyHandler is executed.
func GetUser[T User](ctx *context.Context) T {
	if v := ctx.Values().Get(userContextKey); v != nil {
		if t, ok := v.(T); ok {
			return t
		}
	}

	var empty T
	return empty
}

// GetUser accepts the iris Context and returns the T custom user/claims struct value.
// It's only available after Auth.VerifyHandler is executed.
func (s *Auth[T]) GetUser(ctx *context.Context) T {
	return GetUser[T](ctx)
}
