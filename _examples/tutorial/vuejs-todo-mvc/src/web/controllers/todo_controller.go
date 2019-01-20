package controllers

import (
	"github.com/kataras/iris/_examples/tutorial/vuejs-todo-mvc/src/todo"

	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"github.com/kataras/iris/sessions"
	"github.com/kataras/iris/websocket"
)

// TodoController is our TODO app's web controller.
type TodoController struct {
	Service todo.Service

	Session *sessions.Session
}

// BeforeActivation called once before the server ran, and before
// the routes and dependencies binded.
// You can bind custom things to the controller, add new methods, add middleware,
// add dependencies to the struct or the method(s) and more.
func (c *TodoController) BeforeActivation(b mvc.BeforeActivation) {
	// this could be binded to a controller's function input argument
	// if any, or struct field if any:
	b.Dependencies().Add(func(ctx iris.Context) (items []todo.Item) {
		ctx.ReadJSON(&items)
		return
	})
}

// Get handles the GET: /todos route.
func (c *TodoController) Get() []todo.Item {
	return c.Service.Get(c.Session.ID())
}

// PostItemResponse the response data that will be returned as json
// after a post save action of all todo items.
type PostItemResponse struct {
	Success bool `json:"success"`
}

var emptyResponse = PostItemResponse{Success: false}

// Post handles the POST: /todos route.
func (c *TodoController) Post(newItems []todo.Item) PostItemResponse {
	if err := c.Service.Save(c.Session.ID(), newItems); err != nil {
		return emptyResponse
	}

	return PostItemResponse{Success: true}
}

func (c *TodoController) GetSync(conn websocket.Connection) {
	// join to the session in order to send "saved"
	// events only to a single user, that means
	// that if user has opened more than one browser window/tab
	// of the same session then the changes will be reflected to one another.
	conn.Join(c.Session.ID())
	conn.On("save", func() { // "save" event from client.
		conn.To(c.Session.ID()).Emit("saved", nil) // fire a "saved" event to the rest of the clients w.
	})

	conn.Wait()
}
