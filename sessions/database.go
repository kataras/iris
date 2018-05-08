package sessions

import (
	"sync"
	"time"

	"github.com/kataras/iris/core/memstore"
)

// Database is the interface which all session databases should implement
// By design it doesn't support any type of cookie session like other frameworks.
// I want to protect you, believe me.
// The scope of the database is to store somewhere the sessions in order to
// keep them after restarting the server, nothing more.
//
// Synchronization are made automatically, you can register one using `UseDatabase`.
//
// Look the `sessiondb` folder for databases implementations.
type Database interface {
	// Acquire receives a session's lifetime from the database,
	// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
	Acquire(sid string, expires time.Duration) LifeTime
	// Set sets a key value of a specific session.
	// The "immutable" input argument depends on the store, it may not implement it at all.
	Set(sid string, lifetime LifeTime, key string, value interface{}, immutable bool)
	// Get retrieves a session value based on the key.
	Get(sid string, key string) interface{}
	// Visit loops through all session keys and values.
	Visit(sid string, cb func(key string, value interface{}))
	// Len returns the length of the session's entries (keys).
	Len(sid string) int
	// Delete removes a session key value based on its key.
	Delete(sid string, key string) (deleted bool)
	// Clear removes all session key values but it keeps the session entry.
	Clear(sid string)
	// Release destroys the session, it clears and removes the session entry,
	// session manager will create a new session ID on the next request after this call.
	Release(sid string)
}

type mem struct {
	values map[string]*memstore.Store
	mu     sync.RWMutex
}

var _ Database = (*mem)(nil)

func newMemDB() Database { return &mem{values: make(map[string]*memstore.Store)} }

func (s *mem) Acquire(sid string, expires time.Duration) LifeTime {
	s.mu.Lock()
	s.values[sid] = new(memstore.Store)
	s.mu.Unlock()
	return LifeTime{}
}

// immutable depends on the store, it may not implement it at all.
func (s *mem) Set(sid string, lifetime LifeTime, key string, value interface{}, immutable bool) {
	s.mu.RLock()
	s.values[sid].Save(key, value, immutable)
	s.mu.RUnlock()
}

func (s *mem) Get(sid string, key string) interface{} {
	s.mu.RLock()
	v := s.values[sid].Get(key)
	s.mu.RUnlock()

	return v
}

func (s *mem) Visit(sid string, cb func(key string, value interface{})) {
	s.values[sid].Visit(cb)
}

func (s *mem) Len(sid string) int {
	s.mu.RLock()
	n := s.values[sid].Len()
	s.mu.RUnlock()

	return n
}

func (s *mem) Delete(sid string, key string) (deleted bool) {
	s.mu.RLock()
	deleted = s.values[sid].Remove(key)
	s.mu.RUnlock()
	return
}

func (s *mem) Clear(sid string) {
	s.mu.Lock()
	s.values[sid].Reset()
	s.mu.Unlock()
}

func (s *mem) Release(sid string) {
	s.mu.Lock()
	delete(s.values, sid)
	s.mu.Unlock()
}
