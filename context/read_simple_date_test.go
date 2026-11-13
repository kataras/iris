package context_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/types"
)

type (
	testSimpleDateQuery struct {
		Limit int              `url:"limit"`
		Start types.SimpleDate `url:"startdate"`
		End   types.SimpleDate `url:"enddate"`
	}

	testSimpleDatePointerQuery struct {
		Limit int               `url:"limit"`
		Start *types.SimpleDate `url:"startdate"`
	}

	testSimpleDateParams struct {
		Day types.SimpleDate `param:"day"`
	}

	testSimpleDateURL struct {
		Day   types.SimpleDate `param:"day"`
		Start types.SimpleDate `url:"startdate"`
	}

	testSimpleDateForm struct {
		Limit int              `form:"limit"`
		Start types.SimpleDate `form:"startdate"`
	}

	testSimpleDateHeader struct {
		Day types.SimpleDate `header:"X-Day"`
	}

	testSimpleDateMsgPack struct {
		Start types.SimpleDate `msgpack:"start"`
	}
)

// newReadSimpleDateApp writes the bound values on success and a 400 with the
// binder's error on failure.
func newReadSimpleDateApp() *iris.Application {
	app := iris.New()

	fail := func(ctx iris.Context, err error) {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.WriteString(err.Error())
	}

	app.Get("/query", func(ctx iris.Context) {
		var q testSimpleDateQuery
		if err := ctx.ReadQuery(&q); err != nil {
			fail(ctx, err)
			return
		}
		ctx.Writef("limit=%d start=%q end=%q", q.Limit, q.Start, q.End)
	})
	app.Get("/query-pointer", func(ctx iris.Context) {
		var q testSimpleDatePointerQuery
		if err := ctx.ReadQuery(&q); err != nil {
			fail(ctx, err)
			return
		}
		if q.Start == nil {
			ctx.Writef("limit=%d start=nil", q.Limit)
			return
		}
		ctx.Writef("limit=%d start=%q", q.Limit, *q.Start)
	})
	app.Get("/params/{day}", func(ctx iris.Context) {
		var p testSimpleDateParams
		if err := ctx.ReadParams(&p); err != nil {
			fail(ctx, err)
			return
		}
		ctx.Writef("day=%q", p.Day)
	})
	app.Get("/url/{day}", func(ctx iris.Context) {
		var u testSimpleDateURL
		if err := ctx.ReadURL(&u); err != nil {
			fail(ctx, err)
			return
		}
		ctx.Writef("day=%q start=%q", u.Day, u.Start)
	})
	app.Post("/form", func(ctx iris.Context) {
		var f testSimpleDateForm
		if err := ctx.ReadForm(&f); err != nil {
			fail(ctx, err)
			return
		}
		ctx.Writef("limit=%d start=%q", f.Limit, f.Start)
	})
	app.Get("/header", func(ctx iris.Context) {
		var h testSimpleDateHeader
		if err := ctx.ReadHeader(&h); err != nil {
			fail(ctx, err)
			return
		}
		ctx.Writef("day=%q", h.Day)
	})

	app.Get("/msgpack", func(ctx iris.Context) {
		start, _ := types.ParseSimpleDate(ctx.Query().GetString("start"))
		ctx.MsgPack(testSimpleDateMsgPack{Start: start})
	})
	app.Post("/msgpack", func(ctx iris.Context) {
		var v testSimpleDateMsgPack
		if err := ctx.ReadMsgPack(&v); err != nil {
			fail(ctx, err)
			return
		}
		ctx.Writef("start=%q", v.Start)
	})

	return app
}

// ReadQuery fills a types.SimpleDate field from a YYYY-MM-DD value. It used to
// fail with "schema: converter not found for types.SimpleDate".
// Issue #2486.
func TestReadQuerySimpleDate(t *testing.T) {
	e := httptest.New(t, newReadSimpleDateApp())

	e.GET("/query").WithQuery("startdate", "2024-09-01").WithQuery("enddate", "2024-09-30").WithQuery("limit", 3).
		Expect().Status(httptest.StatusOK).Body().IsEqual(`limit=3 start="2024-09-01" end="2024-09-30"`)

	// An absent key and an empty value leave the date zero and are not an
	// error, the same as the int field next to it.
	e.GET("/query").Expect().Status(httptest.StatusOK).Body().IsEqual(`limit=0 start="" end=""`)
	e.GET("/query").WithQuery("startdate", "").WithQuery("limit", "").
		Expect().Status(httptest.StatusOK).Body().IsEqual(`limit=0 start="" end=""`)

	// Anything that is not YYYY-MM-DD is a conversion error naming the key.
	for _, bad := range []string{"01.09.2024", "2024-9-1", "2024-09-01T00:00:00Z", "2024-02-30", "null"} {
		e.GET("/query").WithQuery("startdate", bad).
			Expect().Status(httptest.StatusBadRequest).Body().Contains(`error converting value for "startdate"`)
	}
}

// A *types.SimpleDate field is filled the same way and stays nil when its key
// is absent, which is how an optional date tells "not sent" from a date.
// Issue #2486.
func TestReadQuerySimpleDatePointer(t *testing.T) {
	e := httptest.New(t, newReadSimpleDateApp())

	e.GET("/query-pointer").WithQuery("startdate", "2024-09-01").WithQuery("limit", 3).
		Expect().Status(httptest.StatusOK).Body().IsEqual(`limit=3 start="2024-09-01"`)
	e.GET("/query-pointer").WithQuery("limit", 3).
		Expect().Status(httptest.StatusOK).Body().IsEqual(`limit=3 start=nil`)
	e.GET("/query-pointer").WithQuery("startdate", "yesterday").
		Expect().Status(httptest.StatusBadRequest).Body().Contains(`error converting value for "startdate"`)
}

// ReadParams and ReadURL fill a types.SimpleDate field from a path parameter.
// Issue #2486.
func TestReadParamsSimpleDate(t *testing.T) {
	e := httptest.New(t, newReadSimpleDateApp())

	e.GET("/params/2024-09-01").Expect().Status(httptest.StatusOK).Body().IsEqual(`day="2024-09-01"`)
	e.GET("/params/yesterday").Expect().Status(httptest.StatusBadRequest).
		Body().Contains(`error converting value for "day"`)

	e.GET("/url/2024-09-01").WithQuery("startdate", "2024-09-30").
		Expect().Status(httptest.StatusOK).Body().IsEqual(`day="2024-09-01" start="2024-09-30"`)
	e.GET("/url/2024-09-01").Expect().Status(httptest.StatusOK).Body().IsEqual(`day="2024-09-01" start=""`)
}

// ReadForm fills a types.SimpleDate field from a form value.
// Issue #2486.
func TestReadFormSimpleDate(t *testing.T) {
	e := httptest.New(t, newReadSimpleDateApp())

	e.POST("/form").WithFormField("startdate", "2024-09-01").WithFormField("limit", 3).
		Expect().Status(httptest.StatusOK).Body().IsEqual(`limit=3 start="2024-09-01"`)
	e.POST("/form").WithFormField("startdate", "").WithFormField("limit", "").
		Expect().Status(httptest.StatusOK).Body().IsEqual(`limit=0 start=""`)
	e.POST("/form").WithFormField("startdate", "yesterday").
		Expect().Status(httptest.StatusBadRequest).Body().Contains(`error converting value for "startdate"`)
}

// ReadHeader fills a types.SimpleDate field from a request header.
// Issue #2486.
func TestReadHeaderSimpleDate(t *testing.T) {
	e := httptest.New(t, newReadSimpleDateApp())

	e.GET("/header").WithHeader("X-Day", "2024-09-01").
		Expect().Status(httptest.StatusOK).Body().IsEqual(`day="2024-09-01"`)
	e.GET("/header").Expect().Status(httptest.StatusOK).Body().IsEqual(`day=""`)
}

// What ctx.MsgPack writes for a types.SimpleDate, ctx.ReadMsgPack reads back.
// msgpack decodes a type through its UnmarshalText once it has one, so the
// type needs MarshalText too: without it the date is still written as an
// empty map and reading that back fails.
func TestMsgPackSimpleDateRoundTrip(t *testing.T) {
	e := httptest.New(t, newReadSimpleDateApp())

	for _, start := range []string{"2024-09-01", ""} {
		body := e.GET("/msgpack").WithQuery("start", start).Expect().Status(httptest.StatusOK).Body().Raw()

		e.POST("/msgpack").WithBytes([]byte(body)).
			Expect().Status(httptest.StatusOK).Body().IsEqual(`start="` + start + `"`)
	}
}
