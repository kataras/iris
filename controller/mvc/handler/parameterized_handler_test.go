package handler_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/httptest"

	"uuid"
)

func TestHandlerBy(t *testing.T) {
	type Request struct {
		Firstname string `json:"firstname"` // read json.
		Lastname  string `json:"lastname"`
		Age       int    `json:"age"`
	}

	type RequestParams struct {
		Authenticated bool   `context:"authenticated"` // read from the context.
		CompanyID     string `param:"company_id"`      // read dynamic path parameters.
		Country       string `url:"country"`           // read query.
	}

	type Response struct {
		CompanyID string `json:"company_id"`
		Country   string `json:"country"`

		Fullname string `json:"fullname"`
		IsAdult  bool   `json:"is_adult"`
	}

	newHTTPRequest := func() (RequestParams, Request) {
		params := RequestParams{
			CompanyID: uuid.New().String(),
			Country:   "el-GR",
		}

		req := Request{
			Firstname: "John",
			Lastname:  "Doe",
			Age:       25,
		}

		return params, req
	}

	app := iris.New()

	app.Use(func(ctx iris.Context) {
		ctx.Values().Set("authenticated", true) // set a context value to be used by the next handler, framework is able to bind it through its parameters by the "context" field tag.
		ctx.Next()
	})

	createFn := func(ctx iris.Context, params RequestParams, req Request) (Response, error) {
		if !params.Authenticated {
			return Response{}, fmt.Errorf("unauthorized")
		}

		resp := Response{
			CompanyID: params.CompanyID,
			Country:   params.Country,
			Fullname:  req.Firstname + " " + req.Lastname,
			IsAdult:   req.Age >= 18,
		}
		return resp, nil
	}
	app.Post("/{company_id:uuid}", handler.HandlerBy(createFn, handler.WithStatusCode(iris.StatusCreated)))

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	params, req := newHTTPRequest()
	e := httptest.New(t, app)
	e.POST("/"+params.CompanyID).WithQuery("country", params.Country).WithJSON(req).Expect().Status(iris.StatusCreated).JSON().
		Object().ContainsKey("company_id").HasValue("company_id", params.CompanyID).
		ContainsKey("country").HasValue("country", params.Country).
		ContainsKey("fullname").HasValue("fullname", req.Firstname+" "+req.Lastname).
		ContainsKey("is_adult").HasValue("is_adult", req.Age >= 18)
}
