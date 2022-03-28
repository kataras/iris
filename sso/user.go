//go:build go1.18

package sso

import (
	"github.com/kataras/iris/v12/context"

	"github.com/kataras/jwt"
)

type (
	StandardClaims = jwt.Claims
	User           = interface{} // any type.
)

const accessTokenContextKey = "iris.sso.context.access_token"

func GetAccessToken(ctx *context.Context) string {
	return ctx.Values().GetString(accessTokenContextKey)
}

const standardClaimsContextKey = "iris.sso.context.standard_claims"

func GetStandardClaims(ctx *context.Context) StandardClaims {
	if v := ctx.Values().Get(standardClaimsContextKey); v != nil {
		if c, ok := v.(StandardClaims); ok {
			return c
		}
	}

	return StandardClaims{}
}

func (s *SSO[T]) GetStandardClaims(ctx *context.Context) StandardClaims {
	return GetStandardClaims(ctx)
}

const userContextKey = "iris.sso.context.user"

func GetUser[T User](ctx *context.Context) T {
	if v := ctx.Values().Get(userContextKey); v != nil {
		if t, ok := v.(T); ok {
			return t
		}
	}

	var empty T
	return empty
}

func (s *SSO[T]) GetUser(ctx *context.Context) T {
	return GetUser[T](ctx)
}
