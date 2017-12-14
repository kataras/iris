package uri

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/geekypanda/httpcache/cfg"
)

// URIBuilder is the requested url builder
// which keeps all the information the server
// should know to save or retrieve a cached entry's response
// used on client-side only
// both from net/http and valyala/fasthttp
type URIBuilder struct {
	serverAddr,
	clientMethod,
	clientURI string

	cacheLifetime    time.Duration
	cacheStatuscode  int
	cacheContentType string
}

// ServerAddr sets the server address for the final request url
func (r *URIBuilder) ServerAddr(s string) *URIBuilder {
	r.serverAddr = s
	return r
}

// ClientMethod sets the method which the client should call the remote server's handler
// used to build the final request url too
func (r *URIBuilder) ClientMethod(s string) *URIBuilder {
	r.clientMethod = s
	return r
}

// ClientURI sets the client path for the final request url
func (r *URIBuilder) ClientURI(s string) *URIBuilder {
	r.clientURI = s
	return r
}

// Lifetime sets the cache lifetime for the final request url
func (r *URIBuilder) Lifetime(d time.Duration) *URIBuilder {
	r.cacheLifetime = d
	return r
}

// StatusCode sets the cache status code for the final request url
func (r *URIBuilder) StatusCode(code int) *URIBuilder {
	r.cacheStatuscode = code
	return r
}

// ContentType sets the cache content type for the final request url
func (r *URIBuilder) ContentType(s string) *URIBuilder {
	r.cacheContentType = s
	return r
}

// String returns the full url which should be passed to get a cache entry response back
// (it could be setted by server too but we need some client-freedom on the requested key)
// in order to be sure that the registered cache entries are unique among different clients with the same key
// note1: we do it manually*,
// note2: on fasthttp that is not required because the query args added as expected but we will use it there too to be align with net/http
func (r URIBuilder) String() string {
	return r.build()
}

func (r URIBuilder) build() string {

	remoteURL := r.serverAddr

	// fasthttp appends the "/" in the last uri (with query args also, that's probably a fasthttp bug which I'll fix later)
	// for now lets make that check:

	if !strings.HasSuffix(remoteURL, "/") {
		remoteURL += "/"
	}
	scheme := "http://"
	// validate the remoteURL, should contains a scheme, if not then check if the client has given a scheme and if so
	// use that for the server too
	if !strings.Contains(remoteURL, "://") {
		if strings.Contains(remoteURL, ":443") || strings.Contains(remoteURL, ":https") {
			remoteURL = "https://" + remoteURL
		} else {
			remoteURL = scheme + "://" + remoteURL
		}
	}
	var cacheDurationStr, statusCodeStr string

	if r.cacheLifetime.Seconds() > 0 {
		cacheDurationStr = strconv.Itoa(int(r.cacheLifetime.Seconds()))
	}

	if r.cacheStatuscode > 0 {
		statusCodeStr = strconv.Itoa(r.cacheStatuscode)
	}

	s := remoteURL + "?" + cfg.QueryCacheKey + "=" + url.QueryEscape(r.clientMethod+scheme+r.clientURI)
	if cacheDurationStr != "" {
		s += "&" + cfg.QueryCacheDuration + "=" + url.QueryEscape(cacheDurationStr)
	}
	if statusCodeStr != "" {
		s += "&" + cfg.QueryCacheStatusCode + "=" + url.QueryEscape(statusCodeStr)
	}
	if r.cacheContentType != "" {
		s += "&" + cfg.QueryCacheContentType + "=" + url.QueryEscape(r.cacheContentType)
	}
	return s
}
