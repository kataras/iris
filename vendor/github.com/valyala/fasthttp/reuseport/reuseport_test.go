package reuseport

import (
	"fmt"
	"io/ioutil"
	"net"
	"testing"
	"time"
)

func TestNewListener(t *testing.T) {
	addr := "localhost:10081"
	serversCount := 20
	requestsCount := 1000

	var lns []net.Listener
	doneCh := make(chan struct{}, serversCount)

	for i := 0; i < serversCount; i++ {
		ln, err := Listen("tcp4", addr)
		if err != nil {
			t.Fatalf("cannot create listener %d: %s", i, err)
		}
		go func() {
			serveEcho(t, ln)
			doneCh <- struct{}{}
		}()
		lns = append(lns, ln)
	}

	for i := 0; i < requestsCount; i++ {
		c, err := net.Dial("tcp4", addr)
		if err != nil {
			t.Fatalf("%d. unexpected error when dialing: %s", i, err)
		}
		req := fmt.Sprintf("request number %d", i)
		if _, err = c.Write([]byte(req)); err != nil {
			t.Fatalf("%d. unexpected error when writing request: %s", i, err)
		}
		if err = c.(*net.TCPConn).CloseWrite(); err != nil {
			t.Fatalf("%d. unexpected error when closing write end of the connection: %s", i, err)
		}

		var resp []byte
		ch := make(chan struct{})
		go func() {
			if resp, err = ioutil.ReadAll(c); err != nil {
				t.Fatalf("%d. unexpected error when reading response: %s", i, err)
			}
			close(ch)
		}()
		select {
		case <-ch:
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("%d. timeout when waiting for response: %s", i, err)
		}

		if string(resp) != req {
			t.Fatalf("%d. unexpected response %q. Expecting %q", i, resp, req)
		}
		if err = c.Close(); err != nil {
			t.Fatalf("%d. unexpected error when closing connection: %s", i, err)
		}
	}

	for _, ln := range lns {
		if err := ln.Close(); err != nil {
			t.Fatalf("unexpected error when closing listener: %s", err)
		}
	}

	for i := 0; i < serversCount; i++ {
		select {
		case <-doneCh:
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("timeout when waiting for servers to be closed")
		}
	}
}

func serveEcho(t *testing.T, ln net.Listener) {
	for {
		c, err := ln.Accept()
		if err != nil {
			break
		}
		req, err := ioutil.ReadAll(c)
		if err != nil {
			t.Fatalf("unepxected error when reading request: %s", err)
		}
		if _, err = c.Write(req); err != nil {
			t.Fatalf("unexpected error when writing response: %s", err)
		}
		if err = c.Close(); err != nil {
			t.Fatalf("unexpected error when closing connection: %s", err)
		}
	}
}
