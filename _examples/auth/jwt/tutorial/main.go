package main

import (
	"myapp/api"
	"myapp/domain/repository"

	"github.com/kataras/iris/v12"
)

var (
	userRepo = repository.NewMemoryUserRepository()
	todoRepo = repository.NewMemoryTodoRepository()
)

func main() {
	if err := repository.GenerateSamples(userRepo, todoRepo); err != nil {
		panic(err)
	}

	app := iris.New()
	app.PartyFunc("/", api.NewRouter(userRepo, todoRepo))

	// POST http://localhost:8080/signin (Form: username, password)
	// GET  http://localhost:8080/todos
	// GET  http://localhost:8080/todos/{id}
	// POST http://localhost:8080/todos (JSON, Form or URL: title, body)
	// GET  http://localhost:8080/admin/todos
	app.Listen(":8080")
}
