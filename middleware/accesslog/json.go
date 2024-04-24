package accesslog

import (
	"bytes"
	"io"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// JSON is a Formatter type for JSON logs.
type JSON struct {
	// Indent in spaces.
	// Note that, if set to > 0 then jsoniter is used instead of easyjson.
	Indent     string
	EscapeHTML bool
	HumanTime  bool

	jsoniter jsoniter.API
	ac       *AccessLog
}

// SetOutput creates the json encoder writes to the "dest".
// It's called automatically by the middleware when this Formatter is used.
func (f *JSON) SetOutput(dest io.Writer) {
	f.ac, _ = dest.(*AccessLog)
	if indentStep := strings.Count(f.Indent, " "); indentStep > 0 {
		// Note that: indent setting should always be spaces
		// as the jsoniter does not support other chars.
		f.jsoniter = jsoniter.Config{
			TagKey:        "json",
			IndentionStep: indentStep,
			EscapeHTML:    f.EscapeHTML,
			SortMapKeys:   true,
		}.Froze()
	}
}

var (
	timestampKeyB        = []byte(`"timestamp":`)
	timestampKeyIndentB  = append(timestampKeyB, ' ')
	timestampKeyVB       = append(timestampKeyB, '0')
	timestampIndentKeyVB = append(timestampKeyIndentB, '0')
)

// Format prints the logs in JSON format.
// Writes to the destination directly,
// locks on each Format call.
func (f *JSON) Format(log *Log) (bool, error) {
	if f.jsoniter != nil {
		if f.HumanTime {
			// 1. Don't write the unix timestamp,
			// key will be visible though as we don't omit the field.
			log.Timestamp = 0
		}

		b, err := f.jsoniter.Marshal(log)
		if err != nil {
			return true, err
		}

		if f.HumanTime {
			// 2. Get the time text based on the configuration.
			t := log.Now.Format(log.TimeFormat)

			// 3. Find the "timestamp:$indent"
			// and set it to the text one.
			var (
				oldT  []byte
				tsKey []byte
			)

			if f.Indent != "" {
				oldT = timestampIndentKeyVB
				tsKey = timestampKeyIndentB
			} else {
				oldT = timestampKeyVB
				tsKey = timestampKeyB
			}

			newT := append(tsKey, strconv.Quote(t)...)
			b = bytes.Replace(b, oldT, newT, 1)
		}

		f.ac.Write(append(b, newLine))
		return true, nil
	}

	err := f.writeEasyJSON(log)
	return true, err
}
