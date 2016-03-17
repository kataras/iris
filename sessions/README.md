> This package is converted to work with Iris but it was originaly created by Gorila team, the original package is gorilla/sessions

## Warning

Sessions are not working currently, I am trying to find a way to make them ligher than it's.

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



```

