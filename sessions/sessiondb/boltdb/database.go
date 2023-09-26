package boltdb

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/kataras/iris/v12/core/memstore"
	"github.com/kataras/iris/v12/sessions"

	"github.com/kataras/golog"
	bolt "go.etcd.io/bbolt"
)

// DefaultFileMode used as the default database's "fileMode"
// for creating the sessions directory path, opening and write
// the session boltdb(file-based) storage.
var (
	DefaultFileMode = 0755
)

// Database the BoltDB(file-based) session storage.
type Database struct {
	table []byte
	// Service is the underline BoltDB database connection,
	// it's initialized at `New` or `NewFromDB`.
	// Can be used to get stats.
	Service *bolt.DB
	logger  *golog.Logger
}

var errPathMissing = errors.New("path is required")

// New creates and returns a new BoltDB(file-based) storage
// instance based on the "path".
// Path should include the filename and the directory(aka fullpath), i.e sessions/store.db.
//
// It will remove any old session files.
func New(path string, fileMode os.FileMode) (*Database, error) {
	if path == "" {
		golog.Error(errPathMissing)
		return nil, errPathMissing
	}

	if fileMode == 0 {
		fileMode = os.FileMode(DefaultFileMode)
	}

	// create directories if necessary
	if err := os.MkdirAll(filepath.Dir(path), fileMode); err != nil {
		golog.Errorf("error while trying to create the necessary directories for %s: %v", path, err)
		return nil, err
	}

	service, err := bolt.Open(path, fileMode,
		&bolt.Options{Timeout: 20 * time.Second},
	)
	if err != nil {
		golog.Errorf("unable to initialize the BoltDB-based session database: %v", err)
		return nil, err
	}

	return NewFromDB(service, "sessions")
}

// NewFromDB same as `New` but accepts an already-created custom boltdb connection instead.
func NewFromDB(service *bolt.DB, bucketName string) (*Database, error) {
	bucket := []byte(bucketName)

	err := service.Update(func(tx *bolt.Tx) (err error) {
		_, err = tx.CreateBucketIfNotExists(bucket)
		return
	})

	if err != nil {
		return nil, err
	}

	db := &Database{table: bucket, Service: service}

	// runtime.SetFinalizer(db, closeDB)
	return db, db.cleanup()
}

func (db *Database) getBucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket(db.table)
}

func (db *Database) getBucketForSession(tx *bolt.Tx, sid string) *bolt.Bucket {
	b := db.getBucket(tx).Bucket([]byte(sid))
	if b == nil {
		// session does not exist, it shouldn't happen, session bucket creation happens once at `Acquire`,
		// no need to accept the `bolt.bucket.CreateBucketIfNotExists`'s performance cost.
		db.logger.Debugf("unreachable session access for '%s'", sid)
	}

	return b
}

var (
	expirationBucketName = []byte("expiration")
	delim                = []byte("_")
)

// expiration lives on its own bucket for each session bucket.
func getExpirationBucketName(bsid []byte) []byte {
	return append(bsid, append(delim, expirationBucketName...)...)
}

// Cleanup removes any invalid(have expired) session entries on initialization.
func (db *Database) cleanup() error {
	return db.Service.Update(func(tx *bolt.Tx) error {
		b := db.getBucket(tx)
		c := b.Cursor()
		// loop through all buckets, find one with expiration.
		for bsid, v := c.First(); bsid != nil; bsid, v = c.Next() {
			if len(bsid) == 0 { // empty key, continue to the next session bucket.
				continue
			}

			expirationName := getExpirationBucketName(bsid)
			if bExp := b.Bucket(expirationName); bExp != nil { // has expiration.
				_, expValue := bExp.Cursor().First() // the expiration bucket contains only one key(we don't care, see `Acquire`) value(time.Time) pair.
				if expValue == nil {
					db.logger.Debugf("cleanup: expiration is there but its value is empty '%s'", v) // should never happen.
					continue
				}

				var expirationTime time.Time
				if err := sessions.DefaultTranscoder.Unmarshal(expValue, &expirationTime); err != nil {
					db.logger.Debugf("cleanup: unable to retrieve expiration value for '%s'", v)
					continue
				}

				if expirationTime.Before(time.Now()) {
					// expired, delete the expiration bucket.
					if err := b.DeleteBucket(expirationName); err != nil {
						db.logger.Debugf("cleanup: unable to destroy a session '%s'", bsid)
						return err
					}

					// and the session bucket, if any.
					return b.DeleteBucket(bsid)
				}
			}
		}

		return nil
	})
}

// SetLogger sets the logger once before server ran.
// By default the Iris one is injected.
func (db *Database) SetLogger(logger *golog.Logger) {
	db.logger = logger
}

var expirationKey = []byte("exp") // it can be random.

// Acquire receives a session's lifetime from the database,
// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
func (db *Database) Acquire(sid string, expires time.Duration) (lifetime memstore.LifeTime) {
	bsid := []byte(sid)
	err := db.Service.Update(func(tx *bolt.Tx) (err error) {
		root := db.getBucket(tx)

		if expires > 0 { // should check or create the expiration bucket.
			name := getExpirationBucketName(bsid)
			b := root.Bucket(name)
			if b == nil {
				// not found, create a session bucket and an expiration bucket and save the given "expires" of time.Time,
				// don't return a lifetime, let it empty, session manager will do its job.
				b, err = root.CreateBucket(name)
				if err != nil {
					db.logger.Debugf("unable to create a session bucket for '%s': %v", sid, err)
					return err
				}

				expirationTime := time.Now().Add(expires)
				timeBytes, err := sessions.DefaultTranscoder.Marshal(expirationTime)
				if err != nil {
					db.logger.Debugf("unable to set an expiration value on session expiration bucket for '%s': %v", sid, err)
					return err
				}

				err = b.Put(expirationKey, timeBytes)
				if err == nil {
					// create the session bucket now, so the rest of the calls can be easly get the bucket without any further checks.
					_, err = root.CreateBucket(bsid)
				}

				return err
			}

			// found, get the associated expiration bucket, wrap its value and return.
			_, expValue := b.Cursor().First()
			if expValue == nil {
				return nil // does not expire.
			}

			var expirationTime time.Time
			if err = sessions.DefaultTranscoder.Unmarshal(expValue, &expirationTime); err != nil {
				db.logger.Debugf("acquire: unable to retrieve expiration value for '%s', value was: '%s': %v", sid, expValue, err)
				return
			}

			lifetime = memstore.LifeTime{Time: expirationTime}
			return nil
		}

		// does not expire, just create the session bucket if not exists so we can be ready later on.
		_, err = root.CreateBucketIfNotExists(bsid)
		return
	})
	if err != nil {
		db.logger.Debugf("unable to acquire session '%s': %v", sid, err)
		return memstore.LifeTime{}
	}

	return
}

// OnUpdateExpiration will re-set the database's session's entry ttl.
func (db *Database) OnUpdateExpiration(sid string, newExpires time.Duration) error {
	expirationTime := time.Now().Add(newExpires)
	timeBytes, err := sessions.DefaultTranscoder.Marshal(expirationTime)
	if err != nil {
		return err
	}

	err = db.Service.Update(func(tx *bolt.Tx) error {
		expirationName := getExpirationBucketName([]byte(sid))
		root := db.getBucket(tx)
		b := root.Bucket(expirationName)
		if b == nil {
			// db.logger.Debugf("tried to reset the expiration value for '%s' while its configured lifetime is unlimited or the session is already expired and not found now", sid)
			return sessions.ErrNotFound
		}

		return b.Put(expirationKey, timeBytes)
	})

	if err != nil {
		db.logger.Debugf("unable to reset the expiration value for '%s': %v", sid, err)
	}

	return err
}

func makeKey(key string) []byte {
	return []byte(key)
}

// Set sets a key value of a specific session.
// Ignore the "immutable".
func (db *Database) Set(sid string, key string, value interface{}, ttl time.Duration, immutable bool) error {
	valueBytes, err := sessions.DefaultTranscoder.Marshal(value)
	if err != nil {
		db.logger.Debug(err)
		return err
	}

	err = db.Service.Update(func(tx *bolt.Tx) error {
		b := db.getBucketForSession(tx, sid)
		if b == nil {
			return nil
		}

		// Author's notes:
		// expiration is handlded by the session manager for the whole session, so the `db.Destroy` will be called when and if needed.
		// Therefore we don't have to implement a TTL here, but we need a `db.Cleanup`, as we did previously, method to delete any expired if server restarted
		// (badger does not need a `Cleanup` because we set the TTL based on the lifetime.DurationUntilExpiration()).
		return b.Put(makeKey(key), valueBytes)
	})

	if err != nil {
		db.logger.Debug(err)
	}

	return err
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
	err := db.Service.View(func(tx *bolt.Tx) error {
		b := db.getBucketForSession(tx, sid)
		if b == nil {
			return nil
		}

		valueBytes := b.Get(makeKey(key))
		if len(valueBytes) == 0 {
			return nil
		}

		return sessions.DefaultTranscoder.Unmarshal(valueBytes, outPtr)
	})
	if err != nil {
		db.logger.Debugf("session '%s' key '%s' cannot be retrieved: %v", sid, key, err)
	}

	return err
}

// Visit loops through all session keys and values.
func (db *Database) Visit(sid string, cb func(key string, value interface{})) error {
	err := db.Service.View(func(tx *bolt.Tx) error {
		b := db.getBucketForSession(tx, sid)
		if b == nil {
			return nil
		}

		return b.ForEach(func(k []byte, v []byte) error {
			var value interface{}
			if err := sessions.DefaultTranscoder.Unmarshal(v, &value); err != nil {
				db.logger.Debugf("unable to retrieve value of key '%s' of '%s': %v", k, sid, err)
				return err
			}

			cb(string(k), value)
			return nil
		})
	})

	if err != nil {
		db.logger.Debugf("Database.Visit: %s: %v", sid, err)
	}

	return err
}

// Len returns the length of the session's entries (keys).
func (db *Database) Len(sid string) (n int) {
	err := db.Service.View(func(tx *bolt.Tx) error {
		b := db.getBucketForSession(tx, sid)
		if b == nil {
			return nil
		}

		n = int(int64(b.Stats().KeyN))
		return nil
	})

	if err != nil {
		db.logger.Debugf("Database.Len: %s: %v", sid, err)
	}

	return
}

// Delete removes a session key value based on its key.
func (db *Database) Delete(sid string, key string) (deleted bool) {
	err := db.Service.Update(func(tx *bolt.Tx) error {
		b := db.getBucketForSession(tx, sid)
		if b == nil {
			return sessions.ErrNotFound
		}

		return b.Delete(makeKey(key))
	})

	return err == nil
}

// Clear removes all session key values but it keeps the session entry.
func (db *Database) Clear(sid string) error {
	err := db.Service.Update(func(tx *bolt.Tx) error {
		b := db.getBucketForSession(tx, sid)
		if b == nil {
			return nil
		}

		return b.ForEach(func(k []byte, v []byte) error {
			return b.Delete(k)
		})
	})

	if err != nil {
		db.logger.Debugf("Database.Clear: %s: %v", sid, err)
	}

	return err
}

// Release destroys the session, it clears and removes the session entry,
// session manager will create a new session ID on the next request after this call.
func (db *Database) Release(sid string) error {
	err := db.Service.Update(func(tx *bolt.Tx) error {
		// delete the session bucket.
		b := db.getBucket(tx)
		bsid := []byte(sid)
		// try to delete the associated expiration bucket, if exists, ignore error.
		_ = b.DeleteBucket(getExpirationBucketName(bsid))

		return b.DeleteBucket(bsid)
	})

	if err != nil {
		db.logger.Debugf("Database.Release: %s: %v", sid, err)
	}

	return err
}

// Close shutdowns the BoltDB connection.
func (db *Database) Close() error {
	return closeDB(db)
}

func closeDB(db *Database) error {
	err := db.Service.Close()
	if err != nil {
		db.logger.Warnf("closing the BoltDB connection: %v", err)
	}

	return err
}
