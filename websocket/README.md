## Websocket 


### Package information

This package is (almost) a copy of the official go support for websockets [golang.org/x/net/websocket](golang.org/x/net/websocket), copyrights goes to the Go Authors.

>Additions in order make fully compatible with Iris' Context type out of the box without any unnecessary conversations from the users of this package with Iris.

-----------------------------------

### How to use

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/websocket"
	"io"
)

func echoHandler(ws *websocket.Conn) {
	io.Copy(ws, ws)
}

func main() {
	iris.Handle("/echo", websocket.Handler(echoHandler))
	iris.Get("/*files", iris.Static("."))
	err := iris.Listen(":8080")
	if err != nil {
		panic("Iris Listen: " + err.Error())
	}
}


```