// An AI gateway: relaying an upstream provider stream, with failover.
//
// This is the shape a real proxy takes. A fake provider runs on the same
// process so the example is self-contained, and it fails on demand so the
// failover path is visible rather than theoretical.
//
// The three things worth reading here are the classification (a typed failure
// read from the response, not from an error string), the retry guard (once the
// first delta has gone out, a second attempt would splice two answers into one
// response), and the accounting hook (which runs even when the client hangs up).
//
// Learn: relay a model stream through your own server, classify upstream failures, and fail over only while nothing has reached the client.
// Before: ../../05-response/streaming-ai
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -N -H 'Accept: text/event-stream' http://localhost:8080/chat                # 200, a normal relay from the primary
//	curl -N -H 'Accept: text/event-stream' 'http://localhost:8080/chat?fail=1'       # 200, the primary fails and the secondary answers
//	curl -i -H 'Accept: text/event-stream' 'http://localhost:8080/chat?fail=all'     # an error status with Retry-After, because nothing streamed
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/ai"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/sse"
)

func main() {
	// Two fake providers, so failover has somewhere to go.
	primary := startFakeProvider("primary")
	secondary := startFakeProvider("secondary")
	defer primary.Close()
	defer secondary.Close()

	chain := []backend{
		{name: "primary", url: primary.URL},
		{name: "secondary", url: secondary.URL},
	}

	app := iris.New()
	app.Get("/chat", newGateway(chain).handle)
	app.Listen(":8080")
}

// backend is one upstream the gateway can try.
type backend struct {
	name string
	url  string
}

type gateway struct {
	chain  []backend
	client *http.Client
}

func newGateway(chain []backend) *gateway {
	return &gateway{
		chain: chain,
		// No client-level timeout. http.Client.Timeout covers the whole
		// lifecycle including reading the body, so any non-zero value cuts a long
		// answer off mid-sentence. Cancellation rides the request context.
		//
		// A header timeout is a different thing: it bounds how long a provider
		// may stall before the first byte. Classify reports it as a transient
		// timeout, so the chain falls through to the next backend.
		client: &http.Client{
			Transport: &http.Transport{ResponseHeaderTimeout: 30 * time.Second},
		},
	}
}

func (g *gateway) handle(ctx iris.Context) {
	o := ai.DefaultOptions()
	o.OnFinish = func(_ context.Context, r *ai.Result) {
		// The durable copy. The stream is an optimisation, never the source of
		// truth, and this runs on disconnect too.
		log.Printf("turn: committed=%t disconnected=%t tokens=%d/%d duration=%s err=%v",
			r.Committed, r.Disconnected, r.Usage.InputTokens, r.Usage.OutputTokens,
			r.Duration.Round(time.Millisecond), r.Err)
	}

	res, err := ai.Open(ctx, o)
	if err != nil {
		errors.HandleError(ctx, err)
		return
	}

	err = res.Close(g.relayWithFailover(ctx, res))
	if err == nil || res.Committed() {
		// Either it worked, or the answer had already begun and Close reported
		// the failure in-band. Both are finished responses.
		return
	}

	// Nothing reached the client, so the status code is still ours to spend.
	// This is the branch a gateway forgets, and the symptom is a 200 with an
	// empty body.
	//
	// Report is the same call the streaming path makes, so the client gets one
	// envelope either way: the provider's own words go to the logger and the
	// collector, and the caller gets a canonical code and a sentence about what
	// to do next.
	if ue, ok := ai.AsUpstream(err); ok {
		reported := ue.Report(ctx)
		if reported == nil {
			return // the client itself is gone: nobody is listening.
		}

		if ue.RetryAfter > 0 {
			ctx.SetHeader("Retry-After", strconv.Itoa(int(ue.RetryAfter.Seconds())))
		}

		// StopWithJSON, not an errors.X call: Report already built AND
		// collected this envelope, so all that is left is to write it under
		// the status its own canonical code carries.
		ctx.StopWithJSON(reported.ErrorCode.Status, reported)
		return
	}

	errors.Internal.Err(ctx, err)
}

// relayWithFailover walks the chain until one upstream serves the request.
func (g *gateway) relayWithFailover(ctx iris.Context, res *ai.Response) error {
	var lastErr error

	for _, b := range g.chain {
		// Once a delta has reached the client the status code is spent and the
		// answer has begun. Restarting now would splice two different answers
		// into one response, so this is where failover has to stop.
		if res.Committed() {
			return lastErr
		}

		err := g.relayOnce(ctx, res, b)
		if err == nil {
			return nil
		}
		lastErr = err

		ue, ok := ai.AsUpstream(err)
		if !ok {
			// Not an upstream failure. Relay types everything the provider did
			// wrong, including a stream cut before its [DONE], and a client that
			// hung up; what is left is our decoder's bug. Do not paper over it.
			return err
		}

		log.Printf("upstream %s failed: %s (class=%s, retryable=%t)",
			b.name, ue.Error(), ue.Class, ue.Retryable())

		switch ue.Class {
		case ai.ClassCanceled:
			// Nobody is waiting. A second attempt spends tokens on an answer that
			// will never be read.
			return err
		case ai.ClassFatal:
			// The request is wrong; it would be just as wrong on the next one.
			return err
		}

		// Transient, rate limited or exhausted: the next backend may serve it.
	}

	return lastErr
}

func (g *gateway) relayOnce(ctx iris.Context, res *ai.Response, b backend) error {
	url := b.url + "?" + ctx.Request().URL.RawQuery

	req, err := http.NewRequestWithContext(ctx.Request().Context(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := g.client.Do(req)

	// Classify takes the response, not only the error: a non-2xx comes back with
	// a nil error, so a signature that saw only the error would have to recover
	// the status code from a string.
	if ue := ai.Classify(resp, err); ue != nil {
		return ue
	}

	return ai.Relay(res, resp, decoder(b.name), ai.RelayOptions{
		Read: sse.ReadOptions{Terminators: []string{"[DONE]"}},
	})
}

// decoder turns this provider's dialect into neutral parts. It is application
// code on purpose: the framework has no business knowing what a given vendor
// calls a delta this month.
func decoder(backendName string) ai.Decoder {
	return ai.DecoderFunc(func(ev sse.Event) ([]ai.Part, error) {
		var chunk struct {
			Delta string `json:"delta"`
			Usage *struct {
				In  int64 `json:"in"`
				Out int64 `json:"out"`
			} `json:"usage"`
		}

		if err := json.Unmarshal(ev.Data, &chunk); err != nil {
			return nil, fmt.Errorf("decoding %s event %q: %w", backendName, ev.Data, err)
		}

		var parts []ai.Part
		if chunk.Delta != "" {
			parts = append(parts, ai.Text("m1", chunk.Delta))
		}
		if chunk.Usage != nil {
			parts = append(parts, ai.Meter(ai.Usage{
				InputTokens:  chunk.Usage.In,
				OutputTokens: chunk.Usage.Out,
			}))
		}

		return parts, nil
	})
}

// --------------------------------------------------------------------------
// A fake provider, so the example runs on its own.
// --------------------------------------------------------------------------

type fakeProvider struct {
	URL   string
	close func()
}

func (p *fakeProvider) Close() { p.close() }

func startFakeProvider(name string) *fakeProvider {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch fail := r.URL.Query().Get("fail"); {
		case fail == "all", fail == "1" && name == "primary":
			// The shape a provider actually uses: a nested error object.
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "3")
			w.WriteHeader(http.StatusTooManyRequests)
			io.WriteString(w, `{"error":{"type":"rate_limit_error","message":"`+name+` is out of capacity"}}`)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		rc := http.NewResponseController(w)

		for _, word := range strings.Fields("this answer came from " + name + " one delta at a time") {
			// Written without a space after "data:" on purpose: it is legal, and
			// several gateways do it, which is what breaks a prefix-matching
			// reader.
			fmt.Fprintf(w, "data:{\"delta\":%q}\n\n", word+" ")
			rc.Flush()
			time.Sleep(60 * time.Millisecond)
		}

		io.WriteString(w, "data: {\"usage\":{\"in\":12,\"out\":9}}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
		rc.Flush()
	})

	srv := &http.Server{Handler: mux}
	ln := mustListen()

	go srv.Serve(ln)

	return &fakeProvider{
		URL:   "http://" + ln.Addr().String(),
		close: func() { srv.Close() },
	}
}

// mustListen grabs an ephemeral port for a fake provider.
func mustListen() net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	return ln
}
