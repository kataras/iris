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
	// FromAlias is the from part, if empty this is the first part before @ from the Username field
	FromAlias string
	// UseCommand enable it if you want to send e-mail with the mail command  instead of smtp
	//
	// Host,Port & Password will be ignored
	// ONLY FOR UNIX
	UseCommand bool
}

// DefaultMail returns the default configs for Mail
// returns just an empty Mail struct
func DefaultMail() Mail {
	return Mail{}
}

// IsValid returns true if the mail configs are valid
func (m Mail) IsValid() bool {
	return (m.Host != "" && m.Port > 0 && m.Username != "" && m.Password != "") || m.UseCommand
}
