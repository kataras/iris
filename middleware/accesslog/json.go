package accesslog

import (
	"io"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// JSON is a Formatter type for JSON logs.
type JSON struct {
	// Indent in spaces.
	// Note that, if set to > 0 then jsoniter is used instead of easyjson.
	Indent     string
	EscapeHTML bool

	jsoniter jsoniter.API
	ac       *AccessLog
}

// SetOutput creates the json encoder writes to the "dest".
// It's called automatically by the middleware when this Formatter is used.
func (f *JSON) SetOutput(dest io.Writer) {
	f.ac, _ = dest.(*AccessLog)
	if indentStep := strings.Count(f.Indent, " "); indentStep > 0 {
		f.jsoniter = jsoniter.Config{
			TagKey:        "json",
			IndentionStep: indentStep,
			EscapeHTML:    f.EscapeHTML,
			SortMapKeys:   true,
		}.Froze()
	}
}

// Format prints the logs in JSON format.
// Writes to the destination directly,
// locks on each Format call.
func (f *JSON) Format(log *Log) (bool, error) {
	if f.jsoniter != nil {
		b, err := f.jsoniter.Marshal(log)
		if err != nil {
			return true, err
		}
		f.ac.Write(append(b, newLine))
		return true, nil
	}

	err := f.writeEasyJSON(log)
	return true, err
}
