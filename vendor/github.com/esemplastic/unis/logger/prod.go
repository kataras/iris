package logger

// NewProd returns a new Logger which prints nothing.
func NewProd() Logger {
	return func(errorMessage string) {}
}
