//go:build go1.18

package auth

import (
	stdContext "context"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/x/errors"

	"github.com/kataras/jwt"
)

// VerifiedToken holds the information about a verified token.
type VerifiedToken = jwt.VerifiedToken

// Provider is an interface of T which MUST be completed
// by a custom value type to provide user information to the Auth's
// JWT Token Signer and Verifier.
//
// A provider can implement Transformer and ErrorHandler and ClaimsProvider as well.
type Provider[T User] interface {
	// Signin accepts a username (or email) and a password and should
	// return a valid T value or an error describing
	// the user authentication or verification reason of failure.
	//
	// It's called on auth.SigninHandler
	Signin(ctx stdContext.Context, username, password string) (T, error)

	// ValidateToken accepts the standard JWT claims and the T value obtained
	// by the Signin method and should return a nil error on validation success
	// or a non-nil error for validation failure.
	// It is mostly used to perform checks of the T value's struct fields or
	// the standard claim's (e.g. origin jwt token id).
	// It can be an empty method too which returns a nil error.
	//
	// It's caleld on auth.VerifyHandler.
	ValidateToken(ctx stdContext.Context, standardClaims StandardClaims, t T) error

	// InvalidateToken is optional and can be used to allow tokens to be invalidated
	// from server-side. Commonly, implement when a token and user pair is saved
	// on a persistence storage and server can decide which token is valid or invalid.
	// It can be an empty method too which returns a nil error.
	//
	// It's called on auth.SignoutHandler.
	InvalidateToken(ctx stdContext.Context, standardClaims StandardClaims, t T) error
	// InvalidateTokens is like InvalidateToken but it should invalidate
	// all tokens generated for a specific T value.
	// It can be an empty method too which returns a nil error.
	//
	// It's called on auth.SignoutAllHandler.
	InvalidateTokens(ctx stdContext.Context, t T) error
}

// ClaimsProvider is an optional interface, which may not be used at all.
// If implemented by a Provider, it signs the jwt token
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
