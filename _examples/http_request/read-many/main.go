package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	app.Post("/", logAllBody, logJSON, logFormValues, func(ctx iris.Context) {
		// body, err := ioutil.ReadAll(ctx.Request().Body) once or
		body, err := ctx.GetBody() // as many times as you need.
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		if len(body) == 0 {
			ctx.WriteString(`The body was empty
or iris.WithoutBodyConsumptionOnUnmarshal option is missing from app.Run.
Check the terminal window for any queries logs.`)
		} else {
			ctx.WriteString("OK body is still:\n")
			ctx.Write(body)
		}
	})

	// With ctx.UnmarshalBody, ctx.ReadJSON, ctx.ReadXML, ctx.ReadForm, ctx.FormValues
	// and ctx.GetBody methods the default golang and net/http behavior
	// is to consume the readen data - they are not available on any next handlers in the chain -
	// to change that behavior just pass the `WithoutBodyConsumptionOnUnmarshal` option.
	app.Listen(":8080", iris.WithoutBodyConsumptionOnUnmarshal)
}

func logAllBody(ctx iris.Context) {
	body, err := ctx.GetBody()
	if err == nil && len(body) > 0 {
		ctx.Application().Logger().Infof("logAllBody: %s", string(body))
	}

	ctx.Next()
}

func logJSON(ctx iris.Context) {
	var p interface{}
	if err := ctx.ReadJSON(&p); err == nil {
		ctx.Application().Logger().Infof("logJSON: %#+v", p)
	}

	ctx.Next()
}

func logFormValues(ctx iris.Context) {
	values := ctx.FormValues()
	if values != nil {
		ctx.Application().Logger().Infof("logFormValues: %v", values)
	}

	ctx.Next()
}
