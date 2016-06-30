package sessions

import (
	"encoding/base64"
	"strings"
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
		config         *config.Sessions
		provider       IProvider
		mu             sync.Mutex
		compiledCookie string
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
	manager.compiledCookie = base64.URLEncoding.EncodeToString([]byte(c.Cookie))
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

var dotB = byte('.')

// Start starts the session
func (m *Manager) Start(ctx context.IContext) store.IStore {

	m.mu.Lock()
	var store store.IStore
	requestCtx := ctx.GetRequestCtx()
	cookieValue := string(requestCtx.Request.Header.Cookie(m.compiledCookie))

	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := m.generateSessionID()
		store, _ = m.provider.Init(sid)
		cookie := fasthttp.AcquireCookie()
		// The RFC makes no mention of encoding url value, so here I think to encode both sessionid key and the value using the safe(to put and to use as cookie) url-encoding
		cookie.SetKey(m.compiledCookie)
		cookie.SetValue(base64.URLEncoding.EncodeToString([]byte(sid)))
		cookie.SetPath("/")
		if !m.config.DisableSubdomainPersistance {
			requestDomain := ctx.HostString()

			if portIdx := strings.IndexByte(requestDomain, ':'); portIdx > 0 {
				requestDomain = requestDomain[0:portIdx]
			}
			// RFC2109, we allow level 1 subdomains, but no further
			// if we have localhost.com , we want the localhost.com.
			// so if we have something like: mysubdomain.localhost.com we want the localhost here
			// if we have mysubsubdomain.mysubdomain.localhost.com we want the .mysubdomain.localhost.com here
			// slow things here, especially the 'replace' but this is a good and understable( I hope) way to get the be able to set cookies from subdomains & domain with 1-level limit
			if dotIdx := strings.LastIndexByte(requestDomain, dotB); dotIdx > 0 {
				// is mysubdomain.localhost.com || mysubsubdomain.mysubdomain.localhost.com
				s := requestDomain[0:dotIdx] // set mysubdomain.localhost || mysubsubdomain.mysubdomain.localhost
				if secondDotIdx := strings.LastIndexByte(s, dotB); secondDotIdx > 0 {
					//is mysubdomain.localhost ||  mysubsubdomain.mysubdomain.localhost
					s = s[secondDotIdx+1:] // set to localhost || mysubdomain.localhost
				}
				// replace the s with the requestDomain before the domain's siffux
				subdomainSuff := strings.LastIndexByte(requestDomain, dotB)
				if subdomainSuff > len(s) { // if it is actual exists as subdomain suffix
					requestDomain = strings.Replace(requestDomain, requestDomain[0:subdomainSuff], s, 1) // set to localhost.com || mysubdomain.localhost.com
				}
			}
			cookie.SetDomain("." + requestDomain) // . to allow persistance
		}
		cookie.SetHTTPOnly(true)
		cookie.SetExpire(m.config.Expires)
		requestCtx.Response.Header.SetCookie(cookie)
		fasthttp.ReleaseCookie(cookie)
	} else {
		sid, _ := base64.URLEncoding.DecodeString(cookieValue)
		store, _ = m.provider.Read(string(sid))
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
