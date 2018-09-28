package main

import (
	"fmt"
	"reflect"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	// "github.com/kataras/iris/core/memstore"
	"github.com/kataras/iris/hero"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// Let's see how we can register a custom macro such as ":uint32"  or ":small" for its alias (optionally) for Uint32 types.
	// app.Macros().Register("uint32", "small", false, false, func(paramValue string) bool {
	// 	_, err := strconv.ParseUint(paramValue, 10, 32)
	// 	return err == nil
	// }).
	// 	RegisterFunc("min", func(min uint32) func(string) bool {
	// 		return func(paramValue string) bool {
	// 			n, err := strconv.ParseUint(paramValue, 10, 32)
	// 			if err != nil {
	// 				return false
	// 			}

	// 			return uint32(n) >= min
	// 		}
	// 	})

	/* TODO:
	   somehow define one-time how the parameter should be parsed to a particular type (go std or custom)
	   tip: we can change the original value from string to X using the entry's.ValueRaw
	   ^ Done 27 sep 2018.
	*/

	// app.Macros().Register("uint32", "small", false, false, func(paramValue string) (interface{}, bool) {
	// 	v, err := strconv.ParseUint(paramValue, 10, 32)
	// 	return uint32(v), err == nil
	// }).
	// 	RegisterFunc("min", func(min uint32) func(uint32) bool {
	// 		return func(paramValue uint32) bool {
	// 			return paramValue >= min
	// 		}
	// 	})

	// // optionally, only when mvc or hero features are used for this custom macro/parameter type.
	// context.ParamResolvers[reflect.Uint32] = func(paramIndex int) interface{} {
	// 	/* both works but second is faster, we omit the duplication of the type conversion over and over  as of 27 Sep of 2018 (this patch)*/
	// 	// return func(ctx context.Context) uint32 {
	// 	// 	param := ctx.Params().GetEntryAt(paramIndex)
	// 	// 	paramValueAsUint32, _ := strconv.ParseUint(param.String(), 10, 32)
	// 	// 	return uint32(paramValueAsUint32)
	// 	// }
	// 	return func(ctx context.Context) uint32 {
	// 		return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(uint32)
	// 	} /* TODO: find a way to automative it based on the macro's first return value type, if thats the case then we must not return nil even if not found,
	// 	we must return a value i.e 0 for int for its interface{} */
	// }
	// //

	app.Get("/test_uint32/{myparam1:string}/{myparam2:uint32 min(10)}", hero.Handler(func(myparam1 string, myparam2 uint32) string {
		return fmt.Sprintf("Value of the parameters are: %s:%d\n", myparam1, myparam2)
	}))

	app.Get("/test_string/{myparam1}/{myparam2 prefix(a)}", func(ctx context.Context) {
		var (
			myparam1 = ctx.Params().Get("myparam1")
			myparam2 = ctx.Params().Get("myparam2")
		)

		ctx.Writef("myparam1: %s | myparam2: %s", myparam1, myparam2)
	})

	app.Get("/test_string2/{myparam1}/{myparam2}", func(ctx context.Context) {
		var (
			myparam1 = ctx.Params().Get("myparam1")
			myparam2 = ctx.Params().Get("myparam2")
		)

		ctx.Writef("myparam1: %s | myparam2: %s", myparam1, myparam2)
	})

	app.Get("/test_uint64/{myparam1:string}/{myparam2:uint64}", func(ctx context.Context) {
		// works: ctx.Writef("Value of the parameter is: %s\n", ctx.Params().Get("myparam"))
		// but better and faster because the macro converts the string to uint64 automatically:
		println("type of myparam2 (should be uint64) is: " + reflect.ValueOf(ctx.Params().GetEntry("myparam2").ValueRaw).Kind().String())
		ctx.Writef("Value of the parameters are: %s:%d\n", ctx.Params().Get("myparam1"), ctx.Params().GetUint64Default("myparam2", 0))
	})

	app.Run(iris.Addr(":8080"))
}
