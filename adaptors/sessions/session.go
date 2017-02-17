package sessions

import (
	"strconv"
	"sync"
	"time"

	"github.com/kataras/go-errors"
	"gopkg.in/kataras/iris.v6"
)

type (

	// session is an 'object' which wraps the session provider with its session databases, only frontend user has access to this session object.
	// implements the iris.Session interface
	session struct {
		sid    string
		values map[string]interface{} // here are the real values
		// we could set the flash messages inside values but this will bring us more problems
		// because of session databases and because of
		// users may want to get all sessions and save them or display them
		// but without temp values (flash messages) which are removed after fetching.
		// so introduce a new field here.
		// NOTE: flashes are not managed by third-party, only inside session struct.
		flashes   map[string]*flashMessage
		mu        sync.RWMutex
		createdAt time.Time
		provider  *provider
	}

	flashMessage struct {
		// if true then this flash message is removed on the flash gc
		shouldRemove bool
		value        interface{}
	}
)

var _ iris.Session = &session{}

// ID returns the session's id
func (s *session) ID() string {
	return s.sid
}

// Get returns the value of an entry by its key
func (s *session) Get(key string) interface{} {
	s.mu.RLock()
	value := s.values[key]
	s.mu.RUnlock()

	return value
}

// when running on the session manager removes any 'old' flash messages
func (s *session) runFlashGC() {
	s.mu.Lock()
	for key, v := range s.flashes {
		if v.shouldRemove {
			delete(s.flashes, key)
		}
	}
	s.mu.Unlock()
}

// HasFlash returns true if this request has available flash messages
func (s *session) HasFlash() bool {
	return s.flashes != nil && len(s.flashes) > 0
}

// GetFlash returns a flash message which removed on the next request
//
// To check for flash messages we use the HasFlash() Method
// and to obtain the flash message we use the GetFlash() Method.
// There is also a method GetFlashes() to fetch all the messages.
//
// Fetching a message deletes it from the session.
// This means that a message is meant to be displayed only on the first page served to the user
func (s *session) GetFlash(key string) (v interface{}) {
	s.mu.Lock()
	if valueStorage, found := s.flashes[key]; found {
		valueStorage.shouldRemove = true
		v = valueStorage.value
	}
	s.mu.Unlock()

	return
}

// GetString same as Get but returns as string, if nil then returns an empty string
func (s *session) GetString(key string) string {
	if value := s.Get(key); value != nil {
		if v, ok := value.(string); ok {
			return v
		}
	}

	return ""
}

// GetFlashString same as GetFlash but returns as string, if nil then returns an empty string
func (s *session) GetFlashString(key string) string {
	if value := s.GetFlash(key); value != nil {
		if v, ok := value.(string); ok {
			return v
		}
	}

	return ""
}

var errFindParse = errors.New("Unable to find the %s with key: %s. Found? %#v")

// GetInt same as Get but returns as int, if not found then returns -1 and an error
func (s *session) GetInt(key string) (int, error) {
	v := s.Get(key)
	if vint, ok := v.(int); ok {
		return vint, nil
	} else if vstring, sok := v.(string); sok {
		return strconv.Atoi(vstring)
	}

	return -1, errFindParse.Format("int", key, v)
}

// GetInt64 same as Get but returns as int64, if not found then returns -1 and an error
func (s *session) GetInt64(key string) (int64, error) {
	v := s.Get(key)
	if vint64, ok := v.(int64); ok {
		return vint64, nil
	} else if vint, ok := v.(int); ok {
		return int64(vint), nil
	} else if vstring, sok := v.(string); sok {
		return strconv.ParseInt(vstring, 10, 64)
	}

	return -1, errFindParse.Format("int64", key, v)

}

// GetFloat32 same as Get but returns as float32, if not found then returns -1 and an error
func (s *session) GetFloat32(key string) (float32, error) {
	v := s.Get(key)
	if vfloat32, ok := v.(float32); ok {
		return vfloat32, nil
	} else if vfloat64, ok := v.(float64); ok {
		return float32(vfloat64), nil
	} else if vint, ok := v.(int); ok {
		return float32(vint), nil
	} else if vstring, sok := v.(string); sok {
		vfloat64, err := strconv.ParseFloat(vstring, 32)
		if err != nil {
			return -1, err
		}
		return float32(vfloat64), nil
	}

	return -1, errFindParse.Format("float32", key, v)
}

// GetFloat64 same as Get but returns as float64, if not found then returns -1 and an error
func (s *session) GetFloat64(key string) (float64, error) {
	v := s.Get(key)
	if vfloat32, ok := v.(float32); ok {
		return float64(vfloat32), nil
	} else if vfloat64, ok := v.(float64); ok {
		return vfloat64, nil
	} else if vint, ok := v.(int); ok {
		return float64(vint), nil
	} else if vstring, sok := v.(string); sok {
		return strconv.ParseFloat(vstring, 32)
	}

	return -1, errFindParse.Format("float64", key, v)
}

// GetBoolean same as Get but returns as boolean, if not found then returns -1 and an error
func (s *session) GetBoolean(key string) (bool, error) {
	v := s.Get(key)
	// here we could check for "true", "false" and 0 for false and 1 for true
	// but this may cause unexpected behavior from the developer if they expecting an error
	// so we just check if bool, if yes then return that bool, otherwise return false and an error
	if vb, ok := v.(bool); ok {
		return vb, nil
	}

	return false, errFindParse.Format("bool", key, v)
}

// GetAll returns a copy of all session's values
func (s *session) GetAll() map[string]interface{} {
	items := make(map[string]interface{}, len(s.values))
	s.mu.RLock()
	for key, v := range s.values {
		items[key] = v
	}
	s.mu.RUnlock()
	return items
}

// GetFlashes returns all flash messages as map[string](key) and interface{} value
// NOTE: this will cause at remove all current flash messages on the next request of the same user
func (s *session) GetFlashes() map[string]interface{} {
	flashes := make(map[string]interface{}, len(s.flashes))
	s.mu.Lock()
	for key, v := range s.flashes {
		flashes[key] = v.value
		v.shouldRemove = true
	}
	s.mu.Unlock()
	return flashes
}

// VisitAll loop each one entry and calls the callback function func(key,value)
func (s *session) VisitAll(cb func(k string, v interface{})) {
	for key := range s.values {
		cb(key, s.values[key])
	}
}

// Set fills the session with an entry, it receives a key and a value
// returns an error, which is always nil
func (s *session) Set(key string, value interface{}) {
	s.mu.Lock()
	s.values[key] = value
	s.mu.Unlock()

	s.updateDatabases()
}

// SetFlash sets a flash message by its key.
//
// A flash message is used in order to keep a message in session through one or several requests of the same user.
// It is removed from session after it has been displayed to the user.
// Flash messages are usually used in combination with HTTP redirections,
// because in this case there is no view, so messages can only be displayed in the request that follows redirection.
//
// A flash message has a name and a content (AKA key and value).
// It is an entry of an associative array. The name is a string: often "notice", "success", or "error", but it can be anything.
// The content is usually a string. You can put HTML tags in your message if you display it raw.
// You can also set the message value to a number or an array: it will be serialized and kept in session like a string.
//
// Flash messages can be set using the SetFlash() Method
// For example, if you would like to inform the user that his changes were successfully saved,
// you could add the following line to your Handler:
//
// SetFlash("success", "Data saved!");
//
// In this example we used the key 'success'.
// If you want to define more than one flash messages, you will have to use different keys
func (s *session) SetFlash(key string, value interface{}) {
	s.mu.Lock()
	s.flashes[key] = &flashMessage{value: value}
	s.mu.Unlock()
}

// Delete removes an entry by its key
func (s *session) Delete(key string) {
	s.mu.Lock()
	delete(s.values, key)
	s.mu.Unlock()

	s.updateDatabases()
}

func (s *session) updateDatabases() {
	s.provider.updateDatabases(s.sid, s.values)
}

// DeleteFlash removes a flash message by its key
func (s *session) DeleteFlash(key string) {
	s.mu.Lock()
	delete(s.flashes, key)
	s.mu.Unlock()
}

// Clear removes all entries
func (s *session) Clear() {
	s.mu.Lock()
	for key := range s.values {
		delete(s.values, key)
	}
	s.mu.Unlock()

	s.updateDatabases()
}

// Clear removes all flash messages
func (s *session) ClearFlashes() {
	s.mu.Lock()
	for key := range s.flashes {
		delete(s.flashes, key)
	}
	s.mu.Unlock()
}
