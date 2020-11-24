package basicauth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/kataras/iris/v12/context"
)

type (
	// ErrHTTPVersion is fired when Options.HTTPSOnly was enabled
	// and the current request is a plain http one.
	ErrHTTPVersion struct{}

	// ErrCredentialsForbidden is fired when Options.MaxTries have been consumed
	// by the user and the client is forbidden to retry at least for "Age" time.
	ErrCredentialsForbidden struct {
		Username string
		Password string
		Tries    int
		Age      time.Duration
	}

	// ErrCredentialsMissing is fired when the authorization header is empty or malformed.
	ErrCredentialsMissing struct {
		Header string

		AuthenticateHeader      string
		AuthenticateHeaderValue string
		Code                    int
	}

	// ErrCredentialsInvalid is fired when the user input does not match with an existing user.
	ErrCredentialsInvalid struct {
		Username     string
		Password     string
		CurrentTries int

		AuthenticateHeader      string
		AuthenticateHeaderValue string
		Code                    int
	}

	// ErrCredentialsExpired is fired when the username:password combination is valid
	// but the memory stored user has been expired.
	ErrCredentialsExpired struct {
		Username string
		Password string

		AuthenticateHeader      string
		AuthenticateHeaderValue string
		Code                    int
	}
)

func (e ErrHTTPVersion) Error() string {
	return "http version not supported"
}

func (e ErrCredentialsForbidden) Error() string {
	return fmt.Sprintf("credentials: forbidden <%s:%s> for <%s> after <%d> attempts", e.Username, e.Password, e.Age, e.Tries)
}

func (e ErrCredentialsMissing) Error() string {
	if e.Header != "" {
		return fmt.Sprintf("credentials: malformed <%s>", e.Header)
	}

	return "empty credentials"
}

func (e ErrCredentialsInvalid) Error() string {
	return fmt.Sprintf("credentials: invalid <%s:%s> current tries <%d>", e.Username, e.Password, e.CurrentTries)
}

func (e ErrCredentialsExpired) Error() string {
	return fmt.Sprintf("credentials: expired <%s:%s>", e.Username, e.Password)
}

// DefaultErrorHandler is the default error handler for the Options.ErrorHandler field.
func DefaultErrorHandler(ctx *context.Context, err error) {
	switch e := err.(type) {
	case ErrHTTPVersion:
		ctx.StopWithStatus(http.StatusHTTPVersionNotSupported)
	case ErrCredentialsForbidden:
		// If a (proxy) server receives valid credentials that are inadequate to access a given resource,
		// the server should respond with the 403 Forbidden status code.
		// Unlike 401 Unauthorized or 407 Proxy Authentication Required, authentication is impossible for this user.
		ctx.StopWithStatus(http.StatusForbidden)
	case ErrCredentialsMissing:
		unauthorize(ctx, e.AuthenticateHeader, e.AuthenticateHeaderValue, e.Code)
	case ErrCredentialsInvalid:
		unauthorize(ctx, e.AuthenticateHeader, e.AuthenticateHeaderValue, e.Code)
	case ErrCredentialsExpired:
		unauthorize(ctx, e.AuthenticateHeader, e.AuthenticateHeaderValue, e.Code)
	default:
		// This will never happen.
		ctx.StopWithText(http.StatusInternalServerError, "unknown error: %v", err)
	}
}

// unauthorize sends a 401 status code (or 407 if Proxy was set to true)
// which client should catch and prompt for username:password credentials.
func unauthorize(ctx *context.Context, authHeader, authHeaderValue string, code int) {
	ctx.Header(authHeader, authHeaderValue)
	ctx.StopWithStatus(code)
}
