// Package main shows how you can register a custom parameter type and macro functions that belongs to it.
package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/hero"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	app.Macros().Register("slice", "", false, true, func(paramValue string) (interface{}, bool) {
		return strings.Split(paramValue, "/"), true
	}).RegisterFunc("contains", func(expectedItems []string) func(paramValue []string) bool {
		sort.Strings(expectedItems)
		return func(paramValue []string) bool {
			if len(paramValue) != len(expectedItems) {
				return false
			}

			sort.Strings(paramValue)
			for i := 0; i < len(paramValue); i++ {
				if paramValue[i] != expectedItems[i] {
					return false
				}
			}

			return true
		}
	})

	// In order to use your new param type inside MVC controller's function input argument or a hero function input argument
	// you have to tell the Iris what type it is, the `ValueRaw` of the parameter is the same type
	// as you defined it above with the func(paramValue string) (interface{}, bool).
	// The new value and its type(from string to your new custom type) it is stored only once now,
	// you don't have to do any conversions for simple cases like this.
	context.ParamResolvers[reflect.TypeOf([]string{})] = func(paramIndex int) interface{} {
		return func(ctx context.Context) []string {
			// When you want to retrieve a parameter with a value type that it is not supported by-default, such as ctx.Params().GetInt
			// then you can use the `GetEntry` or `GetEntryAt` and cast its underline `ValueRaw` to the desired type.
			// The type should be the same as the macro's evaluator function (last argument on the Macros#Register) return value.
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.([]string)
		}
	}

	/*
		http://localhost:8080/test_slice_hero/myvaluei1/myavlue2 ->
		myparam's value (a trailing path parameter type) is: []string{"myvalue1", "myavlue2"}
	*/
	app.Get("/test_slice_hero/{myparam:slice}", hero.Handler(func(myparam []string) string {
		return fmt.Sprintf("myparam's value (a trailing path parameter type) is: %#v\n", myparam)
	}))

	/*
		http://localhost:8080/test_slice_contains/notcontains1/value2 ->
		(404) Not Found

		http://localhost:8080/test_slice_contains/value1/value2 ->
		myparam's value (a trailing path parameter type) is: []string{"value1", "value2"}
	*/
	app.Get("/test_slice_contains/{myparam:slice contains([value1,value2])}", func(ctx context.Context) {
		// When it is not a builtin function available to retrieve your value with the type you want, such as ctx.Params().GetInt
		// then you can use the `GetEntry.ValueRaw` to get the real value, which is set-ed by your macro above.
		myparam := ctx.Params().GetEntry("myparam").ValueRaw.([]string)
		ctx.Writef("myparam's value (a trailing path parameter type) is: %#v\n", myparam)
	})

	app.Run(iris.Addr(":8080"))
}
