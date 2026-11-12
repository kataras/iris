package encoding

// DefaultHeaderTag is the default struct tag for header.
// Defaults to "header".
const DefaultHeaderTag = "header"

// HeaderDecoderOptions is the options for the header decoder.
type HeaderDecoderOptions = EntriesDecoderOptions[[]string]

// NewHeaderDecoder returns a new header decoder.
// It returns an error if T is not a pointer to a struct.
// The options can be empty.
// See `HeaderDecoderOptions` struct and its `DecodeKeyValues` method for more.
func NewHeaderDecoder[T any](opts HeaderDecoderOptions) (*KeyValuesDecoder[T], error) {
	return NewKeyValuesDecoder[T](opts.get(DefaultHeaderTag, DefaultQueryFieldSetter))
}
