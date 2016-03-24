// Copyright (c) 2016, Iris Team
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
  "time"
  "testing"
)

func TestMemoryRouterCacheByInterface(t *testing.T) {
  var memoryCache IRouterCache
  memoryCache = NewMemoryRouterCache()
  if memoryCache == nil {
    t.Error("MemoryCache would not be nil")
  }
  memoryCache.SetMaxItems(1)
}

func TestMemoryRouterCacheAddItems(t *testing.T) {
  memoryCache := NewMemoryRouterCache()
  if memoryCache == nil {
    t.Error("MemoryCache would not be nil")
  }

  memoryCache.SetMaxItems(1)
  memoryCache.AddItem("GET", "/", nil)
  memoryCache.AddItem("POST", "/", nil)
  if 1 > memoryCache.MaxItems  {
    t.Errorf("MaxItems would be %d, but has %d", 1, memoryCache.MaxItems)
  }
}

func TestMemoryRouterCacheGetItems(t *testing.T) {
  memoryCache := NewMemoryRouterCache()
  memoryCache.SetMaxItems(1)
  memoryCache.AddItem("GET", "/", &Context{})
  ctx := memoryCache.GetItem("GET", "/")
  if ctx == nil  {
    t.Errorf("Item should not be nil")
  }
}

/**
 * testing SyncMemoryRouterCache
 */
func TestSyncMemoryRouterCacheByInterface(t *testing.T) {
  var syncMemCache IRouterCache
  memoryCache := NewMemoryRouterCache()
  syncMemCache = NewSyncMemoryRouterCache(memoryCache)
  if syncMemCache == nil {
    t.Error("SyncMemoryCache would not be nil")
  }
  syncMemCache.SetMaxItems(1)
}

func TestSyncMemoryRouterCacheAddItems(t *testing.T) {
  memoryCache := NewMemoryRouterCache()
  syncMemCache := NewSyncMemoryRouterCache(memoryCache)
  if syncMemCache == nil {
    t.Error("MemoryCache would not be nil")
  }

  syncMemCache.SetMaxItems(1)
  syncMemCache.AddItem("GET", "/", nil)
  syncMemCache.AddItem("POST", "/", nil)

  // waits 500ms to goroutine process
  time.Sleep(500 * time.Millisecond)

  if 1 != syncMemCache.MaxItems {
    t.Errorf("MaxItems would be %d, but has %d", 1, syncMemCache.MaxItems)
  }
}

func TestSyncMemoryRouterCacheGetItems(t *testing.T) {
  memoryCache := NewMemoryRouterCache()
  syncMemCache := NewSyncMemoryRouterCache(memoryCache)
  syncMemCache.AddItem("GET", "/", &Context{})

  // waits 500ms to goroutine process
  time.Sleep(600 * time.Millisecond)

  if ctx := syncMemCache.GetItem("GET", "/"); ctx == nil  {
    t.Errorf("Item should not be nil")
  }
}
