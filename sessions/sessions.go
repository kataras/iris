package sessions

import (
	"net/http"
	"net/url"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/host"
)

func init() {
	context.SetHandlerName("iris/sessions.*Handler", "iris.session")
}

// A Sessions manager should be responsible to Start/Get a sesion, based
// on a Context, which returns a *Session, type.
// It performs automatic memory cleanup on expired sessions.
// It can accept a `Database` for persistence across server restarts.
// A session can set temporary values (flash messages).
type Sessions struct {
	config   Config
	provider *provider

	cookieOptions []context.CookieOption // options added on each session cookie action.
}

// New returns a new fast, feature-rich sessions manager
// it can be adapted to an iris station
func New(cfg Config) *Sessions {
	var cookieOptions []context.CookieOption
	if cfg.AllowReclaim {
		cookieOptions = append(cookieOptions, context.CookieAllowReclaim(cfg.Cookie))
	}
	if !cfg.DisableSubdomainPersistence {
		cookieOptions = append(cookieOptions, context.CookieAllowSubdomains(cfg.Cookie))
	}
	if cfg.CookieSecureTLS {
		cookieOptions = append(cookieOptions, context.CookieSecure)
	}
	if cfg.Encoding != nil {
		cookieOptions = append(cookieOptions, context.CookieEncoding(cfg.Encoding, cfg.Cookie))
	}

	return &Sessions{
		cookieOptions: cookieOptions,
		config:        cfg.Validate(),
		provider:      newProvider(),
	}
}

// UseDatabase adds a session database to the manager's provider,
// a session db doesn't have write access
func (s *Sessions) UseDatabase(db Database) {
	db.SetLogger(s.config.Logger) // inject the logger.
	host.RegisterOnInterrupt(func() {
		db.Close()
	})
	s.provider.RegisterDatabase(db)
}

// GetCookieOptions returns the cookie options registered
// for this sessions manager based on the configuration.
func (s *Sessions) GetCookieOptions() []context.CookieOption {
	return s.cookieOptions
}

// updateCookie gains the ability of updating the session browser cookie to any method which wants to update it
func (s *Sessions) updateCookie(ctx *context.Context, sid string, expires time.Duration, options ...context.CookieOption) {
	cookie := &http.Cookie{}

	// The RFC makes no mention of encoding url value, so here I think to encode both sessionid key and the value using the safe(to put and to use as cookie) url-encoding
	cookie.Name = s.config.Cookie
	cookie.Value = sid
	cookie.Path = "/"
	cookie.HttpOnly = true

	// MaxAge=0 means no 'Max-Age' attribute specified.
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	// MaxAge>0 means Max-Age attribute present and given in seconds
	if expires >= 0 {
		if expires == 0 { // unlimited life
			cookie.Expires = context.CookieExpireUnlimited
		} else { // > 0
			cookie.Expires = time.Now().Add(expires)
		}
		cookie.MaxAge = int(time.Until(cookie.Expires).Seconds())
	}

	s.upsertCookie(ctx, cookie, options)
}

func (s *Sessions) upsertCookie(ctx *context.Context, cookie *http.Cookie, cookieOptions []context.CookieOption) {
	opts := s.cookieOptions
	if len(cookieOptions) > 0 {
		opts = append(opts, cookieOptions...)
	}

	ctx.UpsertCookie(cookie, opts...)
}

func (s *Sessions) getCookieValue(ctx *context.Context, cookieOptions []context.CookieOption) string {
	c := s.getCookie(ctx, cookieOptions)
	if c == nil {
		return ""
	}
	return c.Value
}

func (s *Sessions) getCookie(ctx *context.Context, cookieOptions []context.CookieOption) *http.Cookie {
	opts := s.cookieOptions
	if len(cookieOptions) > 0 {
		opts = append(opts, cookieOptions...)
	}

	cookie, err := ctx.GetRequestCookie(s.config.Cookie, opts...)
	if err != nil {
		return nil
	}

	cookie.Value, _ = url.QueryUnescape(cookie.Value)
	return cookie
}

// Start creates or retrieves an existing session for the particular request.
// Note that `Start` method will not respect configuration's `AllowReclaim`, `DisableSubdomainPersistence`, `CookieSecureTLS`,
// and `Encoding` settings.
// Register sessions as a middleware through the `Handler` method instead,
// which provides automatic resolution of a *sessions.Session input argument
// on MVC and APIContainer as well.
//
// NOTE: Use `app.Use(sess.Handler())` instead, avoid using `Start` manually.
func (s *Sessions) Start(ctx *context.Context, cookieOptions ...context.CookieOption) *Session {
	// cookieValue := s.getCookieValue(ctx, cookieOptions)
	cookie := s.getCookie(ctx, cookieOptions)
	if cookie != nil {
		sid := cookie.Value
		if sid == "" { // rare case: a client may contains a cookie with session name but with empty value.
			// ctx.RemoveCookie(cookie.Name)
			cookie = nil
		} else if cookie.Expires.Add(time.Second).After(time.Now()) { // rare case: of custom clients that may hold expired cookies.
			s.DestroyByID(sid)
			// ctx.RemoveCookie(cookie.Name)
			cookie = nil
		} else {
			// rare case: new expiration configuration that it's lower
			// than the previous setting.
			expiresTime := time.Now().Add(s.config.Expires)
			if cookie.Expires.After(expiresTime) {
				s.DestroyByID(sid)
				//	ctx.RemoveCookie(cookie.Name)
				cookie = nil
			} else {
				//	untilExpirationDur := time.Until(cookie.Expires)
				// ^ this should be
				return s.provider.Read(s, sid, s.config.Expires) // cookie exists and it's valid, let's return its session.
			}
		}
	}

	// Cookie doesn't exist, let's generate a session and set a cookie.
	sid := s.config.SessionIDGenerator(ctx)

	sess := s.provider.Init(s, sid, s.config.Expires)
	// n := s.provider.db.Len(sid)
	// fmt.Printf("db.Len(%s) = %d\n", sid, n)
	// if n > 0 {
	// 	s.provider.db.Visit(sid, func(key string, value interface{}) {
	// 		fmt.Printf("%s=%s\n", key, value)
	// 	})
	// }
	s.updateCookie(ctx, sid, s.config.Expires, cookieOptions...)
	return sess
}

const sessionContextKey = "iris.session"

// Handler returns a sessions middleware to register on application routes.
// To return the request's Session call the `Get(ctx)` package-level function.
//
// Call `Handler()` once per sessions manager.
func (s *Sessions) Handler(requestOptions ...context.CookieOption) context.Handler {
	return func(ctx *context.Context) {
		session := s.Start(ctx, requestOptions...) // this cookie's end-developer's custom options.

		ctx.Values().Set(sessionContextKey, session)
		ctx.Next()

		s.provider.EndRequest(ctx, session)
	}
}

// Get returns a *Session from the same request life cycle,
// can be used inside a chain of handlers of a route.
//
// The `Sessions.Start` should be called previously,
// e.g. register the `Sessions.Handler` as middleware.
// Then call `Get` package-level function as many times as you want.
// Note: It will return nil if the session got destroyed by the same request.
// If you need to destroy and start a new session in the same request you need to call
// sessions manager's `Start` method after Destroy.
func Get(ctx *context.Context) *Session {
	if v := ctx.Values().Get(sessionContextKey); v != nil {
		if sess, ok := v.(*Session); ok {
			return sess
		}
	}

	// ctx.Application().Logger().Debugf("Sessions: Get: no session found, prior Destroy(ctx) calls in the same request should follow with a Start(ctx) call too")
	return nil
}

// StartWithPath same as `Start` but it explicitly accepts the cookie path option.
func (s *Sessions) StartWithPath(ctx *context.Context, path string) *Session {
	return s.Start(ctx, context.CookiePath(path))
}

// ShiftExpiration move the expire date of a session to a new date
// by using session default timeout configuration.
// It will return `ErrNotImplemented` if a database is used and it does not support this feature, yet.
func (s *Sessions) ShiftExpiration(ctx *context.Context, cookieOptions ...context.CookieOption) error {
	return s.UpdateExpiration(ctx, s.config.Expires, cookieOptions...)
}

// UpdateExpiration change expire date of a session to a new date
// by using timeout value passed by `expires` receiver.
// It will return `ErrNotFound` when trying to update expiration on a non-existence or not valid session entry.
// It will return `ErrNotImplemented` if a database is used and it does not support this feature, yet.
func (s *Sessions) UpdateExpiration(ctx *context.Context, expires time.Duration, cookieOptions ...context.CookieOption) error {
	cookieValue := s.getCookieValue(ctx, cookieOptions)
	if cookieValue == "" {
		return ErrNotFound
	}

	// we should also allow it to expire when the browser closed
	err := s.provider.UpdateExpiration(cookieValue, expires)
	if err == nil || expires == -1 {
		s.updateCookie(ctx, cookieValue, expires, cookieOptions...)
	}

	return err
}

// DestroyListener is the form of a destroy listener.
// Look `OnDestroy` for more.
type DestroyListener func(sid string)

// OnDestroy registers one or more destroy listeners.
// A destroy listener is fired when a session has been removed entirely from the server (the entry) and client-side (the cookie).
// Note that if a destroy listener is blocking, then the session manager will delay respectfully,
// use a goroutine inside the listener to avoid that behavior.
func (s *Sessions) OnDestroy(listeners ...DestroyListener) {
	for _, ln := range listeners {
		s.provider.registerDestroyListener(ln)
	}
}

// Destroy removes the session data, the associated cookie
// and the Context's session value.
// Next calls of `sessions.Get` will occur to a nil Session,
// use `Sessions#Start` method for renewal
// or use the Session's Destroy method which does keep the session entry with its values cleared.
func (s *Sessions) Destroy(ctx *context.Context) {
	cookieValue := s.getCookieValue(ctx, nil)
	if cookieValue == "" { // nothing to destroy
		return
	}

	ctx.Values().Remove(sessionContextKey)

	ctx.RemoveCookie(s.config.Cookie, s.cookieOptions...)
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
