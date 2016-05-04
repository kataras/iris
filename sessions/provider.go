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
	"container/list"
	"sync"
	"time"
)

type IProvider interface {
	Init(string) (IStore, error)
	Read(string) (IStore, error)
	Destroy(string) error
	Update(string) error
	GC(time.Duration)
}

type (
	// Provider implements the IProvider
	Provider struct {
		mu                 sync.Mutex
		sessions           map[string]*list.Element // underline TEMPORARY memory store
		list               *list.List               // for GC
		NewStore           func(sessionId string, cookieLifeDuration time.Duration) IStore
		OnDestroy          func(store IStore) // this is called when .Destroy
		cookieLifeDuration time.Duration
	}
)

var _ IProvider = &Provider{}

// NewProvider returns a new empty Provider
func NewProvider() *Provider {
	provider := &Provider{list: list.New()}
	provider.sessions = make(map[string]*list.Element, 0)
	return provider
}

func (p *Provider) Init(sid string) (IStore, error) {
	p.mu.Lock()

	newSessionStore := p.NewStore(sid, p.cookieLifeDuration)

	elem := p.list.PushBack(newSessionStore)
	p.sessions[sid] = elem
	p.mu.Unlock()
	return newSessionStore, nil
}

func (p *Provider) Read(sid string) (IStore, error) {
	if elem, found := p.sessions[sid]; found {
		return elem.Value.(IStore), nil
	} else {
		// if not found
		sessionStore, err := p.Init(sid)
		return sessionStore, err
	}

	//if nothing was inside the sessions
	return nil, nil
}

// Destroy always returns a nil error, for now.
func (p *Provider) Destroy(sid string) error {
	if elem, found := p.sessions[sid]; found {
		elem.Value.(IStore).Destroy()
		delete(p.sessions, sid)
		p.list.Remove(elem)
	}

	return nil
}

// Update updates the lastAccessedTime, and moves the memory place element to the front
// always returns a nil error, for now
func (p *Provider) Update(sid string) error {
	p.mu.Lock()

	if elem, found := p.sessions[sid]; found {
		elem.Value.(IStore).SetLastAccessedTime(time.Now())
		p.list.MoveToFront(elem)
	}

	p.mu.Unlock()
	return nil
}

// GC clears the memory
func (p *Provider) GC(duration time.Duration) {
	p.mu.Lock()
	p.cookieLifeDuration = duration
	defer p.mu.Unlock() //let's defer it and trust the go

	for {
		elem := p.list.Back()
		if elem == nil {
			break
		}

		// if the time has passed. session was expired, then delete the session and its memory place
		if (elem.Value.(IStore).LastAccessedTime().Unix() + duration.Nanoseconds()) < time.Now().Unix() {
			p.list.Remove(elem)
			delete(p.sessions, elem.Value.(IStore).ID())

		} else {
			break
		}
	}
}
