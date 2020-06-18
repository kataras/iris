// file: main.go

package main

import (
	"github.com/kataras/iris/v12/_examples/dependency-injection/overview/datasource"
	"github.com/kataras/iris/v12/_examples/dependency-injection/overview/repositories"
	"github.com/kataras/iris/v12/_examples/dependency-injection/overview/services"
	"github.com/kataras/iris/v12/_examples/dependency-injection/overview/web/middleware"
	"github.com/kataras/iris/v12/_examples/dependency-injection/overview/web/routes"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// Load the template files.
	app.RegisterView(iris.HTML("./web/views", ".html"))

	// Create our movie repository with some (memory) data from the datasource.
	repo := repositories.NewMovieRepository(datasource.Movies)

	app.Party("/hello").ConfigureContainer(func(r *iris.APIContainer) {
		r.Get("/", routes.Hello)
		r.Get("/{name}", routes.HelloName)
	})

	app.Party("/movies").ConfigureContainer(func(r *iris.APIContainer) {
		// Create our movie service, we will bind it to the movie app's dependencies.
		movieService := services.NewMovieService(repo)
		r.RegisterDependency(movieService)

		// Add the basic authentication(admin:password) middleware
		// for the /movies based requests.
		r.Use(middleware.BasicAuth)

		r.Get("/", routes.Movies)
		r.Get("/{id:uint64}", routes.MovieByID)
		r.Put("/{id:uint64}", routes.UpdateMovieByID)
		r.Delete("/{id:uint64}", routes.DeleteMovieByID)
	})

	// http://localhost:8080/hello
	// http://localhost:8080/hello/iris
	// http://localhost:8080/movies ("admin": "password")
	// http://localhost:8080/movies/1
	app.Listen(
		// Start the web server at localhost:8080
		"localhost:8080",
		// enables faster json serialization and more:
		iris.WithOptimizations,
	)
}
