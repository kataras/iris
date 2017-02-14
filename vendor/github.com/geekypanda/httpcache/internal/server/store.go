package server

import (
	"sync"
	"time"

	"github.com/geekypanda/httpcache/internal"
)

type (

	// Store is the interface of the cache bug, default is memory store for performance reasons
	Store interface {
		// Set adds an entry to the cache by its key
		// entry must contain a valid status code, conten type, a body and optional, the expiration duration
		Set(key string, statusCode int, contentType string, body []byte, expiration time.Duration)
		// Get returns an entry based on its key
		Get(key string) *internal.Entry
		// Remove removes a cache entry for the cache
		// a Set action is needed to re-enable this entry
		// it is used only on manually invalidate cache
		// otherwise a silent update to the underline entry's
		// Response is done
		Remove(key string)
	}

	// memoryStore keeps the cache bag, by default httpcache package provides one global default cache service  which provides these functions:
	// `httpcache.Cache`, `httpcache.Invalidate` and `httpcache.Start`
	// Store and NewStore used only when you want to have two different separate cache bags
	memoryStore struct {
		cache map[string]*internal.Entry
		mu    sync.RWMutex
	}
)

// NewMemoryStore returns a new memory store for the cache ,
// note that httpcache package provides one global default cache service  which provides these functions:
// `httpcache.Cache`, `httpcache.Invalidate` and `httpcache.Start`
//
// If you use only one global cache for all of your routes use the `httpcache.New` instead
func NewMemoryStore() Store {
	return &memoryStore{
		cache: make(map[string]*internal.Entry),
		mu:    sync.RWMutex{},
	}
}

func (s *memoryStore) Set(key string, statusCode int, contentType string, body []byte, expiration time.Duration) {
	e := internal.NewEntry(expiration)
	e.Reset(statusCode, contentType, body, nil)
	s.mu.Lock()
	s.cache[key] = e
	s.mu.Unlock()
}

func (s *memoryStore) Get(key string) *internal.Entry {
	s.mu.RLock()
	if v, ok := s.cache[key]; ok {
		s.mu.RUnlock()
		// println("store.go:107 GET from cache entry: " + key)
		return v
	}
	s.mu.RUnlock()
	return nil
}

func (s *memoryStore) Remove(key string) {
	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()
}

func (s *memoryStore) Clear() {
	s.mu.Lock()
	for k := range s.cache {
		delete(s.cache, k)
	}
	s.mu.Unlock()
}
