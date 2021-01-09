package mvc

import (
	"net/http"
	"path"

	"github.com/kataras/iris/v12/context"
)

// GRPC registers a controller which serves gRPC clients.
// It accepts the controller ptr to a struct value,
// the gRPCServer itself, and a strict option which is explained below.
//
// The differences by a common controller are:
// HTTP verb: only POST (Party.AllowMethods can be used for more),
// method parsing is disabled: path is the function name as it is,
// if 'strictMode' option is true then this controller will only serve gRPC-based clients
// and fires 404 on common HTTP clients,
// otherwise HTTP clients can send and receive JSON (protos contain json struct fields by-default).
type GRPC struct {
	// Server is required and should be gRPC Server derives from google's grpc package.
	Server http.Handler
	// ServiceName is required and should be the name of the service (used to build the gRPC route path),
	// e.g. "helloworld.Greeter".
	// For a controller's method of "SayHello" and ServiceName "helloworld.Greeter",
	// both gRPC and common HTTP request path is: "/helloworld.Greeter/SayHello".
	//
	// Tip: the ServiceName can be fetched through proto's file descriptor, e.g.
	// serviceName := pb.File_helloworld_proto.Services().Get(0).FullName().
	ServiceName string

	// When Strict option is true then this controller will only serve gRPC-based clients
	// and fires 404 on common HTTP clients.
	Strict bool
}

var _ Option = GRPC{}

// Apply parses the controller's methods and registers gRPC handlers to the application.
func (g GRPC) Apply(c *ControllerActivator) {
	defer c.Activated()

	pre := func(ctx *context.Context) {
		if ctx.IsGRPC() { // gRPC, consumes and produces protobuf.
			g.Server.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
			ctx.StopExecution()
			return
		}

		// If strict was false, allow common HTTP clients, consumes and produces JSON.
		ctx.Next()
	}

	for i := 0; i < c.Type.NumMethod(); i++ {
		m := c.Type.Method(i)
		path := path.Join(g.ServiceName, m.Name)
		if g.Strict {
			c.app.Router.HandleMany(http.MethodPost, path, pre)
		} else if route := c.Handle(http.MethodPost, path, m.Name, pre); route != nil {
			bckp := route.Description
			route.Description = "gRPC"
			if g.Strict {
				route.Description += "-only"
			}
			route.Description += " " + bckp // e.g. "gRPC controller"
		}

	}
}
