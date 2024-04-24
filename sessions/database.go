package sessions

import (
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/memstore"

	"github.com/kataras/golog"
)

// ErrNotImplemented is returned when a particular feature is not yet implemented yet.
// It can be matched directly, i.e: `isNotImplementedError := sessions.ErrNotImplemented.Equal(err)`.
var ErrNotImplemented = errors.New("not implemented yet")

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
	// SetLogger should inject a logger to this Database.
	SetLogger(*golog.Logger)
	// Acquire receives a session's lifetime from the database,
	// if the return value is LifeTime{} then the session manager sets the life time based on the expiration duration lives in configuration.
	Acquire(sid string, expires time.Duration) memstore.LifeTime
	// OnUpdateExpiration should re-set the expiration (ttl) of the session entry inside the database,
	// it is fired on `ShiftExpiration` and `UpdateExpiration`.
	// If the database does not support change of ttl then the session entry will be cloned to another one
	// and the old one will be removed, it depends on the chosen database storage.
	//
	// Check of error is required, if error returned then the rest session's keys are not proceed.
	//
	// If a database does not support this feature then an `ErrNotImplemented` will be returned instead.
	OnUpdateExpiration(sid string, newExpires time.Duration) error
	// Set sets a key value of a specific session.
	// The "immutable" input argument depends on the store, it may not implement it at all.
	Set(sid string, key string, value interface{}, ttl time.Duration, immutable bool) error
	// Get retrieves a session value based on the key.
	Get(sid string, key string) interface{}
	// Decode binds the "outPtr" to the value associated to the provided "key".
	Decode(sid, key string, outPtr interface{}) error
	// Visit loops through all session keys and values.
	Visit(sid string, cb func(key string, value interface{})) error
	// Len returns the length of the session's entries (keys).
	Len(sid string) int
	// Delete removes a session key value based on its key.
	Delete(sid string, key string) (deleted bool)
	// Clear removes all session key values but it keeps the session entry.
	Clear(sid string) error
	// Release destroys the session, it clears and removes the session entry,
	// session manager will create a new session ID on the next request after this call.
	Release(sid string) error
	// Close should terminate the database connection. It's called automatically on interrupt signals.
	Close() error
}

// DatabaseRequestHandler is an optional interface that a sessions database
// can implement. It contains a single EndRequest method which is fired
// on the very end of the request life cycle. It should be used to Flush
// any local session's values to the client.
type DatabaseRequestHandler interface {
	EndRequest(ctx *context.Context, session *Session)
}

type mem struct {
	values map[string]*memstore.Store
	mu     sync.RWMutex
}

var _ Database = (*mem)(nil)

func newMemDB() Database { return &mem{values: make(map[string]*memstore.Store)} }

func (s *mem) SetLogger(*golog.Logger) {}

func (s *mem) Acquire(sid string, expires time.Duration) memstore.LifeTime {
	s.mu.Lock()
	s.values[sid] = new(memstore.Store)
	s.mu.Unlock()
	return memstore.LifeTime{}
}

// Do nothing, the `LifeTime` of the Session will be managed by the callers automatically on memory-based storage.
func (s *mem) OnUpdateExpiration(string, time.Duration) error { return nil }

// immutable depends on the store, it may not implement it at all.
func (s *mem) Set(sid string, key string, value interface{}, _ time.Duration, immutable bool) error {
	s.mu.RLock()
	store, ok := s.values[sid]
	s.mu.RUnlock()
	if ok {
		store.Save(key, value, immutable)
	}

	return nil
}

func (s *mem) Get(sid string, key string) interface{} {
	s.mu.RLock()
	store, ok := s.values[sid]
	s.mu.RUnlock()
	if ok {
		return store.Get(key)
	}

	return nil
}

func (s *mem) Decode(sid string, key string, outPtr interface{}) error {
	v := s.Get(sid, key)
	if v != nil {
		reflect.ValueOf(outPtr).Set(reflect.ValueOf(v))
	}
	return nil
}

func (s *mem) Visit(sid string, cb func(key string, value interface{})) error {
	s.mu.RLock()
	store, ok := s.values[sid]
	s.mu.RUnlock()
	if ok {
		store.Visit(cb)
	}

	return nil
}

func (s *mem) Len(sid string) int {
	s.mu.RLock()
	store, ok := s.values[sid]
	s.mu.RUnlock()
	if ok {
		return store.Len()
	}

	return 0
}

func (s *mem) Delete(sid string, key string) (deleted bool) {
	s.mu.RLock()
	store, ok := s.values[sid]
	s.mu.RUnlock()
	if ok {
		deleted = store.Remove(key)
	}

	return
}

func (s *mem) Clear(sid string) error {
	s.mu.RLock()
	store, ok := s.values[sid]
	s.mu.RUnlock()
	if ok {
		store.Reset()
	}

	return nil
}

func (s *mem) Release(sid string) error {
	s.mu.Lock()
	delete(s.values, sid)
	s.mu.Unlock()
	return nil
}

func (s *mem) Close() error { return nil }
