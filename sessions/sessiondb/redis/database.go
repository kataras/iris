package redis

import (
	"time"

	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/sessions"

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
		Driver:    Redigo(),
	}
}

// Database the redis back-end session database for the sessions.
type Database struct {
	c Config
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
		golog.Debugf("error connecting to redis: %v", err)
		return nil
	}
	// runtime.SetFinalizer(db, closeDB)
	return db
}

// Config returns the configuration for the redis server bridge, you can change them.
func (db *Database) Config() *Config {
	return &db.c // 6 Aug 2019 - keep that for no breaking change.
}

// Acquire receives a session's lifetime from the database,
// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
func (db *Database) Acquire(sid string, expires time.Duration) sessions.LifeTime {
	seconds, hasExpiration, found := db.c.Driver.TTL(sid)
	if !found {
		// fmt.Printf("db.Acquire expires: %s. Seconds: %v\n", expires, expires.Seconds())
		// not found, create an entry with ttl and return an empty lifetime, session manager will do its job.
		if err := db.c.Driver.Set(sid, sid, int64(expires.Seconds())); err != nil {
			golog.Debug(err)
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
	return db.c.Driver.UpdateTTLMany(sid, int64(newExpires.Seconds()))
}

func (db *Database) makeKey(sid, key string) string {
	return sid + db.c.Delim + key
}

// Set sets a key value of a specific session.
// Ignore the "immutable".
func (db *Database) Set(sid string, lifetime sessions.LifeTime, key string, value interface{}, immutable bool) {
	valueBytes, err := sessions.DefaultTranscoder.Marshal(value)
	if err != nil {
		golog.Error(err)
		return
	}

	// fmt.Println("database.Set")
	// fmt.Printf("lifetime.DurationUntilExpiration(): %s. Seconds: %v\n", lifetime.DurationUntilExpiration(), lifetime.DurationUntilExpiration().Seconds())
	if err = db.c.Driver.Set(db.makeKey(sid, key), valueBytes, int64(lifetime.DurationUntilExpiration().Seconds())); err != nil {
		golog.Debug(err)
	}
}

// Get retrieves a session value based on the key.
func (db *Database) Get(sid string, key string) (value interface{}) {
	db.get(db.makeKey(sid, key), &value)
	return
}

func (db *Database) get(key string, outPtr interface{}) {
	data, err := db.c.Driver.Get(key)
	if err != nil {
		// not found.
		return
	}

	if err = sessions.DefaultTranscoder.Unmarshal(data.([]byte), outPtr); err != nil {
		golog.Debugf("unable to unmarshal value of key: '%s': %v", key, err)
	}
}

func (db *Database) keys(sid string) []string {
	keys, err := db.c.Driver.GetKeys(sid)
	if err != nil {
		golog.Debugf("unable to get all redis keys of session '%s': %v", sid, err)
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
		golog.Error(err)
	}
	return err == nil
}

// Clear removes all session key values but it keeps the session entry.
func (db *Database) Clear(sid string) {
	keys := db.keys(sid)
	for _, key := range keys {
		if err := db.c.Driver.Delete(key); err != nil {
			golog.Debugf("unable to delete session '%s' value of key: '%s': %v", sid, key, err)
		}
	}
}

// Release destroys the session, it clears and removes the session entry,
// session manager will create a new session ID on the next request after this call.
func (db *Database) Release(sid string) {
	// clear all $sid-$key.
	db.Clear(sid)
	// and remove the $sid.
	db.c.Driver.Delete(sid)
}

// Close terminates the redis connection.
func (db *Database) Close() error {
	return closeDB(db)
}

func closeDB(db *Database) error {
	return db.c.Driver.CloseConnection()
}

var (
	// ErrRedisClosed an error with message 'Redis is already closed'
	ErrRedisClosed = errors.New("Redis is already closed")
	// ErrKeyNotFound an error with message 'Key $thekey doesn't found'
	ErrKeyNotFound = errors.New("Key '%s' doesn't found")
)
