// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
// This source code file is based on the Gorilla's sessions package.
//
package sessions

import (
	"github.com/kataras/iris"
	"github.com/valyala/fasthttp"
	"sync"
	"time"
)

var (
	mutex sync.RWMutex
	data  = make(map[*fasthttp.Request]map[interface{}]interface{})
	datat = make(map[*fasthttp.Request]int64)
)

// Set stores a value for a given key in a given request.
func Set(r fasthttp.Request, key, val interface{}) {
	mutex.Lock()
	if data[&r] == nil {
		data[&r] = make(map[interface{}]interface{})
		datat[&r] = time.Now().Unix()
	}
	data[&r][key] = val
	mutex.Unlock()
}

// Get returns a value stored for a given key in a given request.
func Get(r fasthttp.Request, key interface{}) interface{} {
	mutex.RLock()
	if ctx := data[&r]; ctx != nil {
		value := ctx[key]
		mutex.RUnlock()
		return value
	}
	mutex.RUnlock()
	return nil
}

// GetOk returns stored value and presence state like multi-value return of map access.
func GetOk(r fasthttp.Request, key interface{}) (interface{}, bool) {
	mutex.RLock()
	if _, ok := data[&r]; ok {
		value, ok := data[&r][key]
		mutex.RUnlock()
		return value, ok
	}
	mutex.RUnlock()
	return nil, false
}

// GetAll returns all stored values for the request as a map. Nil is returned for invalid requests.
func GetAll(r fasthttp.Request) map[interface{}]interface{} {
	mutex.RLock()
	if context, ok := data[&r]; ok {
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
func GetAllOk(r fasthttp.Request) (map[interface{}]interface{}, bool) {
	mutex.RLock()
	context, ok := data[&r]
	result := make(map[interface{}]interface{}, len(context))
	for k, v := range context {
		result[k] = v
	}
	mutex.RUnlock()
	return result, ok
}

// Delete removes a value stored for a given key in a given request.
func Delete(r fasthttp.Request, key interface{}) {
	mutex.Lock()
	if data[&r] != nil {
		delete(data[&r], key)
	}
	mutex.Unlock()
}

// Clear removes all values stored for a given request.
//
// This is usually called by a handler wrapper to clean up request
// variables at the end of a request lifetime. See ClearHandler().
func Clear(r fasthttp.Request) {
	mutex.Lock()
	clear(&r)
	mutex.Unlock()
}

// clear is Clear without the lock.
func clear(r *fasthttp.Request) {
	delete(data, r)
	delete(datat, r)
}

func ClearAll() {
	mutex.Lock()
	data = make(map[*fasthttp.Request]map[interface{}]interface{})
	datat = make(map[*fasthttp.Request]int64)
	mutex.Unlock()
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
		data = make(map[*fasthttp.Request]map[interface{}]interface{})
		datat = make(map[*fasthttp.Request]int64)
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

// add some functionality to the *Session

// Set sets a value to a session with it's key
func (s *Session) Set(key interface{}, val interface{}) {
	if s.Values == nil {
		// OH MY GOD  I WAS FORGOT TO WRITE THE ,0 AND SHIT 9 HOURS OF MY LIFE
		// TRYING TO MAKE MY OWN SESSION MANAGER WTF...
		// LETS USE THE GORILAS BETTER ITS WORKING NOW WITH THE BUFFER ,0 !!!
		s.Values = make(map[interface{}]interface{}, 0)
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
		for k := range s.Values {
			s.Delete(k)
		}
	}
}

// SessionWrapper is the Iris' session wrapper for the session
// it contains the name of the session and the Store
type SessionWrapper struct {
	name  string
	store Store
}

// New creates the session by it's name and returns a new ready-to-use iris.Handler
func New(name string, store Store) SessionWrapper {
	return SessionWrapper{name, store}
}

// Get returns a session by it's context
// same as GetSession
func (s SessionWrapper) Get(ctx *iris.Context) (*Session, error) {
	return s.store.Get(ctx.Request, s.name)
}

// GetSession returns a session by it's context
// same as Get
func (s SessionWrapper) GetSession(ctx *iris.Context) (*Session, error) {
	return s.Get(ctx)
}

// Clear remove all items from this handler's session
func (s SessionWrapper) Clear(ctx *iris.Context) {
	Clear(ctx.Request)
}
