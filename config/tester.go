package config

// Tester configuration
type Tester struct {
	ListeningAddr string
	ExplicitURL   bool
	Debug         bool
}

// DefaultTester returns the default configuration for a tester
// the ListeningAddr is used as virtual only when no running server is founded
func DefaultTester() Tester {
	return Tester{ListeningAddr: "iris-go.com:1993", ExplicitURL: false, Debug: false}
}
