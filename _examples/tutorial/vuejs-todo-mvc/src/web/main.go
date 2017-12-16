package main

import (
	"github.com/kataras/iris/_examples/tutorial/vuejs-todo-mvc/src/todo"
	"github.com/kataras/iris/_examples/tutorial/vuejs-todo-mvc/src/web/controllers"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"

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

	sess := sessions.New(sessions.Config{
		Cookie: "_iris_session",
	})

	m := mvc.New(app.Party("/todo"))

	// any dependencies bindings here...
	m.AddDependencies(
		mvc.Session(sess),
		new(todo.MemoryService),
	)

	// controllers registration here...
	m.Register(new(controllers.TodoController))

	// start the web server at http://localhost:8080
	app.Run(iris.Addr(":8080"), iris.WithoutVersionChecker, iris.WithOptimizations)
}
