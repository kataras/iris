package user

import (
	"errors"
	"sync"
	"time"
)

// IDGenerator would be our user ID generator
// but here we keep the order of users by their IDs
// so we will use numbers that can be easly written
// to the browser to get results back from the REST API.
// var IDGenerator = func() string {
// 	return uuid.NewV4().String()
// }

// DataSource is our data store example.
type DataSource struct {
	Users map[int64]Model
	mu    sync.RWMutex
}

// NewDataSource returns a new user data source.
func NewDataSource() *DataSource {
	return &DataSource{
		Users: make(map[int64]Model),
	}
}

// GetBy returns receives a query function
// which is fired for every single user model inside
// our imaginary database.
// When that function returns true then it stops the iteration.
//
// It returns the query's return last known boolean value
// and the last known user model
// to help callers to reduce the loc.
//
// But be carefully, the caller should always check for the "found"
// because it may be false but the user model has actually real data inside it.
//
// It's actually a simple but very clever prototype function
// I'm think of and using everywhere since then,
// hope you find it very useful too.
func (d *DataSource) GetBy(query func(Model) bool) (user Model, found bool) {
	d.mu.RLock()
	for _, user = range d.Users {
		found = query(user)
		if found {
			break
		}
	}
	d.mu.RUnlock()
	return
}

// GetByID returns a user model based on its ID.
func (d *DataSource) GetByID(id int64) (Model, bool) {
	return d.GetBy(func(u Model) bool {
		return u.ID == id
	})
}

// GetByUsername returns a user model based on the Username.
func (d *DataSource) GetByUsername(username string) (Model, bool) {
	return d.GetBy(func(u Model) bool {
		return u.Username == username
	})
}

func (d *DataSource) getLastID() (lastID int64) {
	d.mu.RLock()
	for id := range d.Users {
		if id > lastID {
			lastID = id
		}
	}
	d.mu.RUnlock()

	return lastID
}

// InsertOrUpdate adds or updates a user to the (memory) storage.
func (d *DataSource) InsertOrUpdate(user Model) (Model, error) {
	// no matter what we will update the password hash
	// for both update and insert actions.
	hashedPassword, err := GeneratePassword(user.password)
	if err != nil {
		return user, err
	}
	user.HashedPassword = hashedPassword

	// update
	if id := user.ID; id > 0 {
		_, found := d.GetByID(id)
		if !found {
			return user, errors.New("ID should be zero or a valid one that maps to an existing User")
		}
		d.mu.Lock()
		d.Users[id] = user
		d.mu.Unlock()
		return user, nil
	}

	// insert
	id := d.getLastID() + 1
	user.ID = id
	d.mu.Lock()
	user.CreatedAt = time.Now()
	d.Users[id] = user
	d.mu.Unlock()

	return user, nil
}
