package router

import (
	"regexp"
	"testing"
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
func link(path string, m _macros) {
	tmpl, err := ParsePath(path)
	if err != nil {
		panic(err)
	}
	// link the path,  based on its template with a macro
	// and return a new compiled macro or a list of iris handlers
	// in order to be prepended on the original route or make a different function for that?
	// we'll see.
	_ = tmpl

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
	var m = _macros{
		"id": fromRegexp("[1-9]+$"),
	}

	link(source, m)

	// eval("/api/users/42", result)

	return nil
}

func TestMacros(t *testing.T) {
	if err := testMacros("/api/users/{id:int}/posts"); err != nil {
		t.Fatal(err)
	}
}
