package api

import (
	"github.com/username/project/api/users"
	"github.com/username/project/user"

	"github.com/kataras/iris/v12"
)

// buildRouter is the most important part of your server.
// All root endpoints are registered here.
func (srv *Server) buildRouter() {
	// Add a simple health route.
	srv.Any("/health", func(ctx iris.Context) {
		ctx.Writef("%s\n\nOK", srv.String())
	})

	api := srv.Party("/api")
	api.RegisterDependency(user.NewRepository)

	api.PartyConfigure("/user", new(users.API))
}
