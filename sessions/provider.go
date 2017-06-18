// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sessions

import (
	"sync"
	"time"

	"github.com/cdren/iris/core/memstore"
)

type (
	// provider contains the sessions and external databases (load and update).
	// It's the session memory manager
	provider struct {
		// we don't use RWMutex because all actions have read and write at the same action function.
		// (or write to a *session's value which is race if we don't lock)
		// narrow locks are fasters but are useless here.
		mu        sync.Mutex
		sessions  map[string]*session
		databases []Database
	}
)

// newProvider returns a new sessions provider
func newProvider() *provider {
	return &provider{
		sessions:  make(map[string]*session, 0),
		databases: make([]Database, 0),
	}
}

// RegisterDatabase adds a session database
// a session db doesn't have write access
func (p *provider) RegisterDatabase(db Database) {
	p.mu.Lock() // for any case
	p.databases = append(p.databases, db)
	p.mu.Unlock()
}

// newSession returns a new session from sessionid
func (p *provider) newSession(sid string, expires time.Duration) *session {

	sess := &session{
		sid:      sid,
		provider: p,
		values:   p.loadSessionValuesFromDB(sid),
		flashes:  make(map[string]*flashMessage),
	}

	if expires > 0 { // if not unlimited life duration and no -1 (cookie remove action is based on browser's session)
		time.AfterFunc(expires, func() {
			// the destroy makes the check if this session is exists then or not,
			// this is used to destroy the session from the server-side also
			// it's good to have here for security reasons, I didn't add it on the gc function to separate its action
			p.Destroy(sid)
		})
	}

	return sess
}

// can return nil
func (p *provider) loadSessionValuesFromDB(sid string) memstore.Store {
	var store memstore.Store

	for i, n := 0, len(p.databases); i < n; i++ {
		if dbValues := p.databases[i].Load(sid); dbValues != nil && len(dbValues) > 0 {
			for k, v := range dbValues {
				store.Set(k, v)
			}
		}
	}
	return store
}

func (p *provider) updateDatabases(sid string, store memstore.Store) {

	if l := store.Len(); l > 0 {
		mapValues := make(map[string]interface{}, l)

		store.Visit(func(k string, v interface{}) {
			mapValues[k] = v
		})

		for i, n := 0, len(p.databases); i < n; i++ {
			p.databases[i].Update(sid, mapValues)
		}
	}
}

// Init creates the session  and returns it
func (p *provider) Init(sid string, expires time.Duration) Session {
	newSession := p.newSession(sid, expires)
	p.mu.Lock()
	p.sessions[sid] = newSession
	p.mu.Unlock()
	return newSession
}

// Read returns the store which sid parameter belongs
func (p *provider) Read(sid string, expires time.Duration) Session {
	p.mu.Lock()
	if sess, found := p.sessions[sid]; found {
		sess.runFlashGC() // run the flash messages GC, new request here of existing session
		p.mu.Unlock()
		return sess
	}
	p.mu.Unlock()
	return p.Init(sid, expires) // if not found create new

}

// Destroy destroys the session, removes all sessions and flash values,
// the session itself and updates the registered session databases,
// this called from sessionManager which removes the client's cookie also.
func (p *provider) Destroy(sid string) {
	p.mu.Lock()
	if sess, found := p.sessions[sid]; found {
		sess.values = nil
		sess.flashes = nil
		delete(p.sessions, sid)
		p.updateDatabases(sid, nil)
	}
	p.mu.Unlock()

}

// DestroyAll removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
func (p *provider) DestroyAll() {
	p.mu.Lock()
	for _, sess := range p.sessions {
		delete(p.sessions, sess.ID())
		p.updateDatabases(sess.ID(), nil)
	}
	p.mu.Unlock()

}
