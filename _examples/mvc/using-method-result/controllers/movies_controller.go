// file: controllers/movies_controller.go
//
// This is just an example of usage, don't use it for production, it even doesn't check for
// index exceed!

package controllers

import (
	"github.com/kataras/iris/_examples/mvc/using-method-result/datasource"
	"github.com/kataras/iris/_examples/mvc/using-method-result/models"

	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
)

// MoviesController is our /movies controller.
type MoviesController struct {
	// mvc.C is just a lightweight lightweight alternative
	// to the "mvc.Controller" controller type,
	// use it when you don't need mvc.Controller's fields
	// (you don't need those fields when you return values from the method functions).
	mvc.C
}

// Get returns list of the movies.
// Demo:
// curl -i http://localhost:8080/movies
func (c *MoviesController) Get() []models.Movie {
	return datasource.Movies
}

// GetBy returns a movie.
// Demo:
// curl -i http://localhost:8080/movies/1
func (c *MoviesController) GetBy(id int) models.Movie {
	return datasource.Movies[id]
}

// PutBy updates a movie.
// Demo:
// curl -i -X PUT -F "genre=Thriller" -F "poster=@/Users/kataras/Downloads/out.gif" http://localhost:8080/movies/1
func (c *MoviesController) PutBy(id int) (models.Movie, int) {
	// get the movie
	m := datasource.Movies[id]

	// get the request data for poster and genre
	file, info, err := c.Ctx.FormFile("poster")
	if err != nil {
		return models.Movie{}, iris.StatusInternalServerError
	}
	// we don't need the file so close it now
	file.Close()

	// imagine that is the url of the uploaded file...
	poster := info.Filename
	genre := c.Ctx.FormValue("genre")

	// update the poster
	m.Poster = poster
	m.Genre = genre
	datasource.Movies[id] = m

	return m, iris.StatusOK
}

// DeleteBy deletes a movie.
// Demo:
// curl -i -X DELETE -u admin:password http://localhost:8080/movies/1
func (c *MoviesController) DeleteBy(id int) iris.Map {
	// delete the entry from the movies slice
	deleted := datasource.Movies[id].Name
	datasource.Movies = append(datasource.Movies[:id], datasource.Movies[id+1:]...)
	// and return the deleted movie's name
	return iris.Map{"deleted": deleted}
}
