//go:build go1.18

package sso

import (
	stdContext "context"
	"fmt"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/middleware/jwt"
	"github.com/kataras/iris/v12/x/errors"
)

type VerifiedToken = jwt.VerifiedToken

type Provider[T User] interface { // A provider can implement Transformer and ErrorHandler as well.
	Signin(ctx stdContext.Context, username, password string) (T, error)

	// We could do this instead of transformer below but let's keep separated logic methods:
	// ValidateToken(ctx context.Context, tok *VerifiedToken, t *T) error
	ValidateToken(ctx stdContext.Context, standardClaims StandardClaims, t T) error

	InvalidateToken(ctx stdContext.Context, standardClaims StandardClaims, t T) error
	InvalidateTokens(ctx stdContext.Context, t T) error
}

// ClaimsProvider is an optional interface, which may not be used at all.
// If completed by a Provider, it signs the jwt token
// using these claims to each of the following token types.
type ClaimsProvider interface {
	GetAccessTokenClaims() StandardClaims
	GetRefreshTokenClaims(accessClaims StandardClaims) StandardClaims
}

type Transformer[T User] interface {
	Transform(ctx stdContext.Context, tok *VerifiedToken) (T, error)
}

type TransformerFunc[T User] func(ctx stdContext.Context, tok *VerifiedToken) (T, error)

func (fn TransformerFunc[T]) Transform(ctx stdContext.Context, tok *VerifiedToken) (T, error) {
	return fn(ctx, tok)
}

type ErrorHandler interface {
	InvalidArgument(ctx *context.Context, err error)
	Unauthenticated(ctx *context.Context, err error)
}

type DefaultErrorHandler struct{}

func (e *DefaultErrorHandler) InvalidArgument(ctx *context.Context, err error) {
	errors.InvalidArgument.Details(ctx, "unable to parse body", err.Error())
}

func (e *DefaultErrorHandler) Unauthenticated(ctx *context.Context, err error) {
	errors.Unauthenticated.Err(ctx, err)
}

type provider[T User] struct{}

func newProvider[T User]() *provider[T] {
	return new(provider[T])
}

func (p *provider[T]) Signin(ctx stdContext.Context, username, password string) (T, error) { // fired on SigninHandler.
	// your database...
	var t T
	return t, fmt.Errorf("user not found")
}

func (p *provider[T]) ValidateToken(ctx stdContext.Context, standardClaims StandardClaims, t T) error { // fired on VerifyHandler.
	// your database and checks of blocked tokens...
	return nil
}

func (p *provider[T]) InvalidateToken(ctx stdContext.Context, standardClaims StandardClaims, t T) error { // fired on SignoutHandler.
	return nil
}

func (p *provider[T]) InvalidateTokens(ctx stdContext.Context, t T) error { // fired on SignoutAllHandler.
	return nil
}
