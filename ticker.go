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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

import (
	"time"
)

// Ticker is the timer which is used in cache
type Ticker struct {
	ticker       *time.Ticker
	started      bool
	tickHandlers []func()
}

// NewTicker returns a new Ticker
func NewTicker() *Ticker {
	return &Ticker{tickHandlers: make([]func(), 0), started: false}
}

// OnTick add event handlers/ callbacks which are called on each timer's tick
func (c *Ticker) OnTick(h func()) {
	c.tickHandlers = append(c.tickHandlers, h)
}

// Start starts the timer and execute all listener's when tick
func (c *Ticker) Start(duration time.Duration) {
	if c.started {
		return
	}

	if c.ticker != nil {
		panic("Iris Ticker: Cannot re-start a cache timer, if you stop it, it is not recommented to resume it,\n Just create a new CacheTimer.")
	}

	if duration.Seconds() <= 30 {
		//c.duration = 5 * time.Minute
		panic("Iris Ticker: Please provide a duration that it's longer than 30 seconds.")
	}

	c.ticker = time.NewTicker(duration)

	go func() {
		for t := range c.ticker.C {
			_ = t
			//			c.mu.Lock()
			//			c.mu.Unlock()
			//I can make it a clojure to handle only handlers that are registed before .start() but we are ok with this, it is not map no need to Lock, for now.
			for i := range c.tickHandlers {
				c.tickHandlers[i]()
			}
		}
	}()

	c.started = true
}

// Stop stops the ticker
func (c *Ticker) Stop() {
	if c.started {
		c.ticker.Stop()
		c.started = false
	}
}

// ITick is the interface which all ticker's listeners must implement
type ITick interface {
	OnTick()
}
