package sessions

import "encoding/json"

type (
	// Marshaler is the common marshaler interface, used by transcoder.
	Marshaler interface {
		Marshal(interface{}) ([]byte, error)
	}
	// Unmarshaler is the common unmarshaler interface, used by transcoder.
	Unmarshaler interface {
		Unmarshal([]byte, interface{}) error
	}
	// Transcoder is the interface that transcoders should implement, it includes just the `Marshaler` and the `Unmarshaler`.
	Transcoder interface {
		Marshaler
		Unmarshaler
	}
)

// DefaultTranscoder is the default transcoder across databases, it's the JSON by default.
// Change it if you want a different serialization/deserialization inside your session databases (when `UseDatabase` is used).
var DefaultTranscoder Transcoder = defaultTranscoder{}

type defaultTranscoder struct{}

func (d defaultTranscoder) Marshal(value interface{}) ([]byte, error) {
	if tr, ok := value.(Marshaler); ok {
		return tr.Marshal(value)
	}

	if jsonM, ok := value.(json.Marshaler); ok {
		return jsonM.MarshalJSON()
	}

	return json.Marshal(value)
}

func (d defaultTranscoder) Unmarshal(b []byte, outPtr interface{}) error {
	if tr, ok := outPtr.(Unmarshaler); ok {
		return tr.Unmarshal(b, outPtr)
	}

	if jsonUM, ok := outPtr.(json.Unmarshaler); ok {
		return jsonUM.UnmarshalJSON(b)
	}

	return json.Unmarshal(b, outPtr)
}
