package handler_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/httptest"
)

// testRequiredValidator is an Options.Context.RequestValidator that rejects a
// zero value on a field tagged `validate:"required"` and counts its calls.
type testRequiredValidator struct{ calls atomic.Int64 }

func (v *testRequiredValidator) Struct(ptr any) error {
	v.calls.Add(1)

	val := reflect.Indirect(reflect.ValueOf(ptr))
	typ := val.Type()
	for i := range typ.NumField() {
		if strings.HasPrefix(typ.Field(i).Tag.Get("validate"), "required") && val.Field(i).IsZero() {
			return fmt.Errorf("validator: %s is required", typ.Field(i).Name)
		}
	}

	return nil
}

type (
	testValidatedQuery struct {
		PayIDs string `url:"pay_ids" validate:"required"`
		AccID  int    `url:"acc_id" validate:"required"`
	}

	testValidatedForm struct {
		PayIDs string `form:"pay_ids" validate:"required"`
		AccID  int    `form:"acc_id" validate:"required"`
	}

	testValidatedParams struct {
		PayIDs string `param:"pay_ids" validate:"required"`
		AccID  int    `param:"acc_id" validate:"required"`
	}

	testValidatedHeader struct {
		Tenant string `header:"X-Tenant" validate:"required"`
	}

	testValidatedJSON struct {
		PayIDs string `json:"pay_ids" validate:"required"`
		AccID  int    `json:"acc_id" validate:"required"`
	}

	testValidatedQueryAndJSON struct {
		PayIDs string `url:"pay_ids" validate:"required"`
		AccID  int    `json:"acc_id" validate:"required"`
	}
)

// Options.Context.RequestValidator runs once on a request type bound from the
// URL query, a form, path parameters or headers, the way it already does for
// a JSON body. A request that sends nothing is validated too.
// Issue #1783.
func TestRequestValidatorOnEverySource(t *testing.T) {
	v := new(testRequiredValidator)
	app := iris.New(iris.Options{
		Context: iris.ContextOptions{RequestValidator: v},
	})

	app.Get("/query", handler.Handler(func(ctx context.Context, req testValidatedQuery) (string, error) {
		return "ok", nil
	}))
	app.Post("/form", handler.Handler(func(ctx context.Context, req testValidatedForm) (string, error) {
		return "ok", nil
	}))
	app.Get("/params/{pay_ids}/{acc_id:int}", handler.Handler(func(ctx context.Context, req testValidatedParams) (string, error) {
		return "ok", nil
	}))
	app.Get("/by/{pay_ids}/{acc_id:int}", handler.HandlerBy(func(ctx context.Context, param testValidatedParams, req struct{}) (string, error) {
		return "ok", nil
	}))
	app.Get("/header", handler.Handler(func(ctx context.Context, req testValidatedHeader) (string, error) {
		return "ok", nil
	}))
	app.Post("/json", handler.Handler(func(ctx context.Context, req testValidatedJSON) (string, error) {
		return "ok", nil
	}))
	app.Post("/query-and-json", handler.Handler(func(ctx context.Context, req testValidatedQueryAndJSON) (string, error) {
		return "ok", nil
	}))

	e := httptest.New(t, app)

	tests := []struct {
		name   string
		req    *httptest.Request
		status int
	}{
		{"query, nothing sent", e.GET("/query"), httptest.StatusBadRequest},
		{"query, one of two sent", e.GET("/query").WithQuery("pay_ids", "a"), httptest.StatusBadRequest},
		{"query, both sent", e.GET("/query").WithQuery("pay_ids", "a").WithQuery("acc_id", 2), httptest.StatusOK},
		{"form, nothing sent", e.POST("/form").WithHeader("Content-Type", "application/x-www-form-urlencoded"), httptest.StatusBadRequest},
		{"form, one of two sent", e.POST("/form").WithFormField("pay_ids", "a"), httptest.StatusBadRequest},
		{"form, both sent", e.POST("/form").WithFormField("pay_ids", "a").WithFormField("acc_id", 2), httptest.StatusOK},
		{"path parameters, zero value", e.GET("/params/a/0"), httptest.StatusBadRequest},
		{"path parameters, valid", e.GET("/params/a/2"), httptest.StatusOK},
		{"param of HandlerBy, zero value", e.GET("/by/a/0"), httptest.StatusBadRequest},
		{"param of HandlerBy, valid", e.GET("/by/a/2"), httptest.StatusOK},
		{"header, not sent", e.GET("/header"), httptest.StatusBadRequest},
		{"header, sent", e.GET("/header").WithHeader("X-Tenant", "a"), httptest.StatusOK},
		{"json, one of two sent", e.POST("/json").WithJSON(iris.Map{"pay_ids": "a"}), httptest.StatusBadRequest},
		{"json, both sent", e.POST("/json").WithJSON(iris.Map{"pay_ids": "a", "acc_id": 2}), httptest.StatusOK},
		{"query and json, json field missing", e.POST("/query-and-json").WithQuery("pay_ids", "a").WithJSON(iris.Map{}), httptest.StatusBadRequest},
		{"query and json, both sent", e.POST("/query-and-json").WithQuery("pay_ids", "a").WithJSON(iris.Map{"acc_id": 2}), httptest.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := v.calls.Load()
			tt.req.WithReporter(t).Expect().Status(tt.status)

			if calls := v.calls.Load() - before; calls != 1 {
				t.Errorf("expected the request validator to run once, got %d calls", calls)
			}
		})
	}
}

// Without a request validator the same handlers answer as before:
// an empty query reaches the handler with the zero value.
func TestRequestValidatorNotSet(t *testing.T) {
	app := iris.New()
	app.Get("/query", handler.Handler(func(ctx context.Context, req testValidatedQuery) (string, error) {
		return "ok", nil
	}))
	app.Get("/by/{pay_ids}/{acc_id:int}", handler.HandlerBy(func(ctx context.Context, param testValidatedParams, req struct{}) (string, error) {
		return "ok", nil
	}))

	e := httptest.New(t, app)
	e.GET("/query").Expect().Status(httptest.StatusOK).JSON().Object().IsEqual(iris.Map{"id": "ok"})
	e.GET("/by/a/0").Expect().Status(httptest.StatusOK).JSON().Object().IsEqual(iris.Map{"id": "ok"})
}
