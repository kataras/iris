// Package main implements a client for Greeter service.
package main

import (
	"context"
	"log"
	"os"
	"time"

	pb "github.com/kataras/iris/v12/_examples/mvc/grpc-compatible/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	address     = "localhost:443"
	defaultName = "world"
)

func main() {
	// Set up a connection to the server.
	cred, err := credentials.NewClientTLSFromFile("../server.crt", "localhost")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(cred), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
}
