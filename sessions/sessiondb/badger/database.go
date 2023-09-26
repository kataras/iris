package badger

import (
	"bytes"
	"errors"
	"os"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/memstore"
	"github.com/kataras/iris/v12/sessions"

	"github.com/dgraph-io/badger/v2"
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
	logger  *golog.Logger

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
		return nil, errors.New("directoryPath is empty")
	}

	lindex := directoryPath[len(directoryPath)-1]
	if lindex != os.PathSeparator && lindex != '/' {
		directoryPath += string(os.PathSeparator)
	}
	// create directories if necessary
	if err := os.MkdirAll(directoryPath, os.FileMode(DefaultFileMode)); err != nil {
		return nil, err
	}

	opts := badger.DefaultOptions(directoryPath)
	badgerLogger := context.DefaultLogger("sessionsdb.badger").DisableNewLine()
	opts.Logger = badgerLogger

	service, err := badger.Open(opts)
	if err != nil {
		badgerLogger.Errorf("unable to initialize the badger-based session database: %v\n", err)
		return nil, err
	}

	return NewFromDB(service), nil
}

// NewFromDB same as `New` but accepts an already-created custom badger connection instead.
func NewFromDB(service *badger.DB) *Database {
	db := &Database{Service: service}

	// runtime.SetFinalizer(db, closeDB)
	return db
}

// SetLogger sets the logger once before server ran.
// By default the Iris one is injected.
func (db *Database) SetLogger(logger *golog.Logger) {
	db.logger = logger
}

// Acquire receives a session's lifetime from the database,
// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
func (db *Database) Acquire(sid string, expires time.Duration) memstore.LifeTime {
	txn := db.Service.NewTransaction(true)
	defer txn.Commit()

	bsid := makePrefix(sid)
	item, err := txn.Get(bsid)
	if err == nil {
		// found, return the expiration.
		return memstore.LifeTime{Time: time.Unix(int64(item.ExpiresAt()), 0)}
	}

	// not found, create an entry with ttl and return an empty lifetime, session manager will do its job.
	if err != nil {
		if err == badger.ErrKeyNotFound {
			// create it and set the expiration, we don't care about the value there.
			err = txn.SetEntry(badger.NewEntry(bsid, bsid).WithTTL(expires))
		}
	}

	if err != nil {
		db.logger.Error(err)
	}

	return memstore.LifeTime{} // session manager will handle the rest.
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
func (db *Database) Set(sid string, key string, value interface{}, ttl time.Duration, immutable bool) error {
	valueBytes, err := sessions.DefaultTranscoder.Marshal(value)
	if err != nil {
		db.logger.Error(err)
		return err
	}

	err = db.Service.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(badger.NewEntry(makeKey(sid, key), valueBytes).WithTTL(ttl))
	})

	if err != nil {
		db.logger.Error(err)
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
	err := db.Service.View(func(txn *badger.Txn) error {
		item, err := txn.Get(makeKey(sid, key))
		if err != nil {
			return err
		}

		return item.Value(func(valueBytes []byte) error {
			return sessions.DefaultTranscoder.Unmarshal(valueBytes, outPtr)
		})
	})

	if err != nil && err != badger.ErrKeyNotFound {
		db.logger.Error(err)
	}

	return err
}

// validSessionItem reports whether the current iterator's item key
// is a value of the session id "prefix".
func validSessionItem(key, prefix []byte) bool {
	return len(key) > len(prefix) && bytes.Equal(key[0:len(prefix)], prefix)
}

// Visit loops through all session keys and values.
func (db *Database) Visit(sid string, cb func(key string, value interface{})) error {
	prefix := makePrefix(sid)

	txn := db.Service.NewTransaction(false)
	defer txn.Discard()

	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()

	for iter.Rewind(); ; iter.Next() {
		if !iter.Valid() {
			break
		}

		item := iter.Item()
		key := item.Key()
		if !validSessionItem(key, prefix) {
			continue
		}

		var value interface{}

		err := item.Value(func(valueBytes []byte) error {
			return sessions.DefaultTranscoder.Unmarshal(valueBytes, &value)
		})
		if err != nil {
			db.logger.Errorf("[sessionsdb.badger.Visit] %v", err)
			return err
		}

		cb(string(bytes.TrimPrefix(key, prefix)), value)
	}

	return nil
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

	for iter.Rewind(); ; iter.Next() {
		if !iter.Valid() {
			break
		}

		if validSessionItem(iter.Item().Key(), prefix) {
			n++
		}
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
		db.logger.Error(err)
		return false
	}
	return txn.Commit() == nil
}

// Clear removes all session key values but it keeps the session entry.
func (db *Database) Clear(sid string) error {
	prefix := makePrefix(sid)

	txn := db.Service.NewTransaction(true)
	defer txn.Commit()

	iter := txn.NewIterator(iterOptionsNoValues)
	defer iter.Close()

	for iter.Rewind(); iter.ValidForPrefix(prefix); iter.Next() {
		key := iter.Item().Key()
		if err := txn.Delete(key); err != nil {
			db.logger.Warnf("Database.Clear: %s: %v", key, err)
			return err
		}
	}

	return nil
}

// Release destroys the session, it clears and removes the session entry,
// session manager will create a new session ID on the next request after this call.
func (db *Database) Release(sid string) error {
	// clear all $sid-$key.
	err := db.Clear(sid)
	if err != nil {
		return err
	}
	// and remove the $sid.
	txn := db.Service.NewTransaction(true)
	if err = txn.Delete([]byte(sid)); err != nil {
		db.logger.Warnf("Database.Release.Delete: %s: %v", sid, err)
		return err
	}
	if err = txn.Commit(); err != nil {
		db.logger.Debugf("Database.Release.Commit: %s: %v", sid, err)
		return err
	}

	return nil
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
		db.logger.Warnf("closing the badger connection: %v", err)
	} else {
		atomic.StoreUint32(&db.closed, 1)
	}
	return err
}
