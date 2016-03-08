package iris

import (
	"sync"
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
//
//type CacheOptions struct {
//	// MaxItems max number of total cached routes, 500 = +~20000 bytes = ~0.019073MB
//	// Every time the cache timer reach this number it will reset/clean itself
//	// Default is 0
//	// If <=0 then cache cleans all of items (bag)
//	// Auto cache clean is happening after 5 minutes the last request serve, you can change this number by 'ResetDuration' property
//	// Carefuly MaxItems doesn't means that the items never reach this lengh, only on timer tick this number is checked/consider.
//	MaxItems int
//
//	// ResetDuration change this time.value to determinate how much duration after last request serving the cache must be reseted/cleaned
//	// Default is 5 * time.Minute , Minimum is 30 seconds
//	//
//	// If MaxItems <= 0 then it clears the whole cache bag at this duration.
//	ResetDuration time.Duration
//
//	// Every tick from ticker from ResetDuration
//	// the cache creates a temp items list from cache
//	// and is checking if this is the same as it was before
//	// the ResetDuration time, if yes then does nothing
//	// if is larger then it makes the reset/clean.
//	// This operation/algorithm is handled by each instance byself, which implements the cache.
//}

//func DefaultCacheOptions() CacheOptions {
//	return CacheOptions{0, 5 * time.Minute}
//}

type IRouterCache interface {
	OnTick()
	AddItem(method, url string, route *Route)
	GetItem(method, url string) *Route
	SetMaxItems(maxItems int)
}

//
// MemoryRouterCache creation done with just &MemoryRouterCache{}
type MemoryRouterCache struct {
	//1. map[string] ,key is HTTP Method(GET,POST...)
	//2. map[string]*Route ,key is The Request URL Path
	items    map[string]map[string]*Route
	MaxItems int
	mu       *sync.Mutex
}

func (mc *MemoryRouterCache) SetMaxItems(_itemslen int) {
	mc.MaxItems = _itemslen
}

func NewMemoryRouterCache() *MemoryRouterCache {
	mc := &MemoryRouterCache{mu: &sync.Mutex{}, items: make(map[string]map[string]*Route, 0)}
	mc.resetBag()
	return mc
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

func (mc *MemoryRouterCache) OnTick() {

	mc.mu.Lock()
	if mc.MaxItems == 0 {
		//just reset to complete new maps all methods
		mc.resetBag()
	} else {
		//loop each method on bag and clear it if it's len is more than MaxItems
		for k, v := range mc.items {
			if len(v) >= mc.MaxItems {
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
