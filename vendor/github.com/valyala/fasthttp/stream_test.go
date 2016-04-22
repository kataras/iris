package fasthttp

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"testing"
	"time"
)

func TestNewStreamReader(t *testing.T) {
	ch := make(chan struct{})
	r := NewStreamReader(func(w *bufio.Writer) {
		fmt.Fprintf(w, "Hello, world\n")
		fmt.Fprintf(w, "Line #2\n")
		close(ch)
	})

	data, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	expectedData := "Hello, world\nLine #2\n"
	if string(data) != expectedData {
		t.Fatalf("unexpected data %q. Expecting %q", data, expectedData)
	}

	if err = r.Close(); err != nil {
		t.Fatalf("unexpected error")
	}

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}

func TestStreamReaderClose(t *testing.T) {
	firstLine := "the first line must pass"
	ch := make(chan struct{})
	r := NewStreamReader(func(w *bufio.Writer) {
		fmt.Fprintf(w, "%s", firstLine)
		if err := w.Flush(); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		fmt.Fprintf(w, "the second line must fail")
		if err := w.Flush(); err == nil {
			t.Fatalf("expecting error")
		}
		close(ch)
	})

	result := firstLine + "the"
	buf := make([]byte, len(result))
	n, err := io.ReadFull(r, buf)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if n != len(buf) {
		t.Fatalf("unexpected number of bytes read: %d. Expecting %d", n, len(buf))
	}
	if string(buf) != result {
		t.Fatalf("unexpected result: %q. Expecting %q", buf, result)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}
