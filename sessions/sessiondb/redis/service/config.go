package service

import (
	"time"
)

const (
	// DefaultRedisNetwork the redis network option, "tcp"
	DefaultRedisNetwork = "tcp"
	// DefaultRedisAddr the redis address option, "127.0.0.1:6379"
	DefaultRedisAddr = "127.0.0.1:6379"
	// DefaultRedisIdleTimeout the redis idle timeout option, time.Duration(5) * time.Minute
	DefaultRedisIdleTimeout = time.Duration(5) * time.Minute
)

// Config the redis configuration used inside sessions
type Config struct {
	// Network "tcp"
	Network string
	// Addr "127.0.0.1:6379"
	Addr string
	// Password string .If no password then no 'AUTH'. Default ""
	Password string
	// If Database is empty "" then no 'SELECT'. Default ""
	Database string
	// MaxIdle 0 no limit
	MaxIdle int
	// MaxActive 0 no limit
	MaxActive int
	// IdleTimeout  time.Duration(5) * time.Minute
	IdleTimeout time.Duration
	// Prefix "myprefix-for-this-website". Default ""
	Prefix string
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
	}
}
