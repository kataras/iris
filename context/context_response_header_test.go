package context_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// ResponseHeader reads what this handler set on the response; the request's
// headers are reachable only through Header. A client-sent Authorization
// header must be invisible to ResponseHeader.
func TestResponseHeaderReadsResponseOnly(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.SetHeader("X-Rate-Remaining", "41")

		ctx.JSON(iris.Map{
			"request_auth":  ctx.Header("Authorization"),
			"response_auth": ctx.ResponseHeader("Authorization"),
			"roundtrip":     ctx.ResponseHeader("X-Rate-Remaining"),
		})
	})

	e := httptest.New(t, app)

	obj := e.GET("/").WithHeader("Authorization", "Bearer secret-token").
		Expect().Status(httptest.StatusOK).JSON().Object()
	obj.HasValue("request_auth", "Bearer secret-token")
	obj.HasValue("response_auth", "")
	obj.HasValue("roundtrip", "41")
}

// ResponseContentType and ResponseMediaType read the response writer's
// Content-Type header.
func TestResponseContentTypeAccessors(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		if err := ctx.SetContentType("application/json"); err != nil {
			ctx.StatusCode(httptest.StatusInternalServerError)
			return
		}

		mediaType := ctx.ResponseMediaType()
		parsed := ctx.ResponseContentType().MediaType
		if mediaType != parsed {
			ctx.StatusCode(httptest.StatusInternalServerError)
			ctx.WriteString(mediaType + " != " + parsed)
			return
		}

		ctx.WriteString(mediaType)
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("application/json")
}

// SetHeader sets: a second call with the same name replaces the first value.
// AddHeader adds: a second call appends, which is what a list-valued header
// such as Vary needs. An empty SetHeader value deletes the header.
func TestSetHeaderReplacesAddHeaderAppends(t *testing.T) {
	app := iris.New()
	app.Get("/set", func(ctx iris.Context) {
		ctx.SetHeader("X-Single", "first")
		ctx.SetHeader("X-Single", "second")
		ctx.SetHeader("X-Gone", "value")
		ctx.SetHeader("X-Gone", "")
		ctx.WriteString("ok")
	})
	app.Get("/add", func(ctx iris.Context) {
		ctx.AddHeader("Vary", "Origin")
		ctx.AddHeader("Vary", "Accept-Encoding")
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)

	set := e.GET("/set").Expect().Status(httptest.StatusOK)
	set.Headers().Value("X-Single").Array().IsEqual([]string{"second"})
	set.Headers().NotContainsKey("X-Gone")

	add := e.GET("/add").Expect().Status(httptest.StatusOK)
	add.Headers().Value("Vary").Array().IsEqual([]string{"Origin", "Accept-Encoding"})
}
