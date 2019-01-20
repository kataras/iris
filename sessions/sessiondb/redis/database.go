package redis

import (
	"runtime"
	"time"

	"github.com/kataras/golog"
	"github.com/kataras/iris/sessions"
	"github.com/kataras/iris/sessions/sessiondb/redis/service"
)

// Database the redis back-end session database for the sessions.
type Database struct {
	redis *service.Service
}

var _ sessions.Database = (*Database)(nil)

// New returns a new redis database.
func New(cfg ...service.Config) *Database {
	db := &Database{redis: service.New(cfg...)}
	db.redis.Connect()
	_, err := db.redis.PingPong()
	if err != nil {
		golog.Debugf("error connecting to redis: %v", err)
		return nil
	}
	runtime.SetFinalizer(db, closeDB)
	return db
}

// Config returns the configuration for the redis server bridge, you can change them.
func (db *Database) Config() *service.Config {
	return db.redis.Config
}

// Acquire receives a session's lifetime from the database,
// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
func (db *Database) Acquire(sid string, expires time.Duration) sessions.LifeTime {
	seconds, hasExpiration, found := db.redis.TTL(sid)
	if !found {
		// not found, create an entry with ttl and return an empty lifetime, session manager will do its job.
		if err := db.redis.Set(sid, sid, int64(expires.Seconds())); err != nil {
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
	return db.redis.UpdateTTLMany(sid, int64(newExpires.Seconds()))
}

const delim = "_"

func makeKey(sid, key string) string {
	return sid + delim + key
}

// Set sets a key value of a specific session.
// Ignore the "immutable".
func (db *Database) Set(sid string, lifetime sessions.LifeTime, key string, value interface{}, immutable bool) {
	valueBytes, err := sessions.DefaultTranscoder.Marshal(value)
	if err != nil {
		golog.Error(err)
		return
	}

	if err = db.redis.Set(makeKey(sid, key), valueBytes, int64(lifetime.DurationUntilExpiration().Seconds())); err != nil {
		golog.Debug(err)
	}
}

// Get retrieves a session value based on the key.
func (db *Database) Get(sid string, key string) (value interface{}) {
	db.get(makeKey(sid, key), &value)
	return
}

func (db *Database) get(key string, outPtr interface{}) {
	data, err := db.redis.Get(key)
	if err != nil {
		// not found.
		return
	}

	if err = sessions.DefaultTranscoder.Unmarshal(data.([]byte), outPtr); err != nil {
		golog.Debugf("unable to unmarshal value of key: '%s': %v", key, err)
	}
}

func (db *Database) keys(sid string) []string {
	keys, err := db.redis.GetKeys(sid + delim)
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
	err := db.redis.Delete(makeKey(sid, key))
	if err != nil {
		golog.Error(err)
	}
	return err == nil
}

// Clear removes all session key values but it keeps the session entry.
func (db *Database) Clear(sid string) {
	keys := db.keys(sid)
	for _, key := range keys {
		if err := db.redis.Delete(key); err != nil {
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
	db.redis.Delete(sid)
}

// Close terminates the redis connection.
func (db *Database) Close() error {
	return closeDB(db)
}

func closeDB(db *Database) error {
	return db.redis.CloseConnection()
}
