package routes

import (
	"fmt"

	"github.com/kataras/iris/v12"
)

// GetIndexHandler handles the GET: /
func GetIndexHandler(ctx iris.Context) {
	ctx.ViewData("Title", "Index Page")
	if err := ctx.View("index.html"); err != nil {
		ctx.HTML(fmt.Sprintf("<h3>%s</h3>", err.Error()))
		return
	}
}
