// file: web/routes/movie.go

package routes

import (
	"errors"

	"github.com/kataras/iris/_examples/hero/overview/datamodels"
	"github.com/kataras/iris/_examples/hero/overview/services"

	"github.com/kataras/iris"
)

// Movies returns list of the movies.
// Demo:
// curl -i http://localhost:8080/movies
func Movies(service services.MovieService) (results []datamodels.Movie) {
	return service.GetAll()
}

// MovieByID returns a movie.
// Demo:
// curl -i http://localhost:8080/movies/1
func MovieByID(service services.MovieService, id int64) (movie datamodels.Movie, found bool) {
	return service.GetByID(id) // it will throw 404 if not found.
}

// UpdateMovieByID updates a movie.
// Demo:
// curl -i -X PUT -F "genre=Thriller" -F "poster=@/Users/kataras/Downloads/out.gif" http://localhost:8080/movies/1
func UpdateMovieByID(ctx iris.Context, service services.MovieService, id int64) (datamodels.Movie, error) {
	// get the request data for poster and genre
	file, info, err := ctx.FormFile("poster")
	if err != nil {
		return datamodels.Movie{}, errors.New("failed due form file 'poster' missing")
	}
	// we don't need the file so close it now.
	file.Close()

	// imagine that is the url of the uploaded file...
	poster := info.Filename
	genre := ctx.FormValue("genre")

	return service.UpdatePosterAndGenreByID(id, poster, genre)
}

// DeleteMovieByID deletes a movie.
// Demo:
// curl -i -X DELETE -u admin:password http://localhost:8080/movies/1
func DeleteMovieByID(service services.MovieService, id int64) interface{} {
	wasDel := service.DeleteByID(id)
	if wasDel {
		// return the deleted movie's ID
		return iris.Map{"deleted": id}
	}
	// right here we can see that a method function can return any of those two types(map or int),
	// we don't have to specify the return type to a specific type.
	return iris.StatusBadRequest
}
