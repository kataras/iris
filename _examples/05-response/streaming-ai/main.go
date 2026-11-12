// One ai.Open handler that streams a model's answer or returns it as a single
// JSON document.
//
// The generator here is a fake, because the point is the HTTP shape rather than
// the model: swap fakeModel for a real provider call and nothing else changes.
//
// Learn: serve "stream": true and the plain request from one handler, split
// fenced reasoning from the answer, and account for every turn in OnFinish.
// Before: ../streaming-sse
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -N -X POST http://localhost:8080/chat \
//	  -H 'Content-Type: application/json' \
//	  -d '{"prompt":"hello","stream":true}'   # 200, SSE: reasoning and text deltas, then usage
//	curl -X POST http://localhost:8080/chat \
//	  -H 'Content-Type: application/json' \
//	  -d '{"prompt":"hello"}'                 # 200, one JSON document from the same handler
//	curl -N -X POST 'http://localhost:8080/chat?format=text' \
//	  -H 'Content-Type: application/json' \
//	  -d '{"prompt":"hello","stream":true}'   # 200, plain text for a human at a terminal
//	curl -i -X POST http://localhost:8080/chat \
//	  -H 'Content-Type: application/json' -d '{"prompt":""}'   # 400, Validate rejected the empty prompt
//
// Open http://localhost:8080 for the browser client.
//
// Next: ../../16-advanced/ai-gateway
package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/ai"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	app.Get("/", index)
	app.Post("/chat", chat)

	app.Listen(":8080")
}

// ChatRequest is the request body. Stream mirrors what every OpenAI-shaped API
// puts there.
type ChatRequest struct {
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// Validate runs automatically inside ctx.BindJSON.
func (r ChatRequest) Validate() error {
	if strings.TrimSpace(r.Prompt) == "" {
		return iris.ErrEmptyForm
	}

	return nil
}

func index(ctx iris.Context) {
	ctx.HTML(`<!DOCTYPE html>
<html>
<body>
<p><b>Reasoning:</b> <span id="reasoning" style="color:#888"></span></p>
<p><b>Answer:</b> <span id="answer"></span></p>
<script>
// EventSource cannot POST a body, which is why every LLM client uses fetch.
fetch("/chat", {
	method: "POST",
	headers: {"Content-Type": "application/json"},
	body: JSON.stringify({prompt: "hello", stream: true}),
}).then(async (res) => {
	const reader = res.body.getReader();
	const decoder = new TextDecoder();
	let buffer = "";

	for (;;) {
		const {done, value} = await reader.read();
		if (done) break;

		buffer += decoder.decode(value, {stream: true});
		const frames = buffer.split("\n\n");
		buffer = frames.pop();

		for (const frame of frames) {
			let name = "message", data = "";
			for (const line of frame.split("\n")) {
				if (line.startsWith("event:")) name = line.slice(6).trim();
				else if (line.startsWith("data:")) data += line.slice(5).replace(/^ /, "");
			}
			if (!data) continue;

			if (name === "text") {
				document.getElementById("answer").textContent += JSON.parse(data).delta;
			} else if (name === "reasoning") {
				document.getElementById("reasoning").textContent += JSON.parse(data).delta;
			}
		}
	}
});
</script>
</body>
</html>`)
}

func chat(ctx iris.Context) {
	req, ok := ctx.BindJSON[ChatRequest]()
	if !ok {
		return // ctx.BindJSON has already written the 400.
	}

	o := ai.DefaultOptions()
	o.Stream = &req.Stream

	// One handler, three wire formats. The default is a neutral SSE dialect; a
	// vendor dialect would plug in here the same way.
	if ctx.Query().Get("format") == "text" {
		o.Encoder = ai.PlainText()
	}

	// The accounting leg. It runs once, on success, on failure and on client
	// disconnect alike, under a context that outlives the request: a client that
	// hung up is exactly when recording what the upstream already spent matters.
	o.OnFinish = func(_ context.Context, r *ai.Result) {
		log.Printf("turn done: streaming=%t committed=%t ttfp=%s duration=%s tokens=%d/%d chars=%d err=%v",
			r.Streaming, r.Committed, r.TTFP.Round(time.Millisecond), r.Duration.Round(time.Millisecond),
			r.Usage.InputTokens, r.Usage.OutputTokens, len(r.Text), r.Err)
	}

	res, err := ai.Open(ctx, o)
	if err != nil {
		// Nothing has been written, so this is still an ordinary error
		// response: errors.HandleError renders the envelope and keeps the
		// cause in the log rather than in the body.
		errors.HandleError(ctx, err)
		return
	}

	// Models that fence their reasoning inline emit the tag split across deltas
	// as often as not, so the splitter buffers only what could still become one.
	splitter := ai.TagSplit("m1", "<think>", "</think>")

	for delta := range fakeModel(req.Prompt) {
		for _, part := range splitter.Write(delta) {
			if err := res.Send(part); err != nil {
				res.Close(err)
				return
			}
		}

		if res.Context().Err() != nil {
			res.Close(nil) // the client left.
			return
		}
	}

	for _, part := range splitter.Flush() {
		res.Send(part)
	}

	res.Send(ai.Meter(ai.Usage{InputTokens: int64(len(req.Prompt)), OutputTokens: 24}))

	if err := res.Close(nil); err != nil {
		log.Printf("close: %v", err)
	}
}

// fakeModel stands in for a provider. It emits a reasoning trace fenced in a
// tag, then the answer, one small delta at a time.
func fakeModel(prompt string) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		script := "<think>The user said " + prompt + ". Keep it short.</think>" +
			"Hello. This answer arrives one delta at a time, which is the whole point."

		for chunk := range chunked(script, 7) {
			out <- chunk
			time.Sleep(40 * time.Millisecond)
		}
	}()

	return out
}

// chunked cuts s into pieces of at most n bytes, deliberately without respecting
// the tag boundaries, so the splitter has to do real work.
func chunked(s string, n int) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		for len(s) > 0 {
			size := min(n, len(s))
			out <- s[:size]
			s = s[size:]
		}
	}()

	return out
}
