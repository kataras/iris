package main

import (
	"io"

	pb "grpcexample/helloworld"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"

	"google.golang.org/grpc"
)

type Greeter struct {
	pb.UnimplementedGreeterBidirectionalStreamServer
}

// SayHello implements the proto Bidirectional Stream Greeter service.
func (g *Greeter) SayHello(stream pb.GreeterBidirectionalStream_SayHelloServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		println("Received input: " + in.Name)
		// On client side you can implement the 'read' operation too.
		stream.Send(&pb.HelloReply{Message: "Hello " + in.Name})
	}
}

func main() {
	app := iris.New()

	grpcServer := grpc.NewServer()

	myService := &Greeter{}
	pb.RegisterGreeterBidirectionalStreamServer(grpcServer, myService)

	rootApp := mvc.New(app)
	rootApp.Handle(myService, mvc.GRPC{
		Server:      grpcServer,                              // Required.
		ServiceName: "helloworld.GreeterBidirectionalStream", // Required.
		Strict:      true,                                    // Set it to true on gRPC streaming.
	})

	app.Run(iris.TLS(":443", "../server.crt", "../server.key"))
}
