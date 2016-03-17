> This package is converted to work with Iris but it was originaly created by Gorila team, the original package is gorilla/sessions


## Features

The key features are:

    Simple API: use it as an easy way to set signed (and optionally encrypted) cookies.
    Built-in backends to store sessions in cookies or the filesystem.
    Flash messages: session values that last until read.
    Convenient way to switch session persistency (aka "remember me") and set other attributes.
    Mechanism to rotate authentication and encryption keys.
    Multiple sessions per request, even using different backends.
    Interfaces and infrastructure for custom session backends: sessions from different stores can be retrieved and batch-saved using a common API.

    
## Low-Level usage

```go

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


```

