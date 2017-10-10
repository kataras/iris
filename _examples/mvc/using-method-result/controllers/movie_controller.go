// file: controllers/movie_controller.go

package controllers

import (
	"errors"

	"github.com/kataras/iris/_examples/mvc/using-method-result/models"
	"github.com/kataras/iris/_examples/mvc/using-method-result/services"

	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
)

// MovieController is our /movies controller.
type MovieController struct {
	// mvc.C is just a lightweight lightweight alternative
	// to the "mvc.Controller" controller type,
	// use it when you don't need mvc.Controller's fields
	// (you don't need those fields when you return values from the method functions).
	mvc.C

	// Our MovieService, it's an interface which
	// is binded from the main application.
	Service services.MovieService
}

// Get returns list of the movies.
// Demo:
// curl -i http://localhost:8080/movies
func (c *MovieController) Get() []models.Movie {
	return c.Service.GetAll()
}

// GetBy returns a movie.
// Demo:
// curl -i http://localhost:8080/movies/1
func (c *MovieController) GetBy(id int64) models.Movie {
	m, _ := c.Service.GetByID(id)
	return m
}

// PutBy updates a movie.
// Demo:
// curl -i -X PUT -F "genre=Thriller" -F "poster=@/Users/kataras/Downloads/out.gif" http://localhost:8080/movies/1
func (c *MovieController) PutBy(id int64) (models.Movie, error) {
	// get the request data for poster and genre
	file, info, err := c.Ctx.FormFile("poster")
	if err != nil {
		return models.Movie{}, errors.New("failed due form file 'poster' missing")
	}
	// we don't need the file so close it now.
	file.Close()

	// imagine that is the url of the uploaded file...
	poster := info.Filename
	genre := c.Ctx.FormValue("genre")

	// update the movie and return it.
	return c.Service.InsertOrUpdate(models.Movie{
		ID:     id,
		Poster: poster,
		Genre:  genre,
	})
}

// DeleteBy deletes a movie.
// Demo:
// curl -i -X DELETE -u admin:password http://localhost:8080/movies/1
func (c *MovieController) DeleteBy(id int64) interface{} {
	wasDel := c.Service.DeleteByID(id)
	if wasDel {
		// return the deleted movie's ID
		return iris.Map{"deleted": id}
	}
	// right here we can see that a method function can return any of those two types(map or int),
	// we don't have to specify the return type to a specific type.
	return iris.StatusBadRequest
}
