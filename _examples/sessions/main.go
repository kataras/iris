package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
)

//there is no middleware, use the sessions anywhere you want
func main() {

	var store = sessions.NewCookieStore([]byte("myIrisSecretKey"))
	var mySessions = sessions.New("user_sessions", store)

	iris.Get("/set", func(c *iris.Context) {
		//get the session for this context
		session, err := mySessions.Get(c) // or .GetSession(c), it's the same

		if err != nil {
			c.SendStatus(500, err.Error())
			return
		}
		//set session values
		session.Set("name", "kataras")

		//save them
		session.Save(c)

		//write anthing
		c.Write("All ok session setted to: %s", session.Get("name"))
	})

	iris.Get("/get", func(c *iris.Context) {
		//again get the session for this context
		session, err := mySessions.Get(c)

		if err != nil {
			c.SendStatus(500, err.Error())
			return
		}
		//get the session value
		name := session.GetString("name") // .Get or .GetInt

		c.Write("The name on the /set was: %s", name)
	})

	iris.Get("/clear", func(c *iris.Context) {
		session, err := mySessions.Get(c)
		if err != nil {
			c.SendStatus(500, err.Error())
			return
		}
		//Clear clears all
		//session.Clear()
		session.Delete("name")

	})

	// Use global sessions.Clear() to clear ALL sessions and stores if it's necessary
	//sessions.Clear()

	println("Iris is listening on :8080")
	iris.Listen("8080")

}
