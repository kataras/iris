package controllers

import (
	"github.com/kataras/iris/_examples/mvc/login/datamodels"
	"github.com/kataras/iris/_examples/mvc/login/services"

	"github.com/kataras/iris"
)

// UsersController is our /users API controller.
// GET				/users  | get all
// GET				/users/{id:long} | get by id
// PUT				/users/{id:long} | update by id
// DELETE			/users/{id:long} | delete by id
// Requires basic authentication.
type UsersController struct {
	// Optionally: context is auto-binded by Iris on each request,
	// remember that on each incoming request iris creates a new UserController each time,
	// so all fields are request-scoped by-default, only dependency injection is able to set
	// custom fields like the Service which is the same for all requests (static binding).
	Ctx iris.Context

	// Our UserService, it's an interface which
	// is binded from the main application.
	Service services.UserService
}

// Get returns list of the users.
// Demo:
// curl -i -u admin:password http://localhost:8080/users
//
// The correct way if you have sensitive data:
// func (c *UsersController) Get() (results []viewmodels.User) {
// 	data := c.Service.GetAll()
//
// 	for _, user := range data {
// 		results = append(results, viewmodels.User{user})
// 	}
// 	return
// }
// otherwise just return the datamodels.
func (c *UsersController) Get() (results []datamodels.User) {
	return c.Service.GetAll()
}

// GetBy returns a user.
// Demo:
// curl -i -u admin:password http://localhost:8080/users/1
func (c *UsersController) GetBy(id int64) (user datamodels.User, found bool) {
	u, found := c.Service.GetByID(id)
	if !found {
		// this message will be binded to the
		// main.go -> app.OnAnyErrorCode -> NotFound -> shared/error.html -> .Message text.
		c.Ctx.Values().Set("message", "User couldn't be found!")
	}
	return u, found // it will throw/emit 404 if found == false.
}

// PutBy updates a user.
// Demo:
// curl -i -X PUT -u admin:password -F "username=kataras"
// -F "password=rawPasswordIsNotSafeIfOrNotHTTPs_You_Should_Use_A_client_side_lib_for_hash_as_well"
// http://localhost:8080/users/1
func (c *UsersController) PutBy(id int64) (datamodels.User, error) {
	// username := c.Ctx.FormValue("username")
	// password := c.Ctx.FormValue("password")
	u := datamodels.User{}
	if err := c.Ctx.ReadForm(&u); err != nil {
		return u, err
	}

	return c.Service.Update(id, u)
}

// DeleteBy deletes a user.
// Demo:
// curl -i -X DELETE -u admin:password http://localhost:8080/users/1
func (c *UsersController) DeleteBy(id int64) interface{} {
	wasDel := c.Service.DeleteByID(id)
	if wasDel {
		// return the deleted user's ID
		return map[string]interface{}{"deleted": id}
	}
	// right here we can see that a method function
	// can return any of those two types(map or int),
	// we don't have to specify the return type to a specific type.
	return iris.StatusBadRequest // same as 400.
}
