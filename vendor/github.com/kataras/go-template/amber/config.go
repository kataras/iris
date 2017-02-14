package amber

// Config contains fields useful to configure this template engine
type Config struct {
	// Funcs for the html/template result, amber default funcs are not overrided so use it without worries
	Funcs map[string]interface{}
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{Funcs: make(map[string]interface{})}
}
