package redis

import (
	"crypto/tls"
	"errors"
	"strings"
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
	c      Config
	logger *golog.Logger
}

var _ sessions.Database = (*Database)(nil)

// New returns a new redis database.
func New(cfg ...Config) *Database {
	c := DefaultConfig()
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

	if err := c.Driver.Connect(c); err != nil {
		panic(err)
	}

	db := &Database{c: c}
	_, err := db.c.Driver.PingPong()
	if err != nil {
		panic(err)
	}
	// runtime.SetFinalizer(db, closeDB)
	return db
}

// Config returns the configuration for the redis server bridge, you can change them.
func (db *Database) Config() *Config {
	return &db.c // 6 Aug 2019 - keep that for no breaking change.
}

// SetLogger sets the logger once before server ran.
// By default the Iris one is injected.
func (db *Database) SetLogger(logger *golog.Logger) {
	db.logger = logger
}

// Acquire receives a session's lifetime from the database,
// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
func (db *Database) Acquire(sid string, expires time.Duration) sessions.LifeTime {
	key := db.makeKey(sid, "")
	seconds, hasExpiration, found := db.c.Driver.TTL(key)
	if !found {
		// fmt.Printf("db.Acquire expires: %s. Seconds: %v\n", expires, expires.Seconds())
		// not found, create an entry with ttl and return an empty lifetime, session manager will do its job.
		if err := db.c.Driver.Set(key, sid, int64(expires.Seconds())); err != nil {
			db.logger.Debug(err)
		}

		return sessions.LifeTime{} // session manager will handle the rest.
	}

	if !hasExpiration {
		return sessions.LifeTime{}
	}

	return sessions.LifeTime{Time: time.Now().Add(time.Duration(seconds) * time.Second)}
}

// OnUpdateExpiration will re-set the database's session's entry ttl.
// https://redis.io/commands/expire#refreshing-expires
func (db *Database) OnUpdateExpiration(sid string, newExpires time.Duration) error {
	return db.c.Driver.UpdateTTLMany(db.makeKey(sid, ""), int64(newExpires.Seconds()))
}

func (db *Database) makeKey(sid, key string) string {
	if key == "" {
		return db.c.Prefix + sid
	}
	return db.c.Prefix + sid + db.c.Delim + key
}

// Set sets a key value of a specific session.
// Ignore the "immutable".
func (db *Database) Set(sid string, lifetime *sessions.LifeTime, key string, value interface{}, immutable bool) {
	valueBytes, err := sessions.DefaultTranscoder.Marshal(value)
	if err != nil {
		db.logger.Error(err)
		return
	}

	// fmt.Println("database.Set")
	// fmt.Printf("lifetime.DurationUntilExpiration(): %s. Seconds: %v\n", lifetime.DurationUntilExpiration(), lifetime.DurationUntilExpiration().Seconds())
	if err = db.c.Driver.Set(db.makeKey(sid, key), valueBytes, int64(lifetime.DurationUntilExpiration().Seconds())); err != nil {
		db.logger.Debug(err)
	}
}

// Get retrieves a session value based on the key.
func (db *Database) Get(sid string, key string) (value interface{}) {
	db.get(db.makeKey(sid, key), &value)
	return
}

func (db *Database) get(key string, outPtr interface{}) error {
	data, err := db.c.Driver.Get(key)
	if err != nil {
		// not found.
		return err
	}

	if err = sessions.DefaultTranscoder.Unmarshal(data.([]byte), outPtr); err != nil {
		db.logger.Debugf("unable to unmarshal value of key: '%s': %v", key, err)
		return err
	}

	return nil
}

func (db *Database) keys(sid string) []string {
	keys, err := db.c.Driver.GetKeys(db.makeKey(sid, ""))
	if err != nil {
		db.logger.Debugf("unable to get all redis keys of session '%s': %v", sid, err)
		return nil
	}

	return keys
}

// Visit loops through all session keys and values.
func (db *Database) Visit(sid string, cb func(key string, value interface{})) {
	keys := db.keys(sid)
	for _, key := range keys {
		var value interface{} // new value each time, we don't know what user will do in "cb".
		db.get(key, &value)
		key = strings.TrimPrefix(key, db.c.Prefix+sid+db.c.Delim)
		cb(key, value)
	}
}

// Len returns the length of the session's entries (keys).
func (db *Database) Len(sid string) (n int) {
	return len(db.keys(sid))
}

// Delete removes a session key value based on its key.
func (db *Database) Delete(sid string, key string) (deleted bool) {
	err := db.c.Driver.Delete(db.makeKey(sid, key))
	if err != nil {
		db.logger.Error(err)
	}
	return err == nil
}

// Clear removes all session key values but it keeps the session entry.
func (db *Database) Clear(sid string) {
	keys := db.keys(sid)
	for _, key := range keys {
		if err := db.c.Driver.Delete(key); err != nil {
			db.logger.Debugf("unable to delete session '%s' value of key: '%s': %v", sid, key, err)
		}
	}
}

// Release destroys the session, it clears and removes the session entry,
// session manager will create a new session ID on the next request after this call.
func (db *Database) Release(sid string) {
	// clear all $sid-$key.
	db.Clear(sid)
	// and remove the $sid.
	err := db.c.Driver.Delete(db.c.Prefix + sid)
	if err != nil {
		db.logger.Debugf("Database.Release.Driver.Delete: %s: %v", sid, err)
	}
}

// Close terminates the redis connection.
func (db *Database) Close() error {
	return closeDB(db)
}

func closeDB(db *Database) error {
	return db.c.Driver.CloseConnection()
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
