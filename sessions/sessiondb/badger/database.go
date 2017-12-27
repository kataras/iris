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
	DefaultFileMode = 0755
)

// Database the badger(key-value file-based) session storage.
type Database struct {
	// Service is the underline badger database connection,
	// it's initialized at `New` or `NewFromDB`.
	// Can be used to get stats.
	Service *badger.DB
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

	service, err := badger.Open(opts)

	if err != nil {
		golog.Errorf("unable to initialize the badger-based session database: %v", err)
		return nil, err
	}

	return NewFromDB(service)
}

// NewFromDB same as `New` but accepts an already-created custom badger connection instead.
func NewFromDB(service *badger.DB) (*Database, error) {
	if service == nil {
		return nil, errors.New("underline database is missing")
	}

	db := &Database{Service: service}

	runtime.SetFinalizer(db, closeDB)
	return db, db.Cleanup()
}

// Cleanup removes any invalid(have expired) session entries,
// it's being called automatically on `New` as well.
func (db *Database) Cleanup() (err error) {
	rep := errors.NewReporter()

	txn := db.Service.NewTransaction(true)
	defer txn.Commit(nil)

	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()

	for iter.Rewind(); iter.Valid(); iter.Next() {
		// Remember that the contents of the returned slice should not be modified, and
		// only valid until the next call to Next.
		item := iter.Item()
		b, err := item.Value()

		if rep.AddErr(err) {
			continue
		}

		storeDB, err := sessions.DecodeRemoteStore(b)
		if rep.AddErr(err) {
			continue
		}

		if storeDB.Lifetime.HasExpired() {
			if err := txn.Delete(item.Key()); err != nil {
				rep.AddErr(err)
			}
		}
	}

	return rep.Return()
}

// Async is DEPRECATED
// if it was true then it could use different to update the back-end storage, now it does nothing.
func (db *Database) Async(useGoRoutines bool) *Database {
	return db
}

// Load loads the sessions from the badger(key-value file-based) session storage.
func (db *Database) Load(sid string) (storeDB sessions.RemoteStore) {
	bsid := []byte(sid)

	txn := db.Service.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get(bsid)
	if err != nil {
		// Key not found, don't report this, session manager will create a new session as it should.
		return
	}

	b, err := item.Value()

	if err != nil {
		golog.Errorf("error while trying to get the serialized session(%s) from the remote store: %v", sid, err)
		return
	}

	storeDB, err = sessions.DecodeRemoteStore(b) // decode the whole value, as a remote store
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
			golog.Errorf("error while destroying a session(%s) from badger: %v",
				p.SessionID, err)
		}
		return
	}

	s, err := p.Store.Serialize()
	if err != nil {
		golog.Errorf("error while serializing the remote store: %v", err)
	}

	txn := db.Service.NewTransaction(true)

	err = txn.Set(bsid, s)
	if err != nil {
		txn.Discard()
		golog.Errorf("error while trying to save the session(%s) to the database: %v", p.SessionID, err)
		return
	}
	if err := txn.Commit(nil); err != nil { // Commit will call the Discard automatically.
		golog.Errorf("error while committing the session(%s) changes to the database: %v", p.SessionID, err)
	}
}

func (db *Database) destroy(bsid []byte) error {
	txn := db.Service.NewTransaction(true)

	err := txn.Delete(bsid)
	if err != nil {
		return err
	}

	return txn.Commit(nil)
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
