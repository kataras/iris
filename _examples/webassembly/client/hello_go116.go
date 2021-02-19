// +build js

package main

import (
	"fmt"
	"syscall/js"
	"time"
)

func main() {
	// GOARCH=wasm GOOS=js /home/$yourusername/go1.16/bin/go build -mod=mod -o hello.wasm hello_go116.go
	js.Global().Get("console").Call("log", "Hello Webassembly!")
	message := fmt.Sprintf("Hello, the current time is: %s", time.Now().String())
	js.Global().Get("document").Call("getElementById", "hello").Set("innerText", message)
}
