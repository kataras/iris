package config

// Tester configuration
type Tester struct {
	Debug         bool
	ListeningAddr string
}

// DefaultTester returns the default configuration for a tester
// the ListeningAddr is used as virtual only when no running server is founded
func DefaultTester() Tester {
	return Tester{Debug: false, ListeningAddr: "iris-go.com:1993"}
}
