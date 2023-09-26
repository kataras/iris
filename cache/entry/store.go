package entry

import (
	"sync"
)

// Store is the interface which is responsible to store the cache entries.
type Store interface {
	// Get returns an entry based on its key.
	Get(key string) *Entry
	// Set sets an entry based on its key.
	Set(key string, e *Entry)
	// Delete deletes an entry based on its key.
	Delete(key string)
}

// memStore is the default in-memory store for the cache entries.
type memStore struct {
	entries map[string]*Entry
	mu      sync.RWMutex
}

var _ Store = (*memStore)(nil)

// NewMemStore returns a new in-memory store for the cache entries.
func NewMemStore() Store {
	return &memStore{
		entries: make(map[string]*Entry),
	}
}

// Get returns an entry based on its key.
func (s *memStore) Get(key string) *Entry {
	s.mu.RLock()
	e := s.entries[key]
	s.mu.RUnlock()
	return e
}

// Set sets an entry based on its key.
func (s *memStore) Set(key string, e *Entry) {
	s.mu.Lock()
	s.entries[key] = e
	s.mu.Unlock()
}

// Delete deletes an entry based on its key.
func (s *memStore) Delete(key string) {
	s.mu.Lock()
	delete(s.entries, key)
	s.mu.Unlock()
}
