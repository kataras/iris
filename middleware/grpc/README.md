# gRPC

This middleware serves gRPC and Iris on one port, typically 443. It routes a request to your gRPC server when the request looks like gRPC, and to the Iris router otherwise. Both the server and its clients must run HTTP/2.

### Installation

To use the gRPC middleware, you need to import it in your project:

```go
import grpcWrapper "github.com/kataras/iris/v14/middleware/grpc"
```

### API

#### `func New(grpcServer http.Handler) iris.Handler`

The `New` function returns a new gRPC Iris router wrapper for a gRPC server. It takes an `http.Handler` as an argument, which should be your gRPC server.

##### Parameters

- `grpcServer` (`http.Handler`): The gRPC server handler.

##### Returns

- `iris.Handler`: The Iris request context handler.

### Example

Here is a complete example demonstrating how to use the gRPC middleware with an Iris application:

```go
package main

import (
	"context"
	"log"
	"net"

	"github.com/kataras/iris/v14"
	grpcWrapper "github.com/kataras/iris/v14/middleware/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Define your gRPC service
type greeterServer struct{}

func (s *greeterServer) SayHello(ctx context.Context, req *HelloRequest) (*HelloReply, error) {
	return &HelloReply{Message: "Hello " + req.Name}, nil
}

func main() {
	app := iris.New()

	// Create a new gRPC server
	grpcServer := grpc.NewServer()

	// Register your gRPC service
	RegisterGreeterServer(grpcServer, &greeterServer{})

	// Use the gRPC middleware
	app.UseRouter(grpcWrapper.New(grpcServer))

	// Serve both gRPC and HTTP on the same port
	app.Run(iris.ServerOptions{Addr: ":443", CertFile: "server.crt", KeyFile: "server.key"})
}
```

### Notes

- The Iris server and its clients must run HTTP/2.
- You can register more than one gRPC service on the same gRPC server.
