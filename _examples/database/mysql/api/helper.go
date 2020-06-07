package api

import (
	"log"

	"github.com/kataras/iris/v12"
)

const debug = true

func debugf(format string, args ...interface{}) {
	if !debug {
		return
	}

	log.Printf(format, args...)
}

func writeInternalServerError(ctx iris.Context) {
	ctx.StopWithJSON(iris.StatusInternalServerError, newError(iris.StatusInternalServerError, ctx.Request().Method, ctx.Path(), ""))
}

func writeEntityNotFound(ctx iris.Context) {
	ctx.StopWithJSON(iris.StatusNotFound, newError(iris.StatusNotFound, ctx.Request().Method, ctx.Path(), "entity does not exist"))
}
