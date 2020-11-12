package api

import (
	"myapp/domain/repository"

	"github.com/kataras/iris/v12"
)

// NewRouter accepts some dependencies
// and returns a function which returns the routes on the given Iris Party (group of routes).
func NewRouter(userRepo repository.UserRepository, todoRepo repository.TodoRepository) func(iris.Party) {
	return func(router iris.Party) {
		router.Post("/signin", SignIn(userRepo))

		router.Use(Verify()) // protect the next routes with JWT.

		router.Post("/todos", CreateTodo(todoRepo))
		router.Get("/todos", ListTodos(todoRepo))
		router.Get("/todos/{id}", GetTodo(todoRepo))

		router.Get("/admin/todos", AllowAdmin, ListAllTodos(todoRepo))
	}
}
