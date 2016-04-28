// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package sessions

import (
	"encoding/base64"
	// here care:  .DelCookie, is a new fasthttp function and maybe users have panics if their fasthttp package is not updating as expected.
	"net/url"
	"sync"
	"time"

	"github.com/kataras/iris"
	"github.com/valyala/fasthttp"
)

type (
	IManager interface {
		Start(*iris.Context) ISession
		Destroy(*iris.Context)
		GC()
	}

	Manager struct {
		cookieName   string
		mu           sync.Mutex
		provider     IProvider
		lifeDuration time.Duration
	}
)

var _ IManager = &Manager{}

var (
	providers = make(map[string]IProvider)
)

// NewManager creates & returns a new Manager
func NewManager(providerName string, cookieName string, lifeDuration time.Duration) (*Manager, error) {
	provider, found := providers[providerName]
	if !found {
		return nil, ErrProviderNotFound.Format(providerName)
	}

	if cookieName == "" {
		cookieName = "IrisCookieName"
	}

	manager := &Manager{}
	manager.provider = provider
	manager.cookieName = cookieName
	manager.lifeDuration = lifeDuration

	return manager, nil
}

// New creates & returns a new Manager, like NewManager does but it starts the GC and panics on error
func New(providerName string, cookieName string, lifeDuration time.Duration) *Manager {
	manager, err := NewManager(providerName, cookieName, lifeDuration)
	if err != nil {
		panic(err.Error()) // we have to panic here because we will start GC after and if provider is nil then many panics will come
	}
	//run the GC here
	go manager.GC()
	return manager
}

// Register registers a provider
func Register(providerName string, provider IProvider) {
	if provider == nil || providerName == "" {
		ErrProviderRegister.Panic()
	}

	if _, exists := providers[providerName]; exists {
		ErrProviderAlreadyExists.Panicf(providerName)
	}

	providers[providerName] = provider
}

// Manager implementation

func (m *Manager) generateSessionID() string {
	return base64.URLEncoding.EncodeToString(Random(32))
}

// Start starts the session
func (m *Manager) Start(ctx *iris.Context) ISession {
	var session ISession
	m.mu.Lock()

	//
	// uncomment only for debug
	//reqCtx.Request.Header.VisitAllCookie(func(k []byte, v []byte) {
	//	println(string(k) + " = " + string(v))
	//})
	//

	cookieValue := string(ctx.Request.Header.Cookie(m.cookieName))

	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := m.generateSessionID()
		session, _ = m.provider.Init(sid)
		cookie := &fasthttp.Cookie{}
		cookie.SetKey(m.cookieName)
		cookie.SetValue(url.QueryEscape(sid))
		cookie.SetPath("/")
		cookie.SetHTTPOnly(true)
		exp := time.Now().Add(m.lifeDuration)
		cookie.SetExpire(exp)
		ctx.Response.Header.SetCookie(cookie)
	} else {
		sid, _ := url.QueryUnescape(cookieValue)
		session, _ = m.provider.Read(sid)
	}

	m.mu.Unlock()
	return session
}

// Destroy kills the session and remove the associated cookie
func (m *Manager) Destroy(ctx *iris.Context) {
	cookieValue := string(ctx.Request.Header.Cookie(m.cookieName))
	if cookieValue == "" { // nothing to destroy
		return
	}

	m.mu.Lock()
	m.provider.Destroy(cookieValue)
	ctx.Response.Header.DelCookie(m.cookieName)
	ctx.Request.Header.DelCookie(m.cookieName) // maybe unnecessary
	m.mu.Unlock()
}

// GC tick-tock for the store cleanup
// it's a blocking function, so run it with go routine, it's totally safe
func (m *Manager) GC() {
	m.mu.Lock()

	m.provider.GC(m.lifeDuration)
	// set a timer for the next GC
	time.AfterFunc(m.lifeDuration, func() {
		m.GC()
	}) // or m.expire.Unix() if Nanosecond() doesn't works here
	m.mu.Unlock()
}
