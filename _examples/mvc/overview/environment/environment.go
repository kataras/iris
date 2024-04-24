package environment

import (
	"os"
	"strings"
)

// Available environments example.
const (
	PROD Env = "production"
	DEV  Env = "development"
)

// Env is the environment type.
type Env string

// String just returns the string representation of the Env.
func (e Env) String() string {
	return string(e)
}

// ReadEnv returns the environment of the system environment variable of "key".
// Returns the "def" if not found.
// Reports a panic message if the environment variable found
// but the Env is unknown.
func ReadEnv(key string, def Env) Env {
	v := Getenv(key, def.String())
	if v == "" {
		return def
	}

	env := Env(strings.ToLower(v))
	switch env {
	case PROD, DEV: // allowed.
	default:
		panic("unexpected environment " + v)
	}

	return env
}

// Getenv returns the value of a system environment variable "key".
// Defaults to "def" if not found.
func Getenv(key string, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}
