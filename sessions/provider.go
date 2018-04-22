package sessions

import (
	"sync"
	"time"
)

type (
	// provider contains the sessions and external databases (load and update).
	// It's the session memory manager
	provider struct {
		// we don't use RWMutex because all actions have read and write at the same action function.
		// (or write to a *Session's value which is race if we don't lock)
		// narrow locks are fasters but are useless here.
		mu       sync.Mutex
		sessions map[string]*Session
		db       Database
	}
)

// newProvider returns a new sessions provider
func newProvider() *provider {
	return &provider{
		sessions: make(map[string]*Session, 0),
		db:       newMemDB(),
	}
}

// RegisterDatabase sets a session database.
func (p *provider) RegisterDatabase(db Database) {
	p.mu.Lock() // for any case
	p.db = db
	p.mu.Unlock()
}

// newSession returns a new session from sessionid
func (p *provider) newSession(sid string, expires time.Duration) *Session {
	onExpire := func() {
		p.Destroy(sid)
	}

	lifetime := p.db.Acquire(sid, expires)

	// simple and straight:
	if !lifetime.IsZero() {
		// if stored time is not zero
		// start a timer based on the stored time, if not expired.
		lifetime.Revive(onExpire)
	} else {
		// Remember:  if db not exist or it has been expired
		// then the stored time will be zero(see loadSessionFromDB) and the values will be empty.
		//
		// Even if the database has an unlimited session (possible by a previous app run)
		// priority to the "expires" is given,
		// again if <=0 then it does nothing.
		lifetime.Begin(expires, onExpire)
	}

	sess := &Session{
		sid:      sid,
		provider: p,
		flashes:  make(map[string]*flashMessage),
		Lifetime: lifetime,
	}

	return sess
}

// Init creates the session  and returns it
func (p *provider) Init(sid string, expires time.Duration) *Session {
	newSession := p.newSession(sid, expires)
	p.mu.Lock()
	p.sessions[sid] = newSession
	p.mu.Unlock()
	return newSession
}

// UpdateExpiration update expire date of a session.
// if expires > 0 then it updates the destroy task.
// if expires <=0 then it does nothing, to destroy a session call the `Destroy` func instead.
func (p *provider) UpdateExpiration(sid string, expires time.Duration) bool {
	if expires <= 0 {
		return false
	}

	p.mu.Lock()
	sess, found := p.sessions[sid]
	p.mu.Unlock()
	if !found {
		return false
	}

	sess.Lifetime.Shift(expires)
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
		p.deleteSession(sess)
	}
	p.mu.Unlock()
}

// DestroyAll removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
func (p *provider) DestroyAll() {
	p.mu.Lock()
	for _, sess := range p.sessions {
		p.deleteSession(sess)
	}
	p.mu.Unlock()
}

func (p *provider) deleteSession(sess *Session) {
	delete(p.sessions, sess.sid)
	p.db.Release(sess.sid)
}
