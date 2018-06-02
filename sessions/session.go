package sessions

import (
	"strconv"
	"sync"

	"github.com/kataras/iris/core/errors"
)

type (
	// Session should expose the Sessions's end-user API.
	// It is the session's storage controller which you can
	// save or retrieve values based on a key.
	//
	// This is what will be returned when sess := sessions.Start().
	Session struct {
		sid      string
		isNew    bool
		flashes  map[string]*flashMessage
		mu       sync.RWMutex // for flashes.
		Lifetime LifeTime
		provider *provider
	}

	flashMessage struct {
		// if true then this flash message is removed on the flash gc
		shouldRemove bool
		value        interface{}
	}
)

// Destroy destroys this session, it removes its session values and any flashes.
// This session entry will be removed from the server,
// the registered session databases will be notified for this deletion as well.
//
// Note that this method does NOT remove the client's cookie, although
// it should be reseted if new session is attached to that (client).
//
// Use the session's manager `Destroy(ctx)` in order to remove the cookie as well.
func (s *Session) Destroy() {
	s.provider.deleteSession(s)
}

// ID returns the session's ID.
func (s *Session) ID() string {
	return s.sid
}

// IsNew returns true if this session is
// created by the current application's process.
func (s *Session) IsNew() bool {
	return s.isNew
}

// Get returns a value based on its "key".
func (s *Session) Get(key string) interface{} {
	return s.provider.db.Get(s.sid, key)
}

// when running on the session manager removes any 'old' flash messages.
func (s *Session) runFlashGC() {
	s.mu.Lock()
	for key, v := range s.flashes {
		if v.shouldRemove {
			delete(s.flashes, key)
		}
	}
	s.mu.Unlock()
}

// HasFlash returns true if this session has available flash messages.
func (s *Session) HasFlash() bool {
	s.mu.RLock()
	has := len(s.flashes) > 0
	s.mu.RUnlock()
	return has
}

// GetFlash returns a stored flash message based on its "key"
// which will be removed on the next request.
//
// To check for flash messages we use the HasFlash() Method
// and to obtain the flash message we use the GetFlash() Method.
// There is also a method GetFlashes() to fetch all the messages.
//
// Fetching a message deletes it from the session.
// This means that a message is meant to be displayed only on the first page served to the user.
func (s *Session) GetFlash(key string) interface{} {
	fv, ok := s.peekFlashMessage(key)
	if !ok {
		return nil
	}
	fv.shouldRemove = true
	return fv.value
}

// PeekFlash returns a stored flash message based on its "key".
// Unlike GetFlash, this will keep the message valid for the next requests,
// until GetFlashes or GetFlash("key").
func (s *Session) PeekFlash(key string) interface{} {
	fv, ok := s.peekFlashMessage(key)
	if !ok {
		return nil
	}
	return fv.value
}

func (s *Session) peekFlashMessage(key string) (*flashMessage, bool) {
	s.mu.RLock()
	fv, found := s.flashes[key]
	s.mu.RUnlock()

	if !found {
		return nil, false
	}

	return fv, true
}

// GetString same as Get but returns its string representation,
// if key doesn't exist then it returns an empty string.
func (s *Session) GetString(key string) string {
	if value := s.Get(key); value != nil {
		if v, ok := value.(string); ok {
			return v
		}

		if v, ok := value.(int); ok {
			return strconv.Itoa(v)
		}

		if v, ok := value.(int64); ok {
			return strconv.FormatInt(v, 10)
		}
	}

	return ""
}

// GetStringDefault same as Get but returns its string representation,
// if key doesn't exist then it returns the "defaultValue".
func (s *Session) GetStringDefault(key string, defaultValue string) string {
	if v := s.GetString(key); v != "" {
		return v
	}

	return defaultValue
}

// GetFlashString same as `GetFlash` but returns its string representation,
// if key doesn't exist then it returns an empty string.
func (s *Session) GetFlashString(key string) string {
	return s.GetFlashStringDefault(key, "")
}

// GetFlashStringDefault same as `GetFlash` but returns its string representation,
// if key doesn't exist then it returns the "defaultValue".
func (s *Session) GetFlashStringDefault(key string, defaultValue string) string {
	if value := s.GetFlash(key); value != nil {
		if v, ok := value.(string); ok {
			return v
		}
	}

	return defaultValue
}

var errFindParse = errors.New("Unable to find the %s with key: %s. Found? %#v")

// GetInt same as `Get` but returns its int representation,
// if key doesn't exist then it returns -1 and a non-nil error.
func (s *Session) GetInt(key string) (int, error) {
	v := s.Get(key)

	if vint, ok := v.(int); ok {
		return vint, nil
	}

	if vfloat64, ok := v.(float64); ok {
		return int(vfloat64), nil
	}

	if vint64, ok := v.(int64); ok {
		return int(vint64), nil
	}

	if vstring, sok := v.(string); sok {
		return strconv.Atoi(vstring)
	}

	return -1, errFindParse.Format("int", key, v)
}

// GetIntDefault same as `Get` but returns its int representation,
// if key doesn't exist then it returns the "defaultValue".
func (s *Session) GetIntDefault(key string, defaultValue int) int {
	if v, err := s.GetInt(key); err == nil {
		return v
	}
	return defaultValue
}

// Increment increments the stored int value saved as "key" by +"n".
// If value doesn't exist on that "key" then it creates one with the "n" as its value.
// It returns the new, incremented, value.
func (s *Session) Increment(key string, n int) (newValue int) {
	newValue = s.GetIntDefault(key, 0)
	newValue += n
	s.Set(key, newValue)
	return
}

// Decrement decrements the stored int value saved as "key" by -"n".
// If value doesn't exist on that "key" then it creates one with the "n" as its value.
// It returns the new, decremented, value even if it's less than zero.
func (s *Session) Decrement(key string, n int) (newValue int) {
	newValue = s.GetIntDefault(key, 0)
	newValue -= n
	s.Set(key, newValue)
	return
}

// GetInt64 same as `Get` but returns its int64 representation,
// if key doesn't exist then it returns -1 and a non-nil error.
func (s *Session) GetInt64(key string) (int64, error) {
	v := s.Get(key)

	if vint64, ok := v.(int64); ok {
		return vint64, nil
	}

	if vfloat64, ok := v.(float64); ok {
		return int64(vfloat64), nil
	}

	if vint, ok := v.(int); ok {
		return int64(vint), nil
	}

	if vstring, sok := v.(string); sok {
		return strconv.ParseInt(vstring, 10, 64)
	}

	return -1, errFindParse.Format("int64", key, v)
}

// GetInt64Default same as `Get` but returns its int64 representation,
// if key doesn't exist it returns the "defaultValue".
func (s *Session) GetInt64Default(key string, defaultValue int64) int64 {
	if v, err := s.GetInt64(key); err == nil {
		return v
	}

	return defaultValue
}

// GetFloat32 same as `Get` but returns its float32 representation,
// if key doesn't exist then it returns -1 and a non-nil error.
func (s *Session) GetFloat32(key string) (float32, error) {
	v := s.Get(key)

	if vfloat32, ok := v.(float32); ok {
		return vfloat32, nil
	}

	if vfloat64, ok := v.(float64); ok {
		return float32(vfloat64), nil
	}

	if vint, ok := v.(int); ok {
		return float32(vint), nil
	}

	if vint64, ok := v.(int64); ok {
		return float32(vint64), nil
	}

	if vstring, sok := v.(string); sok {
		vfloat64, err := strconv.ParseFloat(vstring, 32)
		if err != nil {
			return -1, err
		}
		return float32(vfloat64), nil
	}

	return -1, errFindParse.Format("float32", key, v)
}

// GetFloat32Default same as `Get` but returns its float32 representation,
// if key doesn't exist then it returns the "defaultValue".
func (s *Session) GetFloat32Default(key string, defaultValue float32) float32 {
	if v, err := s.GetFloat32(key); err == nil {
		return v
	}

	return defaultValue
}

// GetFloat64 same as `Get` but returns its float64 representation,
// if key doesn't exist then it returns -1 and a non-nil error.
func (s *Session) GetFloat64(key string) (float64, error) {
	v := s.Get(key)

	if vfloat32, ok := v.(float32); ok {
		return float64(vfloat32), nil
	}

	if vfloat64, ok := v.(float64); ok {
		return vfloat64, nil
	}

	if vint, ok := v.(int); ok {
		return float64(vint), nil
	}

	if vint64, ok := v.(int64); ok {
		return float64(vint64), nil
	}

	if vstring, sok := v.(string); sok {
		return strconv.ParseFloat(vstring, 32)
	}

	return -1, errFindParse.Format("float64", key, v)
}

// GetFloat64Default same as `Get` but returns its float64 representation,
// if key doesn't exist then it returns the "defaultValue".
func (s *Session) GetFloat64Default(key string, defaultValue float64) float64 {
	if v, err := s.GetFloat64(key); err == nil {
		return v
	}

	return defaultValue
}

// GetBoolean same as `Get` but returns its boolean representation,
// if key doesn't exist then it returns false and a non-nil error.
func (s *Session) GetBoolean(key string) (bool, error) {
	v := s.Get(key)
	if v == nil {
		return false, errFindParse.Format("bool", key, "nil")
	}

	// here we could check for "true", "false" and 0 for false and 1 for true
	// but this may cause unexpected behavior from the developer if they expecting an error
	// so we just check if bool, if yes then return that bool, otherwise return false and an error.
	if vb, ok := v.(bool); ok {
		return vb, nil
	}
	if vstring, ok := v.(string); ok {
		return strconv.ParseBool(vstring)
	}

	return false, errFindParse.Format("bool", key, v)
}

// GetBooleanDefault same as `Get` but returns its boolean representation,
// if key doesn't exist then it returns the "defaultValue".
func (s *Session) GetBooleanDefault(key string, defaultValue bool) bool {
	/*
		Note that here we can't do more than duplicate the GetBoolean's code, because of the "false".
	*/
	v := s.Get(key)
	if v == nil {
		return defaultValue
	}

	// here we could check for "true", "false" and 0 for false and 1 for true
	// but this may cause unexpected behavior from the developer if they expecting an error
	// so we just check if bool, if yes then return that bool, otherwise return false and an error.
	if vb, ok := v.(bool); ok {
		return vb
	}

	if vstring, ok := v.(string); ok {
		if b, err := strconv.ParseBool(vstring); err == nil {
			return b
		}
	}

	return defaultValue
}

// GetAll returns a copy of all session's values.
func (s *Session) GetAll() map[string]interface{} {
	items := make(map[string]interface{}, s.provider.db.Len(s.sid))
	s.mu.RLock()
	s.provider.db.Visit(s.sid, func(key string, value interface{}) {
		items[key] = value
	})
	s.mu.RUnlock()
	return items
}

// GetFlashes returns all flash messages as map[string](key) and interface{} value
// NOTE: this will cause at remove all current flash messages on the next request of the same user.
func (s *Session) GetFlashes() map[string]interface{} {
	flashes := make(map[string]interface{}, len(s.flashes))
	s.mu.Lock()
	for key, v := range s.flashes {
		flashes[key] = v.value
		v.shouldRemove = true
	}
	s.mu.Unlock()
	return flashes
}

// Visit loops each of the entries and calls the callback function func(key, value).
func (s *Session) Visit(cb func(k string, v interface{})) {
	s.provider.db.Visit(s.sid, cb)
}

func (s *Session) set(key string, value interface{}, immutable bool) {
	s.provider.db.Set(s.sid, s.Lifetime, key, value, immutable)

	s.mu.Lock()
	s.isNew = false
	s.mu.Unlock()
}

// Set fills the session with an entry "value", based on its "key".
func (s *Session) Set(key string, value interface{}) {
	s.set(key, value, false)
}

// SetImmutable fills the session with an entry "value", based on its "key".
// Unlike `Set`, the output value cannot be changed by the caller later on (when .Get)
// An Immutable entry should be only changed with a `SetImmutable`, simple `Set` will not work
// if the entry was immutable, for your own safety.
// Use it consistently, it's far slower than `Set`.
// Read more about muttable and immutable go types: https://stackoverflow.com/a/8021081
func (s *Session) SetImmutable(key string, value interface{}) {
	s.set(key, value, true)
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
// If you want to define more than one flash messages, you will have to use different keys.
func (s *Session) SetFlash(key string, value interface{}) {
	s.mu.Lock()
	s.flashes[key] = &flashMessage{value: value}
	s.mu.Unlock()
}

// Delete removes an entry by its key,
// returns true if actually something was removed.
func (s *Session) Delete(key string) bool {
	removed := s.provider.db.Delete(s.sid, key)
	if removed {
		s.mu.Lock()
		s.isNew = false
		s.mu.Unlock()
	}

	return removed
}

// DeleteFlash removes a flash message by its key.
func (s *Session) DeleteFlash(key string) {
	s.mu.Lock()
	delete(s.flashes, key)
	s.mu.Unlock()
}

// Clear removes all entries.
func (s *Session) Clear() {
	s.mu.Lock()
	s.provider.db.Clear(s.sid)
	s.isNew = false
	s.mu.Unlock()
}

// ClearFlashes removes all flash messages.
func (s *Session) ClearFlashes() {
	s.mu.Lock()
	for key := range s.flashes {
		delete(s.flashes, key)
	}
	s.mu.Unlock()
}
