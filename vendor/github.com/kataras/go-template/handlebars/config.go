package handlebars

// Config for handlebars template engine
type Config struct {
	// Helpers for Handlebars, you can register your own by raymond.RegisterHelper(name string, a interface{}) or RegisterHelpers(map[string]interface{})
	// or just fill this method, do not override it it is not nil by default (because of Iris' helpers (url and urlpath)
	Helpers map[string]interface{}
	Layout  string
}

// DefaultConfig returns the default configs for the handlebars template engine
func DefaultConfig() Config {
	return Config{
		Helpers: make(map[string]interface{}, 0),
		Layout:  "",
	}
}
