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

package memory

import (
	"container/list"
	"sync"
	"time"

	"github.com/kataras/iris/sessions"
)

type (
	// Provider implements the IProvider. It's the Memory provider
	Provider struct {
		mu       sync.Mutex
		sessions map[string]*list.Element // underline memory store
		list     *list.List               // for GC
	}
)

var _ sessions.IProvider = &Provider{}

func (p *Provider) Init(sid string) (sessions.ISession, error) {
	p.mu.Lock()

	values := make(map[interface{}]interface{}, 0)
	newSessionStore := &Store{sid: sid, lastAccessedTime: time.Now(), values: values}
	elem := provider.list.PushBack(newSessionStore)
	provider.sessions[sid] = elem
	p.mu.Unlock()
	return newSessionStore, nil
}

func (p *Provider) Read(sid string) (sessions.ISession, error) {
	if elem, found := provider.sessions[sid]; found {
		return elem.Value.(*Store), nil
	} else {
		// if not found
		sessionStore, err := provider.Init(sid)
		return sessionStore, err
	}

	//if nothing was inside the sessions
	return nil, nil
}

// Destroy always returns a nil error, for now.
func (p *Provider) Destroy(sid string) error {
	if elem, found := provider.sessions[sid]; found {
		delete(provider.sessions, sid)
		provider.list.Remove(elem)
	}

	return nil
}

// Update updates the lastAccessedTime, and moves the memory place element to the front
// always returns a nil error, for now
func (p *Provider) Update(sid string) error {
	p.mu.Lock()

	if elem, found := provider.sessions[sid]; found {
		elem.Value.(*Store).lastAccessedTime = time.Now()
		provider.list.MoveToFront(elem)
	}

	p.mu.Unlock()
	return nil
}

// GC clears the memory
func (p *Provider) GC(duration time.Duration) {
	provider.mu.Lock()
	defer provider.mu.Unlock() //let's defer it and trust the go

	for {
		elem := provider.list.Back()
		if elem == nil {
			break
		}

		// if the time has passed. session was expired, then delete the session and its memory place
		if (elem.Value.(*Store).lastAccessedTime.Unix() + duration.Nanoseconds()) < time.Now().Unix() {
			provider.list.Remove(elem)
			delete(provider.sessions, elem.Value.(*Store).sid)
		} else {
			break
		}
	}
}
