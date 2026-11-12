// Server-Sent Events with the sse package.
//
// sse.Open writes nothing: the protocol headers commit on the first Send, so a
// check that fails after opening still returns an ordinary status code. On that
// first event it also removes any buffering middleware from the response, clears
// the server's absolute write timeout and sets the headers that stop proxies
// buffering the stream.
//
// Learn: stream events that end with a terminal frame, batch small ones, and
// still answer an ordinary 401 when a check fails after Open.
// Before: ../files
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -N http://localhost:8080/events    # 200, ten ticks, one per second, then the terminal frame
//	curl -N http://localhost:8080/progress  # 200, the sentence in batched "word" events
//	curl -i http://localhost:8080/guarded   # 401 after Open: still an ordinary response
//
// Open http://localhost:8080 to watch the events in a browser (EventSource).
//
// Next: ../streaming-ai
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/sse"
)

func main() {
	app := iris.New()

	app.Get("/", index)
	app.Get("/events", events)
	app.Get("/progress", progress)
	app.Get("/guarded", guarded)

	app.Listen(":8080")
}

// events streams ten messages, one per second, then ends the stream.
func events(ctx iris.Context) {
	stream, err := sse.Open(ctx)
	if err != nil {
		// The response cannot flush, so nothing written to it would reach the
		// client. Nothing has been written yet, so an honest answer is still
		// possible: errors.HandleError renders the 500 envelope and logs the
		// cause rather than showing it.
		errors.HandleError(ctx, err)
		return
	}
	// Close writes the terminal frame, on success as well as on failure. Its
	// absence is how a client tells a finished stream from a severed one.
	defer stream.Close(nil)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for i := 1; i <= 10; i++ {
		select {
		case <-ctx.ShuttingDown():
			return // the server is stopping: end cleanly while we still can.
		case <-stream.Context().Done():
			return // the client is gone.
		case <-ticker.C:
		}

		if err := stream.Send(sse.Event{
			Name: "tick",
			Data: fmt.Appendf(nil, "tick %d", i),
		}); err != nil {
			return
		}
	}
}

// progress streams a sentence one word at a time, batched.
//
// Without batching this is one write, one flush and one TCP segment per word.
// The interval is a ceiling rather than a floor: a background ticker drains the
// buffer, so the last word does not wait for a next one that never comes.
func progress(ctx iris.Context) {
	stream, err := sse.Open(ctx)
	if err != nil {
		errors.HandleError(ctx, err)
		return
	}
	defer stream.Close(nil)

	batch := stream.Coalesce(200*time.Millisecond, "word")
	defer batch.Close()

	for _, word := range strings.Fields("the quick brown fox jumps over the lazy dog") {
		if err := batch.Send(sse.Event{Name: "word", Data: []byte(word + " ")}); err != nil {
			return
		}

		time.Sleep(80 * time.Millisecond)
	}
}

// guarded shows the property the whole design rests on: Open commits nothing, so
// a check that runs after it can still refuse the request properly.
func guarded(ctx iris.Context) {
	stream, err := sse.Open(ctx)
	if err != nil {
		errors.HandleError(ctx, err)
		return
	}
	defer stream.Close(nil)

	if ctx.Query().Get("token") == "" {
		// A real status code and a real body, because not one byte has gone
		// out: the 401 renders through the rest/errors envelope like any other.
		errors.Unauthenticated.Message(ctx, "token required")
		return
	}

	stream.Send(sse.Event{Name: "welcome", Data: []byte("you are in")})
}

func index(ctx iris.Context) {
	ctx.HTML(`<!DOCTYPE html>
<html>
<body>
<ul id="log"></ul>
<script>
const es = new EventSource("/events");
es.addEventListener("tick", (e) => {
	const li = document.createElement("li");
	li.textContent = e.data;
	document.getElementById("log").appendChild(li);
});
es.addEventListener("done", () => es.close()); // the terminal frame.
es.onerror = () => es.close();
</script>
</body>
</html>`)
}
