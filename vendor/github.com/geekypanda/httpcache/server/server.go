/*
Remote cache handler runs on different machine
than the cache consumer.

Implementation by simple handler, not a http server,
let server customizations to the end-user.

Methods & actions:

  GET: Retrieve the cached status code, content type, body
  POST: Save a cache entry with its status content, content type
    and body, to the cache key-value store
  DELETE: Remove/Invalidate a cache entry based on its key


A remote entry should have a unique key.
May used across different clients at the same time.

key = the full http url(scheme+host+query args/ all url encoded),

key always sent by the consumer to the producer(client to server).

Remote is based only on net/http,
because it doesn't matters if the client is based on fasthttp or net/http.
It will work the same exactly
because remote service cache is based on http requests coming from the client side(the consumer).
*/

package server

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/geekypanda/httpcache/cfg"
	"github.com/geekypanda/httpcache/nethttp"
)

func getURLParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

func getURLParamInt(r *http.Request, key string) (int, error) {
	return strconv.Atoi(getURLParam(r, key))
}

func getURLParamInt64(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(getURLParam(r, key), 10, 64)
}

const (
	methodGet    = "GET"
	methodPost   = "POST"
	methodDelete = "DELETE"
)

// Handler is the remote cache service's Handler
// (a simple http server using this handler is the server side )
// keeps the cache entries, this time based on keys
// these keys are the request's full encoded url
// more than one clients(other http servers) can call(via http rest api)
// this handler.
//
// one handler per cache service
// yes, you're able to have more than one cache service
// in the same http server
type Handler struct {
	store Store
}

// ServeHTTP serves the cache Service to the outside world,
// it is used only when you want to achieve something like horizontal scaling
// it parses the request and tries to return the response with the cached body of the requested cache key
// server-side function
func (s *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// println("Request to the remote service has been established")
	key := getURLParam(r, cfg.QueryCacheKey)
	if key == "" {
		// println("return because key was empty")
		w.WriteHeader(cfg.FailStatus)
		return
	}

	// we always need the Entry, so get it now
	entry := s.store.Get(key)

	if entry == nil && r.Method != methodPost {
		// if it's nil then means it never setted before
		// it doesn't exists, and client doesn't wants to
		// add a cache entry, so just return
		//
		// no delete action is valid
		// no get action is valid
		// no post action is requested
		w.WriteHeader(cfg.FailStatus)
		return
	}

	switch r.Method {
	case methodGet:
		{
			// get from the cache and send to client
			res, ok := entry.Response()
			if !ok {
				// entry exists but it has been expired
				// return
				w.WriteHeader(cfg.FailStatus)
				return
			}

			// entry exists and response is valid
			// send it to the client
			w.Header().Set(cfg.ContentTypeHeader, res.ContentType())
			w.WriteHeader(res.StatusCode())
			w.Write(res.Body())
		}
	case methodPost:
		{
			// save a new cache entry if entry ==nil or
			// update an existing if entry !=nil

			body, err := ioutil.ReadAll(r.Body)
			if err != nil || len(body) == 0 {
				// println("body's request was empty, return fail")
				w.WriteHeader(cfg.FailStatus)
				return
			}

			statusCode, _ := getURLParamInt(r, cfg.QueryCacheStatusCode)
			contentType := getURLParam(r, cfg.QueryCacheContentType)

			// now that we have the information
			// we want to see if this is a totally new cache entry
			// or just update an existing one with the new information
			// (an update can change the status code, content type
			//     and ofcourse the body and expiration time by header)

			if entry == nil {
				// get the information by its url
				// println("we have a post request method, let's save a cached entry ")
				// get the cache expiration via url param
				expirationSeconds, err := getURLParamInt64(r, cfg.QueryCacheDuration)
				// get the body from the requested body
				// get the expiration from the "cache-control's maxage" if no url param is setted
				if expirationSeconds <= 0 || err != nil {
					expirationSeconds = int64(nethttp.GetMaxAge(r)().Seconds())
				}
				// if not setted then try to get it via
				if expirationSeconds <= 0 {
					expirationSeconds = int64(cfg.MinimumCacheDuration.Seconds())
				}

				cacheDuration := time.Duration(expirationSeconds) * time.Second

				// store by its url+the key in order to be unique key among different servers with the same paths
				s.store.Set(key, statusCode, contentType, body, cacheDuration)
			} else {
				// update an existing one and change its duration  based on the header
				// (if > existing duration)
				entry.Reset(statusCode, contentType, body, nethttp.GetMaxAge(r))
			}

			w.WriteHeader(cfg.SuccessStatus)
		}
	case methodDelete:
		{
			// remove the entry entirely from the cache
			// manually DELETE cache should remove this entirely
			// no just invalidate it
			s.store.Remove(key)
			w.WriteHeader(cfg.SuccessStatus)
		}
	default:
		w.WriteHeader(cfg.FailStatus)
	}

}

// the actual work is done on the handler.go
// here we just provide a helper for the main package to create
// an http.Server and serve a cache remote service, without any user touches

// New returns a http.Server which hosts
// the server-side handler for the remote cache service.
//
// it doesn't listens to the server
func New(addr string, store Store) *http.Server {
	if store == nil {
		store = NewMemoryStore()
	}
	h := &Handler{store: store}
	return &http.Server{
		Addr:    addr,
		Handler: h,
	}
}
