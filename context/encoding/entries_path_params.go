package encoding

// PathParams is a slice of Param.
// Empty if the route is static.
// It's used by the router to store the dynamic path parameters.
type PathParams = Entries[any] // no pointer to Param, as they are small and we don't want to allocate memory for each of them.

// DefaultPathParamTag is the default struct tag to look for path params.
// Defaults to "param".
const DefaultPathParamTag = "param"

// PathParamsDecoderOptions is the options for the path params decoder.
type PathParamsDecoderOptions = EntriesDecoderOptions[any]

// NewPathParamsDecoder returns a new path params decoder.
// It returns an error if T is not a pointer to a struct.
// The options can be empty.
// See `EntriesDecoderOptions` struct and its `DecodeEntries` method for more.
func NewPathParamsDecoder[T any](opts PathParamsDecoderOptions) (*EntriesDecoder[T, any], error) {
	return NewEntriesDecoder[T](opts.get(DefaultPathParamTag, DefaultPathFieldSetter))
}
