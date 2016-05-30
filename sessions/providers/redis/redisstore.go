package redis

import (
	"time"

	"github.com/kataras/iris/sessions/store"
	"github.com/kataras/iris/utils"
)

/*Notes only for me
--------
Here we are setting a structure which keeps the current session's values setted by store.Set(key,value)
this is the RedisValue struct.
if noexists
	RedisValue := RedisValue{sessionid,values)

RedisValue.values[thekey]=thevalue


service.Set(store.sid,RedisValue)

because we are using the same redis service for all sessions, and this is the best way to separate them,
without prefix and all that which I tried and failed to deserialize them correctly if the value is string...
so again we will keep the current server's sessions into memory
and fetch them(the sessions) from the redis at each first session run. Yes this is the fastest way to get/set a session
and at the same time they are keep saved to the redis and the GC will cleanup the memory after a while like we are doing
with the memory provider. Or just have a values field inside the Store and use just it, yes better simpler approach.
Ok then, let's convert it again.
*/

// Values is just a type of a map[interface{}]interface{}
type Values map[interface{}]interface{}

// Store the redis session store
type Store struct {
	sid                string
	lastAccessedTime   time.Time
	values             Values
	cookieLifeDuration time.Duration //used on .Set-> SETEX on redis
}

var _ store.IStore = &Store{}

// NewStore creates and returns a new store based on the session id(string) and the cookie life duration (time.Duration)
func NewStore(sid string, cookieLifeDuration time.Duration) *Store {
	s := &Store{sid: sid, lastAccessedTime: time.Now(), cookieLifeDuration: cookieLifeDuration}
	//fetch the values from this session id and copy-> store them
	val, err := redis.GetBytes(sid)
	if err == nil {
		err = utils.DeserializeBytes(val, &s.values)
		if err != nil {
			//if deserialization failed
			s.values = Values{}
		}

	}
	if s.values == nil {
		//if key/sid wasn't found or was found but no entries in it(L72)
		s.values = Values{}
	}

	return s
}

// serialize the values to be stored as strings inside the Redis, we panic at any serialization error here
func serialize(values Values) []byte {
	val, err := utils.SerializeBytes(values)
	if err != nil {
		panic("On redisstore.serialize: " + err.Error())
	}

	return val
}

// update updates the real redis store
func (s *Store) update() {
	go redis.Set(s.sid, serialize(s.values), s.cookieLifeDuration.Seconds()) //set/update all the values, in goroutine
}

// GetAll returns all values
func (s *Store) GetAll() map[interface{}]interface{} {
	return s.values
}

// VisitAll loop each one entry and calls the callback function func(key,value)
func (s *Store) VisitAll(cb func(k interface{}, v interface{})) {
	for key := range s.values {
		cb(key, s.values[key])
	}
}

// Get returns the value of an entry by its key
func (s *Store) Get(key interface{}) interface{} {
	Provider.Update(s.sid)

	if value, found := s.values[key]; found {
		return value
	}

	return nil
}

// GetString same as Get but returns as string, if nil then returns an empty string
func (s *Store) GetString(key interface{}) string {
	if value := s.Get(key); value != nil {
		if v, ok := value.(string); ok {
			return v
		}
	}

	return ""
}

// GetInt same as Get but returns as int, if nil then returns -1
func (s *Store) GetInt(key interface{}) int {
	if value := s.Get(key); value != nil {
		if v, ok := value.(int); ok {
			return v
		}
	}

	return -1
}

// Set fills the session with an entry, it receives a key and a value
// returns an error, which is always nil
func (s *Store) Set(key interface{}, value interface{}) error {
	s.values[key] = value
	Provider.Update(s.sid)

	s.update()
	return nil
}

// Delete removes an entry by its key
// returns an error, which is always nil
func (s *Store) Delete(key interface{}) error {
	delete(s.values, key)
	Provider.Update(s.sid)
	s.update()
	return nil
}

// Clear removes all entries
// returns an error, which is always nil
func (s *Store) Clear() error {
	//we are not using the Redis.Delete, I made so work for nothing.. we wanted only the .Set at the end...
	for key := range s.values {
		delete(s.values, key)
	}

	Provider.Update(s.sid)
	s.update()
	return nil
}

// ID returns the session id
func (s *Store) ID() string {
	return s.sid
}

// LastAccessedTime returns the last time this session has been used
func (s *Store) LastAccessedTime() time.Time {
	return s.lastAccessedTime
}

// SetLastAccessedTime updates the last accessed time
func (s *Store) SetLastAccessedTime(lastacc time.Time) {
	s.lastAccessedTime = lastacc
}

// Destroy deletes entirely the session, from the memory, the client's cookie and the store
func (s *Store) Destroy() {
	// remove the whole  value which is the s.values from real redis
	redis.Delete(s.sid)
}
