package main // See https://github.com/kataras/iris/issues/1601

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

func main() {
	pathToAccessLog := "./access_log.%Y%m%d%H%M"
	w, err := rotatelogs.New(
		pathToAccessLog,
		rotatelogs.WithMaxAge(24*time.Hour),
		rotatelogs.WithRotationTime(time.Hour))
	if err != nil {
		panic(err)
	}
	ac := accesslog.New()
	ac.SetOutput(w)
	/*
		Use a file directly:
		ac := accesslog.File("./access.log")

		Log after the response was sent:
		ac.Async = true

		Custom Time Format:
		ac.TimeFormat = ""

		Add second output:
		ac.AddOutput(os.Stdout)

		Change format (after output was set):
		ac.SetFormatter(&accesslog.JSON{Indent: "  "})
	*/

	defer ac.Close()
	iris.RegisterOnInterrupt(func() {
		ac.Close()
	})

	app := iris.New()
	// Register the middleware (UseRouter to catch http errors too).
	app.UseRouter(ac.Handler)
	//

	// Register some routes...
	app.HandleDir("/", iris.Dir("./public"))

	app.Get("/user/{username}", userHandler)
	app.Post("/read_body", readBodyHandler)
	app.Get("/html_response", htmlResponse)
	//

	app.Listen(":8080")
}

func readBodyHandler(ctx iris.Context) {
	var request interface{}
	if err := ctx.ReadBody(&request); err != nil {
		ctx.StopWithPlainError(iris.StatusBadRequest, err)
		return
	}

	ctx.JSON(iris.Map{"message": "OK", "data": request})
}

func userHandler(ctx iris.Context) {
	ctx.Writef("Hello, %s!", ctx.Params().Get("username"))
}

func htmlResponse(ctx iris.Context) {
	ctx.HTML("<h1>HTML Response</h1>")
}
