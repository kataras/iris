package main

import (
	"context"

	pb "github.com/kataras/iris/v12/_examples/mvc/grpc-compatible/helloworld"

	"github.com/kataras/iris/v12"
	grpcWrapper "github.com/kataras/iris/v12/middleware/grpc"
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

	// POST: https://localhost/hello
	// with request data: {"name": "John"}
	// and expected output: {"message": "Hello John"}
	app.Run(iris.TLS(":443", "server.crt", "server.key"))
}

func newApp() *iris.Application {
	app := iris.New()

	ctrl := &myController{}
	// Register gRPC server.
	grpcServer := grpc.NewServer()
	pb.RegisterGreeterServer(grpcServer, ctrl)

	// Register MVC application controller.
	mvc.New(app).Handle(ctrl)

	// Serve the gRPC server under the Iris HTTP webserver one,
	// the Iris server should ran under TLS (it's a gRPC requirement).
	app.WrapRouter(grpcWrapper.New(grpcServer))
	return app
}

type myController struct{}

// PostHello implements helloworld.GreeterServer
func (c *myController) PostHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}
