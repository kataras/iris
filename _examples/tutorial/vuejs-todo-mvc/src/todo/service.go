package todo

import (
	"sync"
)

type Service interface {
	Get(owner string) []Item
	Save(owner string, newItems []Item) error
}

type MemoryService struct {
	// key = session id, value the list of todo items that this session id has.
	items map[string][]Item
	// protected by locker for concurrent access.
	mu sync.RWMutex
}

func NewMemoryService() *MemoryService {
	return &MemoryService{
		items: make(map[string][]Item, 0),
	}
}

func (s *MemoryService) Get(sessionOwner string) []Item {
	s.mu.RLock()
	items := s.items[sessionOwner]
	s.mu.RUnlock()

	return items
}

func (s *MemoryService) Save(sessionOwner string, newItems []Item) error {
	var prevID int64
	for i := range newItems {
		if newItems[i].ID == 0 {
			newItems[i].ID = prevID
			prevID++
		}
	}

	s.mu.Lock()
	s.items[sessionOwner] = newItems
	s.mu.Unlock()
	return nil
}
