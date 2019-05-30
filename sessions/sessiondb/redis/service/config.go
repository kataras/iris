package service

import (
	"time"
)

const (
	// DefaultRedisNetwork the redis network option, "tcp".
	DefaultRedisNetwork = "tcp"
	// DefaultRedisAddr the redis address option, "127.0.0.1:6379".
	DefaultRedisAddr = "127.0.0.1:6379"
	// DefaultRedisIdleTimeout the redis idle timeout option, time.Duration(5) * time.Minute.
	DefaultRedisIdleTimeout = time.Duration(5) * time.Minute
	// DefaultDelim ths redis delim option, "-".
	DefaultDelim = "-"
)

// Config the redis configuration used inside sessions
type Config struct {
	// Network protocol. Defaults to "tcp".
	Network string
	// Addr of the redis server. Defaults to "127.0.0.1:6379".
	Addr string
	// Password string .If no password then no 'AUTH'. Defaults to "".
	Password string
	// If Database is empty "" then no 'SELECT'. Defaults to "".
	Database string
	// MaxIdle 0 no limit.
	MaxIdle int
	// MaxActive 0 no limit.
	MaxActive int
	// IdleTimeout time.Duration(5) * time.Minute.
	IdleTimeout time.Duration
	// Prefix "myprefix-for-this-website". Defaults to "".
	Prefix string
	// Delim the delimeter for the values. Defaults to "-".
	Delim string
}

// DefaultConfig returns the default configuration for Redis service.
func DefaultConfig() Config {
	return Config{
		Network:     DefaultRedisNetwork,
		Addr:        DefaultRedisAddr,
		Password:    "",
		Database:    "",
		MaxIdle:     0,
		MaxActive:   0,
		IdleTimeout: DefaultRedisIdleTimeout,
		Prefix:      "",
		Delim:       DefaultDelim,
	}
}
