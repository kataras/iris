package file

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"os"
)

// Structure for the file-storage
type Database struct {
	path string
}

// New returns a new session storage instance
func New(p string) *Database {
	return &Database{path: p}
}

// Load loads the values to the underline
func (d *Database) Load(sid string) map[string]interface{} {
	values := make(map[string]interface{})

	val, err := ioutil.ReadFile(d.path + "/" + sid)
	if err == nil {
		err = DeserializeBytes(val, &values)
	}

	return values

}

// serialize the values to be stored as strings inside the session storage
func serialize(values map[string]interface{}) []byte {
	val, err := SerializeBytes(values)
	if err != nil {
		println("Filestorage serialize error: " + err.Error())
	}

	return val
}

// Update updates the session storage
func (d *Database) Update(sid string, newValues map[string]interface{}) {
	if len(newValues) == 0 {
		go os.Remove(d.path + "/" + sid)
	} else {
		ioutil.WriteFile(d.path+"/"+sid, serialize(newValues), 0600)
	}

}

// SerializeBytes serializa bytes using gob encoder and returns them
func SerializeBytes(m interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(m)
	if err == nil {
		return buf.Bytes(), nil
	}
	return nil, err
}

// DeserializeBytes converts the bytes to an object using gob decoder
func DeserializeBytes(b []byte, m interface{}) error {
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	return dec.Decode(m) //no reference here otherwise doesn't work because of go remote object
}
