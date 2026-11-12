package encoding

// ContextValues is a slice of Entry.
// It's used by the Context to store custom key-values for middleware communication.
type ContextValues = Entries[any]

// DefaultContextValueTag is the default struct tag to look for context values.
// Defaults to "context".
const DefaultContextValueTag = "context"

// ContextValuesDecoderOptions is the options for the context values decoder.
type ContextValuesDecoderOptions = EntriesDecoderOptions[any]

// NewContextValuesDecoder returns a new context values decoder.
// It returns an error if T is not a pointer to a struct.
// The options can be empty.
// See `EntriesDecoderOptions` struct and its `DecodeEntries` method for more.
func NewContextValuesDecoder[T any](opts ContextValuesDecoderOptions) (*EntriesDecoder[T, any], error) {
	return NewEntriesDecoder[T](opts.get(DefaultContextValueTag, DefaultPathFieldSetter))
}
