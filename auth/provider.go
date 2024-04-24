//go:build go1.18
// +build go1.18

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
// A provider can optionally complete the Transformer, ClaimsProvider and
// ErrorHandler all in once when necessary.
// Set a provider using the AddProvider method of Auth type.
//
// Example can be found at: https://github.com/kataras/iris/tree/main/_examples/auth/auth/user_provider.go.
type Provider[T User] interface {
	// Signin accepts a username (or email) and a password and should
	// return a valid T value or an error describing
	// the user authentication or verification reason of failure.
	//
	// The first input argument standard context can be
	// casted to iris.Context if executed through Auth.SigninHandler.
	//
	// It's called on Auth.SigninHandler.
	Signin(ctx stdContext.Context, username, password string) (T, error)

	// ValidateToken accepts the standard JWT claims and the T value obtained
	// by the Signin method and should return a nil error on validation success
	// or a non-nil error for validation failure.
	// It is mostly used to perform checks of the T value's struct fields or
	// the standard claim's (e.g. origin jwt token id).
	// It can be an empty method too which returns a nil error.
	//
	// The first input argument standard context can be
	// casted to iris.Context if executed through Auth.VerifyHandler.
	//
	// It's caleld on Auth.VerifyHandler.
	ValidateToken(ctx stdContext.Context, standardClaims StandardClaims, t T) error

	// InvalidateToken is optional and can be used to allow tokens to be invalidated
	// from server-side. Commonly, implement when a token and user pair is saved
	// on a persistence storage and server can decide which token is valid or invalid.
	// It can be an empty method too which returns a nil error.
	//
	// The first input argument standard context can be
	// casted to iris.Context if executed through Auth.SignoutHandler.
	//
	// It's called on Auth.SignoutHandler.
	InvalidateToken(ctx stdContext.Context, standardClaims StandardClaims, t T) error
	// InvalidateTokens is like InvalidateToken but it should invalidate
	// all tokens generated for a specific T value.
	// It can be an empty method too which returns a nil error.
	//
	// The first input argument standard context can be
	// casted to iris.Context if executed through Auth.SignoutAllHandler.
	//
	// It's called on Auth.SignoutAllHandler.
	InvalidateTokens(ctx stdContext.Context, t T) error
}

// ClaimsProvider is an optional interface, which may not be used at all.
// If implemented by a Provider, it signs the jwt token
// using these claims to each of the following token types.
type ClaimsProvider interface {
	GetAccessTokenClaims() StandardClaims
	GetRefreshTokenClaims(accessClaims StandardClaims) StandardClaims
}

// Transformer is an optional interface which can be implemented by a Provider as well.
// Set a Transformer through Auth.SetTransformer or Auth.SetTransformerFunc or by implementing
// the Transform method inside a Provider which can be registered through the Auth.AddProvider
// method.
//
// A transformer is called on Auth.VerifyHandler before Provider.ValidateToken and it can
// be used to modify the T value based on the token's contents. It is mostly used
// to convert the json claims to T value manually, when they differ.
//
// The first input argument standard context can be
// casted to iris.Context if executed through Auth.VerifyHandler.
type Transformer[T User] interface {
	Transform(ctx stdContext.Context, tok *VerifiedToken) (T, error)
}

// TransformerFunc completes the Transformer interface.
type TransformerFunc[T User] func(ctx stdContext.Context, tok *VerifiedToken) (T, error)

// Transform calls itself.
func (fn TransformerFunc[T]) Transform(ctx stdContext.Context, tok *VerifiedToken) (T, error) {
	return fn(ctx, tok)
}

// ErrorHandler is an optional interface which can be implemented by a Provider as well.
//
// ErrorHandler is the interface which controls the HTTP errors on
// Auth.SigninHandler, Auth.VerifyHandler, Auth.SignoutHandler and
// Auth.SignoutAllHandler handelrs.
type ErrorHandler interface {
	// InvalidArgument should handle any 400 (bad request) errors,
	// e.g. invalid request body.
	InvalidArgument(ctx *context.Context, err error)
	// Unauthenticated should handle any 401 (unauthenticated) errors,
	// e.g. user not found or invalid credentials.
	Unauthenticated(ctx *context.Context, err error)
}

// DefaultErrorHandler is the default registered ErrorHandler which can be
// replaced through the Auth.SetErrorHandler method.
type DefaultErrorHandler struct{}

// InvalidArgument sends 400 (bad request) with "unable to parse body" as its message
// and the "err" value as its details.
func (e *DefaultErrorHandler) InvalidArgument(ctx *context.Context, err error) {
	errors.InvalidArgument.Details(ctx, "unable to parse body", err.Error())
}

// Unauthenticated sends 401 (unauthenticated) with the "err" value as its message.
func (e *DefaultErrorHandler) Unauthenticated(ctx *context.Context, err error) {
	errors.Unauthenticated.Err(ctx, err)
}
