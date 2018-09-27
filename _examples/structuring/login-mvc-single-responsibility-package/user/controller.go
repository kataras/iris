package user

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
)

var (
	// About Code: iris.StatusSeeOther ->
	// When redirecting from POST to GET request you -should- use this HTTP status code,
	// however there're some (complicated) alternatives if you
	// search online or even the HTTP RFC.
	// "See Other" RFC 7231
	pathMyProfile = mvc.Response{Path: "/user/me", Code: iris.StatusSeeOther}
	pathRegister  = mvc.Response{Path: "/user/register"}
)

// Controller is responsible to handle the following requests:
// GET  			/user/register
// POST 			/user/register
// GET 				/user/login
// POST 			/user/login
// GET 				/user/me
// GET				/user/{id:long} | long is a new param type, it's the int64.
// All HTTP Methods /user/logout
type Controller struct {
	AuthController
}

type formValue func(string) string

// BeforeActivation called once before the server start
// and before the controller's registration, here you can add
// dependencies, to this controller and only, that the main caller may skip.
func (c *Controller) BeforeActivation(b mvc.BeforeActivation) {
	// bind the context's `FormValue` as well in order to be
	// acceptable on the controller or its methods' input arguments (NEW feature as well).
	b.Dependencies().Add(func(ctx iris.Context) formValue { return ctx.FormValue })
}

type page struct {
	Title string
}

// GetRegister handles GET:/user/register.
// mvc.Result can accept any struct which contains a `Dispatch(ctx iris.Context)` method.
// Both mvc.Response and mvc.View are mvc.Result.
func (c *Controller) GetRegister() mvc.Result {
	if c.isLoggedIn() {
		return c.logout()
	}

	// You could just use it as a variable to win some time in serve-time,
	// this is an exersise for you :)
	return mvc.View{
		Name: pathRegister.Path + ".html",
		Data: page{"User Registration"},
	}
}

// PostRegister handles POST:/user/register.
func (c *Controller) PostRegister(form formValue) mvc.Result {
	// we can either use the `c.Ctx.ReadForm` or read values one by one.
	var (
		firstname = form("firstname")
		username  = form("username")
		password  = form("password")
	)

	user, err := c.createOrUpdate(firstname, username, password)
	if err != nil {
		return c.fireError(err)
	}

	// setting a session value was never easier.
	c.Session.Set(sessionIDKey, user.ID)
	// succeed, nothing more to do here, just redirect to the /user/me.
	return pathMyProfile
}

// with these static views,
// you can use variables-- that are initialized before server start
// so you can win some time on serving.
// You can do it else where as well but I let them as pracise for you,
// essentially you can understand by just looking below.
var userLoginView = mvc.View{
	Name: PathLogin.Path + ".html",
	Data: page{"User Login"},
}

// GetLogin handles GET:/user/login.
func (c *Controller) GetLogin() mvc.Result {
	if c.isLoggedIn() {
		return c.logout()
	}
	return userLoginView
}

// PostLogin handles POST:/user/login.
func (c *Controller) PostLogin(form formValue) mvc.Result {
	var (
		username = form("username")
		password = form("password")
	)

	user, err := c.verify(username, password)
	if err != nil {
		return c.fireError(err)
	}

	c.Session.Set(sessionIDKey, user.ID)
	return pathMyProfile
}

// AnyLogout handles any method on path /user/logout.
func (c *Controller) AnyLogout() {
	c.logout()
}

// GetMe handles GET:/user/me.
func (c *Controller) GetMe() mvc.Result {
	id, err := c.Session.GetInt64(sessionIDKey)
	if err != nil || id <= 0 {
		// when not already logged in, redirect to login.
		return PathLogin
	}

	u, found := c.Source.GetByID(id)
	if !found {
		// if the  session exists but for some reason the user doesn't exist in the "database"
		// then logout him and redirect to the register page.
		return c.logout()
	}

	// set the model and render the view template.
	return mvc.View{
		Name: pathMyProfile.Path + ".html",
		Data: iris.Map{
			"Title": "Profile of " + u.Username,
			"User":  u,
		},
	}
}

func (c *Controller) renderNotFound(id int64) mvc.View {
	return mvc.View{
		Code: iris.StatusNotFound,
		Name: "user/notfound.html",
		Data: iris.Map{
			"Title": "User Not Found",
			"ID":    id,
		},
	}
}

// Dispatch completes the `mvc.Result` interface
// in order to be able to return a type of `Model`
// as mvc.Result.
// If this function didn't exist then
// we should explicit set the output result to that Model or to an interface{}.
func (u Model) Dispatch(ctx iris.Context) {
	ctx.JSON(u)
}

// GetBy handles GET:/user/{id:long},
// i.e http://localhost:8080/user/1
func (c *Controller) GetBy(userID int64) mvc.Result {
	// we have /user/{id}
	// fetch and render user json.
	user, found := c.Source.GetByID(userID)
	if !found {
		// not user found with that ID.
		return c.renderNotFound(userID)
	}

	// Q: how the hell Model can be return as mvc.Result?
	// A: I told you before on some comments and the docs,
	// any struct that has a `Dispatch(ctx iris.Context)`
	// can be returned as an mvc.Result(see ~20 lines above),
	// therefore we are able to combine many type of results in the same method.
	// For example, here, we return either an mvc.View to render a not found custom template
	// either a user which returns the Model as JSON via its Dispatch.
	//
	// We could also return just a struct value that is not an mvc.Result,
	// if the output result of the `GetBy` was that struct's type or an interface{}
	// and iris would render that with JSON as well, but here we can't do that without complete the `Dispatch`
	// function, because we may return an mvc.View which is an mvc.Result.
	return user
}
