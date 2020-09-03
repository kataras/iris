package redis

import (
	"crypto/tls"
	"errors"
	"time"

	"github.com/kataras/iris/v12/sessions"

	"github.com/kataras/golog"
)

const (
	// DefaultRedisNetwork the redis network option, "tcp".
	DefaultRedisNetwork = "tcp"
	// DefaultRedisAddr the redis address option, "127.0.0.1:6379".
	DefaultRedisAddr = "127.0.0.1:6379"
	// DefaultRedisTimeout the redis idle timeout option, time.Duration(30) * time.Second
	DefaultRedisTimeout = time.Duration(30) * time.Second
	// DefaultDelim ths redis delim option, "-".
	DefaultDelim = "-"
)

// Config the redis configuration used inside sessions
type Config struct {
	// Network protocol. Defaults to "tcp".
	Network string
	// Addr of a single redis server instance.
	// See "Clusters" field for clusters support.
	// Defaults to "127.0.0.1:6379".
	Addr string
	// Clusters a list of network addresses for clusters.
	// If not empty "Addr" is ignored.
	// Currently only Radix() Driver supports it.
	Clusters []string
	// Password string .If no password then no 'AUTH'. Defaults to "".
	Password string
	// If Database is empty "" then no 'SELECT'. Defaults to "".
	Database string
	// MaxActive. Defaults to 10.
	MaxActive int
	// Timeout for connect, write and read, defaults to 30 seconds, 0 means no timeout.
	Timeout time.Duration
	// Prefix "myprefix-for-this-website". Defaults to "".
	Prefix string
	// Delim the delimeter for the keys on the sessiondb. Defaults to "-".
	Delim string

	// TLSConfig will cause Dial to perform a TLS handshake using the provided
	// config. If is nil then no TLS is used.
	// See https://golang.org/pkg/crypto/tls/#Config
	TLSConfig *tls.Config

	// Driver supports `Redigo()` or `Radix()` go clients for redis.
	// Configure each driver by the return value of their constructors.
	//
	// Defaults to `Redigo()`.
	Driver Driver
}

// DefaultConfig returns the default configuration for Redis service.
func DefaultConfig() Config {
	return Config{
		Network:   DefaultRedisNetwork,
		Addr:      DefaultRedisAddr,
		Password:  "",
		Database:  "",
		MaxActive: 10,
		Timeout:   DefaultRedisTimeout,
		Prefix:    "",
		Delim:     DefaultDelim,
		TLSConfig: nil,
		Driver:    Redigo(),
	}
}

// Database the redis back-end session database for the sessions.
type Database struct {
	driver DatabaseDriver
}

var _ sessions.Database = (*Database)(nil)

// New returns a new redis database.
func New(cfg ...Config) *Database {
	c := DefaultConfig()
	var driver DatabaseDriver
	if len(cfg) > 0 {
		c = cfg[0]

		if c.Timeout < 0 {
			c.Timeout = DefaultRedisTimeout
		}

		if c.Network == "" {
			c.Network = DefaultRedisNetwork
		}

		if c.Addr == "" {
			c.Addr = DefaultRedisAddr
		}

		if c.MaxActive == 0 {
			c.MaxActive = 10
		}

		if c.Delim == "" {
			c.Delim = DefaultDelim
		}

		if c.Driver == nil {
			c.Driver = Redigo()
		}
	}
	if _, ok := c.Driver.(*RadixDriverHashed); ok {
		driver = DatabaseDriverHashed(c)
	} else {
		driver = DatabaseDriverString(c)
	}
	if err := driver.Connect(c); err != nil {
		panic(err)
	}

	db := &Database{driver: driver}
	return db
}

// Config returns the configuration for the redis server bridge, you can change them.
func (db *Database) Config() *Config {
	return db.driver.Config() // 6 Aug 2019 - keep that for no breaking change.
}

// SetLogger sets the logger once before server ran.
// By default the Iris one is injected.
func (db *Database) SetLogger(logger *golog.Logger) {
	db.driver.SetLogger(logger)
}

// Acquire receives a session's lifetime from the database,
// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
func (db *Database) Acquire(sid string, expires time.Duration) sessions.LifeTime {
	return db.driver.Acquire(sid, expires)
}

// OnUpdateExpiration will re-set the database's session's entry ttl.
// https://redis.io/commands/expire#refreshing-expires
func (db *Database) OnUpdateExpiration(sid string, newExpires time.Duration) error {
	return db.driver.OnUpdateExpiration(sid, newExpires)
}

func (db *Database) Set(sid string, lifetime *sessions.LifeTime, key string, value interface{}, immutable bool) {
	db.driver.Set(sid, lifetime, key, value, immutable)
	return
}

// Get retrieves a session value based on the key.
func (db *Database) Get(sid string, key string) (value interface{}) {
	return db.driver.Get(sid, key)
}

// Visit loops through all session keys and values.
func (db *Database) Visit(sid string, cb func(key string, value interface{})) {
	db.driver.Visit(sid, cb)
	return
}

// Len returns the length of the session's entries (keys).
func (db *Database) Len(sid string) (n int) {
	return db.driver.Len(sid)
}

// Delete removes a session key value based on its key.
func (db *Database) Delete(sid string, key string) (deleted bool) {
	return db.driver.Delete(sid, key)
}

// Clear removes all session key values but it keeps the session entry.
func (db *Database) Clear(sid string) {
	db.driver.Clear(sid)
}

// Release destroys the session, it clears and removes the session entry,
func (db *Database) Release(sid string) {
	db.driver.Release(sid)
}

// Close terminates the redis connection.
func (db *Database) Close() error {
	return db.driver.Close()
}

var (
	// ErrRedisClosed an error with message 'redis: already closed'
	ErrRedisClosed = errors.New("redis: already closed")
	// ErrKeyNotFound a type of error of non-existing redis keys.
	// The producers(the library) of this error will dynamically wrap this error(fmt.Errorf) with the key name.
	// Usage:
	// if err != nil && errors.Is(err, ErrKeyNotFound) {
	// [...]
	// }
	ErrKeyNotFound = errors.New("key not found")
)
