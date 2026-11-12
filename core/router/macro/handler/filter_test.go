package handler_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// Test the MakeFilter, which overrides the parameter name and value on "decision route handlers" (when more than one route is registered on the same path but different path param types).
func TestMakeFilterWithDecisionOnParamType(t *testing.T) {
	app := iris.New()
	logParams := func(ctx iris.Context) {
		ctx.Writef("Path: %s\nCurrent Route: %s\n", ctx.Path(), ctx.CurrentRoute())
		ctx.Params().ForEach(func(key string, value any) {
			ctx.Writef("key: %s, value: %v\n", key, value)
		})
	}

	// fallbacks first.
	app.Get("/u/{p:path}", logParams)
	app.Get("/u/{username:string}", logParams)
	app.Get("/u/{firstname:alphabetical}", logParams)
	app.Get("/u/{id:int}", logParams)
	app.Get("/u/{uid:uint}", logParams)

	e := httptest.New(t, app)
	expectedPathTypeResponse := "Path: /u/some/path/here\nCurrent Route: GET /u/{p:path}\nkey: p, value: some/path/here\n"
	expectedStringTypeResponse := "Path: /u/abcd123\nCurrent Route: GET /u/{username:string}\nkey: username, value: abcd123\n"
	expectedAlphabeticalTypeResponse := "Path: /u/abcd\nCurrent Route: GET /u/{firstname:alphabetical}\nkey: firstname, value: abcd\n"
	expectedIntTypeResponse := "Path: /u/-42\nCurrent Route: GET /u/{id:int}\nkey: id, value: -42\n"
	expectedUintTypeResponse := "Path: /u/42\nCurrent Route: GET /u/{uid:uint}\nkey: uid, value: 42\n"

	e.GET("/u/some/path/here").Expect().Status(httptest.StatusOK).Body().IsEqual(expectedPathTypeResponse)
	e.GET("/u/abcd123").Expect().Status(httptest.StatusOK).Body().IsEqual(expectedStringTypeResponse)
	e.GET("/u/abcd").Expect().Status(httptest.StatusOK).Body().IsEqual(expectedAlphabeticalTypeResponse)
	e.GET("/u/-42").Expect().Status(httptest.StatusOK).Body().IsEqual(expectedIntTypeResponse)
	e.GET("/u/42").Expect().Status(httptest.StatusOK).Body().IsEqual(expectedUintTypeResponse)
}

// The string funcs (prefix, suffix, contains, min, max, eq, eqor, regexp) are
// inherited by every macro that lacks the func. A typed macro stores a
// non-string value, so the func must be fed the value's text instead of
// panicking inside the reflect call on the first request.
func TestMakeFilterStringFuncsOnTypedMacros(t *testing.T) {
	const id = "a6ce7330-d584-4675-af9e-2c2d2c40de07"

	app := iris.New()
	app.Get("/n/{id:int prefix(4)}", func(ctx iris.Context) {
		ctx.Writef("%d", ctx.Params().GetInt("id"))
	})
	app.Get("/u/{id:uuid prefix(a6)}", func(ctx iris.Context) {
		ctx.WriteString(ctx.Params().GetString("id"))
	})

	e := httptest.New(t, app)
	e.GET("/n/42").Expect().Status(httptest.StatusOK).Body().IsEqual("42")
	e.GET("/n/24").Expect().Status(httptest.StatusNotFound)
	e.GET("/u/" + id).Expect().Status(httptest.StatusOK).Body().IsEqual(id)
	e.GET("/u/b" + id[1:]).Expect().Status(httptest.StatusNotFound)
}

// The "uuid" macro stores the parsed uuid.UUID, like "int" stores an int, so
// the handler reads it with GetUUID, and the text form is the canonical
// lowercase hyphenated one whatever spelling the client sent.
func TestMakeFilterUUIDStoresParsedValue(t *testing.T) {
	const canonical = "a6ce7330-d584-4675-af9e-2c2d2c40de07"

	app := iris.New()
	app.Get("/u/{id:uuid}", func(ctx iris.Context) {
		ctx.Writef("%T %s %s", ctx.Params().Get("id"), ctx.Params().GetString("id"), ctx.Params().GetUUID("id"))
	})

	e := httptest.New(t, app)
	want := "uuid.UUID " + canonical + " " + canonical
	for _, input := range []string{
		canonical,
		"A6CE7330-D584-4675-AF9E-2C2D2C40DE07",
		"urn:uuid:" + canonical,
		"a6ce7330d5844675af9e2c2d2c40de07",
	} {
		e.GET("/u/" + input).Expect().Status(httptest.StatusOK).Body().IsEqual(want)
	}
	e.GET("/u/not-a-uuid").Expect().Status(httptest.StatusNotFound)
}
