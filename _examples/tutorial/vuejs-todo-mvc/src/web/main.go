package main

import (
	"github.com/kataras/iris/_examples/tutorial/vuejs-todo-mvc/src/todo"
	"github.com/kataras/iris/_examples/tutorial/vuejs-todo-mvc/src/web/controllers"

	"github.com/kataras/iris"
	mvc "github.com/kataras/iris/mvc2"
	"github.com/kataras/iris/sessions"
)

func main() {
	app := iris.New()
	// serve our app in public, public folder
	// contains the client-side vue.js application,
	// no need for any server-side template here,
	// actually if you're going to just use vue without any
	// back-end services, you can just stop afer this line and start the server.
	app.StaticWeb("/", "./public")

	sess := sessions.New(sessions.Config{
		Cookie: "_iris_session",
	})

	m := mvc.New()

	// any bindings here...
	m.Bind(mvc.Session(sess))

	m.Bind(new(todo.MemoryService))
	// controllers registration here...
	m.Controller(app.Party("/todo"), new(controllers.TodoController))

	// start the web server at http://localhost:8080
	app.Run(iris.Addr(":8080"), iris.WithoutVersionChecker, iris.WithOptimizations)
}
