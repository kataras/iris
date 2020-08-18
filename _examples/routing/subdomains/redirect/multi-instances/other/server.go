package other

import (
	"time"

	"github.com/kataras/iris/v12"
)

func init() {
	app := iris.New()
	app.SetName("other app")

	app.OnAnyErrorCode(handleErrors)
	app.Get("/", index)
}

func index(ctx iris.Context) {
	ctx.HTML("Other Index (App Name: <b>%s</b> | Host: <b>%s</b>)",
		ctx.Application().String(), ctx.Host())
}

func handleErrors(ctx iris.Context) {
	errCode := ctx.GetStatusCode()
	ctx.JSON(iris.Map{
		"Server":    ctx.Application().String(),
		"Code":      errCode,
		"Message":   iris.StatusText(errCode),
		"Timestamp": time.Now().Unix(),
	})
}
