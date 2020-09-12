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
	staticNow, _ := time.Parse(defaultTimeFormat, "1993-01-01 05:00:00")
	ac.Clock = TClock(staticNow)
	ac.SetFormatter(&CSV{
		Header:       true,
		LatencyRound: time.Second,
		AutoFlush:    true,
	})

	lat, _ := time.ParseDuration("1s")

	print := func() {
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
	print()
	print()

	expected := `Timestamp,Latency,Code,Method,Path,IP,Req Values,In,Out,Request,Response
725864400000,1s,200,GET,/,::1,sleep=1s,573,81,,Index
725864400000,1s,200,GET,/,::1,sleep=1s,573,81,,Index
`

	if got := buf.String(); expected != got {
		t.Fatalf("expected:\n%s\n\nbut got:\n%s", expected, got)
	}
}
