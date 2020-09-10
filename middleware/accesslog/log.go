package accesslog

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/memstore"
)

// Log represents the log data specifically for the accesslog middleware.
type Log struct {
	// The AccessLog instance this Log was created of.
	Logger *AccessLog `json:"-" yaml:"-" toml:"-"`

	// The time the log is created.
	Now time.Time `json:"-" yaml:"-" toml:"-"`
	// TimeFormat selected to print the Time as string,
	// useful on Template Formatter.
	TimeFormat string `json:"-" yaml:"-" toml:"-"`
	// Timestamp the Now's unix timestamp (seconds).
	Timestamp int64 `json:"timestamp"`

	// Request-Response latency.
	Latency time.Duration `json:"latency"`
	// Init request's Method and Path.
	Method string `json:"method"`
	Path   string `json:"path"`
	// The response status code.
	Code int `json:"code"`
	// Sorted URL Query arguments.
	Query []memstore.StringEntry `json:"query,omitempty"`
	// Dynamic path parameters.
	PathParams []memstore.Entry `json:"params,omitempty"`
	// Fields any data information useful to represent this Log.
	Fields []memstore.Entry `json:"fields,omitempty"`

	//  The actual number of bytes received and sent on the network (headers + body).
	BytesReceived int `json:"bytes_received"`
	BytesSent     int `json:"bytes_sent"`

	// The Request and Response raw bodies.
	// If they are escaped (e.g. JSON),
	// A third-party software can read it through:
	// data, _ := strconv.Unquote(log.Request)
	// err := json.Unmarshal([]byte(data), &customStruct)
	Request  string `json:"request"`
	Response string `json:"response"`

	// A copy of the Request's Context when Async is true (safe to use concurrently),
	// otherwise it's the current Context (not safe for concurrent access).
	Ctx *context.Context `json:"-" yaml:"-" toml:"-"`
}

// Clone returns a raw copy value of this Log.
func (l *Log) Clone() Log {
	return *l
}

// RequestValuesLine returns a string line which
// combines the path parameters, query and custom fields.
func (l *Log) RequestValuesLine() string {
	return parseRequestValues(l.Code, l.Ctx.Params(), l.Ctx.URLParamsSorted(), l.Fields)
}

// BytesReceivedLine returns the formatted bytes received length.
func (l *Log) BytesReceivedLine() string {
	return formatBytes(l.BytesReceived)
}

// BytesSentLine returns the formatted bytes sent length.
func (l *Log) BytesSentLine() string {
	return formatBytes(l.BytesSent)
}

func formatBytes(b int) string {
	if b <= 0 {
		return ""
	}

	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

func parseRequestValues(code int, pathParams *context.RequestParams, query []memstore.StringEntry, fields memstore.Store) (requestValues string) {
	var buf strings.Builder

	if !context.StatusCodeNotSuccessful(code) {
		// collect path parameters on a successful request-response only.
		pathParams.Visit(func(key, value string) {
			buf.WriteString(key)
			buf.WriteByte('=')
			buf.WriteString(value)
			buf.WriteByte(' ')
		})
	}

	for _, entry := range query {
		buf.WriteString(entry.Key)
		buf.WriteByte('=')
		buf.WriteString(entry.Value)
		buf.WriteByte(' ')
	}

	for _, entry := range fields {
		buf.WriteString(entry.Key)
		buf.WriteByte('=')
		buf.WriteString(fmt.Sprintf("%v", entry.ValueRaw))
		buf.WriteByte(' ')
	}

	if n := buf.Len(); n > 1 {
		requestValues = buf.String()[0 : n-1] // remove last space.
	}

	return
}

// Formatter is responsible to print a Log to the accesslog's writer.
type Formatter interface {
	// SetOutput should inject the accesslog's direct output,
	// if this "dest" is used then the Formatter
	// should manually control its concurrent use.
	SetOutput(dest io.Writer)
	// Format should print the Log.
	// Returns nil error on handle successfully,
	// otherwise the log will be printed using the default formatter
	// and the error will be printed to the Iris Application's error log level.
	// Should return true if this handled the logging, otherwise false to
	// continue with the default print format.
	Format(log *Log) (bool, error)
}

var (
	_ Formatter = (*JSON)(nil)
	_ Formatter = (*Template)(nil)
)

// JSON is a Formatter type for JSON logs.
type JSON struct {
	Prefix, Indent string
	EscapeHTML     bool

	enc *json.Encoder
	mu  sync.Mutex
}

// SetOutput creates the json encoder writes to the "dest".
// It's called automatically by the middleware when this Formatter is used.
func (f *JSON) SetOutput(dest io.Writer) {
	if dest == nil {
		return
	}

	// All logs share the same accesslog's writer and it cannot change during serve-time.
	enc := json.NewEncoder(dest)
	enc.SetEscapeHTML(f.EscapeHTML)
	enc.SetIndent(f.Prefix, f.Indent)
	f.enc = enc
}

// Format prints the logs in JSON format.
// Writes to the destination directly,
// locks on each Format call.
func (f *JSON) Format(log *Log) (bool, error) {
	f.mu.Lock()
	err := f.enc.Encode(log)
	f.mu.Unlock()

	return true, err
}

// Template is a Formatter.
// It's used to print the Log in a text/template way.
// The caller has full control over the printable result;
// certain fields can be ignored, change the display order and e.t.c.
type Template struct {
	// Custom template source.
	// Use this or `Tmpl/TmplName` fields.
	Text string
	// Custom template to use, overrides the `Text` field if not nil.
	Tmpl *template.Template
	// If not empty then this named template/block
	// is response to hold the log result.
	TmplName string

	dest io.Writer
	mu   sync.Mutex
}

// SetOutput creates the default template if missing
// when this formatter is registered.
func (f *Template) SetOutput(dest io.Writer) {
	if f.Tmpl == nil {
		text := f.Text
		if f.Text == "" {
			text = defaultTmplText
		}

		f.Tmpl = template.Must(template.New("").Parse(text))
	}

	f.dest = dest
}

const defaultTmplText = "{{.Now.Format .TimeFormat}}|{{.Latency}}|{{.Method}}|{{.Path}}|{{.RequestValuesLine}}|{{.Code}}|{{.BytesReceivedLine}}|{{.BytesSentLine}}|{{.Request}}|{{.Response}}|\n"

// Format prints the logs in text/template format.
func (f *Template) Format(log *Log) (bool, error) {
	var err error

	// A template may be executed safely in parallel, although if parallel
	// executions share a Writer the output may be interleaved.
	f.mu.Lock()
	if f.TmplName != "" {
		err = f.Tmpl.ExecuteTemplate(f.dest, f.TmplName, log)
	} else {
		err = f.Tmpl.Execute(f.dest, log)
	}
	f.mu.Unlock()

	return true, err
}
