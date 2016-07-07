package memory

import (
	"sync"
	"time"

	"github.com/kataras/iris/sessions/store"
)

// Store the memory store, contains the session id and the values
type Store struct {
	sid              string
	lastAccessedTime time.Time
	values           map[string]interface{} // here is the real memory store
	mu               sync.Mutex
}

var _ store.IStore = &Store{}

// GetAll returns all values
func (s *Store) GetAll() map[string]interface{} {
	return s.values
}

// VisitAll loop each one entry and calls the callback function func(key,value)
func (s *Store) VisitAll(cb func(k string, v interface{})) {
	for key := range s.values {
		cb(key, s.values[key])
	}
}

// Get returns the value of an entry by its key
func (s *Store) Get(key string) interface{} {
	Provider.Update(s.sid)
	if value, found := s.values[key]; found {
		return value
	}
	return nil
}

// GetString same as Get but returns as string, if nil then returns an empty string
func (s *Store) GetString(key string) string {
	if value := s.Get(key); value != nil {
		if v, ok := value.(string); ok {
			return v
		}

	}

	return ""
}

// GetInt same as Get but returns as int, if nil then returns -1
func (s *Store) GetInt(key string) int {
	if value := s.Get(key); value != nil {
		if v, ok := value.(int); ok {
			return v
		}
	}

	return -1
}

// Set fills the session with an entry, it receives a key and a value
// returns an error, which is always nil
func (s *Store) Set(key string, value interface{}) error {
	s.mu.Lock()
	s.values[key] = value
	s.mu.Unlock()
	Provider.Update(s.sid)
	return nil
}

// Delete removes an entry by its key
// returns an error, which is always nil
func (s *Store) Delete(key string) error {
	s.mu.Lock()
	delete(s.values, key)
	s.mu.Unlock()
	Provider.Update(s.sid)
	return nil
}

// Clear removes all entries
// returns an error, which is always nil
func (s *Store) Clear() error {
	s.mu.Lock()
	for key := range s.values {
		delete(s.values, key)
	}
	s.mu.Unlock()
	Provider.Update(s.sid)
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

// Destroy deletes all keys
func (s *Store) Destroy() {
	// clears without provider's update.
	s.mu.Lock()
	for key := range s.values {
		delete(s.values, key)
	}
	s.mu.Unlock()
}
