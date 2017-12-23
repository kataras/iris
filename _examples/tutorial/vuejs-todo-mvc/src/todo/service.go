package todo

type Service interface {
	Get(owner string) []Item
	Save(owner string, newItems []Item) error
}

type MemoryService struct {
	items map[string][]Item
}

func NewMemoryService() *MemoryService {
	return &MemoryService{make(map[string][]Item, 0)}
}

func (s *MemoryService) Get(sessionOwner string) (items []Item) {
	return s.items[sessionOwner]
}

func (s *MemoryService) Save(sessionOwner string, newItems []Item) error {
	var prevID int64
	for i := range newItems {
		if newItems[i].ID == 0 {
			newItems[i].ID = prevID
			prevID++
		}
	}

	s.items[sessionOwner] = newItems
	return nil
}
