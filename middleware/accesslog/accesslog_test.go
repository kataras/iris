package accesslog

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/memstore"
)

func TestAccessLogPrint_Simple(t *testing.T) {
	t.Parallel()
	const goroutinesN = 42

	w := new(bytes.Buffer)
	ac := New(w)
	ac.TimeFormat = "2006-01-02 15:04:05"
	ac.Clock = TClock(time.Time{})

	if !ac.LockWriter { // should be true because we register a *bytes.Buffer.
		t.Fatalf("expected LockRriter to be true")
	}

	var (
		expected string
		wg       = new(sync.WaitGroup)
	)

	for i := 0; i < goroutinesN; i++ {
		wg.Add(1)
		expected += "0001-01-01 00:00:00|1s|200|GET|/path_value?url_query=url_query_value|::1|path_param=path_param_value url_query=url_query_value custom=custom_value|||Incoming|Outcoming|\n"

		go func() {
			defer wg.Done()

			ac.Print(
				nil,
				1*time.Second,
				ac.TimeFormat,
				200,
				"GET",
				"/path_value?url_query=url_query_value",
				"::1",
				"Incoming",
				"Outcoming",
				0,
				0,
				&context.RequestParams{
					Store: []memstore.Entry{
						{Key: "path_param", ValueRaw: "path_param_value"},
					},
				}, []memstore.StringEntry{
					{Key: "url_query", Value: "url_query_value"},
				}, []memstore.Entry{
					{Key: "custom", ValueRaw: "custom_value"},
				})
		}()
	}

	wg.Wait()

	if got := w.String(); expected != got {
		t.Fatalf("expected printed result to be:\n'%s'\n\nbut got:\n'%s'", expected, got)
	}
}

func TestAccessLogBroker(t *testing.T) {
	w := new(bytes.Buffer)
	ac := New(w)
	ac.Clock = TClock(time.Time{})
	broker := ac.Broker()

	closed := make(chan struct{})
	wg := new(sync.WaitGroup)
	n := 4
	wg.Add(4)
	go func() {
		defer wg.Done()

		i := 0
		ln := broker.NewListener()
		for {
			select {
			case <-closed:
				broker.CloseListener(ln)
				t.Log("Log Listener Closed")
				return
			case log := <-ln:
				lat := log.Latency
				t.Log(lat.String())
				wg.Done()
				if expected := time.Duration(i) * time.Second; expected != lat {
					panic(fmt.Sprintf("expected latency: %s but got: %s", expected, lat))
				}
				i++
				time.Sleep(1350 * time.Millisecond)
				if i == 2 {
					time.Sleep(2 * time.Second) // "random" sleep even more.
				}
				if log.Latency != lat {
					panic("expected logger to wait for notifier before release the log")
				}
			}
		}
	}()

	time.Sleep(time.Second)

	printLog := func(lat time.Duration) {
		err := ac.Print(
			nil,
			lat,
			"",
			0,
			"",
			"",
			"",
			"",
			"",
			0,
			0,
			&context.RequestParams{},
			nil,
			nil,
		)

		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < n; i++ {
		printLog(time.Duration(i) * time.Second)
	}

	// wait for all listeners to finish.
	wg.Wait()

	// wait for close messages.
	wg.Add(1)
	close(closed)
	wg.Wait()
}

type noOpFormatter struct{}

func (*noOpFormatter) SetOutput(io.Writer) {}

// Format prints the logs in text/template format.
func (*noOpFormatter) Format(*Log) (bool, error) {
	return true, nil
}

// go test -run=^$ -bench=BenchmarkAccessLogAfter -benchmem
func BenchmarkAccessLogAfter(b *testing.B) {
	benchmarkAccessLogAfter(b, true, false)
}

func BenchmarkAccessLogAfterPrint(b *testing.B) {
	benchmarkAccessLogAfter(b, false, false)
}

func benchmarkAccessLogAfter(b *testing.B, withLogStruct, async bool) {
	ac := New(ioutil.Discard)
	ac.Clock = TClock(time.Time{})
	ac.BytesReceived = false
	ac.BytesSent = false
	ac.BodyMinify = false
	ac.RequestBody = false
	ac.ResponseBody = false
	ac.KeepMultiLineError = true
	ac.Async = false
	ac.IP = false
	ac.LockWriter = async
	if withLogStruct {
		ac.SetFormatter(new(noOpFormatter)) // just to create the log structure, here we test the log creation time only.
	}

	ctx := new(context.Context)
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		b.Fatal(err)
	}
	ctx.ResetRequest(req)
	recorder := httptest.NewRecorder()
	w := context.AcquireResponseWriter()
	w.BeginResponse(recorder)
	ctx.ResetResponseWriter(w)

	wg := new(sync.WaitGroup)
	if async {
		wg.Add(b.N)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if async {
			go func() {
				ac.after(ctx, time.Millisecond, "GET", "/")
				wg.Done()
			}()
		} else {
			ac.after(ctx, time.Millisecond, "GET", "/")
		}
	}
	b.StopTimer()
	if async {
		wg.Wait()
	}
	w.EndResponse()
}
