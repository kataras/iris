package main

import (
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
)

func main() {

	var store = sessions.NewCookieStore([]byte("myIrisSecretKey"))
	var sessionName = "user_sessions"

	//iris.Use(sessions.New(sessionName, store))
	iris.Get("/home", func(c *iris.Context) {
		c.Write("test home")
	})
	iris.Get("/set", func(c *iris.Context) {
		//session := sessions.GetSession(c, sessionName)
		sessions.Set(c.Request, sessionName, sessions.NewSession(store, sessionName))
		session := sessions.Get(c.Request, sessionName)
		session.(*sessions.Session).Set("foo", "bar")
		store.Save(c.Request, c.ResponseWriter, session.(*sessions.Session))
		fmt.Printf("\n%T Point to: %v:", session, session)
		c.Write("dsa")
		//session.Set("foo", "bar")
		//c.Write("foo setted to: " + session.GetString("foo"))
	})

	iris.Get("/get", func(c *iris.Context) {
		//		var foo string = " no session key given "
		//session := sessions.GetSession(c, sessionName)
		v, _ := store.Get(c.Request, "foo")
		fmt.Printf("FROM STORE GET: %T Point to: %v\n", v, v)
		session := sessions.Get(c.Request, sessionName)
		session.(*sessions.Session).Get("foo")
		fmt.Printf("\n%T Point to: %v:", session, session)
		c.Write("dsa")
		//	if session != nil {
		//	println("session no nil")
		//	foo = session.GetString("foo")
		//}
		//c.Write(foo)
	})

	iris.Get("/clear", func(c *iris.Context) {
		session := sessions.GetSession(c, sessionName)
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
