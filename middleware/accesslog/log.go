package accesslog

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/memstore"
)

// Log represents the log data specifically for the accesslog middleware.
//easyjson:json
type Log struct {
	// The AccessLog instance this Log was created of.
	Logger *AccessLog `json:"-" yaml:"-" toml:"-"`

	// The time the log is created.
	Now time.Time `json:"-" yaml:"-" toml:"-"`
	// TimeFormat selected to print the Time as string,
	// useful on Template Formatter.
	TimeFormat string `json:"-" yaml:"-" toml:"-"`
	// Timestamp the Now's unix timestamp (milliseconds).
	Timestamp int64 `json:"timestamp" csv:"timestamp"`

	// Request-Response latency.
	Latency time.Duration `json:"latency" csv:"latency"`
	// The response status code.
	Code int `json:"code" csv:"code"`
	// Init request's Method and Path.
	Method string `json:"method" csv:"method"`
	Path   string `json:"path" csv:"path"`
	// The Remote Address.
	IP string `json:"ip,omitempty" csv:"ip,omitempty"`
	// Sorted URL Query arguments.
	Query []memstore.StringEntry `json:"query,omitempty" csv:"query,omitempty"`
	// Dynamic path parameters.
	PathParams memstore.Store `json:"params,omitempty" csv:"params,omitempty"`
	// Fields any data information useful to represent this Log.
	Fields memstore.Store `json:"fields,omitempty" csv:"fields,omitempty"`
	// The Request and Response raw bodies.
	// If they are escaped (e.g. JSON),
	// A third-party software can read it through:
	// data, _ := strconv.Unquote(log.Request)
	// err := json.Unmarshal([]byte(data), &customStruct)
	Request  string `json:"request,omitempty" csv:"request,omitempty"`
	Response string `json:"response,omitempty" csv:"response,omitempty"`
	//  The actual number of bytes received and sent on the network (headers + body or body only).
	BytesReceived int `json:"bytes_received,omitempty" csv:"bytes_received,omitempty"`
	BytesSent     int `json:"bytes_sent,omitempty" csv:"bytes_sent,omitempty"`

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
	return parseRequestValues(l.Code, l.PathParams, l.Query, l.Fields)
}

// BytesReceivedLine returns the formatted bytes received length.
func (l *Log) BytesReceivedLine() string {
	if !l.Logger.BytesReceived && !l.Logger.BytesReceivedBody {
		return ""
	}
	return formatBytes(l.BytesReceived)
}

// BytesSentLine returns the formatted bytes sent length.
func (l *Log) BytesSentLine() string {
	if !l.Logger.BytesSent && !l.Logger.BytesSentBody {
		return ""
	}
	return formatBytes(l.BytesSent)
}

func formatBytes(b int) string {
	if b <= 0 {
		return "0 B"
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

func parseRequestValues(code int, pathParams memstore.Store, query []memstore.StringEntry, fields memstore.Store) (requestValues string) {
	var buf strings.Builder

	if !context.StatusCodeNotSuccessful(code) {
		// collect path parameters on a successful request-response only.
		for _, entry := range pathParams {
			buf.WriteString(entry.Key)
			buf.WriteByte('=')
			buf.WriteString(fmt.Sprintf("%v", entry.ValueRaw))
			buf.WriteByte(' ')
		}
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

type (
	// Formatter is responsible to print a Log to the accesslog's writer.
	Formatter interface {
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

	// Flusher can be implemented by a Formatter
	// to call its Flush method on AccessLog.Close
	// and on panic errors.
	Flusher interface{ Flush() error }
)

var (
	_ Formatter = (*JSON)(nil)
	_ Formatter = (*Template)(nil)
	_ Formatter = (*CSV)(nil)
)
