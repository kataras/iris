package context_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

type (
	// testQueryFormTag carries one binding tag per field: `form`.
	testQueryFormTag struct {
		ID   int    `form:"hosp_id" json:"hosp_id"`
		Name string `form:"hosp_name" json:"hosp_name"`
	}

	testQueryTagOrder struct {
		Both     string `url:"by_url" form:"by_form" json:"both"`
		Excluded string `url:"-" form:"excluded" json:"excluded"`
		NoForm   string `form:"-" json:"no_form"`
		Plain    string
		JSONOnly string `json:"json_only"`
	}

	testQueryFormRequired struct {
		Token string `form:"token,required" json:"token"`
		Other string `form:"other" json:"other"`
	}
)

func newReadQueryFormTagApp() *iris.Application {
	app := iris.New()

	read := func(ptr any, bind func(iris.Context, any) error) iris.Handler {
		return func(ctx iris.Context) {
			if err := bind(ctx, ptr); err != nil && !iris.IsErrPath(err) {
				ctx.StopWithError(iris.StatusBadRequest, err)
				return
			}

			ctx.JSON(ptr)
		}
	}

	readQuery := func(ctx iris.Context, ptr any) error { return ctx.ReadQuery(ptr) }
	readBody := func(ctx iris.Context, ptr any) error { return ctx.ReadBody(ptr) }

	app.Get("/query", func(ctx iris.Context) { read(new(testQueryFormTag), readQuery)(ctx) })
	app.Any("/body", func(ctx iris.Context) { read(new(testQueryFormTag), readBody)(ctx) })
	app.Get("/query/order", func(ctx iris.Context) { read(new(testQueryTagOrder), readQuery)(ctx) })
	app.Get("/body/order", func(ctx iris.Context) { read(new(testQueryTagOrder), readBody)(ctx) })
	app.Get("/query/required", func(ctx iris.Context) { read(new(testQueryFormRequired), readQuery)(ctx) })
	app.Get("/body/required", func(ctx iris.Context) { read(new(testQueryFormRequired), readBody)(ctx) })

	return app
}

// ctx.ReadQuery, and ctx.ReadBody on a GET, bind a field that has no `url`
// tag by its `form` tag, so one `form` tag serves a GET query and a POST form.
// Issue #1725.
func TestReadQueryFormTagFallback(t *testing.T) {
	e := httptest.New(t, newReadQueryFormTagApp())

	for _, path := range []string{"/query", "/body"} {
		e.GET(path).WithQuery("hosp_id", 100).WithQuery("hosp_name", "get test name").
			Expect().Status(httptest.StatusOK).JSON().
			IsEqual(testQueryFormTag{ID: 100, Name: "get test name"})
	}

	// A form post reads its body by the same tag.
	e.POST("/body").WithQuery("hosp_id", 100).WithQuery("hosp_name", "get test name").
		WithFormField("hosp_name", "post test name").
		Expect().Status(httptest.StatusOK).JSON().
		IsEqual(testQueryFormTag{Name: "post test name"})

	// ctx.ReadBody reads the query on another method too, when the request
	// has a query and no content type it knows.
	e.DELETE("/body").WithQuery("hosp_id", 100).WithQuery("hosp_name", "get test name").
		Expect().Status(httptest.StatusOK).JSON().
		IsEqual(testQueryFormTag{ID: 100, Name: "get test name"})
}

// A `url` tag keeps its key, url:"-" and form:"-" keep the field out of the query.
// A field with no `url` and no `form` tag is bound by its name, as before.
func TestReadQueryFormTagFallbackOrder(t *testing.T) {
	e := httptest.New(t, newReadQueryFormTagApp())

	for _, path := range []string{"/query/order", "/body/order"} {
		e.GET(path).
			WithQuery("by_url", "from by_url").WithQuery("by_form", "from by_form").
			WithQuery("excluded", "x").WithQuery("Excluded", "x").
			WithQuery("NoForm", "x").WithQuery("no_form", "x").
			WithQuery("plain", "by field name").
			WithQuery("jsononly", "by field name").WithQuery("json_only", "x").
			Expect().Status(httptest.StatusOK).JSON().
			IsEqual(testQueryTagOrder{Both: "from by_url", Plain: "by field name", JSONOnly: "by field name"})
	}
}

// What the fallback changes for a field that already has a `form` tag and no
// `url` tag: its Go field name is no longer a query key, and the "required"
// option of the `form` tag now applies to a query that is not empty.
func TestReadQueryFormTagFallbackChanges(t *testing.T) {
	e := httptest.New(t, newReadQueryFormTagApp())

	for _, path := range []string{"/query", "/body"} {
		// Before the fallback these two bound ID and Name.
		e.GET(path).WithQuery("id", 100).WithQuery("name", "get test name").
			Expect().Status(httptest.StatusOK).JSON().IsEqual(testQueryFormTag{})

		e.GET(path).WithQuery("ID", 100).WithQuery("Name", "get test name").
			Expect().Status(httptest.StatusOK).JSON().IsEqual(testQueryFormTag{})
	}

	for _, path := range []string{"/query/required", "/body/required"} {
		// Before the fallback this was 200 with an empty token.
		e.GET(path).WithQuery("other", "1").
			Expect().Status(httptest.StatusBadRequest).Body().Contains("token is empty")

		e.GET(path).WithQuery("other", "1").WithQuery("token", "abc").
			Expect().Status(httptest.StatusOK).JSON().
			IsEqual(testQueryFormRequired{Token: "abc", Other: "1"})
	}

	// ctx.ReadQuery does not decode an empty query, so nothing is asked of it.
	e.GET("/query/required").Expect().Status(httptest.StatusOK).JSON().
		IsEqual(testQueryFormRequired{})
}
