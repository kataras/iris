// Package jsonutil is the single source of the framework's encoding/json/v2
// policy. Every Iris package that marshals or unmarshals JSON — the Context
// writers and readers, the rest/errors renderers, the sse and ai encoders,
// the accesslog formatter and the HTTP client — composes its option set from
// the defaults here, so the two halves of the framework can never disagree
// about what its JSON looks like on the wire.
//
// The package imports only the standard library on purpose: client/ ships
// with a deliberately slim dependency graph and still needs the same policy.
package jsonutil

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"strconv"
	"sync"
	"time"
)

// MarshalDurationAsNanos and UnmarshalDurationFromNanos preserve the old
// encoding/json representation of time.Duration — a plain int64 nanosecond
// count. encoding/json/v2 refuses to handle time.Duration without an explicit
// choice ("no default representation"), which would turn every struct
// carrying a Duration into a runtime error.
var (
	MarshalDurationAsNanos = json.WithMarshalers(json.MarshalFunc(func(d time.Duration) ([]byte, error) {
		return strconv.AppendInt(nil, int64(d), 10), nil
	}))

	UnmarshalDurationFromNanos = json.WithUnmarshalers(json.UnmarshalFunc(func(b []byte, d *time.Duration) error {
		i, err := strconv.ParseInt(string(b), 10, 64)
		if err != nil {
			return err
		}
		*d = time.Duration(i)
		return nil
	}))
)

// DefaultMarshalOptions preserve the parts of the old encoding/json output
// that responses depend on: sorted map keys (byte-stable output for iris.Map
// bodies, ETags and golden tests), invalid UTF-8 mangled to U+FFFD instead
// of failing the response (a stray byte in a DB-sourced string must not turn
// a working endpoint into a 500) and time.Duration as int64 nanoseconds.
//
// Duplicate object names are rejected on output, the v2 default, matching
// DefaultUnmarshalOptions: a body the framework writes must be readable by
// its own client, and the read-side JSON-smuggling defense has no write-side
// hole. rest/types.Merged merges overlapping keys instead of emitting both.
var DefaultMarshalOptions = json.JoinOptions(
	json.Deterministic(true),
	jsontext.AllowInvalidUTF8(true),
	MarshalDurationAsNanos,
)

// DefaultMarshalOptionsSlice is DefaultMarshalOptions ready to pass
// variadically with zero allocations. Never append to it.
var DefaultMarshalOptionsSlice = []json.Options{DefaultMarshalOptions}

// DefaultUnmarshalOptions preserve the old encoding/json input behavior the
// framework keeps on purpose: case-insensitive member matching (deployed
// clients bind under the old rules) and time.Duration from int64 nanoseconds.
// The v2 strictness that IS kept — rejecting duplicate names and invalid
// UTF-8 — needs no option; it is the v2 default.
//
// Case-insensitive matching is v1's strings.EqualFold, delimiters included.
// Plain v2 folding also ignores '_' and '-', so a camelCased "vO2PEAK" matches
// both a "VO2PEAK" and a "VO2_PEAK" field and the whole body fails as an
// ambiguous name; MatchCaseSensitiveDelimiter keeps the two apart.
var DefaultUnmarshalOptions = json.JoinOptions(
	UnmarshalDurationFromNanos,
	json.MatchCaseInsensitiveNames(true),
	jsonv1.MatchCaseSensitiveDelimiter(true),
)

// DefaultUnmarshalOptionsSlice is DefaultUnmarshalOptions ready to pass
// variadically with zero allocations. Never append to it.
var DefaultUnmarshalOptionsSlice = []json.Options{DefaultUnmarshalOptions}

// maxPooledBufferCap keeps a rare huge payload from parking its backing
// array in the pool for the process lifetime.
const maxPooledBufferCap = 1 << 16

var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

// AcquireBuffer returns an empty scratch buffer for marshaling.
// Release it with ReleaseBuffer once its bytes have been fully consumed —
// the caller must not retain a slice of its contents past the release.
func AcquireBuffer() *bytes.Buffer {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// ReleaseBuffer returns a buffer to the pool, dropping oversized ones.
func ReleaseBuffer(buf *bytes.Buffer) {
	if buf.Cap() <= maxPooledBufferCap {
		bufPool.Put(buf)
	}
}
