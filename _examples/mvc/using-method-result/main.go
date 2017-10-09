// file: main.go

package main

import (
	"github.com/kataras/iris/_examples/mvc/using-method-result/controllers"
	"github.com/kataras/iris/_examples/mvc/using-method-result/middleware"

	"github.com/kataras/iris"
)

func main() {
	app := iris.New()
	// Load the template files.
	app.RegisterView(iris.HTML("./views", ".html"))

	// Register our controllers.
	app.Controller("/hello", new(controllers.HelloController))
	// Add the basic authentication(admin:password) middleware
	// for the /movies based requests.
	app.Controller("/movies", new(controllers.MoviesController), middleware.BasicAuth)

	// Start the web server at localhost:8080
	// http://localhost:8080/hello
	// http://localhost:8080/hello/iris
	// http://localhost:8080/movies/1
	app.Run(
		iris.Addr("localhost:8080"),
		iris.WithoutVersionChecker,
		iris.WithoutServerError(iris.ErrServerClosed),
		iris.WithOptimizations, // enables faster json serialization and more
	)
}
