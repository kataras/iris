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

package iris

// cache.go: from the version 1, we don't need this atm, but we keep it.

import (
	"sync"
)

type (
	// IContextCache is the interface of the ContextCache & SyncContextCache
	IContextCache interface {
		OnTick()
		AddItem(method, url string, ctx *Context) // This is the faster method, just set&return just a *Context, I tried to return only params and middleware but it's add 10.000 nanoseconds, also +2k bytes of memory . So let's keep it as I did thee first time, I don't know what do do to make it's performance even better... It doesn't go much best, can't use channels because the performance will be get very low, locks are better for this purpose.
		GetItem(method, url string) *Context
		SetMaxItems(maxItems int)
	}

	// ContextCache creation done with just &ContextCache{}
	ContextCache struct {
		//1. map[string] ,key is HTTP Method(GET,POST...)
		//2. map[string]*Context ,key is The Request URL Path
		//the map in this case is the faster way, I tried with array of structs but it's 100 times slower on > 1 core because of async goroutes on addItem I sugges, so we keep the map
		items    map[string]map[string]*Context
		MaxItems int
	}

	// SyncContextCache is the cache version of routine-thread-safe ContextCache
	SyncContextCache struct {
		*ContextCache
		//we need this mutex if we have running the iris at > 1 core, because we use map but maybe at the future I will change it.
		mu *sync.RWMutex
	}
)

var _ IContextCache = &ContextCache{}
var _ IContextCache = &SyncContextCache{}

// SetMaxItems receives int and set max cached items to this number
func (mc *ContextCache) SetMaxItems(_itemslen int) {
	mc.MaxItems = _itemslen
}

// NewContextCache returns the cache for a router, is used on the MemoryRouter
func NewContextCache() *ContextCache {
	mc := &ContextCache{items: make(map[string]map[string]*Context, 0)}
	mc.resetBag()
	return mc
}

// NewSyncContextCache returns the cache for a router, it's based on the one-thread ContextCache
func NewSyncContextCache(underlineCache *ContextCache) *SyncContextCache {
	mc := &SyncContextCache{ContextCache: underlineCache, mu: new(sync.RWMutex)}
	mc.resetBag()
	return mc
}

// AddItem adds an item to the bag/cache, is a goroutine.
func (mc *ContextCache) AddItem(method, url string, ctx *Context) {
	mc.items[method][url] = ctx
}

// AddItem adds an item to the bag/cache, is a goroutine.
func (mc *SyncContextCache) AddItem(method, url string, ctx *Context) {
	go func(method, url string, c *Context) { //for safety on multiple fast calls
		mc.mu.Lock()
		mc.items[method][url] = ctx
		mc.mu.Unlock()
	}(method, url, ctx)
}

// GetItem returns an item from the bag/cache, if not exists it returns just nil.
func (mc *ContextCache) GetItem(method, url string) *Context {
	if ctx := mc.items[method][url]; ctx != nil {
		return ctx
	}

	return nil
}

// GetItem returns an item from the bag/cache, if not exists it returns just nil.
func (mc *SyncContextCache) GetItem(method, url string) *Context {
	mc.mu.RLock()
	if ctx := mc.items[method][url]; ctx != nil {
		mc.mu.RUnlock()
		return ctx
	}
	mc.mu.RUnlock()
	return nil
}

// DoOnTick raised every time the ticker ticks, can be called independed, it's just check for items len and resets the cache
func (mc *ContextCache) DoOnTick() {

	if mc.MaxItems == 0 {
		//just reset to complete new maps all methods
		mc.resetBag()
	} else {
		//loop each method on bag and clear it if it's len is more than MaxItems
		for k, v := range mc.items {
			if len(v) >= mc.MaxItems {
				//we just create a new map, no delete each manualy because this number maybe be very long.
				mc.items[k] = make(map[string]*Context, 0)
			}
		}
	}
}

// OnTick is the implementation of the ITick
// it makes the ContextCache a ticker's listener
func (mc *ContextCache) OnTick() {
	mc.DoOnTick()
}

// OnTick is the implementation of the ITick
// it makes the ContextCache a ticker's listener
func (mc *SyncContextCache) OnTick() {
	mc.mu.Lock()
	mc.ContextCache.DoOnTick()
	mc.mu.Unlock()
}

// resetBag clears the cached items
func (mc *ContextCache) resetBag() {
	for _, m := range HTTPMethods.All {
		mc.items[m] = make(map[string]*Context, 0)
	}
}
