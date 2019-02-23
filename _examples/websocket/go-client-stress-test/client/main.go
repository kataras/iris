package main

import (
	"bufio"
	"context"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/websocket"
)

var (
	url = "ws://localhost:8080"
	f   *os.File
)

const totalClients = 16000 // max depends on the OS.
const verbose = false

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
	log.Println("-- Running...")
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
		go connect(wg, 14*time.Second+waitTime)
	}

	for i := 0; i < totalClients/4; i++ {
		wg.Add(1)
		waitTime := time.Duration(rand.Intn(10)) * time.Millisecond
		time.Sleep(waitTime)
		go connect(wg, 9*time.Second+waitTime)
	}

	for i := 0; i < totalClients/4; i++ {
		wg.Add(1)
		waitTime := time.Duration(rand.Intn(5)) * time.Millisecond
		time.Sleep(waitTime)
		go connect(wg, 7*time.Second+waitTime)
	}

	wg.Wait()

	log.Printf("execution time [%s]", time.Since(start))
	log.Println()

	if connectionFailures > 0 {
		log.Printf("Finished with %d/%d connection failures.", connectionFailures, totalClients)
	}

	if n := len(connectErrors); n > 0 {
		log.Printf("Finished with %d connect errors: ", n)
		var lastErr error
		var sameC int

		for i, err := range connectErrors {
			if lastErr != nil {
				if lastErr.Error() == err.Error() {
					sameC++
					continue
				} else {
					_, ok := lastErr.(*net.OpError)
					if ok {
						if _, ok = err.(*net.OpError); ok {
							sameC++
							continue
						}
					}
				}
			}

			if sameC > 0 {
				log.Printf("and %d more like this...\n", sameC)
				sameC = 0
				continue
			}

			log.Printf("[%d] - %v\n", i+1, err)
			lastErr = err
		}
	}

	if n := len(disconnectErrors); n > 0 {
		log.Printf("Finished with %d disconnect errors\n", n)
		for i, err := range disconnectErrors {
			if err == websocket.ErrAlreadyDisconnected && i > 0 {
				continue
			}

			log.Printf("[%d] - %v\n", i+1, err)
		}
	}

	if connectionFailures == 0 && len(connectErrors) == 0 && len(disconnectErrors) == 0 {
		log.Println("ALL OK.")
	}

	log.Println("-- Finished.")
}

func connect(wg *sync.WaitGroup, alive time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), alive)
	defer cancel()

	c, err := websocket.Dial(ctx, url, websocket.ConnectionConfig{})
	if err != nil {
		atomic.AddUint64(&connectionFailures, 1)
		collectError("connect", err)
		wg.Done()
		return err
	}

	c.OnError(func(err error) {
		log.Printf("error: %v", err)
	})

	disconnected := false
	c.OnDisconnect(func() {
		if verbose {
			log.Printf("I am disconnected after [%s].\n", alive)
		}

		disconnected = true
	})

	c.On("chat", func(message string) {
		if verbose {
			log.Printf("\n%s\n", message)
		}
	})

	if alive > 0 {
		go func() {
			time.Sleep(alive)
			if err := c.Disconnect(); err != nil {
				collectError("disconnect", err)
			}

			wg.Done()
		}()

	}

	scanner := bufio.NewScanner(f)
	for !disconnected {
		if !scanner.Scan() {
			return scanner.Err()
		}

		if text := scanner.Text(); len(text) > 1 {
			c.Emit("chat", text)
		}
	}

	return nil
}
