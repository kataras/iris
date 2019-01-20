// file: main.go

package main

import (
	"github.com/kataras/iris/_examples/hero/overview/datasource"
	"github.com/kataras/iris/_examples/hero/overview/repositories"
	"github.com/kataras/iris/_examples/hero/overview/services"
	"github.com/kataras/iris/_examples/hero/overview/web/middleware"
	"github.com/kataras/iris/_examples/hero/overview/web/routes"

	"github.com/kataras/iris"
	"github.com/kataras/iris/hero"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// Load the template files.
	app.RegisterView(iris.HTML("./web/views", ".html"))

	// Create our movie repository with some (memory) data from the datasource.
	repo := repositories.NewMovieRepository(datasource.Movies)
	// Create our movie service, we will bind it to the movie app's dependencies.
	movieService := services.NewMovieService(repo)
	hero.Register(movieService)

	// Serve our routes with hero handlers.
	app.PartyFunc("/hello", func(r iris.Party) {
		r.Get("/", hero.Handler(routes.Hello))
		r.Get("/{name}", hero.Handler(routes.HelloName))
	})

	app.PartyFunc("/movies", func(r iris.Party) {
		// Add the basic authentication(admin:password) middleware
		// for the /movies based requests.
		r.Use(middleware.BasicAuth)

		r.Get("/", hero.Handler(routes.Movies))
		r.Get("/{id:long}", hero.Handler(routes.MovieByID))
		r.Put("/{id:long}", hero.Handler(routes.UpdateMovieByID))
		r.Delete("/{id:long}", hero.Handler(routes.DeleteMovieByID))
	})

	// http://localhost:8080/hello
	// http://localhost:8080/hello/iris
	// http://localhost:8080/movies
	// http://localhost:8080/movies/1
	app.Run(
		// Start the web server at localhost:8080
		iris.Addr("localhost:8080"),
		// skip err server closed when CTRL/CMD+C pressed:
		iris.WithoutServerError(iris.ErrServerClosed),
		// enables faster json serialization and more:
		iris.WithOptimizations,
	)
}
