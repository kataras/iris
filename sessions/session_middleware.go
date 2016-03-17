package sessions

import (
	"github.com/kataras/iris"
	"net/http"
	"sync"
)

type sessionWrapper struct {
	sessionName string
	req         *http.Request
	store       Store
}

var bag map[interface{}]interface{}
var mu sync.Mutex

// Get gets a value from a key and a request which can be nil
func Get(req *http.Request, key interface{}) interface{} {
	if bag == nil {
		return nil
	}
	mu.Lock()
	v := bag[key]
	mu.Unlock()
	return v
}

// Set assign a value to a key and, request can be nil
func Set(req *http.Request, key interface{}, val interface{}) {
	mu.Lock()
	if bag == nil {
		bag = make(map[interface{}]interface{})
	}
	bag[key] = val
	mu.Unlock()
}

// DeleteDanger removes without other checking a pair by its key
func DeleteDanger(key interface{}) {
	delete(bag, key)
}

// Clear removes all pairs from the bag
func Clear() {
	mu.Lock()
	if bag != nil && len(bag) > 0 {
		for k, _ := range bag {
			DeleteDanger(k)
		}
	}
	mu.Unlock()
}

// GetSession returns a session by it's name
func GetSession(name string) *Session {
	if s := Get(nil, name); s != nil {

		return s.(*Session)
	}
	return nil
}

// Set sets a value to a session with it's key
func (s *Session) Set(key interface{}, val interface{}) {
	if s.Values == nil {
		s.Values = make(map[interface{}]interface{})
	}

	s.Values[key] = val
}

// Get returns a value from a key
func (s *Session) Get(key interface{}) interface{} {
	return s.Values[key]
}

// GetString same as Get but returns string
// if nothing found returns empty string ""
func (s *Session) GetString(key interface{}) string {
	if s == nil || s.Values[key] == nil {
		return ""
	}
	return s.Values[key].(string)
}

// GetInt same as Get but returns int
// if nothing found returns -1
func (s *Session) GetInt(key interface{}) int {
	if s == nil || s.Values[key] == nil {
		return -1
	}
	return s.Values[key].(int)
}

// Delete removes without other checking a pair by its key
func (s *Session) Delete(key interface{}) {
	delete(s.Values, key)
}

// Clear remove all pairs from the session
func (s *Session) Clear() {
	if s.Values != nil && len(s.Values) > 0 {
		for k, _ := range s.Values {
			s.Delete(k)
		}
	}
}

type sessionMiddlewareWrapper struct {
	name    string
	session *Session
}

// New creates the session by it's name and returns a new ready-to-use iris.Handler
func New(name string, store *CookieStore) sessionMiddlewareWrapper {
	m := sessionMiddlewareWrapper{name, NewSession(store, name)}
	Set(nil, name, m.session)
	return m
}

// Clear remove all items from this handler's session
func (m sessionMiddlewareWrapper) Clear() {
	Clear()
}

// Serve is the Middleware handler
func (m sessionMiddlewareWrapper) Serve(ctx *iris.Context) {
	//care here maybe error if you use the middleware without a session

	// Use before hook to save out the session
	ctx.PreWrite(func(iris.ResponseWriter) {
		if m.session.Values != nil && len(m.session.Values) > 0 {
			m.session.store.Save(ctx.Request, ctx.ResponseWriter, m.session)
		}
	})

	ctx.Next()
}
