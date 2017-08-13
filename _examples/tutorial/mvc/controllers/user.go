package controllers

import (
	"time"

	"github.com/kataras/iris/_examples/tutorial/mvc/persistence"
)

// User is our user example controller.
type User struct {
	Controller

	// all fields with pointers(*)
	// that are not nil
	// and all fields with
	// that are tagged with iris:"persistence"`
	// are being persistence and kept
	// between the requests, meaning that
	// they will not be reset-ed on each new request,
	// they will be the same for all requests.
	CreatedAt time.Time `iris:"persistence"`
	Title     string    `iris:"persistence"`
	DB        *persistence.Database
}

func NewUserController(db *persistence.Database) *User {
	return &User{
		CreatedAt: time.Now(),
		Title:     "User page",
		DB:        db,
	}
}

// Get serves using the User controller when HTTP Method is "GET".
func (c *User) Get() {
	c.Tmpl = "user/index.html"
	c.Data["title"] = c.Title
	c.Data["username"] = "kataras " + c.Params.Get("userid")
	c.Data["connstring"] = c.DB.Connstring
	c.Data["uptime"] = time.Now().Sub(c.CreatedAt).Seconds()
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
