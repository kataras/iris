package main

import (
	"fmt"
	"github.com/kataras/go-websocket"
	"html/template"
	"net/http"
)

type clientPage struct {
	Title string
	Host  string
}

var host = "localhost:8080"

func main() {

	// serve our javascript files
	staticHandler := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", staticHandler))

	// or this ( import "github.com/kataras/go-fs"): it's the same thing
	// http.Handle("/static/", fs.DirHandler("./static", "/static/"))

	// Start Websocket example code

	// init our websocket
	ws := websocket.New(websocket.Config{}) // with the default configuration
	// the path which the websocket client should listen/registed to ->
	http.Handle("/my_endpoint", ws.Handler()) // See ./templates/client.html line: 21

	// serve our client-side source code go-websocket. See ./templates/client.html line: 19
	http.Handle(websocket.ClientSourcePath, websocket.ClientSourceHandler) // if you run more than one websocket servers, you don't have to call it more than once.

	ws.OnConnection(handleWebsocketConnection) // register the connection handler, which will fire on each new connected websocket client.

	// start the websocket server, you can do it whereven you want but I am choosing to make it  here
	ws.Serve()

	// End Websocket example code

	// parse our view (the template file)
	clientHTML, err := template.New("").ParseFiles("./templates/client.html")
	if err != nil {
		panic(err)
	}

	// serve our html page to /
	http.Handle("/", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" || req.URL.Path == "" {
			err := clientHTML.ExecuteTemplate(res, "client.html", clientPage{"Client Page", host})
			if err != nil {
				res.Write([]byte(err.Error()))
			}
		}
	}))

	println("Server is up & running, open two browser tabs/or/and windows and navigate to " + host)
	http.ListenAndServe(host, nil)
}

var myChatRoom = "room1"

func handleWebsocketConnection(c websocket.Connection) {

	c.Join(myChatRoom)

	c.On("chat", func(message string) {
		// to all except this connection ->
		//c.To(websocket.Broadcast).Emit("chat", "Message from: "+c.ID()+"-> "+message)

		// to the client ->
		//c.Emit("chat", "Message from myself: "+message)

		//send the message to the whole room,
		//all connections are inside this room will receive this message
		c.To(myChatRoom).Emit("chat", "From: "+c.ID()+": "+message)
	})

	c.OnDisconnect(func() {
		fmt.Printf("\nConnection with ID: %s has been disconnected!", c.ID())
	})
}
