package badger

import (
	"bytes"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/sessions"

	"github.com/dgraph-io/badger"
	"github.com/kataras/golog"
)

// DefaultFileMode used as the default database's "fileMode"
// for creating the sessions directory path, opening and write the session file.
var (
	DefaultFileMode = 0755
)

// Database the badger(key-value file-based) session storage.
type Database struct {
	// Service is the underline badger database connection,
	// it's initialized at `New` or `NewFromDB`.
	// Can be used to get stats.
	Service *badger.DB

	closed uint32 // if 1 is closed.
}

var _ sessions.Database = (*Database)(nil)

// New creates and returns a new badger(key-value file-based) storage
// instance based on the "directoryPath".
// DirectoryPath should is the directory which the badger database will store the sessions,
// i.e ./sessions
//
// It will remove any old session files.
func New(directoryPath string) (*Database, error) {
	if directoryPath == "" {
		return nil, errors.New("directoryPath is missing")
	}

	lindex := directoryPath[len(directoryPath)-1]
	if lindex != os.PathSeparator && lindex != '/' {
		directoryPath += string(os.PathSeparator)
	}
	// create directories if necessary
	if err := os.MkdirAll(directoryPath, os.FileMode(DefaultFileMode)); err != nil {
		return nil, err
	}

	opts := badger.DefaultOptions
	opts.Dir = directoryPath
	opts.ValueDir = directoryPath

	service, err := badger.Open(opts)

	if err != nil {
		golog.Errorf("unable to initialize the badger-based session database: %v", err)
		return nil, err
	}

	return NewFromDB(service), nil
}

// NewFromDB same as `New` but accepts an already-created custom badger connection instead.
func NewFromDB(service *badger.DB) *Database {
	db := &Database{Service: service}

	runtime.SetFinalizer(db, closeDB)
	return db
}

// Acquire receives a session's lifetime from the database,
// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
func (db *Database) Acquire(sid string, expires time.Duration) sessions.LifeTime {
	txn := db.Service.NewTransaction(true)
	defer txn.Commit(nil)

	bsid := makePrefix(sid)
	item, err := txn.Get(bsid)
	if err == nil {
		// found, return the expiration.
		return sessions.LifeTime{Time: time.Unix(int64(item.ExpiresAt()), 0)}
	}

	// not found, create an entry with ttl and return an empty lifetime, session manager will do its job.
	if err != nil {
		if err == badger.ErrKeyNotFound {
			// create it and set the expiration, we don't care about the value there.
			err = txn.SetWithTTL(bsid, bsid, expires)
		}
	}

	if err != nil {
		golog.Error(err)
	}

	return sessions.LifeTime{} // session manager will handle the rest.
}

// OnUpdateExpiration not implemented here, yet.
// Note that this error will not be logged, callers should catch it manually.
func (db *Database) OnUpdateExpiration(sid string, newExpires time.Duration) error {
	return sessions.ErrNotImplemented
}

var delim = byte('_')

func makePrefix(sid string) []byte {
	return append([]byte(sid), delim)
}

func makeKey(sid, key string) []byte {
	return append(makePrefix(sid), []byte(key)...)
}

// Set sets a key value of a specific session.
// Ignore the "immutable".
func (db *Database) Set(sid string, lifetime sessions.LifeTime, key string, value interface{}, immutable bool) {
	valueBytes, err := sessions.DefaultTranscoder.Marshal(value)
	if err != nil {
		golog.Error(err)
		return
	}

	err = db.Service.Update(func(txn *badger.Txn) error {
		dur := lifetime.DurationUntilExpiration()
		return txn.SetWithTTL(makeKey(sid, key), valueBytes, dur)
	})

	if err != nil {
		golog.Error(err)
	}
}

// Get retrieves a session value based on the key.
func (db *Database) Get(sid string, key string) (value interface{}) {
	err := db.Service.View(func(txn *badger.Txn) error {
		item, err := txn.Get(makeKey(sid, key))
		if err != nil {
			return err
		}

		// return item.Value(func(valueBytes []byte) {
		// 	if err := sessions.DefaultTranscoder.Unmarshal(valueBytes, &value); err != nil {
		// 		golog.Error(err)
		// 	}
		// })

		valueBytes, err := item.Value()
		if err != nil {
			return err
		}

		return sessions.DefaultTranscoder.Unmarshal(valueBytes, &value)
	})

	if err != nil && err != badger.ErrKeyNotFound {
		golog.Error(err)
		return nil
	}

	return
}

// Visit loops through all session keys and values.
func (db *Database) Visit(sid string, cb func(key string, value interface{})) {
	prefix := makePrefix(sid)

	txn := db.Service.NewTransaction(false)
	defer txn.Discard()

	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()

	for iter.Rewind(); iter.ValidForPrefix(prefix); iter.Next() {
		item := iter.Item()
		var value interface{}

		// err := item.Value(func(valueBytes []byte) {
		// 	if err := sessions.DefaultTranscoder.Unmarshal(valueBytes, &value); err != nil {
		// 		golog.Error(err)
		// 	}
		// })

		// if err != nil {
		// 	golog.Error(err)
		// 	continue
		// }

		valueBytes, err := item.Value()
		if err != nil {
			golog.Error(err)
			continue
		}

		if err = sessions.DefaultTranscoder.Unmarshal(valueBytes, &value); err != nil {
			golog.Error(err)
			continue
		}

		cb(string(bytes.TrimPrefix(item.Key(), prefix)), value)
	}
}

var iterOptionsNoValues = badger.IteratorOptions{
	PrefetchValues: false,
	PrefetchSize:   100,
	Reverse:        false,
	AllVersions:    false,
}

// Len returns the length of the session's entries (keys).
func (db *Database) Len(sid string) (n int) {
	prefix := makePrefix(sid)

	txn := db.Service.NewTransaction(false)
	iter := txn.NewIterator(iterOptionsNoValues)

	for iter.Rewind(); iter.ValidForPrefix(prefix); iter.Next() {
		n++
	}

	iter.Close()
	txn.Discard()
	return
}

// Delete removes a session key value based on its key.
func (db *Database) Delete(sid string, key string) (deleted bool) {
	txn := db.Service.NewTransaction(true)
	err := txn.Delete(makeKey(sid, key))
	if err != nil {
		golog.Error(err)
	}
	txn.Commit(nil)
	return err == nil
}

// Clear removes all session key values but it keeps the session entry.
func (db *Database) Clear(sid string) {
	prefix := makePrefix(sid)

	txn := db.Service.NewTransaction(true)
	defer txn.Commit(nil)

	iter := txn.NewIterator(iterOptionsNoValues)
	defer iter.Close()

	for iter.Rewind(); iter.ValidForPrefix(prefix); iter.Next() {
		txn.Delete(iter.Item().Key())
	}
}

// Release destroys the session, it clears and removes the session entry,
// session manager will create a new session ID on the next request after this call.
func (db *Database) Release(sid string) {
	// clear all $sid-$key.
	db.Clear(sid)
	// and remove the $sid.
	txn := db.Service.NewTransaction(true)
	txn.Delete([]byte(sid))
	txn.Commit(nil)
}

// Close shutdowns the badger connection.
func (db *Database) Close() error {
	return closeDB(db)
}

func closeDB(db *Database) error {
	if atomic.LoadUint32(&db.closed) > 0 {
		return nil
	}
	err := db.Service.Close()
	if err != nil {
		golog.Warnf("closing the badger connection: %v", err)
	} else {
		atomic.StoreUint32(&db.closed, 1)
	}
	return err
}
