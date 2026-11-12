package mvc_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/httptest"

	"uuid"
)

// The declared V and the macro's stored type may disagree (an untyped "{id}"
// stores a string, the uuid macro stores a uuid.UUID). PathParam[T, V].Get
// must convert instead of panicking on the type assertion.
func TestPathParamGetUUIDFromStringDoesNotPanic(t *testing.T) {
	type Customer struct {
		ID string `json:"customer_id"`
	}

	p := mvc.PathParam[Customer, uuid.UUID]{
		RawPathParam: &mvc.RawPathParam{
			Name:      "customer_id",
			Type:      "uuid",
			RoutePath: "/{customer_id:uuid}",
		},
	}

	app := iris.New()
	app.Get("/customers/{customer_id:uuid}", func(ctx iris.Context) {
		id := p.Get(ctx)
		ctx.WriteString(id.String())
	})

	e := httptest.New(t, app)

	uid := uuid.New().String()
	e.GET("/customers/" + uid).Expect().
		Status(httptest.StatusOK).Body().IsEqual(uid)
}
