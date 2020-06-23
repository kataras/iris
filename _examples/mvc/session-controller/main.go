package main

import (
	"fmt"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/kataras/iris/v12/sessions"
)

// VisitController handles the root route.
type VisitController struct {
	// the current request session, automatically binded.
	Session *sessions.Session

	// A time.time which is binded from the MVC application manually.
	StartTime time.Time
}

// Get handles index
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
	// Configure sessions manager as we used to.
	sess := sessions.New(sessions.Config{Cookie: "mysession_cookie_name"})
	app.Use(sess.Handler())

	visitApp := mvc.New(app)
	visitApp.Register(time.Now())
	// The `VisitController.Session` is automatically binded to the current `sessions.Session`.

	visitApp.Handle(new(VisitController))

	return app
}

func main() {
	app := newApp()

	// 1. Prepare a client, e.g. your browser
	// 2. navigate to http://localhost:8080
	// 3. refresh the page some times
	// 4. close the browser
	// 5. re-open the browser (if it wasn't in private mode) and re-play 2.
	app.Listen(":8080")
}
