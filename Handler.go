package gapi

import (

)
/*Example

import (
	"github.com/kataras/gapi"
)

type UserHandler struct {
	gapi.Handler `GET:"/api/users/{userId(int)}"`
}

func (u *UserHandler) Handle(ctx *gapi.Context) {
	defer ctx.Close()
	var userId, _ = ctx.ParamInt("userId")

	println(userId)

}

...

api:= gapi.New()
registerError := api.RegisterHandler(new(UserHandler))

*/

type Handler interface {
	Handle(ctx * Context)
}