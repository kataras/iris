package iris

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type (
	// CacheService is the cache service which caches the whole response body
	CacheService interface {
		// Start is the method which the CacheService starts the GC(check if expiration of each entry is passed , if yes then delete it from cache)
		Start(time.Duration)
		// Cache accepts a route's handler which will cache its response and a time.Duration(int64) which is the expiration duration
		Cache(HandlerFunc, time.Duration) HandlerFunc
		// ServeRemoteCache creates & returns a new handler which saves cache by POST method and serves a cache entry by GET method to clients
		// usually set it with iris.Any,
		// but developer is able to set different paths for save or get cache entries: using the iris.Post/.Get(...,iris.ServeRemote())
		//
		// IT IS NOT READY FOR PRODUCTION YET, READ THE HISTORY.md for the available working cache methods
		ServeRemoteCache() HandlerFunc
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
				if now.Before(v.expires) {
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

// GetRemoteCacheKey returns the context's cache key,
// differs from GetCacheKey is that this method parses the query arguments
// because this key must be sent to an external server
func GetRemoteCacheKey(ctx *Context) string {
	return url.QueryEscape(ctx.Request.URI().String())
}

// RemoteCache accepts the remote server address and path of the external cache service, the body handler and optional an expiration
// the last 2 receivers works like .Cache(...) function
//
// Note: Remotecache is a global function, usage:
// app.Get("/", iris.RemoteCache("http://127.0.0.1:8888/cache", bodyHandler, time.Duration(15)*time.Second))
//
// IT IS NOT READY FOR PRODUCTION YET, READ THE HISTORY.md for the available working cache methods
func RemoteCache(cacheServerAddr string, bodyHandler HandlerFunc, expiration time.Duration) HandlerFunc {
	client := fasthttp.Client{}
	//	buf := utils.NewBufferPool(10)
	cacheDurationStr := fmt.Sprintf("%f", expiration.Seconds())
	h := func(ctx *Context) {
		req := fasthttp.AcquireRequest()
		req.SetRequestURI(cacheServerAddr)
		req.Header.SetMethodBytes(MethodGetBytes)
		req.URI().QueryArgs().Add("cache_key", GetRemoteCacheKey(ctx))

		res := fasthttp.AcquireResponse()
		err := client.DoTimeout(req, res, time.Duration(5)*time.Second)
		if err != nil || res.StatusCode() == StatusBadRequest {
			// if not found on cache, then execute the handler and save the cache to the remote server
			bodyHandler.Serve(ctx)
			// save to the remote cache
			req.Header.SetMethodBytes(MethodPostBytes)

			req.URI().QueryArgs().Add("cache_duration", cacheDurationStr)
			statusCode := strconv.Itoa(ctx.Response.StatusCode())
			req.URI().QueryArgs().Add("cache_status_code", statusCode)
			cType := string(ctx.Response.Header.Peek(contentType))
			req.URI().QueryArgs().Add("cache_content_type", cType)
			postArgs := fasthttp.AcquireArgs()
			postArgs.SetBytesV("cache_body", ctx.Response.Body())

			go func() {
				client.DoTimeout(req, res, time.Duration(5)*time.Second)
				fasthttp.ReleaseArgs(postArgs)
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(res)
			}()

		} else {
			// get the status code , content type and the write the response body
			statusCode := res.StatusCode()
			cType := res.Header.ContentType()
			ctx.SetStatusCode(statusCode)
			ctx.Response.Header.SetContentTypeBytes(cType)

			ctx.RequestCtx.Write(res.Body())

			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(res)
		}

	}
	return h
}

// ServeRemoteCache usage: iris.Any("/cacheservice", iris.ServeRemote())
// client does an http request to retrieve cached body from the external/remote server which keeps the cache service.
//
// if is GET method request then gets from cache
// if it's POST method request then its saves to the cache
// if it's DELETE method request then its invalidates/removes from cache manually
// the content type and the status are setted inside the caller's handler
// this is not like cs.Cache, it's useful only when you separate your servers to achieve horizontal scaling
//
// Note that it depends on a station instance's cache service.
// Do not try to call it from default' station if you use the form of app := iris.New(),
// use the app.ServeRemoteCache instead of iris.Cache
//
// IT IS NOT READY FOR PRODUCTION YET, READ THE HISTORY.md for the available working cache methods
func ServeRemoteCache() HandlerFunc {
	return Default.CacheService.ServeRemoteCache()
}

// ServeRemoteCache usage: iris.Any("/cacheservice", iris.ServeRemoteCache())
// client does an http request to retrieve cached body from the external/remote server which keeps the cache service.
//
// if is GET method request then gets from cache
// if it's POST method request then its saves to the cache
// if it's DELETE method request then its invalidates/removes from cache manually
// the content type and the status are setted inside the caller's handler
// this is not like cs.Cache, it's useful only when you separate your servers to achieve horizontal scaling
//
// Note that it depends on a station instance's cache service.
// Do not try to call it from default' station if you use the form of app := iris.New(),
// use the app.ServeRemoteCache instead of iris.Cache
//
// IT IS NOT READY FOR PRODUCTION YET, READ THE HISTORY.md for the available working cache methods
func (cs *cacheService) ServeRemoteCache() HandlerFunc {
	h := func(ctx *Context) {
		key := ctx.URLParam("cache_key")
		if key == "" {
			ctx.SetStatusCode(StatusBadRequest)
			return
		}

		if ctx.IsGet() {
			if v := cs.get(key); v != nil {
				ctx.SetStatusCode(v.statusCode)
				ctx.SetContentType(v.contentType)
				ctx.RequestCtx.Write(v.value)
				return
			}
		} else if ctx.IsPost() {
			// get the cache expiration via url param
			expirationSeconds, err := ctx.URLParamInt64("cache_duration")
			// get the body from the post arguments or requested body
			body := ctx.PostArgs().Peek("cache_body")
			if len(body) == 0 {
				body = ctx.Request.Body()
				if len(body) == 0 {
					ctx.SetStatusCode(StatusBadRequest)
					return
				}
			}
			// get the expiration from the "cache-control's maxage" if no url param is setted
			if expirationSeconds <= 0 || err != nil {
				expirationSeconds = ctx.MaxAge()
			}

			// if not setted then try to get it via
			if expirationSeconds <= 0 {
				expirationSeconds = 5 * 60 // 5 minutes
			}

			cacheDuration := time.Duration(expirationSeconds) * time.Second
			statusCode, err := ctx.URLParamInt("cache_status_code")
			if err != nil {
				statusCode = StatusOK
			}
			cType := ctx.URLParam("cache_content_type")
			if cType == "" {
				cType = contentHTML
			}

			cs.set(key, statusCode, cType, body, cacheDuration)

			ctx.SetStatusCode(StatusOK)
			return
		} else if ctx.IsDelete() {
			cs.remove(key)
			ctx.SetStatusCode(StatusOK)
			return
		}

		ctx.SetStatusCode(StatusBadRequest)
	}

	return cs.Cache(h, -1)
}
