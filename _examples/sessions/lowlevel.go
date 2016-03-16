package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
)

func main() {
	store := sessions.NewCookieStore([]byte("myIrisSecretKey"))
	//iris.Use(sessions.Session("my_session", store))

	iris.UseFunc(func(c *iris.Context) {
		// Get a session. We're ignoring the error resulted from decoding an
		// existing session: Get() always returns a session, even if empty.

		session, _ := store.Get(c.Request, "my_session")
		// Set some session values.
		session.Values["foo"] = "bar"
		session.Values[42] = 2032
		// Save it before we write to the response/return from the handler.
		session.Save(c.Request, c.ResponseWriter)
		c.Next()
	})

	iris.Get("/home", func(c *iris.Context) {
		session, _ := store.Get(c.Request, "my_session")
		c.Write(session.Values["foo"].(string))
	})

	println("Iris is listening on :8080")
	iris.Listen("8080")
}
