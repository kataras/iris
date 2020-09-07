package accesslog

import (
	"bytes"
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
		expected += "0001-01-01 00:00:00|1s|GET|/path_value?url_query=url_query_value|path_param=path_param_value url_query=url_query_value custom=custom_value|200|Incoming|Outcoming|\n"

		go func() {
			defer wg.Done()

			ac.Print(
				nil,
				1*time.Second,
				ac.TimeFormat,
				200,
				"GET",
				"/path_value?url_query=url_query_value",
				"Incoming",
				"Outcoming",
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
