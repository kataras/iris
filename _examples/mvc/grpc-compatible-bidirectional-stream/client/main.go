package main

import (
	"context"
	"time"

	pb "grpcexample/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	// Set up a connection to the server.
	cred, err := credentials.NewClientTLSFromFile("../server.crt", "localhost")
	if err != nil {
		panic(err)
	}

	conn, err := grpc.Dial("localhost:443", grpc.WithTransportCredentials(cred), grpc.WithBlock())
	defer conn.Close()

	client := pb.NewGreeterBidirectionalStreamClient(conn)
	stream, err := client.SayHello(context.Background())
	if err != nil {
		panic(err)
	}

	waitCh := make(chan struct{})

	// Implement the send channel.
	// As an exercise you can implement the read channel one (reading from server, see the server/main.go).
	go func() {
		for {
			println("Sleeping for 2 seconds...")
			time.Sleep(2 * time.Second)
			println("Sending a <test> msg...")
			msg := &pb.HelloRequest{Name: "test"}

			err = stream.Send(msg)
			if err != nil {
				panic("stream.Send: " + err.Error())
			}
		}
	}()

	<-waitCh
	stream.CloseSend()
}
