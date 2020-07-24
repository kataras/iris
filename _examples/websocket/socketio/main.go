// Package main runs a go-socket.io based websocket server.
// An Iris compatible clone of: https://github.com/googollee/go-socket.io#example,
// use of `iris.FromStd` to convert its handler.
package main

import (
	"fmt"
	"log"

	socketio "github.com/googollee/go-socket.io"
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())
		return nil
	})
	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		fmt.Println("notice:", msg)
		s.Emit("reply", "have "+msg)
	})
	server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
		s.SetContext(msg)
		return "recv " + msg
	})
	server.OnEvent("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		s.Close()
		return last
	})
	server.OnError("/", func(s socketio.Conn, e error) {
		fmt.Println("meet error:", e)
	})
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Println("closed", reason)
	})
	go server.Serve()
	defer server.Close()

	app.HandleMany("GET POST", "/socket.io/{any:path}", iris.FromStd(server))
	app.HandleDir("/", iris.Dir("./asset"))

	app.Listen(":8000", iris.WithoutPathCorrection)
}

/*
If you want to enable CORS in your websocket handler,
please follow this post: https://github.com/googollee/go-socket.io/issues/242
*/
