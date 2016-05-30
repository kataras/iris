package sessions

import (
	"encoding/base64"
	"net/url"
	"sync"
	"time"

	"github.com/kataras/iris/config"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/sessions/store"
	"github.com/kataras/iris/utils"
	"github.com/valyala/fasthttp"
)

type (
	// IManager is the interface which Manager should implement
	IManager interface {
		Start(context.IContext) store.IStore
		Destroy(context.IContext)
		GC()
	}
	// Manager implements the IManager interface
	// contains the cookie's name, the provider and a duration for GC and cookie life expire
	Manager struct {
		config   *config.Sessions
		provider IProvider
		mu       sync.Mutex
	}
)

var _ IManager = &Manager{}

var (
	continueOnError = true
	providers       = make(map[string]IProvider)
)

// newManager creates & returns a new Manager
func newManager(c config.Sessions) (*Manager, error) {
	provider, found := providers[c.Provider]
	if !found {
		return nil, ErrProviderNotFound.Format(c.Provider)
	}

	manager := &Manager{}
	manager.config = &c
	manager.provider = provider

	return manager, nil
}

// Register registers a provider
func Register(provider IProvider) {
	if provider == nil {
		ErrProviderRegister.Panic()
	}
	providerName := provider.Name()

	if _, exists := providers[providerName]; exists {
		if !continueOnError {
			ErrProviderAlreadyExists.Panicf(providerName)
		} else {
			// do nothing it's a map it will overrides the existing provider.
		}
	}

	providers[providerName] = provider
}

// Manager implementation

func (m *Manager) generateSessionID() string {
	return base64.URLEncoding.EncodeToString(utils.Random(32))
}

// Start starts the session
func (m *Manager) Start(ctx context.IContext) store.IStore {

	m.mu.Lock()
	var store store.IStore
	requestCtx := ctx.GetRequestCtx()
	cookieValue := string(requestCtx.Request.Header.Cookie(m.config.Cookie))

	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := m.generateSessionID()
		store, _ = m.provider.Init(sid)
		cookie := fasthttp.AcquireCookie()
		cookie.SetKey(m.config.Cookie)
		cookie.SetValue(url.QueryEscape(sid))
		cookie.SetPath("/")
		cookie.SetHTTPOnly(true)
		cookie.SetExpire(m.config.Expires)
		requestCtx.Response.Header.SetCookie(cookie)
		fasthttp.ReleaseCookie(cookie)
		//println("manager.go:156-> Setting cookie with lifetime: ", m.lifeDuration.Seconds())
	} else {
		sid, _ := url.QueryUnescape(cookieValue)
		store, _ = m.provider.Read(sid)
	}

	m.mu.Unlock()
	return store
}

// Destroy kills the session and remove the associated cookie
func (m *Manager) Destroy(ctx context.IContext) {
	cookieValue := string(ctx.GetRequestCtx().Request.Header.Cookie(m.config.Cookie))
	if cookieValue == "" { // nothing to destroy
		return
	}

	m.mu.Lock()
	m.provider.Destroy(cookieValue)

	ctx.RemoveCookie(m.config.Cookie)

	m.mu.Unlock()
}

// GC tick-tock for the store cleanup
// it's a blocking function, so run it with go routine, it's totally safe
func (m *Manager) GC() {
	m.mu.Lock()

	m.provider.GC(m.config.GcDuration)
	// set a timer for the next GC
	time.AfterFunc(m.config.GcDuration, func() {
		m.GC()
	}) // or m.expire.Unix() if Nanosecond() doesn't works here
	m.mu.Unlock()
}
