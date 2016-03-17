> This package is converted to work with Iris but it was originaly created by Gorila team, the original package is gorilla/sessions

## Features

The key features are:

    Simple API: use it as an easy way to set signed (and optionally encrypted) cookies.
    Built-in backends to store sessions in cookies or the filesystem.
    Flash messages: session values that last until read.
    Convenient way to switch session persistency (aka "remember me") and set other attributes.
    Mechanism to rotate authentication and encryption keys.
    Multiple sessions per request, even using different backends.
    Interfaces and infrastructure for custom session backends: sessions from different stores can be 	retrieved and batch-saved using a common API.

    
## Usage

```go

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
		session, err := mySessions.Get(c)

		if err != nil {
			c.SendStatus(500, err.Error())
			return
		}
		//set session values
		session.Set("name", "kataras")

		//save them
		session.Save(c)

		//write anthing
		c.Write("All ok session setted to: ", session.Get("name"))
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

		c.Write("The name on the /set was: ", name)
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



```

