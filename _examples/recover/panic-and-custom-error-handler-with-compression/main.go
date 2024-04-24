package main

import (
	"errors"
	"fmt"

	"github.com/kataras/iris/v12"
	//	"github.com/kataras/iris/v12/context"
)

func main() {
	app := iris.New()
	app.UseRouter(iris.Compression)
	app.UseRouter(myErrorHandler)

	app.Get("/", handler)

	app.Listen(":8080")
}

func myErrorHandler(ctx iris.Context) {
	recorder := ctx.Recorder()

	defer func() {
		var err error

		if v := recover(); v != nil { // panic
			if panicErr, ok := v.(error); ok {
				err = panicErr
			} else {
				err = errors.New(fmt.Sprint(v))
			}
		} else { // custom error.
			err = ctx.GetErr()
		}

		if err != nil {
			// To keep compression after reset:
			// clear body and any headers created between recorder and handler.
			recorder.ResetBody()
			recorder.ResetHeaders()
			//

			// To disable compression after reset:
			// recorder.Reset()
			// recorder.ResponseWriter.(*context.CompressResponseWriter).Disabled = true
			//

			ctx.StopWithJSON(iris.StatusInternalServerError, iris.Map{
				"message": err.Error(),
			})
		}
	}()

	ctx.Next()
}

func handler(ctx iris.Context) {
	ctx.WriteString("Content may fall")
	ctx.Header("X-Test", "value")

	// ctx.SetErr(fmt.Errorf("custom error message"))
	panic("errr!")
}
