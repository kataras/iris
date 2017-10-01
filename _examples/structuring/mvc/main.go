package main

import (
	"github.com/kataras/iris"

	"github.com/kataras/iris/_examples/structuring/mvc/app"
)

func main() {
	// http://localhost:8080
	// http://localhost:8080/follower/42
	// http://localhost:8080/following/42
	// http://localhost:8080/like/42
	app.Boot(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed), iris.WithoutVersionChecker)
}
