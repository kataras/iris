package controllers

import (
	"github.com/kataras/iris/_examples/tutorial/vuejs-todo-mvc/src/todo"

	"github.com/kataras/iris"
	mvc "github.com/kataras/iris/mvc2"
	"github.com/kataras/iris/sessions"
)

// TodoController is our TODO app's web controller.
type TodoController struct {
	service todo.Service

	session *sessions.Session
}

// OnActivate called once before the server ran, can bind custom
// things to the controller.
func (c *TodoController) OnActivate(ca *mvc.ControllerActivator) {
	// this could be binded to a controller's function input argument
	// if any, or struct field if any:
	ca.Bind(func(ctx iris.Context) todo.Item {
		// ctx.ReadForm(&item)
		var (
			owner = ctx.PostValue("owner")
			body  = ctx.PostValue("body")
			state = ctx.PostValue("state")
		)

		return todo.Item{
			OwnerID:      owner,
			Body:         body,
			CurrentState: todo.ParseState(state),
		}
	})

}

// Get handles the GET: /todo route.
func (c *TodoController) Get() []todo.Item {
	return c.service.GetByOwner(c.session.ID())
}

// PutCompleteBy handles the PUT: /todo/complete/{id:long} route.
func (c *TodoController) PutCompleteBy(id int64) int {
	item, found := c.service.GetByID(id)
	if !found {
		return iris.StatusNotFound
	}

	if item.OwnerID != c.session.ID() {
		return iris.StatusForbidden
	}

	if !c.service.Complete(item) {
		return iris.StatusBadRequest
	}

	return iris.StatusOK
}

// Post handles the POST: /todo route.
func (c *TodoController) Post(newItem todo.Item) int {
	if newItem.OwnerID != c.session.ID() {
		return iris.StatusForbidden
	}

	if err := c.service.Save(newItem); err != nil {
		return iris.StatusBadRequest
	}
	return iris.StatusOK
}
