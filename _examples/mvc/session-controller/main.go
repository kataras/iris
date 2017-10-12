// +build go1.9

package main

import (
	"fmt"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
)

// VisitController handles the root route.
type VisitController struct {
	iris.C

	// the sessions manager, we need that to set `Session`.
	// It's binded from `app.Controller`.
	Manager *sessions.Sessions
	// the current request session,
	// its initialization happens at the `BeginRequest`.
	Session *sessions.Session

	// A time.time which is binded from the `app.Controller`,
	// order of binded fields doesn't matter.
	StartTime time.Time
}

// BeginRequest is executed for each Get, Post, Put requests,
// can be used to share context, common data
// or to cancel the request via `ctx.StopExecution()`.
func (c *VisitController) BeginRequest(ctx iris.Context) {
	// always call the embedded `BeginRequest` before everything else.
	c.C.BeginRequest(ctx)

	if c.Manager == nil {
		ctx.Application().Logger().Errorf(`VisitController: sessions manager is nil, you should bind it`)
		// dont run the main method handler and any "done" handlers.
		ctx.StopExecution()
		return
	}

	// set the `c.Session` we will use that in our Get method.
	c.Session = c.Manager.Start(ctx)
}

// Get handles
// Method: GET
// Path: http://localhost:8080
func (c *VisitController) Get() string {
	// get the visits, before calcuate this new one.
	visits, _ := c.Session.GetIntDefault("visits", 0)

	// increment the visits and store to the session.
	visits++
	c.Session.Set("visits", visits)

	// write the current, updated visits.
	since := time.Now().Sub(c.StartTime).Seconds()
	return fmt.Sprintf("%d visit from my current session in %0.1f seconds of server's up-time",
		visits, since)
}

var (
	manager = sessions.New(sessions.Config{Cookie: "mysession_cookie_name"})
)

func main() {
	app := iris.New()

	// bind our session manager, which is required, to the `VisitController.Manager`
	// and the time.Now() to the `VisitController.StartTime`.
	app.Controller("/", new(VisitController),
		manager,
		time.Now())

	// 1. open the browser (no in private mode)
	// 2. navigate to http://localhost:8080
	// 3. refresh the page some times
	// 4. close the browser
	// 5. re-open the browser and re-play 2.
	app.Run(iris.Addr(":8080"), iris.WithoutVersionChecker)
}
