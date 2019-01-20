package main

// developers can use any library to add a custom cookie encoder/decoder.
// At this example we use the gorilla's securecookie package:
// $ go get github.com/gorilla/securecookie
// $ go run main.go

import (
	"github.com/kataras/iris"

	"github.com/kataras/iris/sessions"

	"github.com/gorilla/securecookie"
)

func newApp() *iris.Application {
	app := iris.New()

	cookieName := "mycustomsessionid"
	// AES only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	hashKey := []byte("the-big-and-secret-fash-key-here")
	blockKey := []byte("lot-secret-of-characters-big-too")
	secureCookie := securecookie.New(hashKey, blockKey)

	mySessions := sessions.New(sessions.Config{
		Cookie:       cookieName,
		Encode:       secureCookie.Encode,
		Decode:       secureCookie.Decode,
		AllowReclaim: true,
	})

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("You should navigate to the /set, /get, /delete, /clear,/destroy instead")
	})
	app.Get("/set", func(ctx iris.Context) {

		//set session values
		s := mySessions.Start(ctx)
		s.Set("name", "iris")

		//test if setted here
		ctx.Writef("All ok session setted to: %s", s.GetString("name"))
	})

	app.Get("/get", func(ctx iris.Context) {
		// get a specific key, as string, if no found returns just an empty string
		s := mySessions.Start(ctx)
		name := s.GetString("name")

		ctx.Writef("The name on the /set was: %s", name)
	})

	app.Get("/delete", func(ctx iris.Context) {
		// delete a specific key
		s := mySessions.Start(ctx)
		s.Delete("name")
	})

	app.Get("/clear", func(ctx iris.Context) {
		// removes all entries
		mySessions.Start(ctx).Clear()
	})

	app.Get("/update", func(ctx iris.Context) {
		// updates expire date with a new date
		mySessions.ShiftExpiration(ctx)
	})

	app.Get("/destroy", func(ctx iris.Context) {
		//destroy, removes the entire session data and cookie
		mySessions.Destroy(ctx)
	})
	// Note about destroy:
	//
	// You can destroy a session outside of a handler too, using the:
	// mySessions.DestroyByID
	// mySessions.DestroyAll

	return app
}

func main() {
	app := newApp()
	app.Run(iris.Addr(":8080"))
}
