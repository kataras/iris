package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))

	m := mvc.New(app)
	m.Handle(new(controller))

	app.Listen(":8080")
}

type errorResponse struct {
	Code    int
	Message string
}

/*
// Note: if a struct implements the standard go error, so it's an error
// and its Error() is not empty, then its text will be rendered instead,
// override any Dispatch method.
func (e errorResponse) Error() string {
	return e.Message
}
*/

// implements mvc.Result.
func (e errorResponse) Dispatch(ctx iris.Context) {
	// If u want to use mvc.Result on any method without an output return value
	// go for it:
	//
	view := mvc.View{Code: e.Code, Data: e} // use Code and Message as the template data.
	switch e.Code {
	case iris.StatusNotFound:
		view.Name = "404"
	default:
		view.Name = "500"
	}
	view.Dispatch(ctx)

	// Otherwise use ctx methods:
	//
	// ctx.StatusCode(e.Code)
	// switch e.Code {
	// case iris.StatusNotFound:
	// 	// use Code and Message as the template data.
	// if err := ctx.View("404.html", e)
	// default:
	// if err := ctx.View("500.html", e)
	// }
}

type controller struct{}

type user struct {
	ID uint64 `json:"id"`
}

func (c *controller) GetBy(userid uint64) mvc.Result {
	if userid != 1 {
		return errorResponse{
			Code:    iris.StatusNotFound,
			Message: "User Not Found",
		}
	}

	return mvc.Response{
		Object: user{ID: userid},
	}
}
