package iris

import (
	"errors"
	"sync"
	"time"
)

/* My Greek notes for nearly future features
//edw 9a ew ta options gia to cache
//auta ta options 9a einai genika
//meta 9a kanw akoma ena func i struct
//pou 9a einai to memory cache giati isws
//sto mellon valw kai kati san to redis
//h na borei o xristis mesw nosqlh otidipote
//to default tha einai to memory cache gia to router
//episis sto router na suenxisw tin fash me to interface
//kai na exw 2 structs me
//GenericRouter i SimpleRouter
// kai CachedRouter
//h efoson to cache einai mono sto find
//isws kanw sto find kati na to kanw type
// kai na tou valw functions ? kai auto ginete
//opws ekana sto iris Handler 9a dw.

//telika apofasisa oti to cache 9a exei timer
// sto AddItem dn 9a xreiazezete na vlepoume to len kai na kanoume reset
//to reset 9a ginete mono sto timer me vasi twn maxitems an einai panw apo 0 aliws ola clear.*/

type CacheOptions struct {
	// MaxItems max number of total cached routes, 500 = +~20000 bytes = ~0.019073MB
	// Every time the cache timer reach this number it will reset/clean itself
	// Default is 0
	// If <=0 then cache cleans all of items (bag)
	// Auto cache clean is happening after 5 minutes the last request serve, you can change this number by 'ResetDuration' property
	// Carefuly MaxItems doesn't means that the items never reach this lengh, only on timer tick this number is checked/consider.
	MaxItems int

	// ResetDuration change this time.value to determinate how much duration after last request serving the cache must be reseted/cleaned
	// Default is 5 * time.Minute , Minimum is 30 seconds
	//
	// If MaxItems <= 0 then it clears the whole cache bag at this duration.
	ResetDuration time.Duration

	// Every tick from ticker from ResetDuration
	// the cache creates a temp items list from cache
	// and is checking if this is the same as it was before
	// the ResetDuration time, if yes then does nothing
	// if is larger then it makes the reset/clean.
	// This operation/algorithm is handled by each instance byself, which implements the cache.
}

func DefaultCacheOptions() CacheOptions {
	return CacheOptions{0, 5 * time.Minute}
}

// CacheTimer
//
//
type CacheTimer struct {
	duration     time.Duration
	ticker       *time.Ticker
	started      bool
	tickHandlers []func()
}

func NewCacheTimer() *CacheTimer {
	return &CacheTimer{tickHandlers: make([]func(), 0), started: false}
}

// OnTick add event handlers/ callbacks which are called on each timer's tick
func (c *CacheTimer) OnTick(h func()) {
	c.tickHandlers = append(c.tickHandlers, h)
}

func (c *CacheTimer) Start() {
	if c.started {
		return
	}

	if c.ticker != nil {
		panic("Iris CacheTimer: Cannot re-start a cache timer, if you stop it, it is not recommented to resume it,\n Just create a new CacheTimer.")
	}

	if c.duration.Seconds() < 30 {
		c.duration = 5 * time.Minute
	}

	c.ticker = time.NewTicker(c.duration)

	go func() {
		for t := range c.ticker.C {
			_ = t
			//			c.mu.Lock()
			//			c.mu.Unlock()
			//I can make it a clojure to handle only handlers that are registed before .start() but we are ok with this, it is not map no need to Lock, for now.
			for i := 0; i < len(c.tickHandlers); i++ {
				c.tickHandlers[i]()
			}
		}
	}()

	c.started = true
}

func (c *CacheTimer) Stop() {
	if c.started {
		c.ticker.Stop()
		c.started = false
	}
}

//
//

type cacheState uint8

const (
	// Init is the default value of the state inside 'Cache' means enabled (by default) without manualy call .Enable.
	// We need this in TryStart, we need to call .Enable because only then the 'Cache' is setted it's map/bag/cache.
	Init cacheState = iota
	Disabled
	Enabled
)

//
// MemoryRouterCache creation done with just &MemoryRouterCache{}
type MemoryRouterCache struct {
	options CacheOptions
	timer   *CacheTimer
	//1. map[string] ,key is HTTP Method(GET,POST...)
	//2. map[string]*Route ,key is The Request URL Path
	items map[string]map[string]*Route
	mu    *sync.Mutex
	state cacheState // we want enable by default so 0 by default (logicaly)
}

func (mc *MemoryRouterCache) SetOptions(_options CacheOptions) {
	mc.options = _options
}

// AddItem adds an item to the bag/cache, is a goroutine.
func (mc *MemoryRouterCache) AddItem(method, url string, route *Route) {
	//don't check for timer or nil items, just panic if something goes whrong.
	go func(method, url string, route *Route) { //for safety on multiple fast calls
		mc.mu.Lock()
		mc.items[method][url] = route
		mc.mu.Unlock()
	}(method, url, route)
}

// GetItem returns an item from the bag/cache, if not exists it returns just nil.
func (mc *MemoryRouterCache) GetItem(method, url string) *Route {
	//Don't check for anything else, make it as fast as it can be.
	mc.mu.Lock()
	if v := mc.items[method][url]; v != nil {
		mc.mu.Unlock()
		return v
	}
	mc.mu.Unlock()
	return nil
}

func (mc *MemoryRouterCache) onTick() {

	mc.mu.Lock()
	if mc.options.MaxItems == 0 {
		//just reset to complete new maps all methods
		mc.resetBag()
	} else {
		//loop each method on bag and clear it if it's len is more than MaxItems
		for k, v := range mc.items {
			if len(v) >= mc.options.MaxItems {
				//we just create a new map, no delete each manualy because this number maybe be very long.
				mc.items[k] = make(map[string]*Route, 0)
			}
		}
	}

	mc.mu.Unlock()
}

func (mc *MemoryRouterCache) resetBag() {
	for _, m := range HTTPMethods.ANY {
		mc.items[m] = make(map[string]*Route, 0)
	}
}

// Enable enables the memory cache but it doesn't starts the timer.
// Use the Start() to start the timer when you 're ready.
func (mc *MemoryRouterCache) Enable() {
	if mc.state != Enabled { // Init or Disabled
		if mc.options.ResetDuration == 0 { // == nil if we had a pointer, just checks if option reset duration has setted.
			mc.options = DefaultCacheOptions()
		}

		if mc.timer == nil {
			mc.timer = NewCacheTimer()
			mc.mu = &sync.Mutex{}
			mc.timer.OnTick(mc.onTick)
		}

		if mc.items == nil {
			mc.items = make(map[string]map[string]*Route, 0)
			mc.resetBag()
		}
		mc.state = Enabled

	}

}

// Disable disables the memory cache and stops the timer if started.
func (mc *MemoryRouterCache) Disable() {
	if mc.state == Enabled && mc.timer != nil {
		if mc.timer.started {
			mc.timer.Stop()
		}
		mc.items = nil
		//do no nil it we need to check for default enable on the TryStart mc.timer = nil
	}

	mc.state = Disabled

}

// Start starts the actual work of the memory router cache, it starts the timer.
func (mc *MemoryRouterCache) Start() {
	if err := mc.TryStart(); err != nil {
		panic(err.Error())
	}
}

// TryStart starts the actual work of the memory router cache, if it is enabled, if it is not enabled
// then just returns an error without panic, useful to use inside the Router.HandleFunc to start
func (mc *MemoryRouterCache) TryStart() error {

	if mc.state == Disabled { // if it's enabled which means that the .Enable called before but for some reason the timer is nil?
		return errors.New("Iris MemoryRouterCache: Timer is nil, please call .Enable() first and don't change any other values.")
	} else if mc.state == Init {
		// means that we didn't call the .Enable manualy but before TryStart but the memory cache is enabled ( by default) so enable and run it.
		//this is called by default (logicaly) only the first time of the TryStart
		mc.Enable()
	}

	if !mc.timer.started {
		mc.timer.Start()
	}

	return nil
}
