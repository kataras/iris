package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
)

func main() {

	var store = sessions.NewCookieStore([]byte("myIrisSecretKey"))
	var sessionName = "user_sessions"

	iris.Use(sessions.New(sessionName, store))

	iris.Get("/set", func(c *iris.Context) {
		session := sessions.GetSession(sessionName)
		session.Set("foo", "bar")
		c.Write("foo setted to: " + session.GetString("foo"))
	})

	iris.Get("/get", func(c *iris.Context) {
		var foo string = " no session key given "
		session := sessions.GetSession(sessionName)
		if session != nil {
			foo = session.GetString("foo")
		}
		c.Write(foo)
	})

	iris.Get("/clear", func(c *iris.Context) {
		session := sessions.GetSession(sessionName)
		if session != nil {
			//Clear clears all
			//session.Clear()
			session.Delete("foo")
		}
	})

	// Use global sessions.Clear() to clear ALL sessions and stores if it's necessary
	//sessions.Clear()

	println("Iris is listening on :8080")
	iris.Listen("8080")
}
