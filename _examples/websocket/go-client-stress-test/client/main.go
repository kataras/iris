package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"
	// _ "github.com/kataras/iris/websocket2"
	"../../../../ws1m"
	//"github.com/kataras/iris/websocket"
)

var (
	url = "ws://localhost:8080/socket"
	f   *os.File
)

const totalClients = 100000

var connectionFailures uint64

var (
	disconnectErrors []error
	connectErrors    []error
	errMu            sync.Mutex
)

func collectError(op string, err error) {
	errMu.Lock()
	defer errMu.Unlock()
	switch op {
	case "disconnect":
		disconnectErrors = append(disconnectErrors, err)
	case "connect":
		connectErrors = append(connectErrors, err)
	}

}

func main() {
	var err error
	f, err = os.Open("./test.data")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	start := time.Now()

	wg := new(sync.WaitGroup)
	for i := 0; i < totalClients/4; i++ {
		wg.Add(1)
		go connect(wg, 5*time.Second)
	}

	for i := 0; i < totalClients/4; i++ {
		wg.Add(1)
		waitTime := time.Duration(rand.Intn(5)) * time.Millisecond
		time.Sleep(waitTime)
		go connect(wg, 7*time.Second+waitTime)
	}

	for i := 0; i < totalClients/4; i++ {
		wg.Add(1)
		waitTime := time.Duration(rand.Intn(10)) * time.Millisecond
		time.Sleep(waitTime)
		go connect(wg, 10*time.Second+waitTime)
	}

	for i := 0; i < totalClients/4; i++ {
		wg.Add(1)
		waitTime := time.Duration(rand.Intn(20)) * time.Millisecond
		time.Sleep(waitTime)
		go connect(wg, 25*time.Second+waitTime)
	}

	wg.Wait()
	fmt.Println("--------================--------------")
	fmt.Printf("execution time [%s]", time.Since(start))
	fmt.Println()

	if connectionFailures > 0 {
		fmt.Printf("Finished with %d/%d connection failures. Please close the server-side manually.\n", connectionFailures, totalClients)
	}

	if n := len(connectErrors); n > 0 {
		fmt.Printf("Finished with %d connect errors:\n", n)
		var lastErr error
		var sameC int

		for i, err := range connectErrors {
			if lastErr != nil {
				if lastErr.Error() == err.Error() {
					sameC++
					continue
				}
			}

			if sameC > 0 {
				fmt.Printf("and %d more like this...\n", sameC)
				sameC = 0
				continue
			}

			fmt.Printf("[%d] - %v\n", i+1, err)
			lastErr = err
		}
	}

	if n := len(disconnectErrors); n > 0 {
		fmt.Printf("Finished with %d disconnect errors\n", n)
		for i, err := range disconnectErrors {
			if err == websocket.ErrAlreadyDisconnected {
				continue
			}

			fmt.Printf("[%d] - %v\n", i+1, err)
		}
	}

	if connectionFailures == 0 && len(connectErrors) == 0 && len(disconnectErrors) == 0 {
		fmt.Println("ALL OK.")
	}

	fmt.Println("--------================--------------")
}

func connect(wg *sync.WaitGroup, alive time.Duration) {
	c, err := websocket.Dial(nil, url, websocket.ConnectionConfig{})
	if err != nil {
		atomic.AddUint64(&connectionFailures, 1)
		collectError("connect", err)
		wg.Done()
		return
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
			collectError("disconnect", err)
		}

		wg.Done()
	}()

	scanner := bufio.NewScanner(f)
	for !disconnected {
		if !scanner.Scan() || scanner.Err() != nil {
			break
		}

		if text := scanner.Text(); len(text) > 1 {
			c.Emit("chat", text)
		}
	}
}
