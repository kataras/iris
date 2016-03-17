package sessions

import (
	_ "fmt"
	"github.com/kataras/iris"
	"net/http"
	"sync"
	"time"
)

type sessionWrapper struct {
	sessionName string
	req         *http.Request
	store       Store
}

var (
	mutex sync.RWMutex
	data  = make(map[*http.Request]map[interface{}]interface{})
	datat = make(map[*http.Request]int64)
)

// Set stores a value for a given key in a given request.
func Set(r *http.Request, key, val interface{}) {
	mutex.Lock()
	if data[r] == nil {
		data[r] = make(map[interface{}]interface{})
		datat[r] = time.Now().Unix()
	}
	data[r][key] = val
	mutex.Unlock()
}

// Get returns a value stored for a given key in a given request.
func Get(r *http.Request, key interface{}) interface{} {
	mutex.RLock()
	if ctx := data[r]; ctx != nil {
		value := ctx[key]
		mutex.RUnlock()
		return value
	}
	mutex.RUnlock()
	return nil
}

// GetOk returns stored value and presence state like multi-value return of map access.
func GetOk(r *http.Request, key interface{}) (interface{}, bool) {
	mutex.RLock()
	if _, ok := data[r]; ok {
		value, ok := data[r][key]
		mutex.RUnlock()
		return value, ok
	}
	mutex.RUnlock()
	return nil, false
}

// GetAll returns all stored values for the request as a map. Nil is returned for invalid requests.
func GetAll(r *http.Request) map[interface{}]interface{} {
	mutex.RLock()
	if context, ok := data[r]; ok {
		result := make(map[interface{}]interface{}, len(context))
		for k, v := range context {
			result[k] = v
		}
		mutex.RUnlock()
		return result
	}
	mutex.RUnlock()
	return nil
}

// GetAllOk returns all stored values for the request as a map and a boolean value that indicates if
// the request was registered.
func GetAllOk(r *http.Request) (map[interface{}]interface{}, bool) {
	mutex.RLock()
	context, ok := data[r]
	result := make(map[interface{}]interface{}, len(context))
	for k, v := range context {
		result[k] = v
	}
	mutex.RUnlock()
	return result, ok
}

// Delete removes a value stored for a given key in a given request.
func Delete(r *http.Request, key interface{}) {
	mutex.Lock()
	if data[r] != nil {
		delete(data[r], key)
	}
	mutex.Unlock()
}

// Clear removes all values stored for a given request.
//
// This is usually called by a handler wrapper to clean up request
// variables at the end of a request lifetime. See ClearHandler().
func Clear(r *http.Request) {
	mutex.Lock()
	clear(r)
	mutex.Unlock()
}

// clear is Clear without the lock.
func clear(r *http.Request) {
	delete(data, r)
	delete(datat, r)
}

// Purge removes request data stored for longer than maxAge, in seconds.
// It returns the amount of requests removed.
//
// If maxAge <= 0, all request data is removed.
//
// This is only used for sanity check: in case context cleaning was not
// properly set some request data can be kept forever, consuming an increasing
// amount of memory. In case this is detected, Purge() must be called
// periodically until the problem is fixed.
func Purge(maxAge int) int {
	mutex.Lock()
	count := 0
	if maxAge <= 0 {
		count = len(data)
		data = make(map[*http.Request]map[interface{}]interface{})
		datat = make(map[*http.Request]int64)
	} else {
		min := time.Now().Unix() - int64(maxAge)
		for r := range data {
			if datat[r] < min {
				clear(r)
				count++
			}
		}
	}
	mutex.Unlock()
	return count
}

// GetSession returns a session by it's name
func GetSession(ctx *iris.Context, name string) *Session {
	//it's always not nil
	if s := Get(ctx.Request, name); s != nil {
		return s.(*Session)
	}

	return nil
}

// Set sets a value to a session with it's key
func (s *Session) Set(key interface{}, val interface{}) {
	if s.Values == nil {
		s.Values = make(map[interface{}]interface{})
	}
	s.writeInThisReq = true
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
	name  string
	store Store
}

// New creates the session by it's name and returns a new ready-to-use iris.Handler
func New(name string, store *CookieStore) sessionMiddlewareWrapper {
	m := sessionMiddlewareWrapper{name, store}

	return m
}

// Clear remove all items from this handler's session
func (m sessionMiddlewareWrapper) Clear(req *http.Request) {
	Clear(req)
}

// Serve is the Middleware handler
func (m sessionMiddlewareWrapper) Serve(ctx *iris.Context) {
	//care here maybe error if you use the middleware without a session
	var s *Session = NewSession(m.store, m.name)
	st := GetSession(ctx, m.name)
	if st == nil {
		Set(ctx.Request, m.name, s)
	} else {
		//s = st
	}
	//s := NewSession(m.store, m.name)
	//Set(ctx.Request, m.name, s)

	/*if st := Get(ctx.Request, m.name); st != nil {
		s = st.(*Session)
	} else {
		s = NewSession(m.store, m.name)
		Set(ctx.Request, m.name, s)
	}*/

	// Use before hook to save out the session
	/*ctx.PreWrite(func(iris.ResponseWriter) {
		if s.writeInThisReq {
			fmt.Println("\nsave")
			m.store.Save(ctx.Request, ctx.ResponseWriter, s)
			fmt.Printf("\nAfter save Store: %T Point to: %v %s", m.store, m.store, m.store)
		}
	})*/
	s.writeInThisReq = false

	ctx.Next()
}
