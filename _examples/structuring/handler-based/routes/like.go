package routes

import (
	"github.com/kataras/iris"
)

// GetLikeHandler handles the GET: /like/{id}
func GetLikeHandler(ctx iris.Context) {
	id, _ := ctx.Params().GetInt64("id")
	ctx.Writef("from "+ctx.GetCurrentRoute().Path()+" with ID: %d", id)
}
