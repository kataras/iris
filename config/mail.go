package config

// Mail keeps the configs for mail sender service
type Mail struct {
	// Host is the server mail host, IP or address
	Host string
	// Port is the listening port
	Port int
	// Username is the auth username@domain.com for the sender
	Username string
	// Password is the auth password for the sender
	Password string
}

// DefaultMail returns the default configs for Mail
// returns just an empty Mail struct
func DefaultMail() Mail {
	return Mail{}
}
