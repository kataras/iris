package logger

// Config are the options of the logger middlweare
// contains 4 bools
// Status, IP, Method, Path
// if set to true then these will print
type Config struct {
	// Status displays status code (bool)
	Status bool
	// IP displays request's remote address (bool)
	IP bool
	// Method displays the http method (bool)
	Method bool
	// Path displays the request path (bool)
	Path bool
}

// DefaultConfig returns an options which all properties are true except EnableColors
func DefaultConfig() Config {
	return Config{true, true, true, true}
}
