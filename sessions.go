package iris

import (
	"container/list"
	"encoding/base64"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/config"
	"github.com/kataras/iris/utils"
	"github.com/valyala/fasthttp"
)

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------SessionDatabase implementation---------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// SessionDatabase is the interface which all session databases should implement
// By design it doesn't support any type of cookie store like other frameworks, I want to protect you, believe me, no context access (although we could)
// The scope of the database is to store somewhere the sessions in order to keep them after restarting the server, nothing more.
// the values are stored by the underline session, the check for new sessions, or 'this session value should added' are made automatically by Iris, you are able just to set the values to your backend database with Load function.
// session database doesn't have any write or read access to the session, the loading of the initial data is done by the Load(string) map[string]interfface{} function
// synchronization are made automatically, you can register more than one session database but the first non-empty Load return data will be used as the session values.
type SessionDatabase interface {
	Load(string) map[string]interface{}
	Update(string, map[string]interface{})
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------Session implementation-----------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// session is an 'object' which wraps the session provider with its session databases, only frontend user has access to this session object.
// this is really used on context and everywhere inside Iris
type session struct {
	sid              string
	values           map[string]interface{} // here is the real values
	mu               sync.Mutex
	lastAccessedTime time.Time
	createdAt        time.Time
	provider         *sessionProvider
}

// ID returns the session's id
func (s *session) ID() string {
	return s.sid
}

// Get returns the value of an entry by its key
func (s *session) Get(key string) interface{} {
	s.provider.update(s.sid)
	if value, found := s.values[key]; found {
		return value
	}
	return nil
}

// GetString same as Get but returns as string, if nil then returns an empty string
func (s *session) GetString(key string) string {
	if value := s.Get(key); value != nil {
		if v, ok := value.(string); ok {
			return v
		}

	}

	return ""
}

// GetInt same as Get but returns as int, if nil then returns -1
func (s *session) GetInt(key string) int {
	if value := s.Get(key); value != nil {
		if v, ok := value.(int); ok {
			return v
		}
	}

	return -1
}

// GetAll returns all session's values
func (s *session) GetAll() map[string]interface{} {
	return s.values
}

// VisitAll loop each one entry and calls the callback function func(key,value)
func (s *session) VisitAll(cb func(k string, v interface{})) {
	for key := range s.values {
		cb(key, s.values[key])
	}
}

// Set fills the session with an entry, it receives a key and a value
// returns an error, which is always nil
func (s *session) Set(key string, value interface{}) {
	s.mu.Lock()
	s.values[key] = value
	s.mu.Unlock()
	s.provider.update(s.sid)
}

// Delete removes an entry by its key
// returns an error, which is always nil
func (s *session) Delete(key string) {
	s.mu.Lock()
	delete(s.values, key)
	s.mu.Unlock()
	s.provider.update(s.sid)
}

// Clear removes all entries
func (s *session) Clear() {
	s.mu.Lock()
	for key := range s.values {
		delete(s.values, key)
	}
	s.mu.Unlock()
	s.provider.update(s.sid)
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------sessionProvider implementation---------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

type (
	// sessionProvider contains the temp sessions memory and the databases
	sessionProvider struct {
		mu        sync.Mutex
		sessions  map[string]*list.Element // underline TEMPORARY memory store used to give advantage on sessions used more times than others
		list      *list.List               // for GC
		databases []SessionDatabase
		expires   time.Duration
	}
)

func (p *sessionProvider) registerDatabase(db SessionDatabase) {
	p.mu.Lock() // for any case
	p.databases = append(p.databases, db)
	p.mu.Unlock()
}

func (p *sessionProvider) newSession(sid string) *session {

	sess := &session{
		sid:              sid,
		provider:         p,
		lastAccessedTime: time.Now(),
		values:           p.loadSessionValues(sid),
	}
	if p.expires > 0 { // if not unlimited life duration
		time.AfterFunc(p.expires, func() {
			// the destroy makes the check if this session is exists then or not,
			// this is used to destroy the session from the server-side also
			// it's good to have here for security reasons, I didn't add it on the gc function to separate its action
			p.destroy(sid)

		})
	}

	return sess

}

func (p *sessionProvider) loadSessionValues(sid string) map[string]interface{} {

	for i, n := 0, len(p.databases); i < n; i++ {
		if dbValues := p.databases[i].Load(sid); dbValues != nil && len(dbValues) > 0 {
			return dbValues // return the first non-empty from the registered stores.
		}
	}
	values := make(map[string]interface{})
	return values
}

func (p *sessionProvider) updateDatabases(sid string, newValues map[string]interface{}) {
	for i, n := 0, len(p.databases); i < n; i++ {
		p.databases[i].Update(sid, newValues)
	}
}

// Init creates the session  and returns it
func (p *sessionProvider) init(sid string) *session {
	newSession := p.newSession(sid)
	elem := p.list.PushBack(newSession)
	p.mu.Lock()
	p.sessions[sid] = elem
	p.mu.Unlock()
	return newSession
}

// Read returns the store which sid parameter is belongs
func (p *sessionProvider) read(sid string) *session {
	p.mu.Lock()
	if elem, found := p.sessions[sid]; found {
		p.mu.Unlock() // yes defer is slow
		elem.Value.(*session).lastAccessedTime = time.Now()
		return elem.Value.(*session)
	}
	p.mu.Unlock()
	// if not found create new
	sess := p.init(sid)
	return sess
}

// Destroy destroys the session, removes all sessions values, the session itself and updates the registered session databases, this called from sessionManager which removes the client's cookie also.
func (p *sessionProvider) destroy(sid string) {
	p.mu.Lock()
	if elem, found := p.sessions[sid]; found {
		sess := elem.Value.(*session)
		sess.values = nil
		p.updateDatabases(sid, nil)
		delete(p.sessions, sid)
		p.list.Remove(elem)
	}
	p.mu.Unlock()
}

// Update updates the lastAccessedTime, and moves the memory place element to the front
// always returns a nil error, for now
func (p *sessionProvider) update(sid string) {
	p.mu.Lock()
	if elem, found := p.sessions[sid]; found {
		sess := elem.Value.(*session)
		sess.lastAccessedTime = time.Now()
		p.list.MoveToFront(elem)
		p.updateDatabases(sid, sess.values)
	}
	p.mu.Unlock()
}

// GC clears the memory
func (p *sessionProvider) gc(duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for {
		elem := p.list.Back()
		if elem == nil {
			break
		}

		// if the time has passed. session was expired, then delete the session and its memory place
		// we are not destroy the session completely for the case this is re-used after
		sess := elem.Value.(*session)
		if time.Now().After(sess.lastAccessedTime.Add(duration)) {
			p.list.Remove(elem)
		} else {
			break
		}
	}
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------sessionsManager implementation---------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

type (
	// sessionsManager implements the ISessionsManager interface
	// contains the cookie's name, the provider and a duration for GC and cookie life expire
	sessionsManager struct {
		config   *config.Sessions
		provider *sessionProvider
	}
)

// newSessionsManager creates & returns a new SessionsManager and start its GC
func newSessionsManager(c *config.Sessions) *sessionsManager {
	if c.DecodeCookie {
		c.Cookie = base64.URLEncoding.EncodeToString([]byte(c.Cookie)) // change the cookie's name/key to a more safe(?)
		// get the real value for your tests by:
		//sessIdKey := url.QueryEscape(base64.URLEncoding.EncodeToString([]byte(iris.Config.Sessions.Cookie)))
	}
	manager := &sessionsManager{config: c, provider: &sessionProvider{list: list.New(), sessions: make(map[string]*list.Element, 0), databases: make([]SessionDatabase, 0), expires: c.Expires}}
	//run the GC here
	go manager.gc()
	return manager
}

func (m *sessionsManager) registerDatabase(db SessionDatabase) {
	m.provider.expires = m.config.Expires // updae the expires confiuration field for any case
	m.provider.registerDatabase(db)
}

func (m *sessionsManager) generateSessionID() string {
	return base64.URLEncoding.EncodeToString(utils.Random(32))
}

// Start starts the session
func (m *sessionsManager) start(ctx *Context) *session {
	var session *session

	cookieValue := ctx.GetCookie(m.config.Cookie)

	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := m.generateSessionID()
		session = m.provider.init(sid)
		cookie := fasthttp.AcquireCookie()
		// The RFC makes no mention of encoding url value, so here I think to encode both sessionid key and the value using the safe(to put and to use as cookie) url-encoding
		cookie.SetKey(m.config.Cookie)
		cookie.SetValue(sid)
		cookie.SetPath("/")
		if !m.config.DisableSubdomainPersistence {
			requestDomain := ctx.HostString()
			if portIdx := strings.IndexByte(requestDomain, ':'); portIdx > 0 {
				requestDomain = requestDomain[0:portIdx]
			}

			if requestDomain == "0.0.0.0" || requestDomain == "127.0.0.1" {
				// for these type of hosts, we can't allow subdomains persistance,
				// the web browser doesn't understand the mysubdomain.0.0.0.0 and mysubdomain.127.0.0.1 as scorrectly ubdomains because of the many dots
				// so don't set a domain here

			} else if strings.Count(requestDomain, ".") > 0 { // there is a problem with .localhost setted as the domain, so we check that first

				// RFC2109, we allow level 1 subdomains, but no further
				// if we have localhost.com , we want the localhost.com.
				// so if we have something like: mysubdomain.localhost.com we want the localhost here
				// if we have mysubsubdomain.mysubdomain.localhost.com we want the .mysubdomain.localhost.com here
				// slow things here, especially the 'replace' but this is a good and understable( I hope) way to get the be able to set cookies from subdomains & domain with 1-level limit
				if dotIdx := strings.LastIndexByte(requestDomain, '.'); dotIdx > 0 {
					// is mysubdomain.localhost.com || mysubsubdomain.mysubdomain.localhost.com
					s := requestDomain[0:dotIdx] // set mysubdomain.localhost || mysubsubdomain.mysubdomain.localhost
					if secondDotIdx := strings.LastIndexByte(s, '.'); secondDotIdx > 0 {
						//is mysubdomain.localhost ||  mysubsubdomain.mysubdomain.localhost
						s = s[secondDotIdx+1:] // set to localhost || mysubdomain.localhost
					}
					// replace the s with the requestDomain before the domain's siffux
					subdomainSuff := strings.LastIndexByte(requestDomain, '.')
					if subdomainSuff > len(s) { // if it is actual exists as subdomain suffix
						requestDomain = strings.Replace(requestDomain, requestDomain[0:subdomainSuff], s, 1) // set to localhost.com || mysubdomain.localhost.com
					}
				}
				// finally set the .localhost.com (for(1-level) || .mysubdomain.localhost.com (for 2-level subdomain allow)
				cookie.SetDomain("." + requestDomain) // . to allow persistance
			}

		}
		cookie.SetHTTPOnly(true)
		if m.config.Expires == 0 {
			// unlimited life
			cookie.SetExpire(config.CookieExpireNever)
		} else {
			cookie.SetExpire(time.Now().Add(m.config.Expires))
		}

		ctx.SetCookie(cookie)
		fasthttp.ReleaseCookie(cookie)
	} else {
		session = m.provider.read(cookieValue)
	}
	return session
}

// Destroy kills the session and remove the associated cookie
func (m *sessionsManager) destroy(ctx *Context) {
	cookieValue := ctx.GetCookie(m.config.Cookie)
	if cookieValue == "" { // nothing to destroy
		return
	}
	ctx.RemoveCookie(m.config.Cookie)
	m.provider.destroy(cookieValue)
}

// GC tick-tock for the store cleanup
// it's a blocking function, so run it with go routine, it's totally safe
func (m *sessionsManager) gc() {
	m.provider.gc(m.config.GcDuration)
	// set a timer for the next GC
	time.AfterFunc(m.config.GcDuration, func() {
		m.gc()
	})
}
