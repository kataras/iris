package redis

import (
	"github.com/kataras/golog"
	"github.com/kataras/iris/v12/sessions"
	"time"
)

type DatabaseHashed struct {
	c      Config
	logger *golog.Logger
}

// Config returns the configuration for the redis server bridge, you can change them.
func (db *DatabaseHashed) Config() *Config {
	return &db.c // 6 Aug 2019 - keep that for no breaking change.
}

func (db *DatabaseHashed) SetLogger(logger *golog.Logger) {
	db.logger = logger
}

func (db *DatabaseHashed) Connect(c Config) error {
	err := db.c.Driver.Connect(c)
	if err != nil {
		return err
	}

	_, err = db.c.Driver.PingPong()
	return err
}

func (db *DatabaseHashed) makeKey(sid, key string) string {
	if key == "" {
		return db.c.Prefix + sid
	}
	return db.c.Prefix + sid + db.c.Delim + key
}

// Acquire receives a session's lifetime from the database,
// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
func (db *DatabaseHashed) Acquire(sid string, expires time.Duration) sessions.LifeTime {
	seconds, hasExpiration, found := db.c.Driver.TTL(sid)
	if !found {
		// fmt.Printf("db.Acquire expires: %s. Seconds: %v\n", expires, expires.Seconds())
		// not found, create an entry with ttl and return an empty lifetime, session manager will do its job.
		valueBytes, _ := sessions.DefaultTranscoder.Marshal(sid)
		if err := db.c.Driver.Set(sid, sid, valueBytes, int64(expires.Seconds())); err != nil {
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
func (db *DatabaseHashed) OnUpdateExpiration(sid string, newExpires time.Duration) error {
	return db.c.Driver.UpdateTTLMany(sid, int64(newExpires.Seconds()))
}

// Set sets a key value of a specific session.
// Ignore the "immutable".
func (db *DatabaseHashed) Set(sid string, lifetime *sessions.LifeTime, key string, value interface{}, immutable bool) {
	valueBytes, err := sessions.DefaultTranscoder.Marshal(value)
	if err != nil {
		db.logger.Error(err)
		return
	}

	//fmt.Println("database.Set ", sid, "--", key, "---", int64(lifetime.DurationUntilExpiration().Seconds()))
	// fmt.Printf("lifetime.DurationUntilExpiration(): %s. Seconds: %v\n", lifetime.DurationUntilExpiration(), lifetime.DurationUntilExpiration().Seconds())
	if err = db.c.Driver.Set(sid, key, valueBytes, int64(lifetime.DurationUntilExpiration().Seconds())); err != nil {
		db.logger.Debug(err)
	}
}

func (db *DatabaseHashed) get(sid, key string, outPtr interface{}) {
	data, err := db.c.Driver.Get(sid, key)
	if err != nil {
		// not found.
		return
	}

	if err = sessions.DefaultTranscoder.Unmarshal(data.([]byte), outPtr); err != nil {
		db.logger.Debugf("unable to unmarshal value of key: '%s': %v", key, err)
	}
}

func (db *DatabaseHashed) keys(sid string) []string {
	keys, err := db.c.Driver.GetKeys(sid)
	if err != nil {
		db.logger.Debugf("unable to get all redis keys of session '%s': %v", sid, err)
		return nil
	}

	return keys
}

// Get retrieves a session value based on the key.
func (db *DatabaseHashed) Get(sid string, key string) (value interface{}) {
	db.get(sid, key, &value)
	return
}

// Visit loops through all session keys and values.
func (db *DatabaseHashed) Visit(sid string, cb func(key string, value interface{})) {
	keys := db.keys(sid)
	for _, key := range keys {
		var value interface{} // new value each time, we don't know what user will do in "cb".
		db.get(sid, key, &value)
		cb(key, value)
	}
}

// Len returns the length of the session's entries (keys).
func (db *DatabaseHashed) Len(sid string) (n int) {
	length, err := db.c.Driver.Len(sid)
	if err != nil {
		db.logger.Debugf("get hash key length of session '%s': %v", sid, err)
		return 0
	}
	return length
}

// Delete removes a session key value based on its key.
func (db *DatabaseHashed) Delete(sid string, key string) (deleted bool) {
	err := db.c.Driver.Delete(sid, key)
	if err != nil {
		db.logger.Error(err)
	}
	return err == nil
}

// Clear removes all session key values but it keeps the session entry.
func (db *DatabaseHashed) Clear(sid string) {
	err := db.c.Driver.Clear(sid)
	if err != nil {
		db.logger.Debugf("unable to delete session '%s': %v", sid, err)
	}
}

// Release destroys the session, it clears and removes the session entry,
// session manager will create a new session ID on the next request after this call.
func (db *DatabaseHashed) Release(sid string) {
	db.Clear(sid)
}

func (db *DatabaseHashed) Close() error {
	return db.c.Driver.CloseConnection()
}
