package main

import (
	"github.com/kataras/iris/_examples/tutorial/vuejs-todo-mvc/src/todo"
	"github.com/kataras/iris/_examples/tutorial/vuejs-todo-mvc/src/web/controllers"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
	"github.com/kataras/iris/websocket"

	"github.com/kataras/iris/mvc"
)

func main() {
	app := iris.New()

	// serve our app in public, public folder
	// contains the client-side vue.js application,
	// no need for any server-side template here,
	// actually if you're going to just use vue without any
	// back-end services, you can just stop afer this line and start the server.
	app.StaticWeb("/", "./public")

	// configure the http sessions.
	sess := sessions.New(sessions.Config{
		Cookie: "iris_session",
	})

	// configure the websocket server.
	ws := websocket.New(websocket.Config{})

	// create a sub router and register the client-side library for the iris websockets,
	// you could skip it but iris websockets supports socket.io-like API.
	todosRouter := app.Party("/todos")
	// http://localhost:8080/todos/iris-ws.js
	// serve the javascript client library to communicate with
	// the iris high level websocket event system.
	todosRouter.Any("/iris-ws.js", websocket.ClientHandler())

	// create our mvc application targeted to /todos relative sub path.
	todosApp := mvc.New(todosRouter)

	// any dependencies bindings here...
	todosApp.Register(
		todo.NewMemoryService(),
		sess.Start,
		ws.Upgrade,
	)

	// controllers registration here...
	todosApp.Handle(new(controllers.TodoController))

	// start the web server at http://localhost:8080
	app.Run(iris.Addr(":8080"))
}
