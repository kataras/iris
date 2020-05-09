package main

// developers can use any library to add a custom cookie encoder/decoder.
// At this example we use the gorilla's securecookie package:
// $ go get github.com/gorilla/securecookie
// $ go run main.go

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/sessions"

	"github.com/kataras/iris/v12/_examples/sessions/overview/example"

	"github.com/gorilla/securecookie"
)

func newApp() *iris.Application {
	app := iris.New()

	cookieName := "_session_id"
	// AES only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	hashKey := securecookie.GenerateRandomKey(64)
	blockKey := securecookie.GenerateRandomKey(32)
	s := securecookie.New(hashKey, blockKey)

	mySessions := sessions.New(sessions.Config{
		Cookie:       cookieName,
		Encoding:     s,
		AllowReclaim: true,
	})

	app = example.NewApp(mySessions)
	return app
}

func main() {
	app := newApp()
	app.Listen(":8080")
}
