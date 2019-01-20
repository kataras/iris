// +build go1.9

package main

import (
	"fmt"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"github.com/kataras/iris/sessions"
)

// VisitController handles the root route.
type VisitController struct {
	// the current request session,
	// its initialization happens by the dependency function that we've added to the `visitApp`.
	Session *sessions.Session

	// A time.time which is binded from the MVC,
	// order of binded fields doesn't matter.
	StartTime time.Time
}

// Get handles
// Method: GET
// Path: http://localhost:8080
func (c *VisitController) Get() string {
	// it increments a "visits" value of integer by one,
	// if the entry with key 'visits' doesn't exist it will create it for you.
	visits := c.Session.Increment("visits", 1)
	// write the current, updated visits.
	since := time.Now().Sub(c.StartTime).Seconds()
	return fmt.Sprintf("%d visit(s) from my current session in %0.1f seconds of server's up-time",
		visits, since)
}

func newApp() *iris.Application {
	app := iris.New()
	sess := sessions.New(sessions.Config{Cookie: "mysession_cookie_name"})

	visitApp := mvc.New(app.Party("/"))
	// bind the current *session.Session, which is required, to the `VisitController.Session`
	// and the time.Now() to the `VisitController.StartTime`.
	visitApp.Register(
		// if dependency is a function which accepts
		// a Context and returns a single value
		// then the result type of this function is resolved by the controller
		// and on each request it will call the function with its Context
		// and set the result(the *sessions.Session here) to the controller's field.
		//
		// If dependencies are registered without field or function's input arguments as
		// consumers then those dependencies are being ignored before the server ran,
		// so you can bind many dependecies and use them in different controllers.
		sess.Start,
		time.Now(),
	)
	visitApp.Handle(new(VisitController))

	return app
}

func main() {
	app := newApp()

	// 1. open the browser (no in private mode)
	// 2. navigate to http://localhost:8080
	// 3. refresh the page some times
	// 4. close the browser
	// 5. re-open the browser and re-play 2.
	app.Run(iris.Addr(":8080"))
}
