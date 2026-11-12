package encoding

// DefaultQueryTag is the default struct tag for url query.
// Defaults to "url".
const DefaultQueryTag = "url"

// QueryDecoderOptions is the options for the url query decoder.
// See `NewQueryDecoder` package-level function for more.
type QueryDecoderOptions = EntriesDecoderOptions[[]string]

// NewQueryDecoder returns a new url query decoder.
// It returns an error if T is not a pointer to a struct.
// The options can be empty.
// See `QueryDecoderOptions` struct and its `DecodeKeyValues` method for more.
func NewQueryDecoder[T any](opts QueryDecoderOptions) (*KeyValuesDecoder[T], error) {
	return NewKeyValuesDecoder[T](opts.get(DefaultQueryTag, DefaultQueryFieldSetter))
}
