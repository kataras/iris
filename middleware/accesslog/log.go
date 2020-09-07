package accesslog

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/memstore"
)

// Log represents the log data specifically for the accesslog middleware.
type Log struct {
	// The AccessLog instance this Log was created of.
	Logger *AccessLog `json:"-"`

	// The time the log is created.
	Now time.Time `json:"-"`
	// Timestamp the Now's unix timestamp (seconds).
	Timestamp int64 `json:"timestamp"`

	// Request-Response latency.
	Latency time.Duration `json:"latency"`
	// Init request's Method and Path.
	Method string `json:"method"`
	Path   string `json:"path"`
	// Sorted URL Query arguments.
	Query []memstore.StringEntry
	// Dynamic path parameters.
	PathParams []memstore.Entry
	// Fields any data information useful to represent this Log.
	Fields context.Map `json:"fields,omitempty"`

	// The Request and Response raw bodies.
	// If they are escaped (e.g. JSON),
	// A third-party software can read it through:
	// data, _ := strconv.Unquote(log.Request)
	// err := json.Unmarshal([]byte(data), &customStruct)
	Request  string `json:"request"`
	Response string `json:"response"`

	// A copy of the Request's Context when Async is true (safe to use concurrently),
	// otherwise it's the current Context (not safe for concurrent access).
	Ctx *context.Context `json:"-"`
}

// Formatter is responsible to print a Log to the accesslog's writer.
type Formatter interface {
	// Format should print the Log.
	// Returns true on handle successfully,
	// otherwise the log will be printed using the default formatter.
	Format(log *Log) bool
	// SetWriter should inject the accesslog's output.
	SetOutput(dest io.Writer)
}

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
func (f *JSON) Format(log *Log) bool {
	f.mu.Lock()
	if f.enc == nil {
		// If no custom writer is given then f.enc is nil,
		// this code block should only be executed once.
		// Also, the app's logger's writer cannot change during serve-time,
		// so all logs share the same instance output.
		f.SetOutput(log.Ctx.Application().Logger().Printer)
	}
	err := f.enc.Encode(log)
	f.mu.Unlock()
	return err == nil
}
