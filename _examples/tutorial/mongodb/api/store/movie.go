package storeapi

import (
	"github.com/kataras/iris/_examples/tutorial/mongodb/httputil"
	"github.com/kataras/iris/_examples/tutorial/mongodb/store"

	"github.com/kataras/iris"
)

type MovieHandler struct {
	service store.MovieService
}

func NewMovieHandler(service store.MovieService) *MovieHandler {
	return &MovieHandler{service: service}
}

func (h *MovieHandler) GetAll(ctx iris.Context) {
	movies, err := h.service.GetAll(nil)
	if err != nil {
		httputil.InternalServerErrorJSON(ctx, err, "Server was unable to retrieve all movies")
		return
	}

	if movies == nil {
		// will return "null" if empty, with this "trick" we return "[]" json.
		movies = make([]store.Movie, 0)
	}

	ctx.JSON(movies)
}

func (h *MovieHandler) Get(ctx iris.Context) {
	id := ctx.Params().Get("id")

	m, err := h.service.GetByID(nil, id)
	if err != nil {
		if err == store.ErrNotFound {
			ctx.NotFound()
		} else {
			httputil.InternalServerErrorJSON(ctx, err, "Server was unable to retrieve movie [%s]", id)
		}
		return
	}

	ctx.JSON(m)
}

func (h *MovieHandler) Add(ctx iris.Context) {
	m := new(store.Movie)

	err := ctx.ReadJSON(m)
	if err != nil {
		httputil.FailJSON(ctx, iris.StatusBadRequest, err, "Malformed request payload")
		return
	}

	err = h.service.Create(nil, m)
	if err != nil {
		httputil.InternalServerErrorJSON(ctx, err, "Server was unable to create a movie")
		return
	}

	ctx.StatusCode(iris.StatusCreated)
	ctx.JSON(m)
}

func (h *MovieHandler) Update(ctx iris.Context) {
	id := ctx.Params().Get("id")

	var m store.Movie
	err := ctx.ReadJSON(&m)
	if err != nil {
		httputil.FailJSON(ctx, iris.StatusBadRequest, err, "Malformed request payload")
		return
	}

	err = h.service.Update(nil, id, m)
	if err != nil {
		if err == store.ErrNotFound {
			ctx.NotFound()
			return
		}
		httputil.InternalServerErrorJSON(ctx, err, "Server was unable to update movie [%s]", id)
		return
	}
}

func (h *MovieHandler) Delete(ctx iris.Context) {
	id := ctx.Params().Get("id")

	err := h.service.Delete(nil, id)
	if err != nil {
		if err == store.ErrNotFound {
			ctx.NotFound()
			return
		}
		httputil.InternalServerErrorJSON(ctx, err, "Server was unable to delete movie [%s]", id)
		return
	}
}
