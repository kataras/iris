package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/kataras/iris/websocket"
)

var (
	url = "ws://localhost:8080/socket"
	f   *os.File
)

const totalClients = 1200

func main() {
	var err error
	f, err = os.Open("./test.data")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	wg := new(sync.WaitGroup)
	for i := 0; i < totalClients/2; i++ {
		wg.Add(1)
		go connect(wg, 5*time.Second)
	}

	for i := 0; i < totalClients/2; i++ {
		wg.Add(1)
		waitTime := time.Duration(rand.Intn(10)) * time.Millisecond
		time.Sleep(waitTime)
		go connect(wg, 10*time.Second+waitTime)
	}

	wg.Wait()
	fmt.Println("ALL OK.")
	time.Sleep(5 * time.Second)
}

func connect(wg *sync.WaitGroup, alive time.Duration) {

	c, err := websocket.Dial(url, websocket.ConnectionConfig{})
	if err != nil {
		panic(err)
	}

	c.OnError(func(err error) {
		fmt.Printf("error: %v", err)
	})

	disconnected := false
	c.OnDisconnect(func() {
		fmt.Printf("I am disconnected after [%s].\n", alive)
		disconnected = true
	})

	c.On("chat", func(message string) {
		fmt.Printf("\n%s\n", message)
	})

	go func() {
		time.Sleep(alive)
		if err := c.Disconnect(); err != nil {
			panic(err)
		}

		wg.Done()
	}()

	scanner := bufio.NewScanner(f)
	for !disconnected {
		if !scanner.Scan() || scanner.Err() != nil {
			break
		}

		c.Emit("chat", scanner.Text())
	}
}
