package accesslog_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context/encoding"
	"github.com/kataras/iris/v14/middleware/accesslog"
)

// forgedRecord is what an attacker puts after a line break: a complete record
// in the default layout, for a request that was never made.
const forgedRecord = "2026-08-01 12:00:00|1ms|200|GET|/admin|127.0.0.1|"

// userTemplateText reads every client-controlled value of the Log by itself
// instead of through RequestValuesLine, as a custom template is free to do.
const userTemplateText = `{{.Method}} {{.Path}} {{.IP}}` +
	`{{range .PathParams}} {{.Key}}={{.Value}}{{end}}` +
	`{{range .Query}} {{.Key}}={{.Value}}{{end}}` +
	`{{range .Fields}} {{.Key}}={{.Value}}{{end}}` +
	` {{.Request}} {{.Response}}` + "\n"

// newHostileLog returns an AccessLog with a fixed clock and every column on.
func newHostileLog(w io.Writer, f accesslog.Formatter) *accesslog.AccessLog {
	ac := accesslog.New(w)
	ac.ResponseBody = true

	now, _ := time.Parse("2006-01-02 15:04:05", "1993-01-01 05:00:00")
	ac.Clock = accesslog.TClock(now)

	if f != nil {
		ac.SetFormatter(f)
	}

	return ac
}

// printHostile writes one log in which every value a client controls carries a
// line break: the method, the path, the IP (it can come from a header), a path
// parameter, a query value, the fields (that is where an application logs the
// user agent and other headers) and both bodies.
func printHostile(t *testing.T, ac *accesslog.AccessLog) {
	t.Helper()

	err := ac.Print(nil,
		time.Second,
		"2006-01-02 15:04:05",
		200,
		"GE\nT",
		"/a\n"+forgedRecord,
		"::1\r\n::2",
		"{\n\t\"a\": 1\n}",
		"one\r\ntwo",
		12,
		34,
		encoding.PathParams{{Key: "id", Value: "4\n2"}},
		encoding.StringEntries{{Key: "q", Value: "x\ny"}},
		accesslog.Fields{
			{Key: "ua", Value: "curl\n8"},
			{Key: "tags", Value: []string{"a\nb"}},
			{Key: "n", Value: 42},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

// formatHostile returns what the formatter "f" writes for printHostile.
func formatHostile(t *testing.T, f accesslog.Formatter) string {
	t.Helper()

	w := new(bytes.Buffer)
	ac := newHostileLog(w, f)
	printHostile(t, ac)

	if err := ac.Close(); err != nil {
		t.Fatal(err)
	}

	return w.String()
}

// What the default format writes for printHostile: one line, with each line
// break and tab turned to a space.
const hostileDefaultLine = "1993-01-01 05:00:00|1s|200|GE T|/a " + forgedRecord + "|::1  ::2|" +
	"id=4 2 q=x y ua=curl 8 tags=[a b] n=42|12 B|34 B|{  \"a\": 1 }|one  two|\n"

// The same log through userTemplateText.
const hostileUserLine = "GE T /a " + forgedRecord + " ::1  ::2 id=4 2 q=x y ua=curl 8 tags=[a b] n=42" +
	" {  \"a\": 1 } one  two\n"

// The default format already sanitized its values. It must keep writing the
// same bytes.
func TestDefaultFormatIsUnchanged(t *testing.T) {
	if got := formatHostile(t, nil); got != hostileDefaultLine {
		t.Fatalf("default format:\nexpected: %q\n     got: %q", hostileDefaultLine, got)
	}
}

// JSON escapes control characters by itself and is one record per line for it.
// The values must reach it as the request sent them, byte for byte.
func TestJSONFormatIsUnchanged(t *testing.T) {
	const expected = `{"timestamp":725864400000,"latency":1000000000,"code":200,"method":"GE\nT",` +
		`"path":"/a\n2026-08-01 12:00:00|1ms|200|GET|/admin|127.0.0.1|","ip":"::1\r\n::2",` +
		`"query":[{"key":"q","value":"x\ny"}],"params":[{"key":"id","value":"4\n2"}],` +
		`"fields":[{"key":"ua","value":"curl\n8"},{"key":"tags","value":["a\nb"]},{"key":"n","value":42}],` +
		`"request":"{\n\t\"a\": 1\n}","response":"one\r\ntwo","bytes_received":12,"bytes_sent":34}` + "\n"

	if got := formatHostile(t, new(accesslog.JSON)); got != expected {
		t.Fatalf("JSON format:\nexpected: %q\n     got: %q", expected, got)
	}
}

// A formatter that writes text lines used to print the values as they were
// received, so a line break in any of them started a second line.
func TestTextFormattersWriteOneLine(t *testing.T) {
	tests := []struct {
		name      string
		formatter accesslog.Formatter
		expected  string
	}{
		{
			// The built-in template has the columns of the default format.
			name:      "Template",
			formatter: new(accesslog.Template),
			expected:  hostileDefaultLine,
		},
		{
			name:      "Template with Text",
			formatter: &accesslog.Template{Text: userTemplateText},
			expected:  hostileUserLine,
		},
		{
			name: "Template with Tmpl and TmplName",
			formatter: &accesslog.Template{
				Tmpl:     template.Must(template.New("line").Parse(userTemplateText)),
				TmplName: "line",
			},
			expected: hostileUserLine,
		},
		{
			name:      "CSV",
			formatter: new(accesslog.CSV),
			expected: "725864400000,1s,200,GE T,/a " + forgedRecord + ",::1  ::2," +
				"id=4 2 q=x y ua=curl 8 tags=[a b] n=42,12,34,\"{  \"\"a\"\": 1 }\",one  two\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatHostile(t, tt.formatter)
			if n := strings.Count(got, "\n"); n != 1 {
				t.Errorf("one log wrote %d lines", n)
			}
			if got != tt.expected {
				t.Errorf("expected: %q\n     got: %q", tt.expected, got)
			}
		})
	}
}

// The Template formatter sanitizes a copy. The Broker's listeners receive the
// Log after the formatter ran, and they still read what the request sent.
func TestTemplateLeavesTheLogToTheBroker(t *testing.T) {
	ac := newHostileLog(io.Discard, new(accesslog.Template))
	defer ac.Close()

	ln := ac.Broker().NewListener()
	printHostile(t, ac)

	select {
	case log := <-ln:
		if expected := "/a\n" + forgedRecord; log.Path != expected {
			t.Errorf("listener: expected path %q but got %q", expected, log.Path)
		}
		if log.Method != "GE\nT" || log.IP != "::1\r\n::2" || log.Request != "{\n\t\"a\": 1\n}" || log.Query[0].Value != "x\ny" {
			t.Errorf("listener: the formatter changed the Log: %#v", log)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the listener did not receive the log")
	}
}

// LogText renders the line Format writes, and does not touch the Log either.
func TestTemplateLogTextLeavesTheLogAlone(t *testing.T) {
	tmpl := &accesslog.Template{Text: userTemplateText}
	ac := newHostileLog(io.Discard, tmpl)
	defer ac.Close()

	log := &accesslog.Log{
		Logger:     ac,
		Method:     "GET",
		Path:       "/a\n" + forgedRecord,
		IP:         "::1",
		PathParams: encoding.PathParams{{Key: "id", Value: "4\n2"}},
		Query:      encoding.StringEntries{{Key: "q", Value: "x\ny"}},
		Fields:     accesslog.Fields{{Key: "tags", Value: []string{"a\nb"}}},
		Request:    "{\n}",
	}

	got, err := tmpl.LogText(log)
	if err != nil {
		t.Fatal(err)
	}

	if expected := "GET /a " + forgedRecord + " ::1 id=4 2 q=x y tags=[a b] { } \n"; got != expected {
		t.Errorf("expected: %q\n     got: %q", expected, got)
	}

	if log.Path != "/a\n"+forgedRecord || log.Request != "{\n}" ||
		log.PathParams[0].Value != "4\n2" || log.Query[0].Value != "x\ny" {
		t.Errorf("LogText changed the Log: %#v", log)
	}
	if tags, ok := log.Fields[0].Value.([]string); !ok || tags[0] != "a\nb" {
		t.Errorf("LogText changed a field of the Log: %#v", log.Fields[0].Value)
	}
}

// A key is the client's too: a query key is, and so is the key of a field or
// of a path parameter that an application takes from the request. A value that
// is not a string and prints clean keeps its type next to a sanitized key, so
// a template can still range over it.
func TestTemplateSanitizesEntryKeys(t *testing.T) {
	const text = `{{range .PathParams}}{{.Key}}={{.Value}} {{end}}` +
		`{{range .Query}}{{.Key}}={{.Value}} {{end}}` +
		`{{range .Fields}}{{.Key}}={{range .Value}}<{{.}}>{{end}} {{end}}` + "\n"

	w := new(bytes.Buffer)
	ac := newHostileLog(w, &accesslog.Template{Text: text})

	err := ac.Print(nil, time.Second, "", 200, "GET", "/", "", "", "", 0, 0,
		encoding.PathParams{{Key: "i\nd", Value: 42}},
		encoding.StringEntries{{Key: "q\r\n" + forgedRecord, Value: "x"}},
		accesslog.Fields{{Key: "ta\ngs", Value: []string{"a", "b"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err = ac.Close(); err != nil {
		t.Fatal(err)
	}

	if expected, got := "i d=42 q  "+forgedRecord+"=x ta gs=<a><b> \n", w.String(); got != expected {
		t.Errorf("expected: %q\n     got: %q", expected, got)
	}
}

// A control character in front of a formula sign is dropped by the sanitizer,
// which must not let the formula through the spreadsheet escape: the value is
// escaped as the request sent it and again as it is written.
func TestCSVFormulaEscapeSurvivesSanitizing(t *testing.T) {
	tests := []struct{ path, expected string }{
		{"=1+1", "'=1+1"},
		{"\x00=1+1", "'=1+1"},
		{"\t=1+1", "' =1+1"},
		{"\r=1+1", "' =1+1"},
		{"/plain", "/plain"},
	}

	for _, tt := range tests {
		w := new(bytes.Buffer)
		ac := newHostileLog(w, new(accesslog.CSV))
		ac.IP = false

		if err := ac.Print(nil, time.Second, "", 200, "GET", tt.path, "", "", "", 0, 0, nil, nil, nil); err != nil {
			t.Fatal(err)
		}
		ac.Close()

		if expected, got := "725864400000,1s,200,GET,"+tt.expected+",0,0\n", w.String(); got != expected {
			t.Errorf("path %q:\nexpected: %q\n     got: %q", tt.path, expected, got)
		}
	}
}

// The same over the wire: one request, with %0A in its path and its query and a
// line break in its body, is one line in the log whatever the formatter is.
func TestRequestCannotForgeALogLine(t *testing.T) {
	tests := []struct {
		name      string
		formatter accesslog.Formatter
	}{
		{"default format", nil},
		{"JSON", new(accesslog.JSON)},
		{"Template", new(accesslog.Template)},
		{"Template with Text", &accesslog.Template{Text: userTemplateText}},
		{"CSV", new(accesslog.CSV)},
	}

	const target = "/a%0A2026-08-01%2012:00:00%7C1ms%7C200%7CGET%7C/admin%7C127.0.0.1%7C" +
		"?q=x%0A2026-08-01%2012:00:00%7C1ms%7C200%7CGET%7C/admin%7C127.0.0.1%7C"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := new(bytes.Buffer)
			ac := newHostileLog(w, tt.formatter)
			// What an application does to log a header or the user agent.
			ac.AddFields(func(ctx iris.Context, f *accesslog.Fields) {
				f.Set("search", ctx.Query().Get("q"))
			})

			app := iris.New()
			app.Logger().SetOutput(io.Discard)
			app.UseRouter(ac.Handler)
			app.Post("/{p:path}", func(ctx iris.Context) {
				body, _ := ctx.Body()
				ctx.Write(body) // the response body carries the line break too.
			})
			if err := app.Build(); err != nil {
				t.Fatal(err)
			}

			srv := httptest.NewServer(app)
			resp, err := srv.Client().Post(srv.URL+target, "application/json", strings.NewReader("{\n\"a\": 1\n}"))
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			srv.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d", resp.StatusCode)
			}
			if err = ac.Close(); err != nil {
				t.Fatal(err)
			}

			got := w.String()
			if !strings.Contains(got, "/admin") {
				t.Fatalf("the request did not reach the log with its path decoded: %q", got)
			}
			if n := strings.Count(got, "\n"); n != 1 {
				t.Errorf("one request wrote %d log lines:\n%s", n, got)
			}
		})
	}
}
