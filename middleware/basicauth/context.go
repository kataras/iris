package basicauth

import (
	"time"

	"github.com/kataras/iris/v14/context"
)

const (
	// userContextKey holds the raw user value of type U, for GetUser.
	// ctx.User() sees the same value through Context.SetUser.
	userContextKey = "iris.basicauth.user"
	// authorizedAtContextKey holds the first login time, for AuthorizedAt.
	authorizedAtContextKey = "iris.basicauth.authorized_at"
)

// GetUser returns the authenticated user stored in the request by a basicauth middleware,
// as the type U that its Options.Allow returned.
// It reports false when the request was not authenticated, when the stored user
// is not a U or after ctx.Logout().
//
// ctx.User() returns the same value as a context.User, which is enough when a
// handler only needs the username; GetUser exists for the fields of your own type.
//
// Usage:
//
//	user, ok := basicauth.GetUser[MyUser](ctx)
func GetUser[U any](ctx *context.Context) (U, bool) {
	return ctx.ValueAs[U](userContextKey)
}

// AuthorizedAt returns the time the request's credentials were first accepted
// by a basicauth middleware, as implied by its Options.MaxAge entry.
// It reports false when the request was not authenticated or when MaxAge is zero,
// as the middleware keeps no per-login time without an expiration.
//
// Usage:
//
//	at, ok := basicauth.AuthorizedAt(ctx)
func AuthorizedAt(ctx *context.Context) (time.Time, bool) {
	at, ok := ctx.ValueAs[time.Time](authorizedAtContextKey)
	if !ok || at.IsZero() {
		return time.Time{}, false
	}

	return at, true
}

// setContext stores the basicauth values of an authenticated request.
func setContext(ctx *context.Context, user any, authorizedAt time.Time) {
	ctx.Values().Set(userContextKey, user)
	if !authorizedAt.IsZero() {
		ctx.Values().Set(authorizedAtContextKey, authorizedAt)
	}

	ctx.SetUser(user)
}

// clearContext removes the basicauth values of a request, see logout.
func clearContext(ctx *context.Context) {
	ctx.Values().Remove(userContextKey)
	ctx.Values().Remove(authorizedAtContextKey)
	ctx.SetUser(nil)
}
