package boltdb

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/coreos/bbolt"
	"github.com/kataras/golog"
	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/sessions"
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
}

var (
	// ErrOptionsMissing returned on `New` when path or tableName are empty.
	ErrOptionsMissing = errors.New("required options are missing")
)

// New creates and returns a new BoltDB(file-based) storage
// instance based on the "path".
// Path should include the filename and the directory(aka fullpath), i.e sessions/store.db.
//
// It will remove any old session files.
func New(path string, fileMode os.FileMode, bucketName string) (*Database, error) {

	if path == "" || bucketName == "" {
		return nil, ErrOptionsMissing
	}

	if fileMode <= 0 {
		fileMode = os.FileMode(DefaultFileMode)
	}

	// create directories if necessary
	if err := os.MkdirAll(filepath.Dir(path), fileMode); err != nil {
		golog.Errorf("error while trying to create the necessary directories for %s: %v", path, err)
		return nil, err
	}

	service, err := bolt.Open(path, 0600,
		&bolt.Options{Timeout: 15 * time.Second},
	)

	if err != nil {
		golog.Errorf("unable to initialize the BoltDB-based session database: %v", err)
		return nil, err
	}

	return NewFromDB(service, bucketName)
}

// NewFromDB same as `New` but accepts an already-created custom boltdb connection instead.
func NewFromDB(service *bolt.DB, bucketName string) (*Database, error) {
	if bucketName == "" {
		return nil, ErrOptionsMissing
	}
	bucket := []byte(bucketName)

	service.Update(func(tx *bolt.Tx) (err error) {
		_, err = tx.CreateBucketIfNotExists(bucket)
		return
	})

	db := &Database{table: bucket, Service: service}

	runtime.SetFinalizer(db, closeDB)
	return db, db.Cleanup()
}

// Cleanup removes any invalid(have expired) session entries,
// it's being called automatically on `New` as well.
func (db *Database) Cleanup() error {
	err := db.Service.Update(func(tx *bolt.Tx) error {
		b := db.getBucket(tx)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if len(k) == 0 { // empty key, continue to the next pair
				continue
			}

			storeDB, err := sessions.DecodeRemoteStore(v)
			if err != nil {
				continue
			}

			if storeDB.Lifetime.HasExpired() {
				if err := c.Delete(); err != nil {
					golog.Warnf("troubles when cleanup a session remote store from BoltDB: %v", err)
				}
			}
		}

		return nil
	})

	return err
}

// Async is DEPRECATED
// if it was true then it could use different to update the back-end storage, now it does nothing.
func (db *Database) Async(useGoRoutines bool) *Database {
	return db
}

// Load loads the sessions from the BoltDB(file-based) session storage.
func (db *Database) Load(sid string) (storeDB sessions.RemoteStore) {
	bsid := []byte(sid)
	err := db.Service.View(func(tx *bolt.Tx) (err error) {
		// db.getSessBucket(tx, sid)
		b := db.getBucket(tx)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if len(k) == 0 { // empty key, continue to the next pair
				continue
			}

			if bytes.Equal(k, bsid) { // session id should be the name of the key-value pair
				storeDB, err = sessions.DecodeRemoteStore(v) // decode the whole value, as a remote store
				break
			}
		}
		return
	})

	if err != nil {
		golog.Errorf("error while trying to load from the remote store: %v", err)
	}

	return
}

// Sync syncs the database with the session's (memory) store.
func (db *Database) Sync(p sessions.SyncPayload) {
	db.sync(p)
}

func (db *Database) sync(p sessions.SyncPayload) {
	bsid := []byte(p.SessionID)

	if p.Action == sessions.ActionDestroy {
		if err := db.destroy(bsid); err != nil {
			golog.Errorf("error while destroying a session(%s) from boltdb: %v",
				p.SessionID, err)
		}
		return
	}

	s, err := p.Store.Serialize()
	if err != nil {
		golog.Errorf("error while serializing the remote store: %v", err)
	}

	err = db.Service.Update(func(tx *bolt.Tx) error {
		return db.getBucket(tx).Put(bsid, s)
	})
	if err != nil {
		golog.Errorf("error while writing the session bucket: %v", err)
	}
}

func (db *Database) destroy(bsid []byte) error {
	return db.Service.Update(func(tx *bolt.Tx) error {
		return db.getBucket(tx).Delete(bsid)
	})
}

// we store the whole data to the key-value pair of the root bucket
// so we don't need a separate bucket for each session
// this method could be faster if we had large data to store
// but with sessions we recommend small amount of data, so the method finally chosen
// is faster (decode/encode the whole store + lifetime and return it as it's)
//
// func (db *Database) getSessBucket(tx *bolt.Tx, sid string) (*bolt.Bucket, error) {
// 	table, err := db.getBucket(tx).CreateBucketIfNotExists([]byte(sid))
// 	return table, err
// }

func (db *Database) getBucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket(db.table)
}

// Len reports the number of sessions that are stored to the this BoltDB table.
func (db *Database) Len() (num int) {
	db.Service.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := db.getBucket(tx)
		if b == nil {
			return nil
		}

		b.ForEach(func([]byte, []byte) error {
			num++
			return nil
		})
		return nil
	})
	return
}

// Close shutdowns the BoltDB connection.
func (db *Database) Close() error {
	return closeDB(db)
}

func closeDB(db *Database) error {
	err := db.Service.Close()
	if err != nil {
		golog.Warnf("closing the BoltDB connection: %v", err)
	}

	return err
}
