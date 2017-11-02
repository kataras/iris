package leveldb

import (
	"bytes"
	"runtime"

	"github.com/kataras/golog"
	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/sessions"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var (
	// Options used to open the leveldb database, defaults to leveldb's default values.
	Options = &opt.Options{}
	// WriteOptions used to put and delete, defaults to leveldb's default values.
	WriteOptions = &opt.WriteOptions{}
	// ReadOptions used to iterate over the database, defaults to leveldb's default values.
	ReadOptions = &opt.ReadOptions{}
)

// Database the LevelDB(file-based) session storage.
type Database struct {
	// Service is the underline LevelDB database connection,
	// it's initialized at `New` or `NewFromDB`.
	// Can be used to get stats.
	Service *leveldb.DB
}

// New creates and returns a new LevelDB(file-based) storage
// instance based on the "directoryPath".
// DirectoryPath should is the directory which the leveldb database will store the sessions,
// i.e ./sessions/
//
// It will remove any old session files.
func New(directoryPath string) (*Database, error) {

	if directoryPath == "" {
		return nil, errors.New("dir is missing")
	}

	// Second parameter is a "github.com/syndtr/goleveldb/leveldb/opt.Options{}"
	// user can change the `Options` or create the sessiondb via `NewFromDB`
	// if wants to use a customized leveldb database
	// or an existing one, we don't require leveldb options at the constructor.
	//
	// The leveldb creates the directories, if necessary.
	service, err := leveldb.OpenFile(directoryPath, Options)

	if err != nil {
		golog.Errorf("unable to initialize the LevelDB-based session database: %v", err)
		return nil, err
	}

	return NewFromDB(service)
}

// NewFromDB same as `New` but accepts an already-created custom leveldb connection instead.
func NewFromDB(service *leveldb.DB) (*Database, error) {
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
	iter := db.Service.NewIterator(nil, ReadOptions)
	for iter.Next() {
		// Remember that the contents of the returned slice should not be modified, and
		// only valid until the next call to Next.
		k := iter.Key()

		if len(k) > 0 {
			v := iter.Value()
			storeDB, err := sessions.DecodeRemoteStore(v)
			if err != nil {
				continue
			}

			if storeDB.Lifetime.HasExpired() {
				if err := db.Service.Delete(k, WriteOptions); err != nil {
					golog.Warnf("troubles when cleanup a session remote store from LevelDB: %v", err)
				}
			}
		}

	}
	iter.Release()
	return iter.Error()
}

// Async is DEPRECATED
// if it was true then it could use different to update the back-end storage, now it does nothing.
func (db *Database) Async(useGoRoutines bool) *Database {
	return db
}

// Load loads the sessions from the LevelDB(file-based) session storage.
func (db *Database) Load(sid string) (storeDB sessions.RemoteStore) {
	bsid := []byte(sid)

	iter := db.Service.NewIterator(nil, ReadOptions)
	for iter.Next() {
		// Remember that the contents of the returned slice should not be modified, and
		// only valid until the next call to Next.
		k := iter.Key()

		if len(k) > 0 {
			v := iter.Value()
			if bytes.Equal(k, bsid) { // session id should be the name of the key-value pair
				store, err := sessions.DecodeRemoteStore(v) // decode the whole value, as a remote store
				if err != nil {
					golog.Errorf("error while trying to load from the remote store: %v", err)
				} else {
					storeDB = store
				}
				break
			}
		}

	}

	iter.Release()
	if err := iter.Error(); err != nil {
		golog.Errorf("error while trying to iterate over the database: %v", err)
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
			golog.Errorf("error while destroying a session(%s) from leveldb: %v",
				p.SessionID, err)
		}
		return
	}

	s, err := p.Store.Serialize()
	if err != nil {
		golog.Errorf("error while serializing the remote store: %v", err)
	}

	err = db.Service.Put(bsid, s, WriteOptions)

	if err != nil {
		golog.Errorf("error while writing the session(%s) to the database: %v", p.SessionID, err)
	}
}

func (db *Database) destroy(bsid []byte) error {
	return db.Service.Delete(bsid, WriteOptions)
}

// Close shutdowns the LevelDB connection.
func (db *Database) Close() error {
	return closeDB(db)
}

func closeDB(db *Database) error {
	err := db.Service.Close()
	if err != nil {
		golog.Warnf("closing the LevelDB connection: %v", err)
	}

	return err
}
