package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/kataras/iris/websocket"
)

const (
	url    = "ws://localhost:8080/socket"
	prompt = ">> "
)

/*
How to run:
Start the server, if it is not already started by executing `go run ../server/main.go`
And open two or more terminal windows and start the clients:
$ go run main.go
>> hi!
*/
func main() {
	c, err := websocket.Dial(nil, url, websocket.ConnectionConfig{})
	if err != nil {
		panic(err)
	}

	c.OnError(func(err error) {
		fmt.Printf("error: %v", err)
	})

	c.OnDisconnect(func() {
		fmt.Println("Server was force-closed[see ../server/main.go#L17] this connection after 20 seconds, therefore I am disconnected.")
		os.Exit(0)
	})

	c.On("chat", func(message string) {
		fmt.Printf("\n%s\n", message)
	})

	fmt.Println("Start by typing a message to send")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(prompt)
		if !scanner.Scan() || scanner.Err() != nil {
			break
		}
		msgToSend := scanner.Text()
		if msgToSend == "exit" {
			break
		}

		c.Emit("chat", msgToSend)
	}

	fmt.Println("Terminated.")
}
