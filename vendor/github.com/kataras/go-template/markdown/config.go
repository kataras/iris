package markdown

// Config for markdown template engine
type Config struct {
	// Sanitize if true then returns safe html, default is false
	Sanitize bool
}

// DefaultConfig returns the default configs for the markdown template engine
func DefaultConfig() Config {
	return Config{
		Sanitize: false,
	}
}
