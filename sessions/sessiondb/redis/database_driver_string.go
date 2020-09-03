package redis

import (
	"github.com/kataras/golog"
	"github.com/kataras/iris/v12/sessions"
	"strings"
	"time"
)

type DatabaseString struct {
	c      Config
	logger *golog.Logger
}

// Config returns the configuration for the redis server bridge, you can change them.
func (db *DatabaseString) Config() *Config {
	return &db.c // 6 Aug 2019 - keep that for no breaking change.
}

func (db *DatabaseString) SetLogger(logger *golog.Logger) {
	db.logger = logger
}

func (db *DatabaseString) Connect(c Config) error {
	err := db.c.Driver.Connect(c)
	if err != nil {
		return err
	}

	_, err = db.c.Driver.PingPong()
	return err
}

func MakeKey(sid, key, prefix, delim string) string {
	if key == "" {
		return prefix + sid
	}
	return prefix + sid + delim + key
}

// Acquire receives a session's lifetime from the database,
// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
func (db *DatabaseString) Acquire(sid string, expires time.Duration) sessions.LifeTime {
	key := MakeKey(sid, "", db.c.Prefix, db.c.Delim)
	seconds, hasExpiration, found := db.c.Driver.TTL(key)
	if !found {
		// fmt.Printf("db.Acquire expires: %s. Seconds: %v\n", expires, expires.Seconds())
		// not found, create an entry with ttl and return an empty lifetime, session manager will do its job.
		if err := db.c.Driver.Set(sid, "", sid, int64(expires.Seconds())); err != nil {
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
func (db *DatabaseString) OnUpdateExpiration(sid string, newExpires time.Duration) error {
	return db.c.Driver.UpdateTTLMany(MakeKey(sid, "", db.c.Prefix, db.c.Delim), int64(newExpires.Seconds()))
}

// Set sets a key value of a specific session.
// Ignore the "immutable".
func (db *DatabaseString) Set(sid string, lifetime *sessions.LifeTime, key string, value interface{}, immutable bool) {
	valueBytes, err := sessions.DefaultTranscoder.Marshal(value)
	if err != nil {
		db.logger.Error(err)
		return
	}

	// fmt.Println("database.Set")
	// fmt.Printf("lifetime.DurationUntilExpiration(): %s. Seconds: %v\n", lifetime.DurationUntilExpiration(), lifetime.DurationUntilExpiration().Seconds())
	if err = db.c.Driver.Set(sid, key, valueBytes, int64(lifetime.DurationUntilExpiration().Seconds())); err != nil {
		db.logger.Debug(err)
	}
}

func (db *DatabaseString) get(key string, outPtr interface{}) error {
	data, err := db.c.Driver.GetByKey(key)
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

func (db *DatabaseString) keys(sid string) []string {
	keys, err := db.c.Driver.GetKeys(MakeKey(sid, "", db.c.Prefix, db.c.Delim))
	if err != nil {
		db.logger.Debugf("unable to get all redis keys of session '%s': %v", sid, err)
		return nil
	}

	return keys
}

// Get retrieves a session value based on the key.
func (db *DatabaseString) Get(sid string, key string) (value interface{}) {
	db.get(MakeKey(sid, key, db.c.Prefix, db.c.Delim), &value)
	return
}

// Visit loops through all session keys and values.
func (db *DatabaseString) Visit(sid string, cb func(key string, value interface{})) {
	keys := db.keys(sid)
	for _, key := range keys {
		var value interface{} // new value each time, we don't know what user will do in "cb".
		db.get(key, &value)
		key = strings.TrimPrefix(key, db.c.Prefix+sid+db.c.Delim)
		cb(key, value)
	}
}

// Len returns the length of the session's entries (keys).
func (db *DatabaseString) Len(sid string) (n int) {
	return len(db.keys(sid))
}

// Delete removes a session key value based on its key.
func (db *DatabaseString) Delete(sid string, key string) (deleted bool) {
	err := db.c.Driver.DeleteByKey(MakeKey(sid, key, db.c.Prefix, db.c.Delim))
	if err != nil {
		db.logger.Error(err)
	}
	return err == nil
}

// Clear removes all session key values but it keeps the session entry.
func (db *DatabaseString) Clear(sid string) {
	keys := db.keys(sid)
	for _, key := range keys {
		if err := db.c.Driver.DeleteByKey(key); err != nil {
			db.logger.Debugf("unable to delete session '%s' value of key: '%s': %v", sid, key, err)
		}
	}
}

// Release destroys the session, it clears and removes the session entry,
// session manager will create a new session ID on the next request after this call.
func (db *DatabaseString) Release(sid string) {
	// clear all $sid-$key.
	db.Clear(sid)
	// and remove the $sid.
	err := db.c.Driver.DeleteByKey(db.c.Prefix + sid)
	if err != nil {
		db.logger.Debugf("Database.Release.Driver.Delete: %s: %v", sid, err)
	}
}

func (db *DatabaseString) Close() error {
	return db.c.Driver.CloseConnection()
}
