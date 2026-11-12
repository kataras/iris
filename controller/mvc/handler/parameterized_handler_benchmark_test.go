package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"

	"uuid"
)

func BenchmarkHandlerBy(b *testing.B) {
	type Request struct {
		Firstname string `json:"firstname"` // read json.
		Lastname  string `json:"lastname"`
		Age       int    `json:"age"`
	}

	type RequestParams struct {
		CompanyID string `param:"company_id"` // read dynamic path parameters.
		Country   string `url:"country"`      // read query.
	}

	type Response struct {
		CompanyID string `json:"company_id"`
		Country   string `json:"country"`

		Fullname string `json:"fullname"`
		IsAdult  bool   `json:"is_adult"`
	}

	newHTTPRequest := func() *http.Request {
		req := Request{
			Firstname: "John",
			Lastname:  "Doe",
			Age:       25,
		}
		reqBytes, err := json.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}

		return httptest.NewRequest("POST", fmt.Sprintf("http://example.com/%s?country=%s", uuid.New().String(), "el-GR"), bytes.NewReader(reqBytes))
	}

	b.Run("iris std", func(b *testing.B) {
		app := iris.New()

		createHandler := func(ctx iris.Context) {
			var params RequestParams
			// bind dynamic path parameter(s).
			if err := ctx.ReadParams(&params); err != nil {
				errors.HandleError(ctx, err)
				return
			}

			// bind query.
			if err := ctx.ReadQuery(&params); err != nil {
				errors.HandleError(ctx, err)
				return
			}

			var req Request
			if err := ctx.ReadJSON(&req); err != nil {
				errors.HandleError(ctx, err)
				return
			}

			resp := Response{
				CompanyID: params.CompanyID,
				Country:   params.Country,
				Fullname:  req.Firstname + " " + req.Lastname,
				IsAdult:   req.Age >= 18,
			}
			ctx.StatusCode(iris.StatusCreated)
			if err := ctx.JSON(resp); err != nil {
				errors.HandleError(ctx, err)
				return
			}
		}
		app.Post("/{company_id:uuid}", createHandler)

		if err := app.Build(); err != nil {
			b.Fatal(err)
		}

		request := newHTTPRequest()

		// b.ReportAllocs()
		for b.Loop() {
			recorder := httptest.NewRecorder()
			app.ServeHTTP(recorder, request)
		}
	})

	b.Run("iris rest handler by", func(b *testing.B) {
		app := iris.New()

		createFn := func(ctx iris.Context, params RequestParams, req Request) (Response, error) {
			resp := Response{
				CompanyID: params.CompanyID,
				Fullname:  req.Firstname + " " + req.Lastname,
				Country:   params.Country,
				IsAdult:   req.Age >= 18,
			}
			return resp, nil
		}
		app.Post("/{company_id:uuid}", handler.HandlerBy(createFn, handler.WithStatusCode(iris.StatusCreated)))

		if err := app.Build(); err != nil {
			b.Fatal(err)
		}

		request := newHTTPRequest()

		// b.ReportAllocs()
		for b.Loop() {
			recorder := httptest.NewRecorder()
			app.ServeHTTP(recorder, request)
		}
	})

	// Before new param decoder:
	// BenchmarkHandlerBy/iris_std-20           				9468645              2622 ns/op            2079 B/op         25 allocs/op
	// BenchmarkHandlerBy/iris_rest_handler_by-20               8906832              2633 ns/op            2162 B/op         27 allocs/op
	// After new param decoder:
	// BenchmarkHandlerBy/iris_std-20           				9554769              2451 ns/op            2073 B/op         25 allocs/op
	// BenchmarkHandlerBy/iris_rest_handler_by-20              12306890              1910 ns/op            1684 B/op         15 allocs/op
	// Winner: iris_rest_handler_by with a 22% faster response time and 18% less memory allocations.
}
