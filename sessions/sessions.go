// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sessions

import (
	"net/http"
	"strings"
	"time"
)

type (

	// Sessions must be implemented within a session manager.
	//
	// A Sessions should be responsible to Start a sesion based
	// on raw http.ResponseWriter and http.Request, which should return
	// a compatible Session interface, type. If the external session manager
	// doesn't qualifies, then the user should code the rest of the functions with empty implementation.
	//
	// Sessions should be responsible to Destroy a session based
	// on the http.ResponseWriter and http.Request, this function should works individually.
	Sessions interface {
		// Start should start the session for the particular net/http request.
		Start(http.ResponseWriter, *http.Request) Session

		// Destroy should kills the net/http session and remove the associated cookie.
		Destroy(http.ResponseWriter, *http.Request)
	} // Sessions is being implemented by Manager

	// Session should expose the Sessions's end-user API.
	// This will be returned at the sess := context.Session().
	Session interface {
		ID() string
		Get(string) interface{}
		HasFlash() bool
		GetFlash(string) interface{}
		GetString(key string) string
		GetFlashString(string) string
		GetInt(key string) (int, error)
		GetInt64(key string) (int64, error)
		GetFloat32(key string) (float32, error)
		GetFloat64(key string) (float64, error)
		GetBoolean(key string) (bool, error)
		GetAll() map[string]interface{}
		GetFlashes() map[string]interface{}
		VisitAll(cb func(k string, v interface{}))
		Set(string, interface{})
		SetFlash(string, interface{})
		Delete(string)
		DeleteFlash(string)
		Clear()
		ClearFlashes()
	} // Session is being implemented inside session.go

	// Manager implements the Sessions interface which Iris uses to start and destroy a session from its Context.
	Manager struct {
		config   Config
		provider *provider
	}
)

// New returns a new fast, feature-rich sessions manager
// it can be adapted to an Iris station
func New(cfg Config) *Manager {
	return &Manager{
		config:   cfg.Validate(),
		provider: newProvider(),
	}
}

var _ Sessions = &Manager{}

// UseDatabase adds a session database to the manager's provider,
// a session db doesn't have write access
func (s *Manager) UseDatabase(db Database) {
	s.provider.RegisterDatabase(db)
}

// Start starts the session for the particular net/http request
func (s *Manager) Start(res http.ResponseWriter, req *http.Request) Session {
	var sess Session

	cookieValue := GetCookie(s.config.Cookie, req)

	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := s.config.SessionIDGenerator()

		sess = s.provider.Init(sid, s.config.Expires)
		cookie := &http.Cookie{}

		// The RFC makes no mention of encoding url value, so here I think to encode both sessionid key and the value using the safe(to put and to use as cookie) url-encoding
		cookie.Name = s.config.Cookie

		cookie.Value = sid
		cookie.Path = "/"
		if !s.config.DisableSubdomainPersistence {

			requestDomain := req.URL.Host
			if portIdx := strings.IndexByte(requestDomain, ':'); portIdx > 0 {
				requestDomain = requestDomain[0:portIdx]
			}
			if IsValidCookieDomain(requestDomain) {

				// RFC2109, we allow level 1 subdomains, but no further
				// if we have localhost.com , we want the localhost.cos.
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
				cookie.Domain = "." + requestDomain // . to allow persistence
			}

		}
		cookie.HttpOnly = true
		// MaxAge=0 means no 'Max-Age' attribute specified.
		// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
		// MaxAge>0 means Max-Age attribute present and given in seconds
		if s.config.Expires >= 0 {
			if s.config.Expires == 0 { // unlimited life
				cookie.Expires = CookieExpireUnlimited
			} else { // > 0
				cookie.Expires = time.Now().Add(s.config.Expires)
			}
			cookie.MaxAge = int(cookie.Expires.Sub(time.Now()).Seconds())
		}

		// set the cookie to secure if this is a tls wrapped request
		// and the configuration allows it.
		if req.TLS != nil && s.config.CookieSecureTLS {
			cookie.Secure = true
		}

		// encode the session id cookie client value right before send it.
		cookie.Value = s.encodeCookieValue(cookie.Value)

		AddCookie(cookie, res)
	} else {

		cookieValue = s.decodeCookieValue(cookieValue)

		sess = s.provider.Read(cookieValue, s.config.Expires)
	}
	return sess
}

// Destroy remove the session data and remove the associated cookie.
func (s *Manager) Destroy(res http.ResponseWriter, req *http.Request) {
	cookieValue := GetCookie(s.config.Cookie, req)
	// decode the client's cookie value in order to find the server's session id
	// to destroy the session data.
	cookieValue = s.decodeCookieValue(cookieValue)
	if cookieValue == "" { // nothing to destroy
		return
	}
	RemoveCookie(s.config.Cookie, res, req)

	s.provider.Destroy(cookieValue)
}

// DestroyByID removes the session entry
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
//
// It's safe to use it even if you are not sure if a session with that id exists.
//
// Note: the sid should be the original one (i.e: fetched by a store )
// it's not decoded.
func (s *Manager) DestroyByID(sid string) {
	s.provider.Destroy(sid)
}

// DestroyAll removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
func (s *Manager) DestroyAll() {
	s.provider.DestroyAll()
}

// let's keep these funcs simple, we can do it with two lines but we may add more things in the future.
func (s *Manager) decodeCookieValue(cookieValue string) string {
	var cookieValueDecoded *string
	if decode := s.config.Decode; decode != nil {
		err := decode(s.config.Cookie, cookieValue, &cookieValueDecoded)
		if err == nil {
			cookieValue = *cookieValueDecoded
		} else {
			cookieValue = ""
		}
	}
	return cookieValue
}

func (s *Manager) encodeCookieValue(cookieValue string) string {
	if encode := s.config.Encode; encode != nil {
		newVal, err := encode(s.config.Cookie, cookieValue)
		if err == nil {
			cookieValue = newVal
		} else {
			cookieValue = ""
		}
	}

	return cookieValue
}
