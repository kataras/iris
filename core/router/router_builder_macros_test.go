package router_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/core/router/macro"
	"github.com/kataras/iris/v14/httptest"
)

// Each application owns its macros: what one of them registers through
// app.Macros() stays in it. They used to share the macro.Defaults pointer, so
// an error handler, a macro function or a parameter type registered on one
// application changed every other application of the process.
func TestMacrosArePerApplication(t *testing.T) {
	t.Run("error handler", func(t *testing.T) {
		register := func(app *iris.Application) {
			app.Get("/users/{name:string}", func(ctx iris.Context) {
				ctx.Writef("string %s", ctx.Params().GetString("name"))
			})
			app.Get("/users/{id:int}", func(ctx iris.Context) {
				ctx.Writef("int %d", ctx.Params().GetInt("id"))
			})
		}

		first := iris.New()
		first.Macros().SetErrorHandler(func(ctx iris.Context, _ int, _ error) {
			ctx.StopWithText(iris.StatusBadRequest, "handled by first")
		})
		register(first)

		second := iris.New()
		register(second)

		// A value that is not an int is offered to the next route of the path,
		// unless an error handler answers for it.
		httptest.New(t, second).GET("/users/bob").Expect().
			Status(iris.StatusOK).Body().IsEqual("string bob")
		httptest.New(t, first).GET("/users/bob").Expect().
			Status(iris.StatusBadRequest).Body().IsEqual("handled by first")
	})

	t.Run("macro function", func(t *testing.T) {
		never := func() func(int) bool {
			return func(int) bool { return false }
		}
		register := func(app *iris.Application) {
			app.Get("/new/{n:int refuse()}", func(ctx iris.Context) { ctx.WriteString("ok") })
			app.Get("/replaced/{n:int min(1)}", func(ctx iris.Context) { ctx.WriteString("ok") })
		}

		first := iris.New()
		// One function the int macro does not have and one it has.
		first.Macros().Get("int").
			RegisterFunc("refuse", never).
			RegisterFunc("min", func(int) func(int) bool { return never() })
		register(first)

		second := iris.New()
		// An unknown function is skipped, so "refuse()" checks nothing here.
		register(second)

		e := httptest.New(t, second)
		e.GET("/new/5").Expect().Status(iris.StatusOK)
		e.GET("/replaced/5").Expect().Status(iris.StatusOK)

		e = httptest.New(t, first)
		e.GET("/new/5").Expect().Status(iris.StatusNotFound)
		e.GET("/replaced/5").Expect().Status(iris.StatusNotFound)

		// The package level ones are left as they were too.
		tmpl, err := macro.Parse("/{n:int refuse()}", *macro.Defaults)
		if err != nil {
			t.Fatal(err)
		}
		if n := len(tmpl.Params[0].Funcs); n != 0 {
			t.Fatalf("macro.Int got the function registered through app.Macros(): %d funcs", n)
		}
	})

	t.Run("parameter type", func(t *testing.T) {
		first, second := iris.New(), iris.New()
		if first.Macros().Register("first_only", "", "", false, false, nil) == nil {
			t.Fatal("first: the macro was not registered")
		}

		if second.Macros().Get("first_only") != nil {
			t.Fatal("second: got the macro registered on first")
		}
		if macro.Defaults.Get("first_only") != nil {
			t.Fatal("macro.Defaults: got the macro registered on first")
		}

		if !second.Macros().Unregister("uuid") {
			t.Fatal("second: uuid was not unregistered")
		}
		if first.Macros().Get("uuid") == nil || macro.Defaults.Get("uuid") == nil {
			t.Fatal("unregistering uuid from second removed it elsewhere")
		}
	})
}

// What is registered on macro.Defaults before an application is created is
// part of that application: a parameter type, its functions and its error
// handler.
func TestMacrosStartFromDefaults(t *testing.T) {
	even := macro.Defaults.Register("even_test_only", "", 0, false, false, macro.Int.Evaluator)
	if even == nil {
		t.Fatal("the macro was not registered on macro.Defaults")
	}
	t.Cleanup(func() { macro.Defaults.Unregister("even_test_only") })

	even.RegisterFunc("even", func() func(int) bool {
		return func(n int) bool { return n%2 == 0 }
	})
	even.SetErrorHandler(func(ctx iris.Context, _ int, _ error) {
		ctx.StopWithText(iris.StatusBadRequest, "not a number")
	})

	app := iris.New()
	app.Get("/{n:even_test_only even()}", func(ctx iris.Context) {
		ctx.Writef("even %d", ctx.Params().GetInt("n"))
	})

	e := httptest.New(t, app)
	e.GET("/4").Expect().Status(iris.StatusOK).Body().IsEqual("even 4")
	e.GET("/5").Expect().Status(iris.StatusNotFound)
	e.GET("/x").Expect().Status(iris.StatusBadRequest).Body().IsEqual("not a number")
}

// A route's parameter still reports the package level macro it was declared
// with: HandleServer asks IsMacro(macro.Path) to find the wildcard the
// developer named, and a documentation generator asks it for the type.
func TestTemplateParamIsMacroOfApplication(t *testing.T) {
	app := iris.New()
	r := app.Get("/files/{id:int}/{p:path}", func(iris.Context) {})

	params := r.Tmpl().Params
	if !params[0].IsMacro(macro.Int) || !params[0].IsMacro(app.Macros().Get("int")) {
		t.Fatal("{id:int} is not reported as the int macro")
	}
	if !params[1].IsMacro(macro.Path) || !params[1].IsMacro(app.Macros().Get("path")) {
		t.Fatal("{p:path} is not reported as the path macro")
	}
	if params[0].IsMacro(macro.Path) || params[1].IsMacro(macro.Int) || params[0].IsMacro(nil) {
		t.Fatal("a parameter is reported as a macro of another type")
	}
}
