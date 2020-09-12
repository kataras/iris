package accesslog

import (
	"encoding/json"
	"io"
	"sync"
)

// JSON is a Formatter type for JSON logs.
type JSON struct {
	Prefix, Indent string
	EscapeHTML     bool

	enc         *json.Encoder
	mu          sync.Mutex
	lockEncoder bool
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
	f.lockEncoder = f.Prefix != "" || f.Indent != ""
	f.enc = enc
}

// Format prints the logs in JSON format.
// Writes to the destination directly,
// locks on each Format call.
func (f *JSON) Format(log *Log) (bool, error) {
	// f.mu.Lock()
	// ^ This lock is not required as the writer is
	// protected with mutex if necessary or configurated to do so.
	// However, if we navigate through the
	// internal encoding's source code we'll see that it
	// uses a field for its indent buffer,
	// therefore it's only useful when Prefix or Indent was not empty.
	if f.lockEncoder {
		f.mu.Lock()
	}
	err := f.enc.Encode(log)
	if f.lockEncoder {
		f.mu.Unlock()
	}

	return true, err
}
