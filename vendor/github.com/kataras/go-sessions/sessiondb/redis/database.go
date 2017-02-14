package redis

import (
	"bytes"
	"encoding/gob"

	"github.com/kataras/go-sessions/sessiondb/redis/service"
)

// Database the redis database for q sessions
type Database struct {
	redis *service.Service
}

// New returns a new redis database
func New(cfg ...service.Config) *Database {
	return &Database{redis: service.New(cfg...)}
}

// Config returns the configuration for the redis server bridge, you can change them
func (d *Database) Config() *service.Config {
	return d.redis.Config
}

// Load loads the values to the underline
func (d *Database) Load(sid string) map[string]interface{} {
	values := make(map[string]interface{})

	if !d.redis.Connected { //yes, check every first time's session for valid redis connection
		d.redis.Connect()
		_, err := d.redis.PingPong()
		if err != nil {
			if err != nil {
				// don't use to get the logger, just prin these to the console... atm
				println("Redis Connection error on Connect: " + err.Error())
				println("But don't panic, auto-switching to memory store right now!")
			}
		}
	}
	//fetch the values from this session id and copy-> store them
	val, err := d.redis.GetBytes(sid)
	if err == nil {
		err = DeserializeBytes(val, &values)
	}

	return values

}

// serialize the values to be stored as strings inside the Redis, we panic at any serialization error here
func serialize(values map[string]interface{}) []byte {
	val, err := SerializeBytes(values)
	if err != nil {
		println("On redisstore.serialize: " + err.Error())
	}

	return val
}

// Update updates the real redis store
func (d *Database) Update(sid string, newValues map[string]interface{}) {
	if len(newValues) == 0 {
		go d.redis.Delete(sid)
	} else {
		go d.redis.Set(sid, serialize(newValues)) //set/update all the values
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
