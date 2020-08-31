package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := newApp()
	// See main_test.go for usage.
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()
	// To automatically decompress using gzip:
	// app.Use(iris.GzipReader)

	app.Use(setAllowedResponses)

	app.Post("/", readBody)

	return app
}

type payload struct {
	Message string `json:"message" xml:"message" msgpack:"message" yaml:"Message" url:"message" form:"message"`
}

func readBody(ctx iris.Context) {
	var p payload

	// Bind request body to "p" depending on the content-type that client sends the data,
	// e.g. JSON, XML, YAML, MessagePack, Protobuf, Form and URL Query.
	err := ctx.ReadBody(&p)
	if err != nil {
		ctx.StopWithProblem(iris.StatusBadRequest,
			iris.NewProblem().Title("Parser issue").Detail(err.Error()))
		return
	}

	// For the sake of the example, log the received payload.
	ctx.Application().Logger().Infof("Received: %#+v", p)

	// Send back the payload depending on the accept content type and accept-encoding of the client,
	// e.g. JSON, XML and so on.
	ctx.Negotiate(p)
}

func setAllowedResponses(ctx iris.Context) {
	// Indicate that the Server can send JSON, XML, YAML and MessagePack for this request.
	ctx.Negotiation().JSON().XML().YAML().MsgPack()
	// Add more, allowed by the server format of responses, mime types here...

	// If client is missing an "Accept: " header then default it to JSON.
	ctx.Negotiation().Accept.JSON()

	ctx.Next()
}
