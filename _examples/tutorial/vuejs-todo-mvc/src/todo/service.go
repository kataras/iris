package todo

type Service interface {
	GetByID(id int64) (Item, bool)
	GetByOwner(owner string) []Item
	Complete(item Item) bool
	Save(newItem Item) error
}

type MemoryService struct {
	items map[int64]Item
}

func (s *MemoryService) getLatestID() (id int64) {
	for k := range s.items {
		if k > id {
			id = k
		}
	}

	return
}

func (s *MemoryService) GetByID(id int64) (Item, bool) {
	item, found := s.items[id]
	return item, found
}

func (s *MemoryService) GetByOwner(owner string) (items []Item) {
	for _, item := range s.items {
		if item.OwnerID != owner {
			continue
		}
		items = append(items, item)
	}
	return
}

func (s *MemoryService) Complete(item Item) bool {
	item.CurrentState = StateCompleted
	return s.Save(item) == nil
}

func (s *MemoryService) Save(newItem Item) error {
	if newItem.ID == 0 {
		// create
		newItem.ID = s.getLatestID() + 1
	}

	//  full replace here for the shake of simplicy)
	s.items[newItem.ID] = newItem
	return nil
}
