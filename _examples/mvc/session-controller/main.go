package main

import (
	"time"

	"github.com/kataras/iris"

	"github.com/kataras/iris/sessions"
)

type VisitController struct {
	iris.SessionController

	StartTime time.Time
}

func (u *VisitController) Get() {
	// get the visits, before calcuate this new one.
	visits, _ := u.Session.GetIntDefault("visits", 0)

	// increment the visits counter and set them to the session.
	visits++
	u.Session.Set("visits", visits)

	// write the current, updated visits
	u.Ctx.Writef("%d visits in %0.1f seconds", visits, time.Now().Sub(u.StartTime).Seconds())
}

func main() {
	mySessionManager := sessions.New(sessions.Config{Cookie: "mysession_cookie_name"})

	app := iris.New()

	// bind our session manager, which is required, to the `VisitController.SessionManager.Manager`
	// and the time.Now() to the `VisitController.StartTime`.
	app.Controller("/", new(VisitController), mySessionManager, time.Now())

	// 1. open the browser (no in private mode)
	// 2. navigate to http://localhost:8080
	// 3. refresh the page some times
	// 4. close the browser
	// 5. re-open the browser and re-play 2.
	app.Run(iris.Addr(":8080"), iris.WithoutVersionChecker)
}
