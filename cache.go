package iris

import (
	"sync"
	"time"
)

type (
	// CacheService is the cache service which caches the whole response body
	// it's an interface because you can even set your own cache service inside framework!
	CacheService interface {
		// Start is the method which the CacheService starts the GC(check if expiration of each entry is passed , if yes then delete it from cache)
		Start(time.Duration)
		// Cache accepts a route's handler which will cache its response and a time.Duration(int64) which is the expiration duration
		Cache(HandlerFunc, time.Duration) HandlerFunc
		// Invalidate accepts a cache key (which can be retrieved by 'GetCacheKey') and remove its cache response body
		InvalidateCache(string)
	}

	cacheService struct {
		cache map[string]*cacheEntry
		mu    sync.RWMutex
		// keep track of the minimum cache duration of all cache entries, this will be used when gcDuration inside .start() is <=time.Second
		lowerExpiration time.Duration
	}

	cacheEntry struct {
		statusCode  int
		contentType string
		value       []byte
		expires     time.Time
	}
)

var _ CacheService = &cacheService{}

func newCacheService() *cacheService {
	cs := &cacheService{
		cache:           make(map[string]*cacheEntry),
		mu:              sync.RWMutex{},
		lowerExpiration: time.Second,
	}

	return cs
}

// Start is not called via newCacheService because
// if gcDuration is  <=time.Second
// then start should check and set the gcDuration from the TOTAL CACHE ENTRIES lowest expiration duration
func (cs *cacheService) Start(gcDuration time.Duration) {

	if gcDuration <= minimumAllowedCacheDuration {
		gcDuration = cs.lowerExpiration
	}

	// start the timer to check for expirated cache entries
	tick := time.Tick(gcDuration)
	go func() {
		for range tick {
			cs.mu.Lock()
			now := time.Now()
			for k, v := range cs.cache {
				if now.After(v.expires) {
					delete(cs.cache, k)
				}
			}
			cs.mu.Unlock()
		}
	}()

}

func (cs *cacheService) get(key string) *cacheEntry {
	cs.mu.RLock()
	if v, ok := cs.cache[key]; ok {
		cs.mu.RUnlock()
		return v
	}
	cs.mu.RUnlock()
	return nil
}

// we don't set it to zero value, just 2050 year is enough xD
var expiresNever = time.Date(2050, time.January, 10, 23, 0, 0, 0, time.UTC)
var minimumAllowedCacheDuration = time.Second

func (cs *cacheService) set(key string, statusCode int, contentType string, value []byte, expiration time.Duration) {
	if statusCode == 0 {
		statusCode = StatusOK
	}
	if contentType == "" {
		contentType = contentText
	}

	entry := &cacheEntry{contentType: contentType, statusCode: statusCode, value: value}

	if expiration <= minimumAllowedCacheDuration {
		// Cache function tries to set the expiration(seconds) from header "cache-control" if expiration <=minimumAllowedCacheDuration
		// but if cache-control is missing then set it to 5 minutes
		expiration = 5 * time.Minute
	}

	entry.expires = time.Now().Add(expiration)

	cs.mu.Lock()
	cs.cache[key] = entry
	cs.mu.Unlock()
}

func (cs *cacheService) remove(key string) {
	cs.mu.Lock()
	delete(cs.cache, key)
	cs.mu.Unlock()
}

// GetCacheKey returns the cache key(string) from a context
// it's just the RequestURI
func GetCacheKey(ctx *Context) string {
	return string(ctx.Request.URI().RequestURI())
}

// InvalidateCache clears the cache body for a specific key(request uri, can be retrieved by GetCacheKey(ctx))
//
// Note that it depends on a station instance's cache service.
// Do not try to call it from default' station if you use the form of app := iris.New(),
// use the app.InvalidateCache instead of iris.InvalidateCache
//
// Example: https://github.com/iris-contrib/examples/tree/master/cache_body
func InvalidateCache(key string) {
	Default.CacheService.InvalidateCache(key)
}

// InvalidateCache clears the cache body for a specific key(request uri, can be retrieved by GetCacheKey(ctx))
//
// Note that it depends on a station instance's cache service.
// Do not try to call it from default' station if you use the form of app := iris.New(),
// use the app.Cache instead of iris.Cache
//
// Example: https://github.com/iris-contrib/examples/tree/master/cache_body
func (cs *cacheService) InvalidateCache(key string) {
	cs.remove(key)
}

// Cache is just a wrapper for a route's handler which you want to enable body caching
// Usage: iris.Get("/", iris.Cache(func(ctx *iris.Context){
//    ctx.WriteString("Hello, world!") // or a template or anything else
// }, time.Duration(10*time.Second))) // duration of expiration
// if <=time.Second then it tries to find it though request header's "cache-control" maxage value
//
// Note that it depends on a station instance's cache service.
// Do not try to call it from default' station if you use the form of app := iris.New(),
// use the app.Cache instead of iris.Cache
//
// Example: https://github.com/iris-contrib/examples/tree/master/cache_body
func Cache(bodyHandler HandlerFunc, expiration time.Duration) HandlerFunc {
	return Default.CacheService.Cache(bodyHandler, expiration)
}

// Cache is just a wrapper for a route's handler which you want to enable body caching
// Usage: iris.Get("/", iris.Cache(func(ctx *iris.Context){
//    ctx.WriteString("Hello, world!") // or a template or anything else
// }, time.Duration(10*time.Second))) // duration of expiration
// if <=time.Second then it tries to find it though request header's "cache-control" maxage value
//
// Note that it depends on a station instance's cache service.
// Do not try to call it from default' station if you use the form of app := iris.New(),
// use the app.Cache instead of iris.Cache
//
// Example: https://github.com/iris-contrib/examples/tree/master/cache_body
func (cs *cacheService) Cache(bodyHandler HandlerFunc, expiration time.Duration) HandlerFunc {

	// the first time the  lowerExpiration should be > time.Second, so:
	if cs.lowerExpiration == time.Second {
		cs.lowerExpiration = expiration
	} else if expiration > time.Second && expiration < cs.lowerExpiration {
		cs.lowerExpiration = expiration
	}

	h := func(ctx *Context) {
		key := GetCacheKey(ctx)
		if v := cs.get(key); v != nil {
			ctx.SetContentType(v.contentType)
			ctx.SetStatusCode(v.statusCode)
			ctx.RequestCtx.Write(v.value)
			return
		}

		// if not found then serve this:
		bodyHandler.Serve(ctx)
		if expiration <= minimumAllowedCacheDuration {
			// try to set the expiraion from header
			expiration = time.Duration(ctx.MaxAge()) * time.Second
		}

		cType := string(ctx.Response.Header.Peek(contentType))
		statusCode := ctx.RequestCtx.Response.StatusCode()
		// and set the cache value as its response body in a goroutine, because we want to exit from the route's handler as soon as possible
		go cs.set(key, statusCode, cType, ctx.Response.Body(), expiration)
	}

	return h
}
