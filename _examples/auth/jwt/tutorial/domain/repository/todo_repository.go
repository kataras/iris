package repository

import (
	"errors"
	"sync"

	"myapp/domain/model"
	"myapp/util"
)

// ErrNotFound indicates that an entry was not found.
// Usage: errors.Is(err, ErrNotFound)
var ErrNotFound = errors.New("not found")

// TodoRepository is responsible for Todo CRUD operations,
// however, for the sake of the example we only implement the Create and Read ones.
type TodoRepository interface {
	Create(userID, title, body string) (model.Todo, error)
	GetByID(id string) (model.Todo, error)
	GetAll() ([]model.Todo, error)
	GetAllByUser(userID string) ([]model.Todo, error)
}

var (
	_ TodoRepository = (*memoryTodoRepository)(nil)
)

type memoryTodoRepository struct {
	todos []model.Todo // map[string]model.Todo
	mu    sync.RWMutex
}

// NewMemoryTodoRepository returns the default in-memory todo repository.
func NewMemoryTodoRepository() TodoRepository {
	r := new(memoryTodoRepository)
	return r
}

func (r *memoryTodoRepository) Create(userID, title, body string) (model.Todo, error) {
	id, err := util.GenerateUUID()
	if err != nil {
		return model.Todo{}, err
	}

	todo := model.Todo{
		ID:        id,
		UserID:    userID,
		Title:     title,
		Body:      body,
		CreatedAt: util.Now().Unix(),
	}

	r.mu.Lock()
	r.todos = append(r.todos, todo)
	r.mu.Unlock()

	return todo, nil
}

func (r *memoryTodoRepository) GetByID(id string) (model.Todo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, todo := range r.todos {
		if todo.ID == id {
			return todo, nil
		}
	}

	return model.Todo{}, ErrNotFound
}

func (r *memoryTodoRepository) GetAll() ([]model.Todo, error) {
	r.mu.RLock()
	tmp := make([]model.Todo, len(r.todos))
	copy(tmp, r.todos)
	r.mu.RUnlock()
	return tmp, nil
}

func (r *memoryTodoRepository) GetAllByUser(userID string) ([]model.Todo, error) {
	// initialize a slice, so we don't have "null" at empty response.
	todos := make([]model.Todo, 0)

	r.mu.RLock()
	for _, todo := range r.todos {
		if todo.UserID == userID {
			todos = append(todos, todo)
		}
	}
	r.mu.RUnlock()

	return todos, nil
}
