package errors_test

import (
	stdcontext "context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

// timeoutErr is what an http.Transport hands back when an upstream stalls,
// wrapped the way application code wraps it: a net.Error whose Timeout() is
// true, behind a fmt.Errorf.
var timeoutErr = fmt.Errorf("relay: %w", &net.OpError{Op: "dial", Net: "tcp", Err: os.ErrDeadlineExceeded})

// reportProbe runs one request through a handler that calls Report and records
// what the logger and the collector saw.
type reportProbe struct {
	logged    error
	collected *errors.Error
	reported  *errors.Error
}

func runReport(t *testing.T, cancelClient bool, err error) reportProbe {
	t.Helper()

	original := errors.LogError
	t.Cleanup(func() { errors.LogError = original })

	var probe reportProbe
	errors.LogError = func(_ iris.Context, err error) { probe.logged = err }

	app := iris.New()
	app.UseRouter(errors.NewOptions().
		SetCollector(func(_ iris.Context, e *errors.Error) { probe.collected = e }).
		Handler())
	app.Get("/", func(ctx iris.Context) {
		if cancelClient {
			// The client went away: its request context is cancelled.
			cancelled, cancel := stdcontext.WithCancel(ctx.Request().Context())
			cancel()
			ctx.SetRequest(ctx.Request().WithContext(cancelled))
		}

		probe.reported = errors.BadGateway.Report(ctx, err)
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app)
	t.Cleanup(srv.Close)

	resp, getErr := http.Get(srv.URL)
	if getErr != nil {
		t.Fatal(getErr)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	return probe
}

// TestReportTimeoutWithALiveClientIsReported: an upstream timeout is a failure
// the client is waiting to hear about and an operator needs to see. Only the
// client's own cancellation means nobody is listening; the shape of the error
// does not get to decide that.
func TestReportTimeoutWithALiveClientIsReported(t *testing.T) {
	probe := runReport(t, false, timeoutErr)

	if probe.reported == nil {
		t.Fatal("Report returned nil for an upstream timeout while the client was still connected")
	}
	if probe.logged == nil {
		t.Error("the logger never saw the timeout")
	}
	if probe.collected == nil {
		t.Error("the collector never saw the timeout")
	}
	if probe.reported.ErrorCode.CanonicalName != errors.BadGateway {
		t.Errorf("canonical name = %q, want the code Report was called under", probe.reported.ErrorCode.CanonicalName)
	}
}

// TestReportSkipsACancelledClient keeps the other half: a client that hung up
// is not paged about.
func TestReportSkipsACancelledClient(t *testing.T) {
	probe := runReport(t, true, timeoutErr)

	if probe.reported != nil {
		t.Errorf("Report returned %+v for a cancelled client, want nil", probe.reported)
	}
	if probe.logged != nil || probe.collected != nil {
		t.Error("the logger or the collector fired for a client that is gone")
	}
}
