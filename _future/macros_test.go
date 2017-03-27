// I would be grateful if I had the chance to see the whole work-in-progress in a codebase when I started.
// You have the chance to learn faster nowdays, don't underestimate that, that's the only reason that this "_future" folder exists now.
//
//
// The whole "router" package is a temp place to test my ideas and implementations for future iris' features.
// Young developers can understand and see how ideas can be transform to real implementations on a software like Iris,
// watching the history of a "dirty" code can be useful for some of you.
//
package router

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/httptest"
)

// No, better to have simple functions, it will be easier for users to understand
// type ParamEvaluator interface {
// 	Eval() func(string) bool
// 	Literal() string
// }

// type IntParam struct {
// }

// func (i IntParam) Literal() string {
// 	return "int"
// }

// func (i IntParam) Eval() func(string) bool {
// 	r, err := regexp.Compile("[1-9]+$")
// 	if err != nil {
// 		panic(err)
// 	}
// 	return r.MatchString
// }

// func (i IntParam) Eq(eqToNumber int) func(int) bool {
// 	return func(param int) bool {
// 		return eqToNumber == param
// 	}
// }

// a helper to return a macro from a simple regexp
// it compiles the regexp  and after returns the macro, for obviously performance reasons.
func fromRegexp(expr string) _macrofn {
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

	// println(tmpl.Params[0].Param.FailStatusCode)
	// link the path,  based on its template with a macro
	// and return a new compiled macro or a list of iris handlers
	// in order to be prepended on the original route or make a different function for that?
	// we'll see.

	var h iris.HandlerFunc // we could add an empty handler but we wouldn't know where to ctx.Next if this path doesn't uses macros.

	createH := func(paramName string, validator _macrofn, failStatus int, prevH iris.HandlerFunc) iris.HandlerFunc {
		return func(ctx *iris.Context) {
			if prevH != nil {
				prevH(ctx)
			}
			paramValue := ctx.Param(paramName)
			if paramValue != "" {
				valid := validator(paramValue)
				if !valid {
					// print("not valid for validator on paramValue= '" + paramValue + "' ctx.Pos = ")
					// println(ctx.Pos) // it should be always 0.
					ctx.EmitError(failStatus)
					return
				}
			}
			// remember: router already matches the path, so here if a path param is missing then it was allowed by the router.
			ctx.Next()
		}
	}

	for i := range tmpl.Params {
		p := tmpl.Params[i]
		if m, found := mac[p.Param.Macro.Name]; found && m.eval != nil {
			prevH := h
			eval := m.eval
			for _, fi := range m.funcs {
				for _, mi := range p.Param.Macro.Funcs {
					hasFunc := fi.name == mi.Name
					if !hasFunc {
						for _, gb := range mac[global_macro].funcs {
							if gb.name == fi.name {
								hasFunc = true
								break
							}
						}
					}

					if hasFunc {
						prevEval := eval
						macroFuncEval := fi.eval(mi.Params)
						eval = func(pvalue string) bool {
							if prevEval(pvalue) {
								return macroFuncEval(pvalue)
							}
							return false
						}
						continue
					}
				}
			}

			h = createH(p.Param.Name, eval, p.Param.FailStatusCode, prevH)
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

// let's give the macro's funcs access to context, it will be great experimental to serve templates just with a path signature
type _macrofn func(pathParamValue string) bool

type _macrofunc struct {
	name string
	eval func([]string) _macrofn
}

type _macro struct {
	funcs []_macrofunc
	eval  _macrofn
}

// macros should be registered before .Listen
type _macros map[string]*_macro

var all_macros = _macros{}

func addMacro(name string, v _macrofn) {
	all_macros[name] = &_macro{eval: v}
}

func addMacroFunc(macroName string, funcName string, v func([]string) _macrofn) {
	if m, found := all_macros[macroName]; found {
		m.funcs = append(m.funcs, _macrofunc{name: funcName, eval: v})
	}
}

const global_macro = "any"

// func(min int, max int) func(paramValue string)bool
func macroFuncFrom(v interface{}) func(params []string) _macrofn {
	// this is executed once on boot time, not at serve time:
	vot := reflect.TypeOf(v)
	numFields := vot.NumIn()

	return func(params []string) _macrofn {
		if len(params) != numFields {
			panic("should accepts _numFields_ args")
		}
		var args []reflect.Value

		// check for accepting arguments
		for i := 0; i < numFields; i++ {
			field := vot.In(i)
			param := params[i]
			// if field.IsVariadic() {
			// 	panic("variadic arguments are not supported") // or they will do ?
			// }
			var val interface{}
			var err error
			switch field.Kind() {
			// these can be transfered to another function with supported type conversions
			// the dev can also be able to modify how a string converted to x kind of type,
			// even custom type, i.e User{}, (I have to give an easy way to do hard things
			//                               but also extensibility for devs that are experienced,
			//                               like I did with the rest of the features).
			case reflect.String:
				val = param
			case reflect.Int:
				val, err = strconv.Atoi(param)
			case reflect.Bool:
				val, err = strconv.ParseBool(param)

			default:
				panic("unsported type!")
			}
			if err != nil {
				panic(err)
			}
			args = append(args, reflect.ValueOf(val))
		}

		// check for the return type (only one ofc, which again is a function but it returns a boolean)
		// which accepts one argument which is the parameter value.
		if vot.NumOut() != 1 {
			panic("expecting to return only one (func)")
		}
		rof := vot.Out(0)
		if rof.Kind() != reflect.Func {
			panic("expecting to return a function!")
		}

		returnRof := rof.Out(0)
		if rof.NumOut() != 1 {
			panic("expecting to return only one (bool)")
		}
		if returnRof.Kind() != reflect.Bool {
			panic("expecting this func to return a boolean")
		}

		if rof.NumIn() != 1 {
			panic("expecting this func to receive one arg")
		}

		vofi := reflect.ValueOf(v).Call(args)[0].Interface()
		var validator _macrofn
		// check for typed and not typed
		if _v, ok := vofi.(_macrofn); ok {
			validator = _v
		} else if _v, ok = vofi.(func(string) bool); ok {
			validator = _v
		}
		//

		// this is executed when a route requested:
		return func(paramValue string) bool {
			return validator(paramValue)
		}
		//
	}
}

func TestMacros(t *testing.T) {
	addMacro("int", fromRegexp("[1-9]+$"))

	// // {id:int range(42,49)}
	// // "hard" manually way(it will not be included on the final feature(;)):
	// addMacroFunc("int", "range", func(params []string) _macrofn {
	// 	// start: .Boot time, before .Listen
	// 	allowedParamsLen := 2
	// 	// params:  42,49 (including first and second)
	// 	if len(params) != allowedParamsLen {
	// 		panic("range accepts two parameters")
	// 	}

	// 	min, err := strconv.Atoi(params[0])
	// 	if err != nil {
	// 		panic("invalid first parameter: " + err.Error())
	// 	}
	// 	max, err := strconv.Atoi(params[1])
	// 	if err != nil {
	// 		panic("invalid second parameter: " + err.Error())
	// 	}
	// 	// end

	// 	return func(paramValue string) bool {
	// 		paramValueInt, err := strconv.Atoi(paramValue)
	// 		if err != nil {
	// 			return false
	// 		}
	// 		if paramValueInt >= min && paramValueInt <= max {
	// 			return true
	// 		}
	// 		return false
	// 	}
	// })
	//
	// {id:int range(42,49)}
	// easy way, same performance as the hard way, no cost while serving requests.
	// ::
	// result should be like that in the final feature implementation, using reflection BEFORE .Listen on .Boot time,
	// so no performance cost(done) =>
	addMacroFunc("int", "range", macroFuncFrom(func(min int, max int) func(string) bool {
		return func(paramValue string) bool {
			paramValueInt, err := strconv.Atoi(paramValue)
			if err != nil {
				return false
			}
			if paramValueInt >= min && paramValueInt <= max {
				return true
			}
			return false
		}
	}))

	addMacroFunc("int", "even", func(params []string) _macrofn {
		return func(paramValue string) bool {
			paramValueInt, err := strconv.Atoi(paramValue)
			if err != nil {
				return false
			}
			if paramValueInt%2 == 0 {
				return true
			}
			return false
		}
	})

	// "any" will contain macros functions
	// which are available to all other, we will need some functions to be 'globally' registered when don't care about.
	addMacro("any", fromRegexp(".*"))
	addMacroFunc("any", "contains", macroFuncFrom(func(text string) _macrofn {
		return func(paramValue string) bool {
			return strings.Contains(paramValue, text)
		}
	}))
	addMacroFunc("any", "suffix", macroFuncFrom(func(text string) _macrofn {
		return func(paramValue string) bool {
			return strings.HasSuffix(paramValue, text)
		}
	}))

	addMacro("string", fromRegexp("[a-zA-Z]+$"))
	// this will 'override' the "any contains"
	// when string macro is used:
	addMacroFunc("string", "contains", macroFuncFrom(func(text string) _macrofn {
		return func(paramValue string) bool {
			// println("from string:contains instead of any:string")
			// println("'" + text + "' vs '" + paramValue + "'")

			return strings.Contains(paramValue, text)
		}
	}))

	path := "/api/users/{id:int range(42,49) even() !600}/posts"
	app := iris.New()
	app.Adapt(httprouter.New())

	hv := link(path, all_macros)
	// 600 is a custom virtual error code to handle "int" param invalids
	// it sends a custom error message with a 404 (not found) http status code.
	app.OnError(600, func(ctx *iris.Context) {
		ctx.SetStatusCode(404) // throw a raw 404 not found
		ctx.Writef("Expecting an integer in range between and 42-49, should be even number too")
		// println("600 -> 404 from " + ctx.Path())
	})
	app.Get("/api/users/:id/posts", hv, func(ctx *iris.Context) {
		ctx.ResponseWriter.WriteString(ctx.Path())
	})

	path2 := "/markdown/{file:any suffix(.md)}"
	hv2 := link(path2, all_macros)
	app.Get("/markdown/*file", hv2, func(ctx *iris.Context) {
		ctx.Markdown(iris.StatusOK, "**hello**")
	})

	// contains a space(on tests)
	path3 := "/hello/{fullname:string contains( )}"
	hv3 := link(path3, all_macros)
	app.Get("/hello/:fullname", hv3, func(ctx *iris.Context) {
		ctx.Writef("hello %s", ctx.Param("fullname"))
	})

	e := httptest.New(app, t)

	e.GET("/api/users/42/posts").Expect().Status(iris.StatusOK).Body().Equal("/api/users/42/posts")
	e.GET("/api/users/50/posts").Expect().Status(iris.StatusNotFound).Body().Equal("Expecting an integer in range between and 42-49, should be even number too") // remember, it accepts 1-9 not matched if zero.
	e.GET("/api/users/0/posts").Expect().Status(iris.StatusNotFound)
	e.GET("/api/users/_/posts").Expect().Status(iris.StatusNotFound)
	e.GET("/api/users/s/posts").Expect().Status(iris.StatusNotFound)
	e.GET("/api/users/posts").Expect().Status(iris.StatusNotFound)
	// macro func invalidate test with a non-zero value between 1-9 but bigger than the max(49)
	e.GET("/api/users/51/posts").Expect().Status(iris.StatusNotFound)
	// macro func invalidate "even" with a non-zero value but 49 is not an even number
	e.GET("/api/users/49/posts").Expect().Status(iris.StatusNotFound)

	// test any and global
	// response with "path language" only no need of handler too.
	// As it goes I love the idea and users will embrace and built awesome things on top of it.
	// maybe I have to 'rename' the final feature on something like iris expression language and document it as much as I can, people will love that
	e.GET("/markdown/something.md").Expect().Status(iris.StatusOK).ContentType("text/html", "utf-8").Body().Equal("<p><strong>hello</strong></p>\n")
	e.GET("/hello/Makis Maropoulos").Expect().Status(iris.StatusOK).Body().Equal("hello Makis Maropoulos")
	e.GET("/hello/MakisMaropoulos").Expect().Status(iris.StatusNotFound) // no space -> invalidate -> fail status code
}
