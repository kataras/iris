/* ticker.go: after version 1, we don't need this atm, but we keep it. */

package utils

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
