package sessions

import (
	"net/http"
	"strings"
	"time"

	"github.com/kataras/iris/context"
)

// A Sessions manager should be responsible to Start a sesion, based
// on a Context, which should return
// a compatible Session interface, type. If the external session manager
// doesn't qualifies, then the user should code the rest of the functions with empty implementation.
//
// Sessions should be responsible to Destroy a session based
// on the Context.
type Sessions struct {
	config   Config
	provider *provider
}

// New returns a new fast, feature-rich sessions manager
// it can be adapted to an iris station
func New(cfg Config) *Sessions {
	return &Sessions{
		config:   cfg.Validate(),
		provider: newProvider(),
	}
}

// UseDatabase adds a session database to the manager's provider,
// a session db doesn't have write access
func (s *Sessions) UseDatabase(db Database) {
	s.provider.RegisterDatabase(db)
}

// updateCookie gains the ability of updating the session browser cookie to any method which wants to update it
func (s *Sessions) updateCookie(ctx context.Context, sid string, expires time.Duration) {
	cookie := &http.Cookie{}

	// The RFC makes no mention of encoding url value, so here I think to encode both sessionid key and the value using the safe(to put and to use as cookie) url-encoding
	cookie.Name = s.config.Cookie

	cookie.Value = sid
	cookie.Path = "/"
	if !s.config.DisableSubdomainPersistence {

		requestDomain := ctx.Host()
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
	if expires >= 0 {
		if expires == 0 { // unlimited life
			cookie.Expires = CookieExpireUnlimited
		} else { // > 0
			cookie.Expires = time.Now().Add(expires)
		}
		cookie.MaxAge = int(cookie.Expires.Sub(time.Now()).Seconds())
	}

	// set the cookie to secure if this is a tls wrapped request
	// and the configuration allows it.
	if ctx.Request().TLS != nil && s.config.CookieSecureTLS {
		cookie.Secure = true
	}

	// encode the session id cookie client value right before send it.
	cookie.Value = s.encodeCookieValue(cookie.Value)
	AddCookie(ctx, cookie, s.config.AllowReclaim)
}

// Start should start the session for the particular request.
func (s *Sessions) Start(ctx context.Context) *Session {
	cookieValue := s.decodeCookieValue(GetCookie(ctx, s.config.Cookie))

	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := s.config.SessionIDGenerator()

		sess := s.provider.Init(sid, s.config.Expires)
		sess.isNew = sess.values.Len() == 0

		s.updateCookie(ctx, sid, s.config.Expires)

		return sess
	}

	sess := s.provider.Read(cookieValue, s.config.Expires)

	return sess
}

// ShiftExpiration move the expire date of a session to a new date
// by using session default timeout configuration.
func (s *Sessions) ShiftExpiration(ctx context.Context) {
	s.UpdateExpiration(ctx, s.config.Expires)
}

// UpdateExpiration change expire date of a session to a new date
// by using timeout value passed by `expires` receiver.
func (s *Sessions) UpdateExpiration(ctx context.Context, expires time.Duration) {
	cookieValue := s.decodeCookieValue(GetCookie(ctx, s.config.Cookie))

	if cookieValue != "" {
		// we should also allow it to expire when the browser closed
		if s.provider.UpdateExpiration(cookieValue, expires) || expires == -1 {
			s.updateCookie(ctx, cookieValue, expires)
		}
	}
}

// Destroy remove the session data and remove the associated cookie.
func (s *Sessions) Destroy(ctx context.Context) {
	cookieValue := GetCookie(ctx, s.config.Cookie)
	// decode the client's cookie value in order to find the server's session id
	// to destroy the session data.
	cookieValue = s.decodeCookieValue(cookieValue)
	if cookieValue == "" { // nothing to destroy
		return
	}
	RemoveCookie(ctx, s.config.Cookie, s.config.AllowReclaim)

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
func (s *Sessions) DestroyByID(sid string) {
	s.provider.Destroy(sid)
}

// DestroyAll removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
func (s *Sessions) DestroyAll() {
	s.provider.DestroyAll()
}

// let's keep these funcs simple, we can do it with two lines but we may add more things in the future.
func (s *Sessions) decodeCookieValue(cookieValue string) string {
	if cookieValue == "" {
		return ""
	}

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

func (s *Sessions) encodeCookieValue(cookieValue string) string {
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
