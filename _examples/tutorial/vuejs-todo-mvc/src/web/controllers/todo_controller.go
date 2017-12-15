package controllers

import (
	"github.com/kataras/iris/_examples/tutorial/vuejs-todo-mvc/src/todo"

	"github.com/kataras/iris"
	mvc "github.com/kataras/iris/mvc2"
	"github.com/kataras/iris/sessions"
)

// TodoController is our TODO app's web controller.
type TodoController struct {
	Service todo.Service

	Session *sessions.Session
}

// BeforeActivate called once before the server ran, and before
// the routes and dependency binder builded.
// You can bind custom things to the controller, add new methods, add middleware,
// add dependencies to the struct or the method(s) and more.
func (c *TodoController) BeforeActivate(ca *mvc.ControllerActivator) {
	// this could be binded to a controller's function input argument
	// if any, or struct field if any:
	ca.Dependencies.Add(func(ctx iris.Context) todo.Item {
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

	// ca.Router.Use(...).Done(...).Layout(...)
	// TODO:(?)
	// m := ca.Method("PutCompleteBy")
	// m.Route.Use(...).Done(...) <- we don't have the route here but I can find something to solve this.
	// m.Dependencies.Add(...)
}

// Get handles the GET: /todo route.
func (c *TodoController) Get() []todo.Item {
	return c.Service.GetByOwner(c.Session.ID())
}

// PutCompleteBy handles the PUT: /todo/complete/{id:long} route.
func (c *TodoController) PutCompleteBy(id int64) int {
	item, found := c.Service.GetByID(id)
	if !found {
		return iris.StatusNotFound
	}

	if item.OwnerID != c.Session.ID() {
		return iris.StatusForbidden
	}

	if !c.Service.Complete(item) {
		return iris.StatusBadRequest
	}

	return iris.StatusOK
}

// Post handles the POST: /todo route.
func (c *TodoController) Post(newItem todo.Item) int {
	if newItem.OwnerID != c.Session.ID() {
		return iris.StatusForbidden
	}

	if err := c.Service.Save(newItem); err != nil {
		return iris.StatusBadRequest
	}
	return iris.StatusOK
}
