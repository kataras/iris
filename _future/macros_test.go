// I would be grateful if I had the chance to see the whole work-in-progress in a codebase when I started.
// You have the chance to learn faster nowdays, don't underestimate that, that's the only reason that this "_future" folder exists now.
//
// The whole "router" package is a temp place to test my ideas and implementations for future iris' features.
// Young developers can understand and see how ideas can be transform to real implementations on a software like Iris,
// watching the history of a "dirty" code can be useful for some of you.

package router

import (
	"regexp"
	"testing"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/httptest"
)

// macros should be registered before .Listen
type _macros map[string]func(string) bool

// a helper to return a macro from a simple regexp
// it compiles the regexp  and after returns the macro, for obviously performance reasons.
func fromRegexp(expr string) func(paramValue string) bool {
	if expr == "" {
		panic("empty expr on regex")
	}

	// add the last $ if missing (and not wildcard(?))
	if i := expr[len(expr)-1]; i != '$' && i != '*' {
		expr += "$"
	}

	r, err := regexp.Compile(expr)
	if err != nil {
		panic(err)
	}

	return r.MatchString
}

// link the path tmpl with macros, at .Boot time, before Listen.
// make it a as middleware from the beginning and prepend that before the main handler.
func link(path string, mac _macros) iris.HandlerFunc {
	tmpl, err := ParsePath(path)
	if err != nil {
		panic(err)
	}
	// link the path,  based on its template with a macro
	// and return a new compiled macro or a list of iris handlers
	// in order to be prepended on the original route or make a different function for that?
	// we'll see.

	var h iris.HandlerFunc // we could add an empty handler but we wouldn't know where to ctx.Next if this path doesn't uses macros.

	createH := func(paramName string, validator func(string) bool, failStatus int, prevH iris.HandlerFunc) iris.HandlerFunc {
		return func(ctx *iris.Context) {
			if prevH != nil {
				prevH(ctx)
			}

			paramValue := ctx.Param(paramName)
			if paramValue != "" {
				valid := validator(paramValue)
				if !valid {
					ctx.EmitError(failStatus)
					return
				}
			}
			// println(ctx.Pos)
			// remember: router already matches the path, so here if a path param is missing then it was allowed by the router.
			ctx.Next()
		}
	}

	for i := range tmpl.Params {
		p := tmpl.Params[i]
		if m := mac[p.Param.Macro.Name]; m != nil {
			prevH := h
			h = createH(p.Param.Name, m, p.Param.FailStatusCode, prevH)
		}
	}

	if h == nil {
		// println("h is nil")
		return func(ctx *iris.Context) {
			ctx.Next() // is ok, the route doesn't contains any valid macros
		}
	}

	return h
}

// eval runs while serving paths
// instead of path it can receive the iris.Context and work as middleware
// if the macro passed completely then do ctx.Next() to continue to the main handler and the following,
// otherwise ctx.EmitError(pathTmpl.FailStatusCode) , which defaults to 404 for normal behavior on not found a route,
// but the developer can change that too,
// for example in order to fire the 402 if the compiled macro(I should think the name later) failed to be evaluted
// then the user should add !+statuscode, i.e "{id:int !402}".
// func eval(path string, tmpl *PathTmpl) bool {
// 	return false
// }
// <--- fun(c)k it, we will do it directly to be iris' middleware or create a new type which will save a macro and tries to eval it with a path
// only for test-cases? and after on iris we can make a middleware from this, I should think it more when I stop the drinking.

func testMacros(source string) error {
	return nil
}

func TestMacros(t *testing.T) {
	var m = _macros{
		"int": fromRegexp("[1-9]+$"),
	}
	path := "/api/users/{id:int}/posts"
	app := iris.New()
	app.Adapt(httprouter.New())

	hv := link(path, m)

	app.Get("/api/users/:id/posts", hv, func(ctx *iris.Context) {
		ctx.ResponseWriter.WriteString(ctx.Path())
	})

	e := httptest.New(app, t)

	e.GET("/api/users/42/posts").Expect().Status(iris.StatusOK).Body().Equal("/api/users/42/posts")
	e.GET("/api/users/0/posts").Expect().Status(iris.StatusNotFound)
	e.GET("/api/users/_/posts").Expect().Status(iris.StatusNotFound)
	e.GET("/api/users/s/posts").Expect().Status(iris.StatusNotFound)
	e.GET("/api/users/posts").Expect().Status(iris.StatusNotFound)
}
