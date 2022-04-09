package api

import (
	"time"

	"github.com/username/project/api/users"
	"github.com/username/project/pkg/database"
	"github.com/username/project/user"

	"github.com/kataras/iris/v12/middleware/modrevision"
)

// buildRouter is the most important part of your server.
// All root endpoints are registered here.
func (srv *Server) buildRouter() {
	// Add a simple health route.
	srv.Any("/health", modrevision.New(modrevision.Options{
		ServerName:   srv.config.ServerName,
		Env:          srv.config.Env,
		Developer:    "kataras",
		TimeLocation: time.FixedZone("Greece/Athens", 10800),
	}))

	api := srv.Party("/api")
	api.RegisterDependency(
		database.Open(srv.config.ConnString),
		user.NewRepository,
	)

	api.PartyConfigure("/user", new(users.API))
}
