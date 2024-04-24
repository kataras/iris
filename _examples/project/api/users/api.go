package users

import (
	"github.com/username/project/user"

	"github.com/kataras/iris/v12"
)

type API struct {
	Users user.Repository // exported field so api/router.go#api.RegisterDependency can bind it.
}

func (api *API) Configure(r iris.Party) {
	r.Post("/signup", api.signUp)
	r.Post("/signin", api.signIn)
	// Add middlewares such as user verification by bearer token here.

	// Authenticated routes...
	r.Get("/", api.getInfo)
}

func (api *API) getInfo(ctx iris.Context) {
	ctx.WriteString("...")
}

func (api *API) signUp(ctx iris.Context) {}
func (api *API) signIn(ctx iris.Context) {}
