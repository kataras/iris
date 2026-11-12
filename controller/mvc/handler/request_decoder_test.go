package handler_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/httptest"

	"uuid"
)

type (
	testParamHandlerRequest struct {
		storeID string
	}

	testRequestDecoderRequest struct {
		firstname string
	}
)

func (r *testParamHandlerRequest) DecodeRequest(ctx iris.Context) error {
	r.storeID = ctx.Params().GetString("store_id")
	return nil
}

func (r *testRequestDecoderRequest) DecodeRequest(ctx iris.Context) error {
	r.firstname = ctx.Query().Get("firstname")
	return nil
}

func testGetRequestDecoder(ctx context.Context, param testParamHandlerRequest, req testRequestDecoderRequest) (string, error) {
	return param.storeID + req.firstname, nil
}

func TestDirectRequestDecoder(t *testing.T) {
	app := iris.New()
	app.Get("/store/{store_id:uuid}", handler.HandlerBy(testGetRequestDecoder))
	e := httptest.New(t, app)

	expectedStoreID := uuid.New().String()
	e.GET("/store/"+expectedStoreID).WithQuery("firstname", "makis").Expect().Status(httptest.StatusOK).
		JSON().Object().IsEqual(iris.Map{"id": expectedStoreID + "makis"})
}

type testParamHandlerInline struct {
	storeID string
}

func testGetRequestDecoderWithInlineParamHandler(ctx context.Context, param testParamHandlerInline, req testRequestDecoderRequest) (string, error) {
	return param.storeID + req.firstname, nil
}

func TestWithParamDecoder_Mix_With_DirectRequestDecoder(t *testing.T) {
	app := iris.New()
	app.Get("/store/{store_id:uuid}", handler.HandlerBy(testGetRequestDecoderWithInlineParamHandler, // the request decoded through its DecodeRequest method.
		handler.WithParamDecoderFunc( // the param decoded through a custom function.
			func(ctx iris.Context, param *testParamHandlerInline) error {
				param.storeID = ctx.Params().GetString("store_id")
				return nil
			})))

	e := httptest.New(t, app)

	expectedStoreID := uuid.New().String()
	e.GET("/store/"+expectedStoreID).WithQuery("firstname", "makis").Expect().Status(httptest.StatusOK).
		JSON().Object().IsEqual(iris.Map{"id": expectedStoreID + "makis"})
}

type testRequestDecoderInline struct {
	fullname string
}

func TestWithRequestDecoder(t *testing.T) {
	app := iris.New()
	app.Get("/store", handler.Handler(func(ctx context.Context, req testRequestDecoderInline) (string, error) {
		return req.fullname, nil
	}, handler.WithRequestDecoderFunc(func(ctx iris.Context, req *testRequestDecoderInline) error {
		req.fullname = fmt.Sprintf("%s %s", ctx.Query().Get("firstname"), ctx.Query().Get("lastname"))
		return nil
	})))

	e := httptest.New(t, app)

	e.GET("/store").WithQuery("firstname", "makis").WithQuery("lastname", "maropoulos").Expect().Status(httptest.StatusOK).
		JSON().Object().IsEqual(iris.Map{"id": "makis maropoulos"})
}
