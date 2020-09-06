package main // See https://github.com/kataras/iris/issues/1601

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
)

func main() {
	app := iris.New()
	ac := accesslog.File("./access.log")
	defer ac.Close()
	iris.RegisterOnInterrupt(func() {
		ac.Close()
	})

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
