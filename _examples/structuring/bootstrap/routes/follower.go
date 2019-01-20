package routes

import (
	"github.com/kataras/iris"
)

// GetFollowerHandler handles the GET: /follower/{id}
func GetFollowerHandler(ctx iris.Context) {
	id, _ := ctx.Params().GetInt64("id")
	ctx.Writef("from "+ctx.GetCurrentRoute().Path()+" with ID: %d", id)
}
