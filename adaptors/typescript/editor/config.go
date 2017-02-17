package editor

import (
	"github.com/imdario/mergo"
	"os"
)

// Default values for the configuration
const (
	DefaultPort = 4444
)

// Config the configs for the Editor plugin
type Config struct {
	// Hostname if empty used the iris server's hostname
	Hostname string
	// Port if 0 4444
	Port int
	// KeyFile the key file(ssl optional)
	KeyFile string
	// CertFile the cert file (ssl optional)
	CertFile string
	// WorkingDir if empty "./"
	WorkingDir string
	// Username defaults to empty, you should set this
	Username string
	// Password defaults to empty, you should set this
	Password string
	// DisableOutput set that to true if you don't care about alm-tools' messages
	// they are useful because that the default value is "false"
	DisableOutput bool
}

// DefaultConfig returns the default configs for the Editor plugin
func DefaultConfig() Config {
	// explicit
	return Config{
		Hostname:      "",
		Port:          4444,
		KeyFile:       "",
		CertFile:      "",
		WorkingDir:    "." + string(os.PathSeparator), // alm-tools should end with path separator.
		Username:      "",
		Password:      "",
		DisableOutput: false,
	}
}

// Merge merges the default with the given config and returns the result
func (c Config) Merge(cfg []Config) (config Config) {

	if len(cfg) > 0 {
		config = cfg[0]
		if err := mergo.Merge(&config, c); err != nil {
			if !c.DisableOutput {
				panic(err)
			}
		}
	} else {
		_default := c
		config = _default
	}

	return
}

// MergeSingle merges the default with the given config and returns the result
func (c Config) MergeSingle(cfg Config) (config Config) {

	config = cfg
	if err := mergo.Merge(&config, c); err != nil {
		if !c.DisableOutput {
			panic(err)
		}
	}

	return
}
