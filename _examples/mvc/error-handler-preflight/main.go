package main

import (
	"fmt"
	"time"

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

type controller struct{}

// Generic response type for JSON results.
type response struct {
	ID        uint64      `json:"id,omitempty"`
	Data      interface{} `json:"data,omitempty"` // {data: result } on fetch actions.
	Code      int         `json:"code,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp int64       `json:"timestamp,omitempty"`
}

func (r *response) Preflight(ctx iris.Context) error {
	if r.ID > 0 {
		r.Timestamp = time.Now().Unix()
	}

	if code := r.Code; code > 0 {
		// You can call ctx.View or mvc.View{...}.Dispatch
		// to render HTML on Code != 200
		// but in order to not proceed with the response resulting
		// as JSON you MUST return the iris.ErrStopExecution error.
		// Example:
		if code != 200 {
			mvc.View{
				/* calls the ctx.StatusCode */
				Code: code,
				/* use any r.Data as the template data
				OR the whole "response" as its data. */
				Data: r,
				/* automatically pick the template per error (just for the sake of the example) */
				Name: fmt.Sprintf("%d", code),
			}.Dispatch(ctx)

			return iris.ErrStopExecution
		}

		ctx.StatusCode(r.Code)
	}

	return nil
}

type user struct {
	ID uint64 `json:"id"`
}

func (c *controller) GetBy(userid uint64) *response {
	if userid != 1 {
		return &response{
			Code:    iris.StatusNotFound,
			Message: "User Not Found",
		}
	}

	return &response{
		ID:   userid,
		Data: user{ID: userid},
	}
}

/*

You can use that `response` structure on non-mvc applications too, using handlers:

c := app.ConfigureContainer()
c.Get("/{id:uint64}", getUserByID)
func getUserByID(id uint64) response {
	if userid != 1 {
		return response{
			Code:    iris.StatusNotFound,
			Message: "User Not Found",
		}
	}

	return response{
		ID:   userid,
		Data: user{ID: userid},
	}
}

*/
