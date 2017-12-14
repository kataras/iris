package django

import "github.com/flosch/pongo2"

type (
	// Value conversion for pongo2.Value
	Value pongo2.Value
	// Error conversion for pongo2.Error
	Error pongo2.Error
	// FilterFunction conversion for pongo2.FilterFunction
	FilterFunction func(in *Value, param *Value) (out *Value, err *Error)
)

// Config for django template engine
type Config struct {
	// Filters for pongo2, map[name of the filter] the filter function . The filters are auto register
	Filters map[string]FilterFunction
	// Globals share context fields between templates. https://github.com/flosch/pongo2/issues/35
	Globals map[string]interface{}
	// DebugTemplates enables template debugging.
	// The verbose error messages will appear in browser instead of quiet passes with error code
	DebugTemplates bool
}

// DefaultConfig returns the default configuration for the django template engine
func DefaultConfig() Config {
	return Config{
		Filters:        make(map[string]FilterFunction),
		Globals:        make(map[string]interface{}, 0),
		DebugTemplates: false,
	}
}
