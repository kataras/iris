package encoding

// DefaultFormTag is the default struct tag for form.
// Defaults to "form".
const DefaultFormTag = "form"

// FormDecoderOptions is the options for the form decoder.
// See `NewFormDecoder` package-level function for more.
type FormDecoderOptions = EntriesDecoderOptions[[]string]

// NewFormDecoder returns a new form decoder.
// It returns an error if T is not a pointer to a struct.
// The options can be empty.
// See `FormDecoderOptions` struct and its `DecodeKeyValues` method for more.
func NewFormDecoder[T any](opts FormDecoderOptions) (*KeyValuesDecoder[T], error) {
	return NewKeyValuesDecoder[T](opts.get(DefaultFormTag, DefaultQueryFieldSetter))
}
