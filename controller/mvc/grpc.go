package mvc

import (
	"fmt"
	"net/http"
	"path"
	"reflect"
	"strings"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
)

// GrpcController holds the controller functionality which serves (g)RPC clients over https subprotocol.
// It accepts the controller ptr to a struct value,
// the gRPCServer itself, and a strict option which is explained below.
//
// The differences between an GRPC-based controller and a common one are:
// HTTP verb: only POST (Router.AllowMethods can be used for more),
// method parsing is disabled: path is the function name as it is,
// if 'strictMode' option is true then this controller will only serve gRPC-based clients
// and fires 404 on common HTTP clients,
// otherwise HTTP clients can send and receive JSON (protos contain json struct fields by-default).
//
// It implements the `Controller` interface.
type GrpcController struct {
	// Server is required and should be gRPC Server derives from google's grpc package.
	server http.Handler
	// ServiceName is required and should be the name of the service (used to build the gRPC route path),
	// e.g. "helloworld.Greeter".
	// For a controller's method of "SayHello" and ServiceName "helloworld.Greeter",
	// both gRPC and common HTTP request path is: "/helloworld.Greeter/SayHello".
	//
	// Tip: the ServiceName can be fetched through proto's file descriptor, e.g.
	// serviceName := pb.File_helloworld_proto.Services().Get(0).FullName().
	serviceName string

	// When Strict option is true then this controller will only serve gRPC-based clients
	// and fires 404 on common HTTP clients. Definitely set this to true for streaming.
	strict bool

	// Holds the (g)rpc proto server struct value.
	// It is used to bind the controller's methods to the gRPC server for even more extensibility when client doesn't send the "application/grpc" header.
	ctrl any
}

var _ router.Controller = (*GrpcController)(nil)

// GRPC returns a new gRPC controller.
// Example:
//
//	ctrl := &myController{}
//	// New gRPC server to your protos.
//	grpcServer := grpc.NewServer()
//	pb.RegisterGreeterServer(grpcServer, ctrl)
//
//	app.RegisterDependency(new(myService)) // Optional, register any dependencies.
//	app.Router("/", mvc.GRPC(ctrl).Server(grpcServer).ServiceName("helloworld.Greeter").Strict(false))
func GRPC(serverStructPtrValue any) *GrpcController {
	return &GrpcController{ctrl: serverStructPtrValue}
}

// Server sets the gRPC server handler. It should be a gRPC Server derives from google's grpc package.
//
// Required.
func (opts *GrpcController) Server(server http.Handler) *GrpcController {
	opts.server = server
	return opts
}

// ServiceName sets the name of the service (used to build the gRPC route path),
// e.g. "helloworld.Greeter".
//
// Required.
func (opts *GrpcController) ServiceName(serviceName string) *GrpcController {
	opts.serviceName = serviceName
	return opts
}

// Strict sets the strict mode. When Strict option is true then this controller will only serve gRPC-based clients
// and fires 404 on common HTTP clients. Definitely set this to true for streaming.
//
// Defaults to false.
func (opts *GrpcController) Strict(strict bool) *GrpcController {
	opts.strict = strict
	return opts
}

// Routing registers the controller's methods to the application's router. It parses the controller's methods and registers gRPC handlers to the application.
// It completes the `Controller` interface.
func (opts *GrpcController) Routing(r router.Router) {
	if opts.server == nil {
		panic(fmt.Sprintf("iris: mvc: GrpcController: %s: no gRPC server; pass one with mvc.GRPC(server, ...)", r.GetRelPath()))
	}

	if opts.serviceName == "" {
		panic(fmt.Sprintf("iris: mvc: GrpcController: %s: no service name; set it with ServiceName(\"package.Service\")", r.GetRelPath()))
	}

	// The service name becomes a route path segment through path.Join, which
	// normalises "..". A name like "../admin" would therefore escape the mount
	// prefix and register routes somewhere the developer did not intend.
	if strings.Contains(opts.serviceName, "..") || strings.ContainsAny(opts.serviceName, "/\\") {
		panic(fmt.Sprintf("iris: mvc: GrpcController: %s: ServiceName must be a plain service identifier, got %q; drop the path separators and the dot segments", r.GetRelPath(), opts.serviceName))
	}

	// resolve any "ctrl"'s  dependencies or if its a function which returns a controller, resolve and return it.
	ctrl, err := r.DI().InvokeReturningFirst(opts.ctrl)
	if err != nil {
		panic(fmt.Errorf("iris: mvc: GrpcController: %s: %T: register its dependencies before this controller: %w", r.GetRelPath(), opts.ctrl, err))
	}

	grpcHandler := func(ctx *context.Context) {
		if ctx.IsGRPC() { // gRPC, consumes and produces protobuf.
			opts.server.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
			ctx.StopExecution()
			return
		}

		// If strict was true fires 404 on common HTTP clients.
		if opts.strict {
			ctx.NotFound()
			ctx.StopExecution()
			return
		}

		// Allow common HTTP clients, consumes and produces JSON.
		ctx.Next()
	}

	// opts.strict = true // by force, see below.

	typ := reflect.TypeOf(ctrl)

	// Register any other custom methods here.
	//
	// Note(@kataras): this runs BEFORE the service routes below, so an r.Use
	// inside Routing also guards them - a route captures the router's middleware
	// when it is created, not when the app is built.
	if c, ok := ctrl.(router.Controller); ok {
		c.Routing(r)
	}

	// Note(@kataras): register in BOTH modes.
	//
	// The route path was computed and then thrown away whenever strict was false
	// - which is the default - so GRPC(ctrl).Server(s).ServiceName("x.Y") with
	// default options silently exposed no endpoints at all. The grpcHandler
	// itself already distinguishes the two modes: in strict mode it answers 404
	// to non-gRPC clients, otherwise it falls through to the next handler.
	for m := range typ.Methods() {
		routePath := path.Join(opts.serviceName, m.Name)
		r.Post(routePath, grpcHandler)
	}

	/* TODO: re-enable this when new subpackages are ready. For now, set strict to true by force ^.
		// If not strict, allow any struct's method to be reachable by both gRPC and common HTTP clients.
		// Using old-fashion reflection, which I don't recommend at all but we don't have any other option here,
		// most people will use it strictly for gRPC otherwise they should choose the new rest subpackage I designed for months for them.

		fullpath := r.GetRelPath() + path
		paramsCount := macro.CountParams(fullpath, *r.Macros())
		handler := injector.MethodHandler(m.Name, paramsCount)

		route := r.Post(path, grpcHandler, handler)
		if route != nil { // the following is useful for end-dev logging.
			// set new description.
			route.Description = fmt.Sprintf("gRPC %s", route.Description) // e.g. "gRPC controller"

			// set new handler name.
			relName := strings.TrimPrefix(reflex.NameOf(typ), "main.")
			route.MainHandlerName = fmt.Sprintf("%s.%s", relName, m.Name)

			// set new debug information.
			sourceFileName, sourceLineNumber := getSourceFileLine(typ, m)
			route.SourceFileName, route.SourceLineNumber = sourceFileName, sourceLineNumber
		}
	}
	*/
}
