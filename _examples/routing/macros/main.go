package main

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	// "github.com/kataras/iris/core/memstore"
	"github.com/kataras/iris/hero"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// Let's see how we can register a custom macro such as ":uint32"  or ":small" for its alias (optionally) for Uint32 types.
	app.Macros().Register("uint32", "small", false, false, func(paramValue string) bool {
		_, err := strconv.ParseUint(paramValue, 10, 32)
		return err == nil
	}).
		RegisterFunc("min", func(min uint32) func(string) bool {
			return func(paramValue string) bool {
				n, err := strconv.ParseUint(paramValue, 10, 32)
				if err != nil {
					return false
				}

				return uint32(n) >= min
			}
		})

		/* TODO:
		   somehow define one-time how the parameter should be parsed to a particular type (go std or custom)
		   tip: we can change the original value from string to X using the entry's.ValueRaw
		*/

	context.ParamResolvers[reflect.Uint32] = func(paramIndex int) interface{} {
		// return func(store memstore.Store) uint32 {
		// 	param, _ := store.GetEntryAt(paramIndex)
		// 	paramValueAsUint32, _ := strconv.ParseUint(param.String(), 10, 32)
		// 	return uint32(paramValueAsUint32)
		// }
		return func(ctx context.Context) uint32 {
			param := ctx.Params().GetEntryAt(paramIndex)
			paramValueAsUint32, _ := strconv.ParseUint(param.String(), 10, 32)
			return uint32(paramValueAsUint32)
		}
	}
	//

	app.Get("/test_uint32/{myparam:uint32 min(10)}", hero.Handler(func(paramValue uint32) string {
		return fmt.Sprintf("Value of the parameter is: %d\n", paramValue)
	}))

	app.Get("test_uint64/{myparam:uint64}", handler)

	app.Run(iris.Addr(":8080"))
}

func handler(ctx context.Context) {
	ctx.Writef("Value of the parameter is: %s\n", ctx.Params().Get("myparam"))
}
