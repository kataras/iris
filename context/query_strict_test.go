package context_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/types"
)

// expectedDate is the marker the date routes write around the value, so a
// handler that wrote nothing cannot satisfy a zero-date assertion.
func expectedDate(date types.SimpleDate) string {
	return `date="` + date.String() + `"`
}

func newQueryStrictApp() *iris.Application {
	app := iris.New()

	app.Get("/date", func(ctx iris.Context) {
		date, ok := ctx.QueryDate("date")
		if !ok {
			return
		}
		ctx.Writef("date=%q", date.String())
	})
	app.Get("/date/required", func(ctx iris.Context) {
		date, ok := ctx.QueryDate("date", iris.QueryRequired())
		if !ok {
			return
		}
		ctx.Writef("date=%q", date.String())
	})
	app.Get("/date/default", func(ctx iris.Context) {
		date, ok := ctx.QueryDate("date", iris.QueryDefault("2020-01-02"))
		if !ok {
			return
		}
		ctx.Writef("date=%q", date.String())
	})
	app.Get("/date/zero", func(ctx iris.Context) {
		date, ok := ctx.QueryDate("date", iris.QueryZeroValues("0", "zero", "0000-00-00", "null"))
		if !ok {
			return
		}
		ctx.Writef("date=%q", date.String())
	})

	app.Get("/date/default-zero", func(ctx iris.Context) {
		date, ok := ctx.QueryDate("date",
			iris.QueryDefault("2020-01-02"), iris.QueryZeroValues("0", "null"))
		if !ok {
			return
		}
		ctx.Writef("date=%q", date.String())
	})
	app.Get("/date/bad-default", func(ctx iris.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				ctx.Writef("%v", recovered)
			}
		}()

		_, _ = ctx.QueryDate("date", iris.QueryDefault("not-a-date"))
	})

	app.Get("/date/empty-default", func(ctx iris.Context) {
		date, ok := ctx.QueryDate("date", iris.QueryDefault(""), iris.QueryRequired())
		if !ok {
			return
		}
		ctx.Writef("date=%q", date.String())
	})

	app.Get("/int", func(ctx iris.Context) {
		n, ok := ctx.QueryInt("limit")
		if !ok {
			return
		}
		ctx.Writef("%d", n)
	})
	app.Get("/int/required", func(ctx iris.Context) {
		n, ok := ctx.QueryInt("limit", iris.QueryRequired())
		if !ok {
			return
		}
		ctx.Writef("%d", n)
	})
	app.Get("/int/default", func(ctx iris.Context) {
		n, ok := ctx.QueryInt("limit", iris.QueryDefault("25"))
		if !ok {
			return
		}
		ctx.Writef("%d", n)
	})

	app.Get("/int/empty-default", func(ctx iris.Context) {
		n, ok := ctx.QueryInt("limit", iris.QueryDefault("  "))
		if !ok {
			return
		}
		ctx.Writef("%d", n)
	})

	app.Get("/bool", func(ctx iris.Context) {
		b, ok := ctx.QueryBool("details")
		if !ok {
			return
		}
		ctx.Writef("%v", b)
	})
	app.Get("/bool/required", func(ctx iris.Context) {
		b, ok := ctx.QueryBool("details", iris.QueryRequired())
		if !ok {
			return
		}
		ctx.Writef("%v", b)
	})
	app.Get("/bool/default", func(ctx iris.Context) {
		b, ok := ctx.QueryBool("details", iris.QueryDefault("true"))
		if !ok {
			return
		}
		ctx.Writef("%v", b)
	})

	// Multi-key routes: the key names the alternatives, tried in order.
	app.Get("/date/keys", func(ctx iris.Context) {
		date, ok := ctx.QueryDate("date day from")
		if !ok {
			return
		}
		ctx.Writef("date=%q", date.String())
	})
	app.Get("/date/keys/required", func(ctx iris.Context) {
		date, ok := ctx.QueryDate("date day from", iris.QueryRequired())
		if !ok {
			return
		}
		ctx.Writef("date=%q", date.String())
	})
	app.Get("/date/keys/zero", func(ctx iris.Context) {
		date, ok := ctx.QueryDate("date day", iris.QueryZeroValues("0", "0000-00-00"))
		if !ok {
			return
		}
		ctx.Writef("date=%q", date.String())
	})
	app.Get("/int/keys", func(ctx iris.Context) {
		n, ok := ctx.QueryInt("limit size")
		if !ok {
			return
		}
		ctx.Writef("%d", n)
	})

	return app
}

// A garbage query parameter is a 400 naming the key, never a silent zero.
func TestQueryStrictGarbage(t *testing.T) {
	e := httptest.New(t, newQueryStrictApp())

	e.GET("/date").WithQuery("date", "not-a-date").Expect().
		Status(httptest.StatusBadRequest).Body().Contains("date")
	// The date parser maps "null" to the zero date without an error. Only
	// QueryZeroValues may declare a raw value to mean "unset", so a reader
	// without it must answer 400 like any other garbage.
	for _, raw := range []string{"null", "NULL"} {
		e.GET("/date").WithQuery("date", raw).Expect().
			Status(httptest.StatusBadRequest).Body().Contains("date")
	}
	e.GET("/int").WithQuery("limit", "ten").Expect().
		Status(httptest.StatusBadRequest).Body().Contains("limit")
	e.GET("/bool").WithQuery("details", "maybe").Expect().
		Status(httptest.StatusBadRequest).Body().Contains("details")
}

// Absent and not required is the zero value with a true report.
func TestQueryStrictAbsent(t *testing.T) {
	e := httptest.New(t, newQueryStrictApp())

	zeroDate := expectedDate(types.SimpleDate{})
	e.GET("/date").Expect().Status(httptest.StatusOK).Body().IsEqual(zeroDate)
	e.GET("/int").Expect().Status(httptest.StatusOK).Body().IsEqual("0")
	e.GET("/bool").Expect().Status(httptest.StatusOK).Body().IsEqual("false")

	// A blank value counts as absent.
	e.GET("/int").WithQuery("limit", "").Expect().Status(httptest.StatusOK).Body().IsEqual("0")
}

// QueryDefault parses its raw value like a query value when the key is absent.
func TestQueryStrictDefault(t *testing.T) {
	e := httptest.New(t, newQueryStrictApp())

	e.GET("/date/default").Expect().Status(httptest.StatusOK).Body().IsEqual(`date="2020-01-02"`)
	e.GET("/date/default").WithQuery("date", "2021-03-04").Expect().
		Status(httptest.StatusOK).Body().IsEqual(`date="2021-03-04"`)
	e.GET("/int/default").Expect().Status(httptest.StatusOK).Body().IsEqual("25")
	e.GET("/int/default").WithQuery("limit", "5").Expect().Status(httptest.StatusOK).Body().IsEqual("5")
	e.GET("/bool/default").Expect().Status(httptest.StatusOK).Body().IsEqual("true")
	e.GET("/bool/default").WithQuery("details", "off").Expect().Status(httptest.StatusOK).Body().IsEqual("false")
}

// QueryRequired turns an absent key into a 400 naming the key.
func TestQueryStrictRequired(t *testing.T) {
	e := httptest.New(t, newQueryStrictApp())

	e.GET("/date/required").Expect().Status(httptest.StatusBadRequest).Body().Contains("date")
	e.GET("/int/required").Expect().Status(httptest.StatusBadRequest).Body().Contains("limit")
	e.GET("/bool/required").Expect().Status(httptest.StatusBadRequest).Body().Contains("details")

	e.GET("/date/required").WithQuery("date", "2020-01-02").Expect().
		Status(httptest.StatusOK).Body().IsEqual(`date="2020-01-02"`)
}

// A QueryZeroValues raw value reads as an absent one, so a legacy "0" or
// "0000-00-00" date filter is not a client error.
func TestQueryStrictZeroValues(t *testing.T) {
	e := httptest.New(t, newQueryStrictApp())

	zeroDate := expectedDate(types.SimpleDate{})
	for _, raw := range []string{"0", "zero", "0000-00-00", "null"} {
		e.GET("/date/zero").WithQuery("date", raw).Expect().
			Status(httptest.StatusOK).Body().IsEqual(zeroDate)
	}

	// A zero placeholder reads as an absent value, so it takes the default.
	for _, raw := range []string{"0", "null", "NULL"} {
		e.GET("/date/default-zero").WithQuery("date", raw).Expect().
			Status(httptest.StatusOK).Body().IsEqual(`date="2020-01-02"`)
	}
	e.GET("/date/default-zero").WithQuery("date", "2021-03-04").Expect().
		Status(httptest.StatusOK).Body().IsEqual(`date="2021-03-04"`)

	// Without the option the same raw value is a 400.
	e.GET("/date").WithQuery("date", "0000-00-00").Expect().Status(httptest.StatusBadRequest)
}

// A blank QueryDefault is no default: it must not become a silent zero, and
// the three readers must agree about it.
func TestQueryStrictEmptyDefault(t *testing.T) {
	e := httptest.New(t, newQueryStrictApp())

	// No default to fall back on, so QueryRequired still applies.
	e.GET("/date/empty-default").Expect().
		Status(httptest.StatusBadRequest).Body().Contains("date")
	e.GET("/date/empty-default").WithQuery("date", "2020-01-02").Expect().
		Status(httptest.StatusOK).Body().IsEqual(`date="2020-01-02"`)

	// And no panic on the reader whose parser rejects a blank value.
	e.GET("/int/empty-default").Expect().Status(httptest.StatusOK).Body().IsEqual("0")
}

// An unparsable QueryDefault is a server-side mistake and panics naming the
// key, the value and the shape expected.
func TestQueryStrictBadDefaultPanics(t *testing.T) {
	e := httptest.New(t, newQueryStrictApp())

	body := e.GET("/date/bad-default").Expect().Status(httptest.StatusOK).Body()
	body.Contains("date")
	body.Contains("not-a-date")
}

// QueryBool accepts the strconv forms plus yes/no/on/off, case-insensitively.
func TestQueryStrictBoolForms(t *testing.T) {
	e := httptest.New(t, newQueryStrictApp())

	for _, raw := range []string{"true", "TRUE", "1", "t", "yes", "YES", "on", "On"} {
		e.GET("/bool").WithQuery("details", raw).Expect().Status(httptest.StatusOK).Body().IsEqual("true")
	}
	for _, raw := range []string{"false", "0", "f", "no", "NO", "off", "Off"} {
		e.GET("/bool").WithQuery("details", raw).Expect().Status(httptest.StatusOK).Body().IsEqual("false")
	}
}

// A key naming several alternatives reads the first one that carries a value,
// in the order stated, not the order the client sent them.
func TestQueryStrictMultipleKeys(t *testing.T) {
	e := httptest.New(t, newQueryStrictApp())
	zeroDate := expectedDate(types.SimpleDate{})

	// The first key wins when it is there.
	e.GET("/date/keys").WithQuery("date", "2020-01-02").Expect().
		Status(httptest.StatusOK).Body().IsEqual(`date="2020-01-02"`)

	// Each later alternative answers when the earlier ones are absent.
	e.GET("/date/keys").WithQuery("day", "2021-03-04").Expect().
		Status(httptest.StatusOK).Body().IsEqual(`date="2021-03-04"`)
	e.GET("/date/keys").WithQuery("from", "2022-05-06").Expect().
		Status(httptest.StatusOK).Body().IsEqual(`date="2022-05-06"`)

	// Stated order decides, not the query string's order.
	e.GET("/date/keys").WithQuery("from", "2022-05-06").WithQuery("date", "2020-01-02").Expect().
		Status(httptest.StatusOK).Body().IsEqual(`date="2020-01-02"`)

	// A blank earlier key is an absent one, so the next alternative answers.
	e.GET("/date/keys").WithQuery("date", "").WithQuery("day", "2021-03-04").Expect().
		Status(httptest.StatusOK).Body().IsEqual(`date="2021-03-04"`)

	// None present is the zero value, exactly as a single absent key is.
	e.GET("/date/keys").Expect().Status(httptest.StatusOK).Body().IsEqual(zeroDate)

	e.GET("/int/keys").WithQuery("size", "5").Expect().
		Status(httptest.StatusOK).Body().IsEqual("5")
}

// The 400 of a multi-key reader names every accepted key when none was sent,
// and only the key that carried the bad value when one was.
func TestQueryStrictMultipleKeysMessages(t *testing.T) {
	e := httptest.New(t, newQueryStrictApp())

	missing := e.GET("/date/keys/required").Expect().
		Status(httptest.StatusBadRequest).Body()
	missing.Contains("date")
	missing.Contains("day")
	missing.Contains("from")

	// Only the offending key is named: the client did not send the others.
	invalid := e.GET("/date/keys").WithQuery("day", "not-a-date").Expect().
		Status(httptest.StatusBadRequest).Body()
	invalid.Contains("day")
	invalid.NotContains("from")
}

// A zero placeholder on an earlier key does not shadow a real value on a later
// one: the placeholder means "unset", so the reader keeps looking.
func TestQueryStrictMultipleKeysZeroValues(t *testing.T) {
	e := httptest.New(t, newQueryStrictApp())
	zeroDate := expectedDate(types.SimpleDate{})

	e.GET("/date/keys/zero").WithQuery("date", "0000-00-00").Expect().
		Status(httptest.StatusOK).Body().IsEqual(zeroDate)
	e.GET("/date/keys/zero").WithQuery("day", "2021-03-04").Expect().
		Status(httptest.StatusOK).Body().IsEqual(`date="2021-03-04"`)
}
