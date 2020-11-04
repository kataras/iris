package main

import (
	"myapp/api"
	"myapp/domain/repository"

	"github.com/kataras/iris/v12"
)

var (
	userRepository = repository.NewMemoryUserRepository()
	todoRepository = repository.NewMemoryTodoRepository()
)

func main() {
	if err := repository.GenerateSamples(userRepository, todoRepository); err != nil {
		panic(err)
	}

	app := iris.New()

	app.Post("/signin", api.SignIn(userRepository))

	verify := api.Verify()

	todosAPI := app.Party("/todos", verify)
	todosAPI.Post("/", api.CreateTodo(todoRepository))
	todosAPI.Get("/", api.ListTodos(todoRepository))
	todosAPI.Get("/{id}", api.GetTodo(todoRepository))

	adminAPI := app.Party("/admin", verify, api.AllowAdmin)
	adminAPI.Get("/todos", api.ListAllTodos(todoRepository))

	// POST http://localhost:8080/signin (Form: username, password)
	// GET  http://localhost:8080/todos
	// GET  http://localhost:8080/todos/{id}
	// POST http://localhost:8080/todos (JSON, Form or URL: title, body)
	// GET  http://localhost:8080/admin/todos
	app.Listen(":8080")
}
