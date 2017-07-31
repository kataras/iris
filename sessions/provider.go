package sessions

import (
	"sync"
	"time"

	"github.com/kataras/iris/core/memstore"
)

type (
	// provider contains the sessions and external databases (load and update).
	// It's the session memory manager
	provider struct {
		// we don't use RWMutex because all actions have read and write at the same action function.
		// (or write to a *Session's value which is race if we don't lock)
		// narrow locks are fasters but are useless here.
		mu        sync.Mutex
		sessions  map[string]*Session
		databases []Database
	}
)

// newProvider returns a new sessions provider
func newProvider() *provider {
	return &provider{
		sessions:  make(map[string]*Session, 0),
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

// startAutoDestroy start a task which destoy the session when expire date is reached,
// but only if `expires` parameter is positive. It updates the expire date of the session from `expires` parameter.
func (p *provider) startAutoDestroy(s *Session, expires time.Duration) bool {
	res := expires > 0
	if res { // if not unlimited life duration and no -1 (cookie remove action is based on browser's session)
		expireDate := time.Now().Add(expires)

		s.expireAt = &expireDate
		s.timer = time.AfterFunc(expires, func() {
			// the destroy makes the check if this session is exists then or not,
			// this is used to destroy the session from the server-side also
			// it's good to have here for security reasons, I didn't add it on the gc function to separate its action
			p.Destroy(s.sid)
		})
	}

	return res
}

// newSession returns a new session from sessionid
func (p *provider) newSession(sid string, expires time.Duration) *Session {
	values, expireAt := p.loadSessionValuesFromDB(sid)

	sess := &Session{
		sid:      sid,
		provider: p,
		values:   values,
		flashes:  make(map[string]*flashMessage),
		expireAt: expireAt,
	}

	if (len(values) > 0) && (sess.expireAt != nil) {
		// Restore expiration state
		// However, if session save in database has no expiration date,
		// therefore the expiration will be reinitialised with session configuration
		expires = sess.expireAt.Sub(time.Now())
	}

	p.startAutoDestroy(sess, expires)

	return sess
}

// can return nil memstore
func (p *provider) loadSessionValuesFromDB(sid string) (memstore.Store, *time.Time) {
	var store memstore.Store
	var expireDate *time.Time

	for i, n := 0, len(p.databases); i < n; i++ {
		dbValues, currentExpireDate := p.databases[i].Load(sid)
		if dbValues != nil && len(dbValues) > 0 {
			for k, v := range dbValues {
				store.Set(k, v)
			}
		}

		if (currentExpireDate != nil) && ((expireDate == nil) || expireDate.After(*currentExpireDate)) {
			expireDate = currentExpireDate
		}
	}

	// Check if session has already expired
	if (expireDate != nil) && expireDate.Before(time.Now()) {
		return nil, nil
	}

	return store, expireDate
}

func (p *provider) updateDatabases(sess *Session, store memstore.Store) {
	if l := store.Len(); l > 0 {
		mapValues := make(map[string]interface{}, l)

		store.Visit(func(k string, v interface{}) {
			mapValues[k] = v
		})

		for i, n := 0, len(p.databases); i < n; i++ {
			p.databases[i].Update(sess.sid, mapValues, sess.expireAt)
		}
	}
}

// Init creates the session  and returns it
func (p *provider) Init(sid string, expires time.Duration) *Session {
	newSession := p.newSession(sid, expires)
	p.mu.Lock()
	p.sessions[sid] = newSession
	p.mu.Unlock()
	return newSession
}

// UpdateExpiraton update expire date of a session, plus it updates destroy task
func (p *provider) UpdateExpiraton(sid string, expires time.Duration) (done bool) {
	if expires <= 0 {
		return false
	}

	p.mu.Lock()
	sess, found := p.sessions[sid]
	p.mu.Unlock()

	if !found {
		return false
	}

	if sess.timer == nil {
		return p.startAutoDestroy(sess, expires)
	} else {
		if expires <= 0 {
			sess.timer.Stop()
			sess.timer = nil
			sess.expireAt = nil
		} else {
			expireDate := time.Now().Add(expires)

			sess.expireAt = &expireDate
			sess.timer.Reset(expires)
		}
	}

	return true
}

// Read returns the store which sid parameter belongs
func (p *provider) Read(sid string, expires time.Duration) *Session {
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
		if sess.timer != nil {
			sess.timer.Stop()
		}

		delete(p.sessions, sid)
		p.updateDatabases(sess, nil)
	}
	p.mu.Unlock()

}

// DestroyAll removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
func (p *provider) DestroyAll() {
	p.mu.Lock()
	for _, sess := range p.sessions {
		if sess.timer != nil {
			sess.timer.Stop()
		}

		delete(p.sessions, sess.ID())
		p.updateDatabases(sess, nil)
	}
	p.mu.Unlock()

}
