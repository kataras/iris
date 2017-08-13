// +build go1.9

package controllers

import (
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
)

// User is our user example controller.
type User struct {
	iris.Controller

	// All fields with pointers(*) that are not nil
	// and all fields that are tagged with iris:"persistence"`
	// are being persistence and kept between the different requests,
	// meaning that these data will not be reset-ed on each new request,
	// they will be the same for all requests.
	CreatedAt      time.Time          `iris:"persistence"`
	Title          string             `iris:"persistence"`
	SessionManager *sessions.Sessions `iris:"persistence"`

	Session *sessions.Session // not persistence
}

func NewUserController(sess *sessions.Sessions) *User {
	return &User{
		SessionManager: sess,
		CreatedAt:      time.Now(),
		Title:          "User page",
	}
}

// Init can be used as a custom function
// to init the new instance of controller
// that is created on each new request.
//
// Useful when more than one methods are using the same
// request data.
func (c *User) Init(ctx iris.Context) {
	c.Session = c.SessionManager.Start(ctx)
	// println("session id: " + c.Session.ID())
}

// Get serves using the User controller when HTTP Method is "GET".
func (c *User) Get() {
	c.Tmpl = "user/index.html"
	c.Data["title"] = c.Title
	c.Data["username"] = "kataras " + c.Params.Get("userid")
	c.Data["uptime"] = time.Now().Sub(c.CreatedAt).Seconds()

	visits, err := c.Session.GetInt("visit_count")
	if err != nil {
		visits = 0
	}
	visits++
	c.Session.Set("visit_count", visits)
	c.Data["visit_count"] = visits
}

/* Can use more than one, the factory will make sure
that the correct http methods are being registed  for this
controller, uncommend these if you want:

func (c *User) Post() {}
func (c *User) Put() {}
func (c *User) Delete() {}
func (c *User) Connect() {}
func (c *User) Head() {}
func (c *User) Patch() {}
func (c *User) Options() {}
func (c *User) Trace() {}
*/

/*
func (c *User) All() {}
//        OR
func (c *User) Any() {}
*/
