package accesslog

import (
	"bytes"
	"testing"
	"time"

	"github.com/kataras/iris/v12/core/memstore"
)

func TestCSV(t *testing.T) {
	buf := new(bytes.Buffer)
	ac := New(buf)
	ac.RequestBody = false
	staticNow, _ := time.Parse(defaultTimeFormat, "1993-01-01 05:00:00")
	ac.Clock = TClock(staticNow)
	ac.SetFormatter(&CSV{
		Header: true,
	})

	lat, _ := time.ParseDuration("1s")

	printFunc := func() {
		ac.Print(
			nil,
			lat,
			"",
			200,
			"GET",
			"/",
			"::1",
			"",
			"Index",
			573,
			81,
			nil,
			[]memstore.StringEntry{{Key: "sleep", Value: "1s"}},
			nil)
	}

	// print twice, the header should only be written once.
	printFunc()
	printFunc()

	expected := `Timestamp,Latency,Code,Method,Path,IP,Req Values,In,Out
725864400000,1s,200,GET,/,::1,sleep=1s,573,81
725864400000,1s,200,GET,/,::1,sleep=1s,573,81
`

	ac.Close()
	if got := buf.String(); expected != got {
		t.Fatalf("expected:\n%s\n\nbut got:\n%s", expected, got)
	}
}
