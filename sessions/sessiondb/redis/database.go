package redis

import (
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/kataras/iris/v12/core/memstore"
	"github.com/kataras/iris/v12/sessions"

	"github.com/kataras/golog"
)

const (
	// DefaultRedisNetwork the redis network option, "tcp".
	DefaultRedisNetwork = "tcp"
	// DefaultRedisAddr the redis address option, "127.0.0.1:6379".
	DefaultRedisAddr = "127.0.0.1:6379"
	// DefaultRedisTimeout the redis idle timeout option, time.Duration(30) * time.Second.
	DefaultRedisTimeout = time.Duration(30) * time.Second
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
	// If not empty "Addr" is ignored and Redis clusters feature is used instead.
	// Note that this field is ignored when setgging a custom `GoRedisClient`.
	Clusters []string
	// Use the specified Username to authenticate the current connection
	// with one of the connections defined in the ACL list when connecting
	// to a Redis 6.0 instance, or greater, that is using the Redis ACL system.
	Username string
	// Optional password. Must match the password specified in the
	// requirepass server configuration option (if connecting to a Redis 5.0 instance, or lower),
	// or the User Password when connecting to a Redis 6.0 instance, or greater,
	// that is using the Redis ACL system.
	Password string
	// If Database is empty "" then no 'SELECT'. Defaults to "".
	Database string
	// Maximum number of socket connections.
	// Default is 10 connections per every CPU as reported by runtime.NumCPU.
	MaxActive int
	// Timeout for connect, write and read, defaults to 30 seconds, 0 means no timeout.
	Timeout time.Duration
	// Prefix "myprefix-for-this-website". Defaults to "".
	Prefix string

	// TLSConfig will cause Dial to perform a TLS handshake using the provided
	// config. If is nil then no TLS is used.
	// See https://golang.org/pkg/crypto/tls/#Config
	TLSConfig *tls.Config

	// A Driver should support be a go client for redis communication.
	// It can be set to a custom one or a mock one (for testing).
	//
	// Defaults to `GoRedis()`.
	Driver Driver
}

// DefaultConfig returns the default configuration for Redis service.
func DefaultConfig() Config {
	return Config{
		Network:   DefaultRedisNetwork,
		Addr:      DefaultRedisAddr,
		Username:  "",
		Password:  "",
		Database:  "",
		MaxActive: 10,
		Timeout:   DefaultRedisTimeout,
		Prefix:    "",
		TLSConfig: nil,
		Driver:    GoRedis(),
	}
}

// Database the redis back-end session database for the sessions.
type Database struct {
	c      Config
	logger *golog.Logger
}

var _ sessions.Database = (*Database)(nil)

// New returns a new redis sessions database.
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

		if c.Driver == nil {
			c.Driver = GoRedis()
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

// SetLogger sets the logger once before server ran.
// By default the Iris one is injected.
func (db *Database) SetLogger(logger *golog.Logger) {
	db.logger = logger
}

func (db *Database) makeSID(sid string) string {
	return db.c.Prefix + sid
}

// SessionIDKey the session ID stored to the redis session itself.
const SessionIDKey = "session_id"

// Acquire receives a session's lifetime from the database,
// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
func (db *Database) Acquire(sid string, expires time.Duration) memstore.LifeTime {
	sidKey := db.makeSID(sid)
	if !db.c.Driver.Exists(sidKey) {
		if err := db.Set(sid, SessionIDKey, sid, 0, false); err != nil {
			db.logger.Debug(err)
		} else if expires > 0 {
			if err := db.c.Driver.UpdateTTL(sidKey, expires); err != nil {
				db.logger.Debug(err)
			}
		}

		return memstore.LifeTime{} // session manager will handle the rest.
	}

	untilExpire := db.c.Driver.TTL(sidKey)
	return memstore.LifeTime{Time: time.Now().Add(untilExpire)}
}

// OnUpdateExpiration will re-set the database's session's entry ttl.
// https://redis.io/commands/expire#refreshing-expires
func (db *Database) OnUpdateExpiration(sid string, newExpires time.Duration) error {
	return db.c.Driver.UpdateTTL(db.makeSID(sid), newExpires)
}

// Set sets a key value of a specific session.
// Ignore the "immutable".
func (db *Database) Set(sid string, key string, value interface{}, _ time.Duration, _ bool) error {
	valueBytes, err := sessions.DefaultTranscoder.Marshal(value)
	if err != nil {
		db.logger.Error(err)
		return err
	}

	if err = db.c.Driver.Set(db.makeSID(sid), key, valueBytes); err != nil {
		db.logger.Debug(err)
		return err
	}

	return nil
}

// Get retrieves a session value based on the key.
func (db *Database) Get(sid string, key string) (value interface{}) {
	if err := db.Decode(sid, key, &value); err == nil {
		return value
	}

	return nil
}

// Decode binds the "outPtr" to the value associated to the provided "key".
func (db *Database) Decode(sid, key string, outPtr interface{}) error {
	sidKey := db.makeSID(sid)
	data, err := db.c.Driver.Get(sidKey, key)
	if err != nil {
		// not found.
		return err
	}

	if err = db.decodeValue(data, outPtr); err != nil {
		db.logger.Debugf("unable to unmarshal value of key: '%s%s': %v", sid, key, err)
		return err
	}

	return nil
}

func (db *Database) decodeValue(val interface{}, outPtr interface{}) error {
	if val == nil {
		return nil
	}

	switch data := val.(type) {
	case []byte:
		// this is the most common type, as we save all values as []byte,
		// the only exception is where the value is string on HGetAll command.
		return sessions.DefaultTranscoder.Unmarshal(data, outPtr)
	case string:
		return sessions.DefaultTranscoder.Unmarshal([]byte(data), outPtr)
	default:
		return fmt.Errorf("unknown value type of %T", data)
	}
}

func (db *Database) keys(fullSID string) []string {
	keys, err := db.c.Driver.GetKeys(fullSID)
	if err != nil {
		db.logger.Debugf("unable to get all redis keys of session '%s': %v", fullSID, err)
		return nil
	}

	return keys
}

// Visit loops through all session keys and values.
func (db *Database) Visit(sid string, cb func(key string, value interface{})) error {
	kv, err := db.c.Driver.GetAll(db.makeSID(sid))
	if err != nil {
		return err
	}

	for k, v := range kv {
		var value interface{} // new value each time, we don't know what user will do in "cb".
		if err = db.decodeValue(v, &value); err != nil {
			db.logger.Debugf("unable to decode %s:%s: %v", sid, k, err)
			return err
		}

		cb(k, value)
	}

	return nil
}

// Len returns the length of the session's entries (keys).
func (db *Database) Len(sid string) int {
	return db.c.Driver.Len(sid)
}

// Delete removes a session key value based on its key.
func (db *Database) Delete(sid string, key string) (deleted bool) {
	err := db.c.Driver.Delete(db.makeSID(sid), key)
	if err != nil {
		db.logger.Error(err)
	}
	return err == nil
}

// Clear removes all session key values but it keeps the session entry.
func (db *Database) Clear(sid string) error {
	sid = db.makeSID(sid)
	keys := db.keys(sid)
	for _, key := range keys {
		if key == SessionIDKey {
			continue
		}
		if err := db.c.Driver.Delete(sid, key); err != nil {
			db.logger.Debugf("unable to delete session '%s' value of key: '%s': %v", sid, key, err)
			return err
		}
	}

	return nil
}

// Release destroys the session, it clears and removes the session entry,
// session manager will create a new session ID on the next request after this call.
func (db *Database) Release(sid string) error {
	err := db.c.Driver.Delete(db.makeSID(sid), "")
	if err != nil {
		db.logger.Debugf("Database.Release.Driver.Delete: %s: %v", sid, err)
	}

	return err
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
