package badger

import (
	"os"
	"runtime"

	"github.com/kataras/golog"
	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/sessions"

	"github.com/dgraph-io/badger"
)

// DefaultFileMode used as the default database's "fileMode"
// for creating the sessions directory path, opening and write the session file.
var (
	DefaultFileMode = 0666
)

// Database the badger(key-value file-based) session storage.
type Database struct {
	// Service is the underline badger database connection,
	// it's initialized at `New` or `NewFromDB`.
	// Can be used to get stats.
	Service *badger.KV
	async   bool
}

// New creates and returns a new badger(key-value file-based) storage
// instance based on the "directoryPath".
// DirectoryPath should is the directory which the badger database will store the sessions,
// i.e ./sessions
//
// It will remove any old session files.
func New(directoryPath string) (*Database, error) {

	if directoryPath == "" {
		return nil, errors.New("dir is missing")
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

	service, err := badger.NewKV(&opts)

	if err != nil {
		golog.Errorf("unable to initialize the badger-based session database: %v", err)
		return nil, err
	}

	return NewFromDB(service)
}

// NewFromDB same as `New` but accepts an already-created custom badger connection instead.
func NewFromDB(service *badger.KV) (*Database, error) {
	if service == nil {
		return nil, errors.New("underline database is missing")
	}

	db := &Database{Service: service}

	runtime.SetFinalizer(db, closeDB)
	return db, db.Cleanup()
}

// Cleanup removes any invalid(have expired) session entries,
// it's being called automatically on `New` as well.
func (db *Database) Cleanup() error {
	rep := errors.NewReporter()

	iter := db.Service.NewIterator(badger.DefaultIteratorOptions)
	for iter.Rewind(); iter.Valid(); iter.Next() {
		// Remember that the contents of the returned slice should not be modified, and
		// only valid until the next call to Next.
		item := iter.Item()
		err := item.Value(func(b []byte) error {
			storeDB, err := sessions.DecodeRemoteStore(b)
			if err != nil {
				return err
			}

			if storeDB.Lifetime.HasExpired() {
				err = db.Service.Delete(item.Key())
			}
			return err
		})

		rep.AddErr(err)
	}

	iter.Close()
	return rep.Return()
}

// Async if true passed then it will use different
// go routines to update the badger(key-value file-based) storage.
func (db *Database) Async(useGoRoutines bool) *Database {
	db.async = useGoRoutines
	return db
}

// Load loads the sessions from the badger(key-value file-based) session storage.
func (db *Database) Load(sid string) (storeDB sessions.RemoteStore) {
	bsid := []byte(sid)
	iter := db.Service.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()

	iter.Seek(bsid)
	if !iter.Valid() {
		return
	}
	item := iter.Item()
	item.Value(func(b []byte) (err error) {
		storeDB, err = sessions.DecodeRemoteStore(b) // decode the whole value, as a remote store
		if err != nil {
			golog.Errorf("error while trying to load from the remote store: %v", err)
		}
		return
	})
	return
}

// Sync syncs the database with the session's (memory) store.
func (db *Database) Sync(p sessions.SyncPayload) {
	if db.async {
		go db.sync(p)
	} else {
		db.sync(p)
	}
}

func (db *Database) sync(p sessions.SyncPayload) {
	bsid := []byte(p.SessionID)

	if p.Action == sessions.ActionDestroy {
		if err := db.destroy(bsid); err != nil {
			golog.Errorf("error while destroying a session(%s) from badger: %v",
				p.SessionID, err)
		}
		return
	}

	s, err := p.Store.Serialize()
	if err != nil {
		golog.Errorf("error while serializing the remote store: %v", err)
	}

	// err = db.Service.Set(bsid, s, meta)
	e := &badger.Entry{
		Key:   bsid,
		Value: s,
	}
	err = db.Service.BatchSet([]*badger.Entry{e})
	if err != nil {
		golog.Errorf("error while writing the session(%s) to the database: %v", p.SessionID, err)
	}
}

func (db *Database) destroy(bsid []byte) error {
	return db.Service.Delete(bsid)
}

// Close shutdowns the badger connection.
func (db *Database) Close() error {
	return closeDB(db)
}

func closeDB(db *Database) error {
	err := db.Service.Close()
	if err != nil {
		golog.Warnf("closing the badger connection: %v", err)
	}
	return err
}
