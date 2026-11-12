package recorder_test

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/middleware/recorder"
)

// TestRecorderDoesNotBufferStreams is the regression test for the bug that a
// recorder installed globally - by accesslog, httpcache or servertiming - turned
// every streaming response into a single lump delivered when the handler
// returned.
//
// It drives a real server rather than httptest.New, because the whole question
// is whether bytes reach the socket before the handler returns, and an
// in-process recorder cannot observe that.
func TestRecorderDoesNotBufferStreams(t *testing.T) {
	release := make(chan struct{})

	app := iris.New()
	app.Use(recorder.Response) // the middleware accesslog installs for you.
	app.Get("/stream", func(ctx iris.Context) {
		ctx.SetContentType("text/event-stream")

		ctx.Writef("data: first\n\n")
		ctx.ResponseWriter().Flush()

		<-release // the handler is still running; the client must already have the event.

		ctx.Writef("data: second\n\n")
		ctx.ResponseWriter().Flush()
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(app)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	type read struct {
		line string
		err  error
	}
	lines := make(chan read, 1)
	br := bufio.NewReader(resp.Body)
	go func() {
		l, err := br.ReadString('\n')
		lines <- read{l, err}
	}()

	select {
	case got := <-lines:
		if got.err != nil {
			t.Fatalf("reading first event: %v", got.err)
		}
		if !strings.Contains(got.line, "first") {
			t.Fatalf("first event = %q, want it to contain \"first\"", got.line)
		}
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("the first event never arrived: the recorder buffered the stream")
	}

	close(release)
}

// TestRecorderStillRecordsOrdinaryResponses guards the other direction: the
// streaming opt-out must not cost a normal handler its recording.
func TestRecorderStillRecordsOrdinaryResponses(t *testing.T) {
	var recorded string

	app := iris.New()
	app.Use(func(ctx iris.Context) {
		rr, isNew := recorder.AcquireResponseRecorder(ctx.ResponseWriter())
		if isNew {
			defer recorder.ReleaseResponseRecorder(rr)
			ctx.SetResponseWriter(rr)
		}
		ctx.Next()
		recorded = string(rr.GetBody())
	})
	app.Get("/plain", func(ctx iris.Context) {
		ctx.WriteString("hello")
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(app)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/plain")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if recorded != "hello" {
		t.Fatalf("recorded body = %q, want %q", recorded, "hello")
	}
}

// TestDisableResponseBufferingUnwrapsEveryWrapper checks the Context helper the
// sse package relies on: after it returns, nothing between the handler and the
// connection is holding bytes.
func TestDisableResponseBufferingUnwrapsEveryWrapper(t *testing.T) {
	app := iris.New()
	app.Use(recorder.Response)
	app.Get("/", func(ctx iris.Context) {
		if err := ctx.DisableResponseBuffering(); err != nil {
			t.Errorf("DisableResponseBuffering: %v", err)
		}

		if _, isRecorder := ctx.ResponseWriter().(*recorder.ResponseRecorder); isRecorder {
			t.Error("the recorder is still installed after DisableResponseBuffering")
		}

		if _, canFlush := context.UnwrapResponseWriter(ctx.ResponseWriter()).(http.Flusher); !canFlush {
			t.Error("the unwrapped writer cannot flush")
		}

		ctx.WriteString("ok")
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(app)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body := make([]byte, 2)
	if _, err := br(resp).Read(body); err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
}

func br(resp *http.Response) *bufio.Reader { return bufio.NewReader(resp.Body) }
