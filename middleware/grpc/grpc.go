package grpc

import (
	"net/http"
	"strings"

	"github.com/kataras/iris/v12/core/router"
)

// New returns a new gRPC Iris router wrapper for a gRPC server.
// useful when you want to share one port (such as 443 for https) between gRPC and Iris.
//
// The Iris server SHOULD run under HTTP/2 and clients too.
//
// Usage:
//  import grpcWrapper "github.com/kataras/iris/v12/middleware/grpc"
//  [...]
//  app := iris.New()
//  grpcServer := grpc.NewServer()
//  app.WrapRouter(grpcWrapper.New(grpcServer))
func New(grpcServer http.Handler) router.WrapperFunc {
	return func(w http.ResponseWriter, r *http.Request, mux http.HandlerFunc) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
			return
		}

		mux.ServeHTTP(w, r)
	}
}
