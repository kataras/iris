package main

import (
	"github.com/kataras/iris"
)

func main() {

	app := iris.New()

	app.Controller("/user", new(UserController))

	// GET http://localhost:9092/user
	// GET http://localhost:9092/user/42
	// POST http://localhost:9092/user
	// PUT http://localhost:9092/user/42
	// DELETE http://localhost:9092/user/42
	// GET http://localhost:9092/user/followers/42
	app.Run(iris.Addr(":9092"))
}

// UserController is our user example controller.
type UserController struct {
	iris.Controller
}

// Get handles GET /user
func (c *UserController) Get() {
	c.Ctx.Writef("Select all users")
}

// GetBy handles GET /user/42
func (c *UserController) GetBy(id int) {
	c.Ctx.Writef("Select user by ID: %d", id)
}

// Post handles POST /user
func (c *UserController) Post() {
	username := c.Ctx.PostValue("username")
	c.Ctx.Writef("Create by user with username: %s", username)
}

// PutBy handles PUT /user/42
func (c *UserController) PutBy(id int) {
	c.Ctx.Writef("Update user by ID: %d", id)
}

// DeleteBy handles DELETE /user/42
func (c *UserController) DeleteBy(id int) {
	c.Ctx.Writef("Delete user by ID: %d", id)
}

// GetFollowersBy handles GET /user/followers/42
func (c *UserController) GetFollowersBy(id int) {
	c.Ctx.Writef("Select all followers by user ID: %d", id)
}
