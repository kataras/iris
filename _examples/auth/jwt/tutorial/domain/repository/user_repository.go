package repository

import (
	"sync"

	"myapp/domain/model"
	"myapp/util"
)

// UserRepository is responsible for User CRUD operations,
// however, for the sake of the example we only implement the Read one.
type UserRepository interface {
	Create(username, password string, roles ...model.Role) (model.User, error)
	// GetByUsernameAndPassword should return a User based on the given input.
	GetByUsernameAndPassword(username, password string) (model.User, bool)
	GetAll() ([]model.User, error)
}

var (
	_ UserRepository = (*memoryUserRepository)(nil)
)

type memoryUserRepository struct {
	// Users represents a user database.
	// For the sake of the tutorial we use a simple slice of users.
	users []model.User
	mu    sync.RWMutex
}

// NewMemoryUserRepository returns the default in-memory user repository.
func NewMemoryUserRepository() UserRepository {
	r := new(memoryUserRepository)
	return r
}

func (r *memoryUserRepository) Create(username, password string, roles ...model.Role) (model.User, error) {
	id, err := util.GenerateUUID()
	if err != nil {
		return model.User{}, err
	}

	hashedPassword, err := util.GeneratePassword(password)
	if err != nil {
		return model.User{}, err
	}

	user := model.User{
		ID:             id,
		Username:       username,
		HashedPassword: hashedPassword,
		Roles:          roles,
	}

	r.mu.Lock()
	r.users = append(r.users, user)
	r.mu.Unlock()

	return user, nil
}

// GetByUsernameAndPassword returns a user from the storage based on the given "username" and "password".
func (r *memoryUserRepository) GetByUsernameAndPassword(username, password string) (model.User, bool) {
	for _, u := range r.users { // our example uses a static slice.
		if u.Username == username {
			// we compare the user input and the stored hashed password.
			ok := util.ValidatePassword(password, u.HashedPassword)
			if ok {
				return u, true
			}
		}
	}

	return model.User{}, false
}

func (r *memoryUserRepository) GetAll() ([]model.User, error) {
	r.mu.RLock()
	tmp := make([]model.User, len(r.users))
	copy(tmp, r.users)
	r.mu.RUnlock()
	return tmp, nil
}
