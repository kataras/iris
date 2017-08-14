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

// newSession returns a new session from sessionid
func (p *provider) newSession(sid string, expires time.Duration) *Session {
	onExpire := func() {
		p.Destroy(sid)
	}

	values, lifetime := p.loadSessionFromDB(sid)
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

	// I ended up without the need of a duration field on lifetime,
	// but these are some of my previous comments, they will be required for any future change.
	//
	// OK I think go has a bug when gob on embedded time.Time
	// even if we  `gob.Register(LifeTime)`
	// the OriginalDuration is not saved to the gob file and it cannot be retrieved, it's always 0.
	// But if we do not embed the `time.Time` inside the `LifeTime` then
	// it's working.
	// i.e type LifeTime struct { time.Time; OriginalDuration time.Duration} -> this doesn't
	//     type LifeTime struct {Time time.Time; OriginalDuration time.Duration} -> this works
	// So we have two options:
	// 1. don't embed the time.Time -> we will have to use lifetime.Time to get its functions, which doesn't seems right to me
	// 2. embed the time.Time and compare their times with `lifetime.After(time.Now().Add(expires))`, it seems right but it
	// 	  should be slower.
	//
	// I'll use the 1. and put some common time.Time functions, like After, IsZero on the `LifeTime` type too.
	//
	// if db exists but its lifetime is bigger than the expires (very raire,
	// the source code should be compatible with the databases,
	// should we print a warning to the user? it is his/her fault
	// use the database's lifetime or the configurated?
	// if values.Len() > 0 && lifetime.OriginalDuration != expires {
	// 	golog.Warnf(`session database: stored expire time(dur=%d) is differnet than the configuration(dur=%d)
	// 		application will use the configurated one`, lifetime.OriginalDuration, expires)
	// 	lifetime.Reset(expires)
	// }

	sess := &Session{
		sid:      sid,
		provider: p,
		values:   values,
		flashes:  make(map[string]*flashMessage),
		lifetime: lifetime,
	}

	return sess
}

func (p *provider) loadSessionFromDB(sid string) (memstore.Store, LifeTime) {
	var store memstore.Store
	var lifetime LifeTime

	firstValidIdx := 1
	for i, n := 0, len(p.databases); i < n; i++ {
		storeDB := p.databases[i].Load(sid)
		if storeDB.Lifetime.HasExpired() { // if expired then skip this db
			firstValidIdx++
			continue
		}

		if lifetime.IsZero() {
			// update the lifetime to the most valid
			lifetime = storeDB.Lifetime
		}

		if n == firstValidIdx {
			// if one database then set the store as it is
			store = storeDB.Values
		} else {
			// else append this database's key-value pairs
			// to the store
			storeDB.Values.Visit(func(key string, value interface{}) {
				store.Set(key, value)
			})
		}
	}

	// Note: if one database and it's being expired then the lifetime will be zero(unlimited)
	// this by itself is wrong but on the `newSession` we make check of this case too and update the lifetime
	// if the configuration has expiration registered.

	/// TODO: bug on destroy doesn't being remove the file
	// we will have to see it, it's not db's problem it's here on provider destroy or lifetime onExpire.
	return store, lifetime
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

	sess.lifetime.Shift(expires)
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
	syncDatabases(p.databases, acquireSyncPayload(sess, ActionDestroy))
}
