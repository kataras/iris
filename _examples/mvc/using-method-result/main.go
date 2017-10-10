// file: main.go

package main

import (
	"github.com/kataras/iris/_examples/mvc/using-method-result/controllers"
	"github.com/kataras/iris/_examples/mvc/using-method-result/datasource"
	"github.com/kataras/iris/_examples/mvc/using-method-result/middleware"
	"github.com/kataras/iris/_examples/mvc/using-method-result/services"

	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	// Load the template files.
	app.RegisterView(iris.HTML("./views", ".html"))

	// Register our controllers.
	app.Controller("/hello", new(controllers.HelloController))

	// Create our movie service (memory), we will bind it to the movie controller.
	service := services.NewMovieServiceFromMemory(datasource.Movies)

	app.Controller("/movies", new(controllers.MovieController),
		// Bind the "service" to the MovieController's Service (interface) field.
		service,
		// Add the basic authentication(admin:password) middleware
		// for the /movies based requests.
		middleware.BasicAuth)

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
