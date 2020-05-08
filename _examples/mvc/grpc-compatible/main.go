package main

import (
	"context"

	pb "github.com/kataras/iris/v12/_examples/mvc/grpc-compatible/helloworld"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"

	"google.golang.org/grpc"
)

// See https://github.com/kataras/iris/issues/1449
// Iris automatically binds the standard "context" context.Context to `iris.Context.Request().Context()`
// and any other structure that is not mapping to a registered dependency
// as a payload depends on the request, e.g XML, YAML, Query, Form, JSON.
//
// Useful to use gRPC services as Iris controllers fast and without wrappers.

func main() {
	app := newApp()
	app.Logger().SetLevel("debug")

	// The Iris server should ran under TLS (it's a gRPC requirement).
	// POST: https://localhost:443/helloworld.Greeter/SayHello
	// with request data: {"name": "John"}
	// and expected output: {"message": "Hello John"}
	app.Run(iris.TLS(":443", "server.crt", "server.key"))
}

func newApp() *iris.Application {
	app := iris.New()
	// app.Configure(iris.WithLowercaseRouting) // OPTIONAL.

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<h1>Index Page</h1>")
	})

	ctrl := &myController{}
	// Register gRPC server.
	grpcServer := grpc.NewServer()
	pb.RegisterGreeterServer(grpcServer, ctrl)

	// serviceName := pb.File_helloworld_proto.Services().Get(0).FullName()

	// Register MVC application controller for gRPC services.
	// You can bind as many mvc gRpc services in the same Party or app,
	// as the ServiceName differs.
	mvc.New(app).Handle(ctrl, mvc.GRPC{
		Server:      grpcServer,           // Required.
		ServiceName: "helloworld.Greeter", // Required.
		Strict:      false,
	})

	return app
}

type myController struct {
	// Ctx iris.Context
}

// SayHello implements helloworld.GreeterServer.
// See https://github.com/kataras/iris/issues/1449#issuecomment-625570442
// for the comments below (https://github.com/iris-contrib/swagger).
//
// @Description greet service
// @Accept  json
// @Produce  json
// @Success 200 {string} string	"Hello {name}"
// @Router /helloworld.Greeter/SayHello [post]
func (c *myController) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}
