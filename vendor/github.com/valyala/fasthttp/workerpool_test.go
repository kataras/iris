package fasthttp

import (
	"fmt"
	"io/ioutil"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valyala/fasthttp/fasthttputil"
)

func TestWorkerPoolStartStopSerial(t *testing.T) {
	testWorkerPoolStartStop(t)
}

func TestWorkerPoolStartStopConcurrent(t *testing.T) {
	concurrency := 10
	ch := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			testWorkerPoolStartStop(t)
			ch <- struct{}{}
		}()
	}
	for i := 0; i < concurrency; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}
}

func testWorkerPoolStartStop(t *testing.T) {
	wp := &workerPool{
		WorkerFunc:      func(conn net.Conn) error { return nil },
		MaxWorkersCount: 10,
		Logger:          defaultLogger,
	}
	for i := 0; i < 10; i++ {
		wp.Start()
		wp.Stop()
	}
}

func TestWorkerPoolMaxWorkersCountSerial(t *testing.T) {
	testWorkerPoolMaxWorkersCountMulti(t)
}

func TestWorkerPoolMaxWorkersCountConcurrent(t *testing.T) {
	concurrency := 4
	ch := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			testWorkerPoolMaxWorkersCountMulti(t)
			ch <- struct{}{}
		}()
	}
	for i := 0; i < concurrency; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}
}

func testWorkerPoolMaxWorkersCountMulti(t *testing.T) {
	for i := 0; i < 5; i++ {
		testWorkerPoolMaxWorkersCount(t)
	}
}

func testWorkerPoolMaxWorkersCount(t *testing.T) {
	ready := make(chan struct{})
	wp := &workerPool{
		WorkerFunc: func(conn net.Conn) error {
			buf := make([]byte, 100)
			n, err := conn.Read(buf)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			buf = buf[:n]
			if string(buf) != "foobar" {
				t.Fatalf("unexpected data read: %q. Expecting %q", buf, "foobar")
			}
			if _, err = conn.Write([]byte("baz")); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			<-ready

			return nil
		},
		MaxWorkersCount: 10,
		Logger:          defaultLogger,
	}
	wp.Start()

	ln := fasthttputil.NewInmemoryListener()

	clientCh := make(chan struct{}, wp.MaxWorkersCount)
	for i := 0; i < wp.MaxWorkersCount; i++ {
		go func() {
			conn, err := ln.Dial()
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if _, err = conn.Write([]byte("foobar")); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			data, err := ioutil.ReadAll(conn)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if string(data) != "baz" {
				t.Fatalf("unexpected value read: %q. Expecting %q", data, "baz")
			}
			if err = conn.Close(); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			clientCh <- struct{}{}
		}()
	}

	for i := 0; i < wp.MaxWorkersCount; i++ {
		conn, err := ln.Accept()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if !wp.Serve(conn) {
			t.Fatalf("worker pool must have enough workers to serve the conn")
		}
	}

	go func() {
		if _, err := ln.Dial(); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}()
	conn, err := ln.Accept()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	for i := 0; i < 5; i++ {
		if wp.Serve(conn) {
			t.Fatalf("worker pool must be full")
		}
	}
	if err = conn.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	close(ready)

	for i := 0; i < wp.MaxWorkersCount; i++ {
		select {
		case <-clientCh:
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	wp.Stop()
}

func TestWorkerPoolPanicErrorSerial(t *testing.T) {
	testWorkerPoolPanicErrorMulti(t)
}

func TestWorkerPoolPanicErrorConcurrent(t *testing.T) {
	concurrency := 10
	ch := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			testWorkerPoolPanicErrorMulti(t)
			ch <- struct{}{}
		}()
	}
	for i := 0; i < concurrency; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}
}

func testWorkerPoolPanicErrorMulti(t *testing.T) {
	var globalCount uint64
	wp := &workerPool{
		WorkerFunc: func(conn net.Conn) error {
			count := atomic.AddUint64(&globalCount, 1)
			switch count % 3 {
			case 0:
				panic("foobar")
			case 1:
				return fmt.Errorf("fake error")
			}
			return nil
		},
		MaxWorkersCount:       1000,
		MaxIdleWorkerDuration: time.Millisecond,
		Logger:                &customLogger{},
	}

	for i := 0; i < 10; i++ {
		testWorkerPoolPanicError(t, wp)
	}
}

func testWorkerPoolPanicError(t *testing.T, wp *workerPool) {
	wp.Start()

	ln := fasthttputil.NewInmemoryListener()

	clientsCount := 10
	clientCh := make(chan struct{}, clientsCount)
	for i := 0; i < clientsCount; i++ {
		go func() {
			conn, err := ln.Dial()
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			data, err := ioutil.ReadAll(conn)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if len(data) > 0 {
				t.Fatalf("unexpected data read: %q. Expecting empty data", data)
			}
			if err = conn.Close(); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			clientCh <- struct{}{}
		}()
	}

	for i := 0; i < clientsCount; i++ {
		conn, err := ln.Accept()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if !wp.Serve(conn) {
			t.Fatalf("worker pool mustn't be full")
		}
	}

	for i := 0; i < clientsCount; i++ {
		select {
		case <-clientCh:
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	wp.Stop()
}
