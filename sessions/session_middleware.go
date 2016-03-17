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

func Get(req *http.Request, key interface{}) interface{} {
	if bag == nil {
		return nil
	}
	mu.Lock()
	v := bag[key]
	mu.Unlock()
	return v
}

func Set(req *http.Request, key interface{}, val interface{}) {
	mu.Lock()
	if bag == nil {
		bag = make(map[interface{}]interface{})
	}
	bag[key] = val
	mu.Unlock()
}

func DeleteDanger(key interface{}) {
	delete(bag, key)
}

func Clear() {
	mu.Lock()
	if bag != nil && len(bag) > 0 {
		for k, _ := range bag {
			DeleteDanger(k)
		}
	}
	mu.Unlock()
}

func GetSession(name string) *Session {
	if s := Get(nil, name); s != nil {

		return s.(*Session)
	}
	return nil
}

func (s *Session) Set(key interface{}, val interface{}) {
	if s.Values == nil {
		s.Values = make(map[interface{}]interface{})
	}

	s.Values[key] = val
}

func (s *Session) Get(key interface{}) interface{} {
	return s.Values[key]
}

func (s *Session) GetString(key interface{}) string {
	if s == nil || s.Values[key] == nil {
		return ""
	}
	return s.Values[key].(string)
}

func (s *Session) Delete(key interface{}) {
	delete(s.Values, key)
}

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

func New(name string, store *CookieStore) sessionMiddlewareWrapper {
	m := sessionMiddlewareWrapper{name, NewSession(store, name)}
	Set(nil, name, m.session)
	return m
}

func (m sessionMiddlewareWrapper) Clear() {
	Clear()
}

// Sessions is the Middleware
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
