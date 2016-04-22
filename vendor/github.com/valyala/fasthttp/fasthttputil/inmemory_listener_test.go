package fasthttputil

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestInmemoryListener(t *testing.T) {
	ln := NewInmemoryListener()

	ch := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			conn, err := ln.Dial()
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			defer conn.Close()
			req := fmt.Sprintf("request_%d", n)
			nn, err := conn.Write([]byte(req))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if nn != len(req) {
				t.Fatalf("unexpected number of bytes written: %d. Expecting %d", nn, len(req))
			}
			buf := make([]byte, 30)
			nn, err = conn.Read(buf)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			buf = buf[:nn]
			resp := fmt.Sprintf("response_%d", n)
			if nn != len(resp) {
				t.Fatalf("unexpected number of bytes read: %d. Expecting %d", nn, len(resp))
			}
			if string(buf) != resp {
				t.Fatalf("unexpected response %q. Expecting %q", buf, resp)
			}
			ch <- struct{}{}
		}(i)
	}

	serverCh := make(chan struct{})
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				close(serverCh)
				return
			}
			defer conn.Close()
			buf := make([]byte, 30)
			n, err := conn.Read(buf)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			buf = buf[:n]
			if !bytes.HasPrefix(buf, []byte("request_")) {
				t.Fatalf("unexpected request prefix %q. Expecting %q", buf, "request_")
			}
			resp := fmt.Sprintf("response_%s", buf[len("request_"):])
			n, err = conn.Write([]byte(resp))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if n != len(resp) {
				t.Fatalf("unexpected number of bytes written: %d. Expecting %d", n, len(resp))
			}
		}
	}()

	for i := 0; i < 10; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	select {
	case <-serverCh:
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}
