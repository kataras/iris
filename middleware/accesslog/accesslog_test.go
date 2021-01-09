package accesslog

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/memstore"
)

func TestAccessLogPrint_Simple(t *testing.T) {
	const goroutinesN = 42

	w := new(bytes.Buffer)
	ac := New(w)
	ac.Async = true
	ac.ResponseBody = true
	ac.Clock = TClock(time.Time{})

	var (
		expected      string
		expectedLines int
		mu            sync.Mutex
	)

	now := time.Now()
	for i := 0; i < goroutinesN; i++ {
		go func() {
			err := ac.Print(
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
				memstore.Store{
					{Key: "path_param", ValueRaw: "path_param_value"},
				},
				[]memstore.StringEntry{
					{Key: "url_query", Value: "url_query_value"},
				},
				[]memstore.Entry{
					{Key: "custom", ValueRaw: "custom_value"},
				})

			if err == nil {
				mu.Lock()
				expected += "0001-01-01 00:00:00|1s|200|GET|/path_value?url_query=url_query_value|::1|path_param=path_param_value url_query=url_query_value custom=custom_value|0 B|0 B|Incoming|Outcoming|\n"
				expectedLines++
				mu.Unlock()
			}
		}()

	}

	// give some time to write at least some messages or all
	// (depends on the machine the test is running).
	time.Sleep(42 * time.Millisecond)
	ac.Close()
	end := time.Since(now)

	if got := atomic.LoadUint32(&ac.remaining); got > 0 { // test wait.
		t.Fatalf("expected remaining: %d but got: %d", 0, got)
	}

	mu.Lock()
	expectedSoFoar := expected
	expectedLinesSoFar := expectedLines
	mu.Unlock()

	if got := w.String(); expectedSoFoar != got {
		gotLines := strings.Count(got, "\n")
		t.Logf("expected printed result to be[%d]:\n'%s'\n\nbut got[%d]:\n'%s'", expectedLinesSoFar, expectedSoFoar, gotLines, got)
		t.Fatalf("expected: %d | got: %d lines", expectedLinesSoFar, gotLines)
	} else {
		t.Logf("We've got [%d/%d] lines of logs in %s", expectedLinesSoFar, goroutinesN, end.String())
	}
}

func TestAccessLogBroker(t *testing.T) {
	w := new(bytes.Buffer)
	ac := New(w)

	ac.Clock = TClock(time.Time{})
	broker := ac.Broker()

	wg := new(sync.WaitGroup)
	n := 4
	wg.Add(n)
	go func() {
		i := 0
		ln := broker.NewListener()

		for log := range ln {
			lat := log.Latency
			t.Log(lat.String())
			wg.Done()
			if expected := time.Duration(i) * time.Second; expected != lat {
				panic(fmt.Sprintf("expected latency: %s but got: %s", expected, lat))
			}
			time.Sleep(1350 * time.Millisecond)
			if log.Latency != lat {
				panic("expected logger to wait for notifier before release the log")
			}
			i++
		}

		if i != n {
			for i < n {
				wg.Done()
				i++
			}
		}

		t.Log("Log Listener Closed: interrupted")
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
			nil,
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

	ac.Close()
}

func TestAccessLogBlank(t *testing.T) {
	w := new(bytes.Buffer)
	ac := New(w)
	clockTime, _ := time.Parse(defaultTimeFormat, "1993-01-01 05:00:00")
	ac.Clock = TClock(clockTime)
	ac.Blank = []byte("<no value>")

	ac.Print(
		nil,
		time.Second,
		defaultTimeFormat,
		200,
		"GET",
		"/",
		"127.0.0.1",
		"",
		"",
		0,
		0,
		nil,
		nil,
		nil,
	)

	ac.Close()
	// the request and bodies length are enabled by-default, zero bytes length
	// are written with 0 B and this cannot changed, so the request field
	// should be written as "<no value>".
	expected := "1993-01-01 05:00:00|1s|200|GET|/|127.0.0.1|0 B|0 B|<no value>|\n"
	if got := w.String(); expected != got {
		t.Fatalf("expected:\n'%s'\n\nbut got:\n'%s'", expected, got)
	}
}

type slowClose struct{ *bytes.Buffer }

func (c *slowClose) Close() error {
	time.Sleep(1 * time.Second)
	return nil
}

func TestAccessLogSetOutput(t *testing.T) {
	var (
		w1 = &bytes.Buffer{}
		w2 = &bytes.Buffer{}
		w3 = &slowClose{&bytes.Buffer{}}
		w4 = &bytes.Buffer{}
	)

	ac := New(w1)
	ac.Clock = TClock(time.Time{})

	n := 40
	expected := strings.Repeat("0001-01-01 00:00:00|1s|200|GET|/|127.0.0.1|0 B|0 B||\n", n)

	printLog := func() {
		err := ac.Print(
			nil,
			time.Second,
			defaultTimeFormat,
			200,
			"GET",
			"/",
			"127.0.0.1",
			"",
			"",
			0,
			0,
			nil,
			nil,
			nil,
		)

		if err != nil {
			t.Fatal(err)
		}
	}

	testSetOutput := func(name string, w io.Writer, withSlowClose bool) {
		wg := new(sync.WaitGroup)
		wg.Add(n / 4)
		for i := 0; i < n/4; i++ {
			go func(i int) {
				defer wg.Done()

				if i%2 == 0 {
					time.Sleep(10 * time.Millisecond)
				}

				if i == 5 {
					if w != nil {
						now := time.Now()
						ac.SetOutput(w)
						if withSlowClose {
							end := time.Since(now)
							if end < time.Second {
								panic(fmt.Sprintf("[%s] [%d]: SetOutput should wait for previous Close. Expected to return a bit after %s but %s", name, i, time.Second, end))
							}
						}
					}
				}

				printLog()
			}(i)
		}

		// wait to finish.
		wg.Wait()
	}

	go testSetOutput("w1", nil, false) // write at least one line and then
	time.Sleep(100 * time.Millisecond) // concurrently
	testSetOutput("w2", w2, false)     // change the writer
	testSetOutput("w3", w3, false)
	testSetOutput("w4", w4, true)

	gotAll := w1.String() + w2.String() + w3.String() + w4.String()

	// test if all content written and we have no loses.
	if expected != gotAll {
		t.Fatalf("expected total written result to be:\n'%s'\n\nbut got:\n'%s'", expected, gotAll)
	}

	// now, check if all have contents, they should because we wait between them,
	// contents spread.
	checkLines := func(name, s string, minimumLines int) {
		if got := strings.Count(s, "\n"); got < minimumLines {
			t.Logf("[%s] expected minimum lines of: %d but got %d", name, minimumLines, got)
		}
	}

	checkLines("w1", w1.String(), 1)
	checkLines("w2", w2.String(), 5)
	checkLines("w3", w3.String(), 5)
	checkLines("w4", w4.String(), 5)

	if err := ac.Close(); err != nil {
		t.Fatalf("On close: %v", err)
	}
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
	ac.BytesReceivedBody = false
	ac.BytesSent = false
	ac.BytesSentBody = false
	ac.BodyMinify = false
	ac.RequestBody = false
	ac.ResponseBody = false
	ac.Async = false
	ac.IP = false
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
