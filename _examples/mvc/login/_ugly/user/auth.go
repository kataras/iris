package user

import (
	"errors"
	"strconv"
	"strings"

	"github.com/kataras/iris"
)

// paths
const (
	PathLogin  = "/user/login"
	PathLogout = "/user/logout"
)

// the session key for the user id comes from the Session.
const (
	sessionIDKey = "UserID"
)

// AuthController is the user authentication controller, a custom shared controller.
type AuthController struct {
	iris.SessionController

	Source *DataSource
	User   Model `iris:"model"`
}

// BeginRequest saves login state to the context, the user id.
func (c *AuthController) BeginRequest(ctx iris.Context) {
	c.SessionController.BeginRequest(ctx)

	if userID := c.Session.Get(sessionIDKey); userID != nil {
		ctx.Values().Set(sessionIDKey, userID)
	}
}

func (c *AuthController) fireError(err error) {
	if err != nil {
		c.Ctx.Application().Logger().Debug(err.Error())

		c.Status = 400
		c.Data["Title"] = "User Error"
		c.Data["Message"] = strings.ToUpper(err.Error())
		c.Tmpl = "shared/error.html"
	}
}

func (c *AuthController) redirectTo(id int64) {
	if id > 0 {
		c.Path = "/user/" + strconv.Itoa(int(id))
	}
}

func (c *AuthController) createOrUpdate(firstname, username, password string) (user Model, err error) {
	username = strings.Trim(username, " ")
	if username == "" || password == "" || firstname == "" {
		return user, errors.New("empty firstname, username or/and password")
	}

	userToInsert := Model{
		Firstname: firstname,
		Username:  username,
		password:  password,
	} // password is hashed by the Source.

	newUser, err := c.Source.InsertOrUpdate(userToInsert)
	if err != nil {
		return user, err
	}

	return newUser, nil
}

func (c *AuthController) isLoggedIn() bool {
	// we don't search by session, we have the user id
	// already by the `SaveState` middleware.
	return c.Values.Get(sessionIDKey) != nil
}

func (c *AuthController) verify(username, password string) (user Model, err error) {
	if username == "" || password == "" {
		return user, errors.New("please fill both username and password fields")
	}

	u, found := c.Source.GetByUsername(username)
	if !found {
		// if user found with that username not found at all.
		return user, errors.New("user with that username does not exist")
	}

	if ok, err := ValidatePassword(password, u.HashedPassword); err != nil || !ok {
		// if user found but an error occurred or the password is not valid.
		return user, errors.New("please try to login with valid credentials")
	}

	return u, nil
}

// if logged in then destroy the session
// and redirect to the login page
// otherwise redirect to the registration page.
func (c *AuthController) logout() {
	if c.isLoggedIn() {
		// c.Manager is the Sessions manager created
		// by the embedded SessionController, automatically.
		c.Manager.DestroyByID(c.Session.ID())
		return
	}

	c.Path = PathLogin
}

// AllowUser will check if this client is a logged user,
// if not then it will redirect that guest to the login page
// otherwise it will allow the execution of the next handler.
func AllowUser(ctx iris.Context) {
	if ctx.Values().Get(sessionIDKey) != nil {
		ctx.Next()
		return
	}
	ctx.Redirect(PathLogin)
}
