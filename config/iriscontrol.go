package config

import "github.com/imdario/mergo"

var (
	// DefaultUsername used for default (basic auth) username in IrisControl's & Editor's default configuration
	DefaultUsername = "iris"
	// DefaultPassword used for default (basic auth) password in IrisControl's & Editor's default configuration
	DefaultPassword = "admin!123"
)

// IrisControl the options which iris control needs
// contains the port (int) and authenticated users with their passwords (map[string]string)
type IrisControl struct {
	// Port the port
	Port int
	// Users the authenticated users, [username]password
	Users map[string]string
}

// DefaultIrisControl returns the default configs for IrisControl plugin
func DefaultIrisControl() IrisControl {
	users := make(map[string]string, 0)
	users[DefaultUsername] = DefaultPassword
	return IrisControl{4000, users}
}

// Merge merges the default with the given config and returns the result
func (c IrisControl) Merge(cfg []IrisControl) (config IrisControl) {

	if len(cfg) > 0 {
		config = cfg[0]
		mergo.Merge(&config, c)
	} else {
		_default := c
		config = _default
	}

	return
}
