package sessions

import (
	"sync"
	"time"

	"github.com/kataras/iris/core/errors"
)

type (
	// provider contains the sessions and external databases (load and update).
	// It's the session memory manager
	provider struct {
		// we don't use RWMutex because all actions have read and write at the same action function.
		// (or write to a *Session's value which is race if we don't lock)
		// narrow locks are fasters but are useless here.
		mu               sync.Mutex
		sessions         map[string]*Session
		db               Database
		destroyListeners []DestroyListener
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

// ErrNotFound can be returned when calling `UpdateExpiration` on a non-existing or invalid session entry.
// It can be matched directly, i.e: `isErrNotFound := sessions.ErrNotFound.Equal(err)`.
var ErrNotFound = errors.New("not found")

// UpdateExpiration resets the expiration of a session.
// if expires > 0 then it will try to update the expiration and destroy task is delayed.
// if expires <= 0 then it does nothing it returns nil, to destroy a session call the `Destroy` func instead.
//
// If the session is not found, it returns a `NotFound` error,  this can only happen when you restart the server and you used the memory-based storage(default),
// because the call of the provider's `UpdateExpiration` is always called when the client has a valid session cookie.
//
// If a backend database is used then it may return an `ErrNotImplemented` error if the underline database does not support this operation.
func (p *provider) UpdateExpiration(sid string, expires time.Duration) error {
	if expires <= 0 {
		return nil
	}

	p.mu.Lock()
	sess, found := p.sessions[sid]
	p.mu.Unlock()
	if !found {
		return ErrNotFound
	}

	sess.Lifetime.Shift(expires)
	return p.db.OnUpdateExpiration(sid, expires)
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

func (p *provider) registerDestroyListener(ln DestroyListener) {
	if ln == nil {
		return
	}
	p.destroyListeners = append(p.destroyListeners, ln)
}

func (p *provider) fireDestroy(sid string) {
	for _, ln := range p.destroyListeners {
		ln(sid)
	}
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
	sid := sess.sid

	delete(p.sessions, sid)
	p.db.Release(sid)
	p.fireDestroy(sid)
}
