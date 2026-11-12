package types

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"

	"github.com/kataras/iris/v14/core/pkg/jsonutil"
)

// Merged holds a slice of JSON-serializable objects to be Merged together.
type Merged struct {
	Values []any
}

var _ json.Marshaler = Merged{}

// MarshalJSON implements the json.Marshaler interface for the merged type.
func (m Merged) MarshalJSON() ([]byte, error) {
	return MergeJSON(m.Values...)
}

// Merge takes any number of JSON-serializable objects and returns a json.Marshaler that combines the objects.
//
// Usage:
//
//	ctx.JSON(types.Merge(obj1, obj2, obj3))
func Merge(values ...any) json.Marshaler {
	return Merged{Values: values}
}

// MergeJSON merges a variadic number of values into a single JSON object.
// Each non-nil value must marshal to a JSON object; a string or []byte value
// is taken as raw JSON. Nil values and empty objects ("{}") contribute
// nothing. Values marshal with the framework's JSON policy (see
// core/pkg/jsonutil), so a struct renders the same here as through ctx.JSON.
//
// A member name present in more than one value appears once in the result,
// in the position of its first occurrence, with the value of its last
// occurrence: the later object wins. The output never carries duplicate
// names, so any strict decoder (the framework's own client included) can
// read it.
//
// Returns an error if any non-nil value does not marshal to a JSON object.
func MergeJSON(values ...any) ([]byte, error) {
	var m objectMerger

	for i, value := range values {
		if value == nil {
			continue
		}

		var (
			part []byte
			err  error
		)

		switch v := value.(type) {
		case string:
			part = []byte(v)
		case []byte:
			part = v
		default:
			part, err = json.Marshal(value, jsonutil.DefaultMarshalOptionsSlice...)
		}

		if err == nil {
			err = m.add(part)
		}

		if err != nil {
			return nil, fmt.Errorf("part: %d: %w", i+1, err)
		}
	}

	return m.bytes()
}

// objectMerger accumulates object members in first-seen order.
type objectMerger struct {
	names  []string
	values map[string]jsontext.Value
}

// add parses one JSON object and merges its members in; a name already
// present takes the new value and keeps its position.
func (m *objectMerger) add(part []byte) error {
	// The part itself may carry duplicate names (raw input, or a custom
	// marshaler); they resolve the same way, last one wins.
	dec := jsontext.NewDecoder(bytes.NewReader(part),
		jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true))

	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if kind := tok.Kind(); kind != '{' {
		return fmt.Errorf("value does not marshal to a JSON object (got %s)", kind)
	}

	for dec.PeekKind() != '}' {
		nameTok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		name := nameTok.String() // before the next read voids the token.

		val, err := dec.ReadValue()
		if err != nil {
			return err
		}

		if m.values == nil {
			m.values = make(map[string]jsontext.Value)
		}
		if _, exists := m.values[name]; !exists {
			m.names = append(m.names, name)
		}
		m.values[name] = val.Clone() // valid only until the next read.
	}

	_, err = dec.ReadToken() // the closing brace.
	return err
}

// bytes renders the merged object, compact (no whitespace), in first-seen
// name order.
func (m *objectMerger) bytes() ([]byte, error) {
	if len(m.names) == 0 {
		return []byte("{}"), nil
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, name := range m.names {
		if i > 0 {
			buf.WriteByte(',')
		}

		quoted, err := jsontext.AppendQuote(nil, name)
		if err != nil {
			return nil, fmt.Errorf("member %q: %w", name, err)
		}
		buf.Write(quoted)
		buf.WriteByte(':')
		buf.Write(m.values[name])
	}
	buf.WriteByte('}')

	return buf.Bytes(), nil
}
