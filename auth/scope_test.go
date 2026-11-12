package auth

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

type scopedUser struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
}

// otherUser is the user type of a VerifyHandler that does not match the one
// Scope was declared with.
type otherUser struct {
	ID string `json:"id"`
}

// verified stands in for VerifyHandler: it stores the user the same way,
// which is all Scope reads.
func verified(u scopedUser) iris.Handler {
	return func(ctx iris.Context) {
		ctx.Values().Set(userContextKey, u)
		ctx.Next()
	}
}

func TestScope(t *testing.T) {
	scope := Scope(func(u scopedUser) string { return u.TenantID })

	app := iris.New()
	app.Get("/with-user", verified(scopedUser{ID: "1", TenantID: "acme"}), scope, func(ctx iris.Context) {
		tenantID, ok := ScopeID[string](ctx)
		if !ok {
			ctx.StopWithStatus(iris.StatusInternalServerError)
			return
		}

		ctx.WriteString(tenantID)
	})

	// A derived zero id is stored as it is: a global admin may legitimately
	// have no tenant, and that is not an authentication failure.
	app.Get("/global-admin", verified(scopedUser{ID: "1"}), scope, func(ctx iris.Context) {
		tenantID, ok := ScopeID[string](ctx)
		if !ok {
			ctx.StopWithStatus(iris.StatusInternalServerError)
			return
		}

		ctx.WriteString("[" + tenantID + "]")
	})

	// No VerifyHandler ran, so there is no user to derive from.
	app.Get("/without-user", scope, func(ctx iris.Context) {
		ctx.WriteString("must not run")
	})

	// A VerifyHandler for another user type is a mis-wired type parameter,
	// not a global administrator: deriving from the zero value here would
	// answer 200 with an empty scope and read across every tenant.
	app.Get("/wrong-user-type", func(ctx iris.Context) {
		ctx.Values().Set(userContextKey, otherUser{ID: "1"})
		ctx.Next()
	}, scope, func(ctx iris.Context) {
		ctx.WriteString("must not run")
	})

	// A read of the wrong type reports false instead of a zero value that
	// the caller cannot tell from a real one.
	app.Get("/wrong-type", verified(scopedUser{ID: "1", TenantID: "acme"}), scope, func(ctx iris.Context) {
		if _, ok := ScopeID[int](ctx); ok {
			ctx.StopWithStatus(iris.StatusInternalServerError)
			return
		}

		ctx.WriteString("false")
	})

	e := httptest.New(t, app)

	e.GET("/with-user").Expect().Status(httptest.StatusOK).Body().IsEqual("acme")
	e.GET("/global-admin").Expect().Status(httptest.StatusOK).Body().IsEqual("[]")
	e.GET("/without-user").Expect().Status(httptest.StatusUnauthorized)
	e.GET("/wrong-user-type").Expect().Status(httptest.StatusUnauthorized)
	e.GET("/wrong-type").Expect().Status(httptest.StatusOK).Body().IsEqual("false")
}
