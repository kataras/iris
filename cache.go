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
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

import (
	"sync"
)

// max items on the cache, if its not defined
const MAX_ITEMS int = 9999999

// IRouterCache is the interface which the MemoryRouter implements
type IRouterCache interface {
	OnTick()
	AddItem(method, url string, ctx *Context)
	GetItem(method, url string) *Context
	SetMaxItems(maxItems int)
	GetMaxItems() int
}

// MemoryRouterCache creation done with just &MemoryRouterCache{}
type MemoryRouterCache struct {
	//1. map[string] ,key is HTTP Method(GET,POST...)
	//2. map[string]*Context ,key is The Request URL Path
	//the map in this case is the faster way, I tried with array of structs but it's 100 times slower on > 1 core because of async goroutes on addItem I sugges, so we keep the map
	items    map[string]map[string]*Context
	// set the limit of items that could be cached
	MaxItems int
}

type SyncMemoryRouterCache struct {
	*MemoryRouterCache
	//we need this mutex if we have running the iris at > 1 core, because we use map but maybe at the future I will change it.
	mu sync.Mutex
}

var _ IRouterCache = &MemoryRouterCache{}
var _ IRouterCache = &SyncMemoryRouterCache{}

// SetMaxItems receives int and set max cached items to this number
func (mc *MemoryRouterCache) SetMaxItems(_itemslen int) {
	mc.MaxItems = _itemslen
}

// GetMaxItems returns the limit of cache items
func (mc MemoryRouterCache) GetMaxItems() int {
	return mc.MaxItems
}

// NewMemoryRouterCache returns the cache for a router, is used on the MemoryRouter
func NewMemoryRouterCache() *MemoryRouterCache {
	mc := &MemoryRouterCache{items: make(map[string]map[string]*Context, 0)}
	mc.MaxItems = MAX_ITEMS
	mc.resetBag()
	return mc
}

// NewMemoryRouterCache returns the cache for a router, it's based on the one-thread MemoryRouterCache
func NewSyncMemoryRouterCache(underlineCache *MemoryRouterCache) *SyncMemoryRouterCache {
	mc := &SyncMemoryRouterCache{MemoryRouterCache: underlineCache, mu: sync.Mutex{}}
	mc.resetBag()
	return mc
}

// AddItem adds an item to the bag/cache, is a goroutine.
func (mc *MemoryRouterCache) AddItem(method, url string, ctx *Context) {
	if len(mc.items[method]) < mc.MaxItems {
		mc.items[method][url] = ctx
	}
}

// AddItem adds an item to the bag/cache, is a goroutine.
func (mc *SyncMemoryRouterCache) AddItem(method, url string, ctx *Context) {
	go func(method, url string, c *Context) { //for safety on multiple fast calls
		if len(mc.items[method]) < mc.MaxItems {
			mc.mu.Lock()
			mc.items[method][url] = c
			mc.mu.Unlock()
		}
	}(method, url, ctx)
}

// GetItem returns an item from the bag/cache, if not exists it returns just nil.
func (mc *MemoryRouterCache) GetItem(method, url string) *Context {
	if ctx := mc.items[method][url]; ctx != nil {
		return ctx
	}
	return nil
}

// GetItem returns an item from the bag/cache, if not exists it returns just nil.
func (mc *SyncMemoryRouterCache) GetItem(method, url string) *Context {
	mc.mu.Lock()
	if ctx := mc.items[method][url]; ctx != nil {
		mc.mu.Unlock()
		return ctx
	}
	mc.mu.Unlock()
	return nil
}

func (mc *MemoryRouterCache) DoOnTick() {
	if mc.MaxItems == 0 {
		//just reset to complete new maps all methods
		mc.resetBag()
		return
	}

	//loop each method on bag and clear it if it's len is more than MaxItems
	for k, v := range mc.items {
		if len(v) >= mc.MaxItems {
			//we just create a new map, no delete each manualy because this number maybe be very long.
			mc.items[k] = make(map[string]*Context, 0)
		}
	}
}

// OnTick is the implementation of the ITick
// it makes the MemoryRouterCache a ticker's listener
func (mc *MemoryRouterCache) OnTick() {
	mc.DoOnTick()
}

// OnTick is the implementation of the ITick
// it makes the MemoryRouterCache a ticker's listener
func (mc *SyncMemoryRouterCache) OnTick() {
	mc.mu.Lock()
	mc.DoOnTick()
	mc.mu.Unlock()
}

// resetBag clears the cached items
func (mc *MemoryRouterCache) resetBag() {
	for _, m := range HTTPMethods.ANY {
		mc.items[m] = make(map[string]*Context, 0)
	}
}
