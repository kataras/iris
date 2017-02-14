package html

// Config for html template engine
type Config struct {
	Left        string
	Right       string
	Layout      string
	Funcs       map[string]interface{}
	LayoutFuncs map[string]interface{}
}

// DefaultConfig returns the default configs for the html template engine
func DefaultConfig() Config {
	return Config{
		Left:        "{{",
		Right:       "}}",
		Layout:      "",
		Funcs:       make(map[string]interface{}, 0),
		LayoutFuncs: make(map[string]interface{}, 0),
	}
}
