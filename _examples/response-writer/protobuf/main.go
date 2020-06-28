package main

import (
	"app/protos"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	app.Get("/", send)
	app.Get("/json", sendAsJSON)
	app.Post("/read", read)
	app.Post("/read_json", readFromJSON)

	app.Listen(":8080")
}

func send(ctx iris.Context) {
	response := &protos.HelloReply{Message: "Hello, World!"}
	ctx.Protobuf(response)
}

func sendAsJSON(ctx iris.Context) {
	response := &protos.HelloReply{Message: "Hello, World!"}
	options := iris.JSON{
		Proto: iris.ProtoMarshalOptions{
			AllowPartial: true,
			Multiline:    true,
			Indent:       "    ",
		},
	}

	ctx.JSON(response, options)
}

func read(ctx iris.Context) {
	var request protos.HelloRequest

	err := ctx.ReadProtobuf(&request)
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	ctx.Writef("HelloRequest.Name = %s", request.Name)
}

func readFromJSON(ctx iris.Context) {
	var request protos.HelloRequest

	err := ctx.ReadJSONProtobuf(&request)
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	ctx.Writef("HelloRequest.Name = %s", request.Name)
}
