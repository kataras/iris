package main

import (
	"strings"

	"github.com/kataras/iris/v12/_examples/mvc/vuejs-todo-mvc/src/todo"
	"github.com/kataras/iris/v12/_examples/mvc/vuejs-todo-mvc/src/web/controllers"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/kataras/iris/v12/sessions"
	"github.com/kataras/iris/v12/websocket"
)

func main() {
	app := iris.New()

	// serve our app in public, public folder
	// contains the client-side vue.js application,
	// no need for any server-side template here,
	// actually if you're going to just use vue without any
	// back-end services, you can just stop afer this line and start the server.
	app.HandleDir("/", iris.Dir("./public"))

	// configure the http sessions.
	sess := sessions.New(sessions.Config{
		Cookie: "iris_session",
	})

	// create a sub router and register the http controllers.
	todosRouter := app.Party("/todos")

	// Register sessions handler.
	// TodoController.Session will automatically
	// filled with the current request's session.
	todosRouter.Use(sess.Handler())

	// create our mvc application targeted to /todos relative sub path.
	todosApp := mvc.New(todosRouter)

	// any dependencies bindings here...
	todosApp.Register(
		todo.NewMemoryService(),
	)

	todosController := new(controllers.TodoController)
	// controllers registration here...
	todosApp.Handle(todosController)

	// Create a sub mvc app for websocket controller.
	// Inherit the parent's dependencies.
	todosWebsocketApp := todosApp.Party("/sync")
	todosWebsocketApp.HandleWebsocket(todosController).
		SetNamespace("todos").
		SetEventMatcher(func(methodName string) (string, bool) {
			return strings.ToLower(methodName), true
		})

	websocketServer := websocket.New(websocket.DefaultGorillaUpgrader, todosWebsocketApp)
	idGenerator := func(ctx iris.Context) string {
		id := sess.Start(ctx).ID()
		return id
	}
	todosWebsocketApp.Router.Get("/", websocket.Handler(websocketServer, idGenerator))

	// start the web server at http://localhost:8080
	app.Listen(":8080")
}
