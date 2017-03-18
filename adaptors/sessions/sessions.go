// Package sessions as originally written by me at https://github.com/kataras/go-sessions
// Based on kataras/go-sessions v1.0.1.
//
// Edited for Iris v6 (or iris vNext) and removed all fasthttp things in order to reduce the
// compiled and go getable size. The 'file' and 'leveldb' databases are missing
// because they written by community, not me, you can still adapt any database with
// .UseDatabase because it expects an interface,
//              find more databases here: https://github.com/kataras/go-sessions/tree/master/sessiondb
package sessions

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"gopkg.in/kataras/iris.v6"
)

type (
	// Sessions is the start point of this package
	// contains all the registered sessions and manages them
	Sessions interface {
		// Adapt is used to adapt this sessions manager as an iris.SessionsPolicy
		// to an Iris station.
		// It's being used by the framework, developers should not actually call this function.
		Adapt(*iris.Policies)

		// UseDatabase ,optionally, adds a session database to the manager's provider,
		// a session db doesn't have write access
		// see https://github.com/kataras/go-sessions/tree/master/sessiondb for its usage.
		UseDatabase(Database)

		// Start starts the session for the particular net/http request
		Start(http.ResponseWriter, *http.Request) iris.Session

		// Destroy deletes all session data and remove the associated cookie.
		Destroy(http.ResponseWriter, *http.Request)

		// DestroyByID removes the session entry
		// from the server-side memory (and database if registered).
		// Client's session cookie will still exist but it will be reseted on the next request.
		//
		// It's safe to use it even if you are not sure if a session with that id exists.
		//
		// Note: the sid should be the original one (i.e: fetched by a store )
		// it's not decoded.
		DestroyByID(string)
		// DestroyAll removes all sessions
		// from the server-side memory (and database if registered).
		// Client's session cookie will still exist but it will be reseted on the next request.
		DestroyAll()
	}

	// sessions contains the cookie's name, the provider and a duration for GC and cookie life expire
	sessions struct {
		config   Config
		provider *provider
	}
)

// New returns a new fast, feature-rich sessions manager
// it can be adapted to an Iris station
func New(cfg Config) Sessions {
	return &sessions{
		config:   cfg.Validate(),
		provider: newProvider(),
	}
}

func (s *sessions) Adapt(frame *iris.Policies) {
	// for newcomers this maybe looks strange:
	// Each policy is an adaptor too, so they all can contain an Adapt.
	// If they contains an Adapt func then the policy is an adaptor too and this Adapt func is called
	// by Iris on .Adapt(...)
	policy := iris.SessionsPolicy{
		Start:   s.Start,
		Destroy: s.Destroy,
	}

	policy.Adapt(frame)

}

// UseDatabase adds a session database to the manager's provider,
// a session db doesn't have write access
func (s *sessions) UseDatabase(db Database) {
	s.provider.RegisterDatabase(db)
}

// Start starts the session for the particular net/http request
func (s *sessions) Start(res http.ResponseWriter, req *http.Request) iris.Session {
	var sess iris.Session

	cookieValue := GetCookie(s.config.Cookie, req)

	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := SessionIDGenerator(s.config.CookieLength)

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
func (s *sessions) Destroy(res http.ResponseWriter, req *http.Request) {
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
func (s *sessions) DestroyByID(sid string) {
	s.provider.Destroy(sid)
}

// DestroyAll removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
func (s *sessions) DestroyAll() {
	s.provider.DestroyAll()
}

// SessionIDGenerator returns a random string, used to set the session id
// you are able to override this to use your own method for generate session ids.
var SessionIDGenerator = func(strLength int) string {
	return base64.URLEncoding.EncodeToString(random(strLength))
}

// let's keep these funcs simple, we can do it with two lines but we may add more things in the future.
func (s *sessions) decodeCookieValue(cookieValue string) string {
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

func (s *sessions) encodeCookieValue(cookieValue string) string {
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
