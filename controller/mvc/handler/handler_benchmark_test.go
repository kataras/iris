package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/rest/errors"
)

func BenchmarkHandler(b *testing.B) {
	type Request struct {
		Firstname string `json:"firstname"`
		Lastname  string `json:"lastname"`
		Age       int    `json:"age"`
	}

	type Response struct {
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
		return httptest.NewRequest("POST", "http://example.com/", bytes.NewReader(reqBytes))
	}

	b.Run("iris std", func(b *testing.B) {
		app := iris.New()

		createHandler := func(ctx iris.Context) {
			var req Request
			if err := ctx.ReadJSON(&req); err != nil {
				errors.HandleError(ctx, err)
				return
			}

			resp := Response{
				Fullname: req.Firstname + " " + req.Lastname,
				IsAdult:  req.Age >= 18,
			}
			ctx.StatusCode(iris.StatusCreated)
			if err := ctx.JSON(resp); err != nil {
				errors.HandleError(ctx, err)
				return
			}
		}
		app.Post("/", createHandler)

		if err := app.Build(); err != nil {
			b.Fatal(err)
		}

		request := newHTTPRequest()
		recorder := httptest.NewRecorder()

		// b.ReportAllocs()
		for b.Loop() {
			app.ServeHTTP(recorder, request)
		}
	})

	b.Run("iris rest handler", func(b *testing.B) {
		app := iris.New()

		createFn := func(ctx iris.Context, req Request) (Response, error) {
			resp := Response{
				Fullname: req.Firstname + " " + req.Lastname,
				IsAdult:  req.Age >= 18,
			}
			return resp, nil
		}
		app.Post("/", handler.Handler(createFn, handler.WithStatusCode(iris.StatusCreated)))

		if err := app.Build(); err != nil {
			b.Fatal(err)
		}

		request := newHTTPRequest()
		recorder := httptest.NewRecorder()

		// b.ReportAllocs()
		for b.Loop() {
			app.ServeHTTP(recorder, request)
		}
	})

	/*
		$ go test -bench=BenchmarkHandler$ -benchmem -benchtime=20s
		goos: windows
		goarch: amd64
		pkg: github.com/kataras/iris/v14/rest
		cpu: 12th Gen Intel(R) Core(TM) i7-12700H
		BenchmarkHandler/iris_std-20            16209397              1502 ns/op            1665 B/op          9 allocs/op
		BenchmarkHandler/iris_rest_handler-20           14887050              1611 ns/op            1761 B/op         11 allocs/op
		PASS
		ok      github.com/kataras/iris/v14/rest        53.449s

		Based on the benchmark results above, the iris_std is the faster and more performant one. Here's why:

		BenchmarkHandler/iris_std-20:

		- 16209397 ops: The number of operations that can be performed per second.
		- 1502 ns/op: Nanoseconds taken for each operation, which is a measure of speed.
		- 1665 B/op: Bytes allocated per operation, indicating memory usage.
		- 9 allocs/op: Number of memory allocations per operation.

		BenchmarkHandler/iris_rest_handler-20:

		- 14887050 ops: The number of operations that can be performed per second.
		- 1611 ns/op: Nanoseconds taken for each operation.
		- 1761 B/op: Bytes allocated per operation.
		- 11 allocs/op: Number of memory allocations per operation.

		Comparing the two:

		- Speed: iris_std is faster than iris_rest_handler as it takes fewer nanoseconds per operation (1502 ns/op vs 1611 ns/op).
		- Memory Efficiency: iris_std is more memory-efficient, using fewer bytes per operation (1665 B/op vs 1761 B/op).
		- Allocation Efficiency: iris_std has fewer allocations per operation (9 allocs/op vs 11 allocs/op),
		  which generally indicates better performance as it suggests less pressure on the garbage collector.

		Conclusion: iris_std is approximately 6.77% faster and 5.45% more memory performant than iris_rest_handler. Which is a very very good number in terms of how
		simple the benchmark was and how much the iris_rest_handler is doing behind the scenes.
	*/
}
