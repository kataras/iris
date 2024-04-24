package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))

	// Hijack each output value of a method (can be used per-party too).
	app.ConfigureContainer().
		UseResultHandler(func(next iris.ResultHandler) iris.ResultHandler {
			return func(ctx iris.Context, v interface{}) error {
				switch val := v.(type) {
				case errorResponse:
					return next(ctx, errorView(val))
				default:
					return next(ctx, v)
				}
			}
		})

	m := mvc.New(app)
	m.Handle(new(controller))

	app.Listen(":8080")
}

func errorView(e errorResponse) mvc.Result {
	switch e.Code {
	case iris.StatusNotFound:
		return mvc.View{Code: e.Code, Name: "404.html", Data: e}
	default:
		return mvc.View{Code: e.Code, Name: "500.html", Data: e}
	}
}

type errorResponse struct {
	Code    int
	Message string
}

type controller struct{}

type user struct {
	ID uint64 `json:"id"`
}

func (c *controller) GetBy(userid uint64) interface{} {
	if userid != 1 {
		return errorResponse{
			Code:    iris.StatusNotFound,
			Message: "User Not Found",
		}
	}

	return user{ID: userid}
}
