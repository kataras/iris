package file

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/kataras/golog"
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

	return &Database{path: path}
}

func (d *Database) sessPath(sid string) string {
	return filepath.Join(d.path, sid)
}

// Load loads the values to the underline
func (d *Database) Load(sid string) map[string]interface{} {
	values := make(map[string]interface{})

	val, err := ioutil.ReadFile(d.sessPath(sid))

	if err == nil {
		err = DeserializeBytes(val, &values)
	}

	if err != nil {
		golog.Errorf("load error: %v", err)
	}

	return values
}

// serialize the values to be stored as strings inside the session file-storage.
func serialize(values map[string]interface{}) []byte {
	val, err := SerializeBytes(values)
	if err != nil {
		golog.Errorf("serialize error: %v", err)
	}

	return val
}

// Update updates the session file-storage.
func (d *Database) Update(sid string, newValues map[string]interface{}) {
	sessPath := d.sessPath(sid)
	if len(newValues) == 0 {
		go os.Remove(sessPath)
		return
	}

	ioutil.WriteFile(sessPath, serialize(newValues), 0666)
}

// SerializeBytes serializes the "m" into bytes using gob encoder and and returns the result.
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
