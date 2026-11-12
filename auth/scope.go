package auth

import (
	stdErrors "errors"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"
)

const scopeContextKey = "iris.auth.context.scope"

// ErrMissingUser is the reason of the 401 that Scope answers when the
// Context carries no verified user of the type Scope derives from: either no
// VerifyHandler ran in front of it, or it verified a different user type.
var ErrMissingUser = stdErrors.New("auth: no verified user of the expected type on the request")

// Scope derives a per-request scope id (a tenant, a customer, an
// organisation) from the authenticated user once, after VerifyHandler, and
// stores it on the Context for every handler and query after it.
//
// It exists so that the scope is derived in one place instead of in every
// handler: a handler that forgets to filter by tenant is a data leak, and a
// handler that derives it again is a second chance to get it wrong.
//
// A missing user answers 401, the same failure VerifyHandler answers with,
// because a scope cannot be derived without one. So does a user of another
// type: reading it untyped and deriving from the zero value would answer 200
// with an empty scope, which is a cross-tenant read wearing the disguise of
// the next paragraph.
//
// A derived zero id is allowed and stored as it is: a global administrator
// may legitimately have no scope, and that is not an authentication failure.
// Guard those requests by checking the id in the handler, not by relying on
// Scope.
//
// The cost per request is one Values read, the derive call and one Values
// set; ScopeID is one map lookup and one type assertion. No reflection is
// involved.
//
// Usage:
//
//	app.Use(a.VerifyHandler(), auth.Scope(func(u Customer) string {
//		return u.TenantID
//	}))
//
//	// ...in a handler, or in an mvc.BeforeWrite hook:
//	tenantID, ok := auth.ScopeID[string](ctx)
func Scope[User any, ID comparable](derive func(User) ID) context.Handler {
	return func(ctx *context.Context) {
		// One typed read, not UserAuthorized followed by GetUser: those are
		// two lookups, and GetUser swallows a failed type assertion.
		user, ok := ctx.ValueAs[User](userContextKey)
		if !ok {
			errors.Unauthenticated.Err(ctx, ErrMissingUser)
			return
		}

		ctx.Values().Set(scopeContextKey, derive(user))
		ctx.Next()
	}
}

// ScopeID returns the id Scope stored on the Context. It reports false when
// Scope did not run or when it stored an id of another type, so that a zero
// id a caller must not trust cannot be mistaken for a real one.
func ScopeID[ID comparable](ctx *context.Context) (ID, bool) {
	return ctx.ValueAs[ID](scopeContextKey)
}
