package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
)

type postValue func(string) string

func main() {
	app := iris.New()

	mvc.New(app.Party("/user")).Register(
		func(ctx iris.Context) postValue {
			return ctx.PostValue
		}).Handle(new(UserController))

	// GET http://localhost:9092/user
	// GET http://localhost:9092/user/42
	// POST http://localhost:9092/user
	// PUT http://localhost:9092/user/42
	// DELETE http://localhost:9092/user/42
	// GET http://localhost:9092/user/followers/42
	app.Run(iris.Addr(":9092"))
}

// UserController is our user example controller.
type UserController struct{}

// Get handles GET /user
func (c *UserController) Get() string {
	return "Select all users"
}

// User is our test User model, nothing tremendous here.
type User struct{ ID int64 }

// GetBy handles GET /user/42, equal to .Get("/user/{id:long}")
func (c *UserController) GetBy(id int64) User {
	// Select User by ID == $id.
	return User{id}
}

// Post handles POST /user
func (c *UserController) Post(post postValue) string {
	username := post("username")
	return "Create by user with username: " + username
}

// PutBy handles PUT /user/42
func (c *UserController) PutBy(id int) string {
	// Update user by ID == $id
	return "User updated"
}

// DeleteBy handles DELETE /user/42
func (c *UserController) DeleteBy(id int) bool {
	// Delete user by ID == %id
	//
	// when boolean then true = iris.StatusOK, false = iris.StatusNotFound
	return true
}

// GetFollowersBy handles GET /user/followers/42
func (c *UserController) GetFollowersBy(id int) []User {
	// Select all followers by user ID == $id
	return []User{ /* ... */ }
}
