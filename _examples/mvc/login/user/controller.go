package user

const (
	pathMyProfile = "/user/me"
	pathRegister  = "/user/register"
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

// GetRegister handles GET:/user/register.
func (c *Controller) GetRegister() {
	if c.isLoggedIn() {
		c.logout()
		return
	}

	c.Data["Title"] = "User Registration"
	c.Tmpl = pathRegister + ".html"
}

// PostRegister handles POST:/user/register.
func (c *Controller) PostRegister() {
	// we can either use the `c.Ctx.ReadForm` or read values one by one.
	var (
		firstname = c.Ctx.FormValue("firstname")
		username  = c.Ctx.FormValue("username")
		password  = c.Ctx.FormValue("password")
	)

	user, err := c.createOrUpdate(firstname, username, password)
	if err != nil {
		c.fireError(err)
		return
	}

	// setting a session value was never easier.
	c.Session.Set(sessionIDKey, user.ID)
	// succeed, nothing more to do here, just redirect to the /user/me.
	c.Path = pathMyProfile
}

// GetLogin handles GET:/user/login.
func (c *Controller) GetLogin() {
	if c.isLoggedIn() {
		c.logout()
		return
	}
	c.Data["Title"] = "User Login"
	c.Tmpl = PathLogin + ".html"
}

// PostLogin handles POST:/user/login.
func (c *Controller) PostLogin() {
	var (
		username = c.Ctx.FormValue("username")
		password = c.Ctx.FormValue("password")
	)

	user, err := c.verify(username, password)
	if err != nil {
		c.fireError(err)
		return
	}

	c.Session.Set(sessionIDKey, user.ID)
	c.Path = pathMyProfile
}

// AnyLogout handles any method on path /user/logout.
func (c *Controller) AnyLogout() {
	c.logout()
}

// GetMe handles GET:/user/me.
func (c *Controller) GetMe() {
	id, err := c.Session.GetInt64(sessionIDKey)
	if err != nil || id <= 0 {
		// when not already logged in.
		c.Path = PathLogin
		return
	}

	u, found := c.Source.GetByID(id)
	if !found {
		// if the  session exists but for some reason the user doesn't exist in the "database"
		// then logout him and redirect to the register page.
		c.logout()
		return
	}

	// set the model and render the view template.
	c.User = u
	c.Data["Title"] = "Profile of " + u.Username
	c.Tmpl = pathMyProfile + ".html"
}

func (c *Controller) renderNotFound(id int64) {
	c.Status = 404
	c.Data["Title"] = "User Not Found"
	c.Data["ID"] = id
	c.Tmpl = "user/notfound.html"
}

// GetBy handles GET:/user/{id:long},
// i.e http://localhost:8080/user/1
func (c *Controller) GetBy(userID int64) {
	// we have /user/{id}
	// fetch and render user json.
	if user, found := c.Source.GetByID(userID); !found {
		// not user found with that ID.
		c.renderNotFound(userID)
	} else {
		c.Ctx.JSON(user)
	}
}
