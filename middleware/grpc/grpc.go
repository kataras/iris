// Package grpc serves a gRPC (or gRPC-Web) server through an Iris route.
//
// Its entry point is New(grpcServer http.Handler), which returns a Handler. The
// single input is the server itself, so it is a plain argument and there is no
// Options struct.
package grpc

import (
	"net/http"

	"github.com/kataras/iris/v14/context"
)

func init() {
	context.SetHandlerName("iris/middleware/grpc.*", "iris.gRPC")
}

// New returns a new gRPC Iris router wrapper for a gRPC server.
// useful when you want to share one port (such as 443 for https) between gRPC and Iris.
//
// The Iris server SHOULD run under HTTP/2 and clients too.
//
// Usage:
//
//	import grpcWrapper "github.com/kataras/iris/v14/middleware/grpc"
//	[...]
//	app := iris.New()
//	grpcServer := grpc.NewServer()
//	app.UseRouter(grpcWrapper.New(grpcServer))
func New(grpcServer http.Handler) context.Handler {
	return func(ctx *context.Context) {
		if ctx.IsGRPC() {
			grpcServer.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
			return
		}

		ctx.Next() // continue to mux.
	}
}
