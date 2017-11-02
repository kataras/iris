package file

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/kataras/golog"
	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/sessions"
)

// DefaultFileMode used as the default database's "fileMode"
// for creating the sessions directory path, opening and write the session file.
var (
	DefaultFileMode = 0755
)

// Database is the basic file-storage session database.
//
// What it does
// It removes old(expired) session files, at init (`Cleanup`).
// It creates a session file on the first inserted key-value session data.
// It removes a session file on destroy.
// It sync the session file to the session's memstore on any other action (insert, delete, clear).
// It automatically remove the session files on runtime when a session is expired.
//
// Remember: sessions are not a storage for large data, everywhere: on any platform on any programming language.
type Database struct {
	dir      string
	fileMode os.FileMode // defaults to DefaultFileMode if missing.
}

// New creates and returns a new file-storage database instance based on the "directoryPath".
// DirectoryPath should is the directory which the leveldb database will store the sessions,
// i.e ./sessions/
//
// It will remove any old session files.
func New(directoryPath string, fileMode os.FileMode) (*Database, error) {
	lindex := directoryPath[len(directoryPath)-1]
	if lindex != os.PathSeparator && lindex != '/' {
		directoryPath += string(os.PathSeparator)
	}

	if fileMode <= 0 {
		fileMode = os.FileMode(DefaultFileMode)
	}

	// create directories if necessary
	if err := os.MkdirAll(directoryPath, fileMode); err != nil {
		return nil, err
	}

	db := &Database{dir: directoryPath, fileMode: fileMode}
	return db, db.Cleanup()
}

// Cleanup removes any invalid(have expired) session files, it's being called automatically on `New` as well.
func (db *Database) Cleanup() error {
	return filepath.Walk(db.dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		sessPath := path
		storeDB, _ := db.load(sessPath) // we don't care about errors here, the file may be not a session a file at all.
		if storeDB.Lifetime.HasExpired() {
			os.Remove(path)
		}
		return nil
	})
}

// FileMode for creating the sessions directory path, opening and write the session file.
//
// Defaults to 0755.
func (db *Database) FileMode(fileMode uint32) *Database {
	db.fileMode = os.FileMode(fileMode)
	return db
}

// Async is DEPRECATED
// if it was true then it could use different to update the back-end storage, now it does nothing.
func (db *Database) Async(useGoRoutines bool) *Database {
	return db
}

func (db *Database) sessPath(sid string) string {
	return filepath.Join(db.dir, sid)
}

// Load loads the values from the storage and returns them
func (db *Database) Load(sid string) sessions.RemoteStore {
	sessPath := db.sessPath(sid)
	store, err := db.load(sessPath)
	if err != nil {
		golog.Error(err.Error())
	}
	return store
}

func (db *Database) load(fileName string) (storeDB sessions.RemoteStore, loadErr error) {
	f, err := os.OpenFile(fileName, os.O_RDONLY, db.fileMode)

	if err != nil {
		// we don't care if filepath doesn't exists yet, it will be created later on.
		return
	}

	defer f.Close()

	contents, err := ioutil.ReadAll(f)

	if err != nil {
		loadErr = errors.New("error while reading the session file's data: %v").Format(err)
		return
	}

	storeDB, err = sessions.DecodeRemoteStore(contents)

	if err != nil { // we care for this error only
		loadErr = errors.New("load error: %v").Format(err)
		return
	}

	return
}

// Sync syncs the database.
func (db *Database) Sync(p sessions.SyncPayload) {
	db.sync(p)
}

func (db *Database) sync(p sessions.SyncPayload) {

	// if destroy then remove the file from the disk
	if p.Action == sessions.ActionDestroy {
		if err := db.destroy(p.SessionID); err != nil {
			golog.Errorf("error while destroying and removing the session file: %v", err)
		}
		return
	}

	if err := db.override(p.SessionID, p.Store); err != nil {
		golog.Errorf("error while writing the session file: %v", err)
	}

}

// good idea but doesn't work, it is not just an array of entries
// which can be appended with the gob...anyway session data should be small so we don't have problem
// with that:

// on insert new data, it appends to the file
// func (db *Database) insert(sid string, entry memstore.Entry) error {
// 	f, err := os.OpenFile(
// 		db.sessPath(sid),
// 		os.O_WRONLY|os.O_CREATE|os.O_RDWR|os.O_APPEND,
// 		db.fileMode,
// 	)

// 	if err != nil {
// 		return err
// 	}

// 	if _, err := f.Write(serializeEntry(entry)); err != nil {
// 		f.Close()
// 		return err
// 	}

// 	return f.Close()
// }

// removes all entries but keeps the file.
// func (db *Database) clearAll(sid string) error {
// 	return ioutil.WriteFile(
// 		db.sessPath(sid),
// 		[]byte{},
// 		db.fileMode,
// 	)
// }

// on update, remove and clear, it re-writes the file to the current values(may empty).
func (db *Database) override(sid string, store sessions.RemoteStore) error {
	s, err := store.Serialize()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(
		db.sessPath(sid),
		s,
		db.fileMode,
	)
}

// on destroy, it removes the file
func (db *Database) destroy(sid string) error {
	return db.expireSess(sid)
}

func (db *Database) expireSess(sid string) error {
	sessPath := db.sessPath(sid)
	return os.Remove(sessPath)
}
