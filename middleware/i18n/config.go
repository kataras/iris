package i18n

// Config the i18n options
type Config struct {
	// Default set it if you want a default language
	//
	// Checked: Configuration state, not at runtime
	Default string
	// URLParameter is the name of the url parameter which the language can be indentified
	//
	// Checked: Serving state, runtime
	URLParameter string
	// Languages is a map[string]string which the key is the language i81n and the value is the file location
	//
	// Example of key is: 'en-US'
	// Example of value is: './locales/en-US.ini'
	Languages map[string]string
}
