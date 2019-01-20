// file: main.go

package main

import (
	"github.com/kataras/iris/_examples/mvc/overview/datasource"
	"github.com/kataras/iris/_examples/mvc/overview/repositories"
	"github.com/kataras/iris/_examples/mvc/overview/services"
	"github.com/kataras/iris/_examples/mvc/overview/web/controllers"
	"github.com/kataras/iris/_examples/mvc/overview/web/middleware"

	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// Load the template files.
	app.RegisterView(iris.HTML("./web/views", ".html"))

	// Serve our controllers.
	mvc.New(app.Party("/hello")).Handle(new(controllers.HelloController))
	// You can also split the code you write to configure an mvc.Application
	// using the `mvc.Configure` method, as shown below.
	mvc.Configure(app.Party("/movies"), movies)

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

// note the mvc.Application, it's not iris.Application.
func movies(app *mvc.Application) {
	// Add the basic authentication(admin:password) middleware
	// for the /movies based requests.
	app.Router.Use(middleware.BasicAuth)

	// Create our movie repository with some (memory) data from the datasource.
	repo := repositories.NewMovieRepository(datasource.Movies)
	// Create our movie service, we will bind it to the movie app's dependencies.
	movieService := services.NewMovieService(repo)
	app.Register(movieService)

	// serve our movies controller.
	// Note that you can serve more than one controller
	// you can also create child mvc apps using the `movies.Party(relativePath)` or `movies.Clone(app.Party(...))`
	// if you want.
	app.Handle(new(controllers.MovieController))
}
