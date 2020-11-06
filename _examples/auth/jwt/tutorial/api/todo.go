package api

import (
	"errors"
	"myapp/domain/repository"

	"github.com/kataras/iris/v12"
)

// TodoRequest represents a Todo HTTP request.
type TodoRequest struct {
	Title string `json:"title" form:"title" url:"title"`
	Body  string `json:"body" form:"body" url:"body"`
}

// CreateTodo handles the creation of a Todo entry.
func CreateTodo(repo repository.TodoRepository) iris.Handler {
	return func(ctx iris.Context) {
		var req TodoRequest
		err := ctx.ReadBody(&req) // will bind the "req" to a JSON, form or url query request data.
		if err != nil {
			ctx.StopWithError(iris.StatusBadRequest, err)
			return
		}

		userID := GetUserID(ctx)
		todo, err := repo.Create(userID, req.Title, req.Body)
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		ctx.StatusCode(iris.StatusCreated)
		ctx.JSON(todo)
	}
}

// GetTodo lists all users todos.
// Parameter: {id}.
func GetTodo(repo repository.TodoRepository) iris.Handler {
	return func(ctx iris.Context) {
		id := ctx.Params().Get("id")
		userID := GetUserID(ctx)

		todo, err := repo.GetByID(id)
		if err != nil {
			code := iris.StatusInternalServerError
			if errors.Is(err, repository.ErrNotFound) {
				code = iris.StatusNotFound
			}

			ctx.StopWithError(code, err)
			return
		}

		if !IsAdmin(ctx) { // admin can access any user's todos.
			if todo.UserID != userID {
				ctx.StopWithStatus(iris.StatusForbidden)
				return
			}
		}

		ctx.JSON(todo)
	}
}

// ListTodos lists todos of the current user.
func ListTodos(repo repository.TodoRepository) iris.Handler {
	return func(ctx iris.Context) {
		userID := GetUserID(ctx)
		todos, err := repo.GetAllByUser(userID)
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		// if len(todos) == 0 {
		// 	ctx.StopWithError(iris.StatusNotFound, fmt.Errorf("no entries found"))
		// 	return
		// }
		// Or let the client decide what to do on empty list.
		ctx.JSON(todos)
	}
}

// ListAllTodos lists all users todos.
// Access: admin.
// Middleware: AllowAdmin.
func ListAllTodos(repo repository.TodoRepository) iris.Handler {
	return func(ctx iris.Context) {
		todos, err := repo.GetAll()
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		ctx.JSON(todos)
	}
}

/* Leave as exercise: use filtering instead...

// ListTodosByUser lists all todos by a specific user.
// Access: admin.
// Middleware: AllowAdmin.
// Parameter: {id}.
func ListTodosByUser(repo repository.TodoRepository) iris.Handler {
	return func(ctx iris.Context) {
		userID := ctx.Params().Get("id")
		todos, err := repo.GetAllByUser(userID)
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		ctx.JSON(todos)
	}
}
*/
