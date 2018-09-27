package main

import (
	"github.com/kataras/iris"

	"github.com/googollee/go-socket.io"
)

/*
	go get -u github.com/googollee/go-socket.io
*/

func main() {
	app := iris.New()
	server, err := socketio.NewServer(nil)
	if err != nil {
		app.Logger().Fatal(err)
	}

	server.On("connection", func(so socketio.Socket) {
		app.Logger().Infof("on connection")
		so.Join("chat")
		so.On("chat message", func(msg string) {
			app.Logger().Infof("emit: %v", so.Emit("chat message", msg))
			so.BroadcastTo("chat", "chat message", msg)
		})
		so.On("disconnection", func() {
			app.Logger().Infof("on disconnect")
		})
	})

	server.On("error", func(so socketio.Socket, err error) {
		app.Logger().Errorf("error: %v", err)
	})

	// serve the socket.io endpoint.
	app.Any("/socket.io/{p:path}", iris.FromStd(server))

	// serve the index.html and the javascript libraries at
	// http://localhost:8080
	app.StaticWeb("/", "./public")

	app.Run(iris.Addr("localhost:8080"), iris.WithoutPathCorrection)
}
