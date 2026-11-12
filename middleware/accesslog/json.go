package accesslog

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"io"

	"github.com/kataras/iris/v14/context/encoding"
	"github.com/kataras/iris/v14/core/pkg/jsonutil"
)

// JSON is a Formatter type for JSON logs.
type JSON struct {
	// Indent in spaces. When set the output is rendered multi-line.
	Indent     string
	EscapeHTML bool
	// HumanTime renders the "timestamp" field as Log.Now formatted with
	// Log.TimeFormat instead of the unix milliseconds integer.
	HumanTime bool

	ac   *AccessLog
	opts json.Options
}

// SetOutput creates the json encoder writes to the "dest".
// It's called automatically by the middleware when this Formatter is used.
func (f *JSON) SetOutput(dest io.Writer) {
	f.ac, _ = dest.(*AccessLog)

	// The framework marshal defaults carry what a log line needs: lenient
	// UTF-8 (a logged body with a stray non-UTF-8 byte must never fail to
	// render; mangled to U+FFFD) and Duration-as-nanos for Fields values.
	opts := []json.Options{jsonutil.DefaultMarshalOptions}
	if f.EscapeHTML {
		opts = append(opts, jsontext.EscapeForHTML(true))
	}
	if f.Indent != "" {
		opts = append(opts, jsontext.Multiline(true), jsontext.WithIndent(f.Indent))
	}
	f.opts = json.JoinOptions(opts...)
}

// jsonEntry is the wire view of a query/param/field entry: key and value
// only. Marshaling encoding.Entry directly would also leak its bookkeeping
// fields (Immutable, Sep) into every log line.
type jsonEntry struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// jsonLog is the wire view of a Log. One view for both compact and indented
// output — the previous implementation had two divergent code paths (a
// hand-edited easyjson writer and encoding/json on the Log itself, which
// disagreed on entry fields and needed a byte-rewrite hack for HumanTime).
type jsonLog struct {
	Timestamp     any         `json:"timestamp"` // int64 unix milliseconds, or a formatted string when HumanTime.
	Latency       int64       `json:"latency"`   // nanoseconds; independent of json/v2's time.Duration format policy.
	Code          int         `json:"code"`
	Method        string      `json:"method"`
	Path          string      `json:"path"`
	IP            string      `json:"ip,omitzero"`
	Query         []jsonEntry `json:"query,omitzero"`
	Params        []jsonEntry `json:"params,omitzero"`
	Fields        []jsonEntry `json:"fields,omitzero"`
	Request       string      `json:"request,omitzero"`
	Response      string      `json:"response,omitzero"`
	BytesReceived int         `json:"bytes_received,omitzero"`
	BytesSent     int         `json:"bytes_sent,omitzero"`
}

func makeJSONEntries[V any](entries encoding.Entries[V]) []jsonEntry {
	if len(entries) == 0 {
		return nil
	}

	out := make([]jsonEntry, len(entries))
	for i, entry := range entries {
		out[i] = jsonEntry{Key: entry.Key, Value: entry.Value}
	}

	return out
}

func makeJSONLog(log *Log, humanTime bool) jsonLog {
	var timestamp any = log.Timestamp
	if humanTime {
		timestamp = log.Now.Format(log.TimeFormat)
	}

	return jsonLog{
		Timestamp:     timestamp,
		Latency:       int64(log.Latency),
		Code:          log.Code,
		Method:        log.Method,
		Path:          log.Path,
		IP:            log.IP,
		Query:         makeJSONEntries(log.Query),
		Params:        makeJSONEntries(log.PathParams),
		Fields:        makeJSONEntries(log.Fields),
		Request:       log.Request,
		Response:      log.Response,
		BytesReceived: log.BytesReceived,
		BytesSent:     log.BytesSent,
	}
}

// Format prints the logs in JSON format.
// Writes to the destination directly,
// locks on each Format call.
func (f *JSON) Format(log *Log) (bool, error) {
	buf := f.ac.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer f.ac.bufPool.Put(buf)

	if err := json.MarshalWrite(buf, makeJSONLog(log, f.HumanTime), f.opts); err != nil {
		return true, err
	}
	buf.WriteByte(newLine)

	_, err := f.ac.Write(buf.Bytes())
	return true, err
}
