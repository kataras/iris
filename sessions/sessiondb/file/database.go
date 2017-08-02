package file

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/kataras/golog"
)

var (
	// PathFileMode for creating the sessions directory path, opening and write the session file.
	// Defaults to 0666.
	PathFileMode uint32 = 0666
)

// Database is the basic file-storage session database.
type Database struct {
	path string
}

// New returns a new file-storage database instance based on the "path".
func New(path string) *Database {
	lindex := path[len(path)-1]
	if lindex != os.PathSeparator && lindex != '/' {
		path += string(os.PathSeparator)
	}
	// create directories if necessary
	os.MkdirAll(path, os.FileMode(PathFileMode))
	return &Database{path: path}
}

func (d *Database) sessPath(sid string) string {
	return filepath.Join(d.path, sid)
}

// Load loads the values to the underline
func (d *Database) Load(sid string) (values map[string]interface{}, expireDate *time.Time) {
	sessPath := d.sessPath(sid)
	f, err := os.OpenFile(sessPath, os.O_RDONLY, os.FileMode(PathFileMode))

	if err != nil {
		// we don't care if filepath doesn't exists yet, it will be created on Update.
		return
	}

	defer f.Close()

	val, err := ioutil.ReadAll(f)

	if err != nil {
		// we don't care if filepath doesn't exists yet, it will be created on Update.
		golog.Errorf("error while reading the session file's data: %v", err)
		return
	}

	if err == nil {
		err = DeserializeBytes(val, &values)
		if err != nil { // we care for this error only
			golog.Errorf("load error: %v", err)
		}
	}

	return // no expiration
}

// serialize the values to be stored as strings inside the session file-storage.
func serialize(values map[string]interface{}) []byte {
	val, err := SerializeBytes(values)
	if err != nil {
		golog.Errorf("serialize error: %v", err)
	}

	return val
}

func (d *Database) expireSess(sid string) {
	go os.Remove(d.sessPath(sid))
}

// Update updates the session file-storage.
func (d *Database) Update(sid string, newValues map[string]interface{}, expireDate *time.Time) {

	if len(newValues) == 0 { // means delete by call
		d.expireSess(sid)
		return
	}

	// delete the file on expiration
	if expireDate != nil && !expireDate.IsZero() {
		now := time.Now()

		if expireDate.Before(now) {
			// already expirated, delete it now and return.
			d.expireSess(sid)
			return
		}
		// otherwise set a timer to delete the file automatically
		afterDur := expireDate.Sub(now)
		time.AfterFunc(afterDur, func() {
			d.expireSess(sid)
		})
	}

	if err := ioutil.WriteFile(d.sessPath(sid), serialize(newValues), os.FileMode(PathFileMode)); err != nil {
		golog.Errorf("error while writing the session to the file: %v", err)
	}
}

// SerializeBytes serializes the "m" into bytes using gob encoder and returns the result.
func SerializeBytes(m interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(m)
	if err == nil {
		return buf.Bytes(), nil
	}
	return nil, err
}

// DeserializeBytes converts the bytes to a go value and puts that to "m" using the gob decoder.
func DeserializeBytes(b []byte, m interface{}) error {
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	return dec.Decode(m) //no reference here otherwise doesn't work because of go remote object
}
