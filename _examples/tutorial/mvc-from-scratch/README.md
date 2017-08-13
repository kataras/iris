# Controllers from scratch

This example folder shows how I started to develop
the Controller idea inside the Iris web framework itself.

Now it's built'n feature and can be used as:

```go
// +build go1.9

// file main.go
package main

import (
    "github.com/kataras/iris/_examples/routing/mvc/persistence"

    "github.com/kataras/iris"
)

func main() {
    app := iris.New()
    app.RegisterView(iris.HTML("./views", ".html"))

    db := persistence.OpenDatabase("a fake db")

    app.Controller("/user/{userid:int}", NewUserController(db))

	// http://localhost:8080/
    // http://localhost:8080/user/42
    app.Run(iris.Addr(":8080"))
}
```

```go
// +build go1.9

// file user_controller.go
package main

import (
    "time"

    "github.com/kataras/iris/_examples/routing/mvc/persistence"

    "github.com/kataras/iris"
)

// User is our user example controller.
type UserController struct {
    iris.Controller

	// All fields that are tagged with iris:"persistence"`
	// are being persistence and kept between the different requests,
	// meaning that these data will not be reset-ed on each new request,
	// they will be the same for all requests.
    CreatedAt time.Time             `iris:"persistence"`
    Title     string                `iris:"persistence"`
    DB        *persistence.Database `iris:"persistence"`
}

func NewUserController(db *persistence.Database) *User {
    return &UserController{
        CreatedAt: time.Now(),
        Title:     "User page",
        DB:        db,
    }
}

// Get serves using the User controller when HTTP Method is "GET".
func (c *UserController) Get() {
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
```

Example can be found at: [_examples/routing/mvc](https://github.com/kataras/iris/tree/master/_examples/routing/mvc).