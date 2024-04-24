package main

import "github.com/kataras/iris/v12"

func main() {
	app := newApp()
	app.Logger().SetLevel("debug")
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()
	// HERE and you are ready to GO:
	app.Use(iris.Compression)

	app.Get("/", send)
	app.Post("/", receive)

	return app
}

type payload struct {
	Username string `json:"username"`
}

func send(ctx iris.Context) {
	ctx.JSON(payload{
		Username: "Makis",
	})
}

func receive(ctx iris.Context) {
	var p payload
	if err := ctx.ReadJSON(&p); err != nil {
		ctx.Application().Logger().Debugf("ReadJSON: %v", err)
	}

	ctx.WriteString(p.Username)
}

/* Manually:
func enableCompression(ctx iris.Context) {
	// Enable writing using compression (deflate, gzip, brotli, snappy, s2):
	err := ctx.CompressWriter(true)
	if err != nil {
		ctx.Application().Logger().Debugf("writer: %v", err)
		// if you REQUIRE server to SEND compressed data then `return` here.
		// return
	}

	// Enable reading and binding request's compressed data:
	err = ctx.CompressReader(true)
	if err != nil &&
		// on GET we don't expect writing with gzip from client
		ctx.Method() != iris.MethodGet  {
		ctx.Application().Logger().Debugf("reader: %v", err)
		// if you REQUIRE server to RECEIVE only
		// compressed data then `return` here.
		// return
	}

	ctx.Next()
}
*/
