package cache

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CacheControlBuilder helps to build the Cache-Control header value.
type CacheControlBuilder struct {
	directives []string
}

// NewCacheControlBuilder creates a new CacheControlBuilder.
func NewCacheControlBuilder() *CacheControlBuilder {
	return &CacheControlBuilder{}
}

// Public sets the Cache-Control directive to 'public'.
// Public indicates that the response may be cached by any cache.
func (b *CacheControlBuilder) Public() *CacheControlBuilder {
	b.directives = append(b.directives, "public")
	return b
}

// Private sets the Cache-Control directive to 'private'.
// Private indicates that the response is intended for a single user and must not be cached by shared caches.
func (b *CacheControlBuilder) Private() *CacheControlBuilder {
	b.directives = append(b.directives, "private")
	return b
}

// NoCache sets the Cache-Control directive to 'no-cache'.
// NoCache indicates that the response must not be used from the cache without revalidation with the origin server.
func (b *CacheControlBuilder) NoCache() *CacheControlBuilder {
	b.directives = append(b.directives, "no-cache")
	return b
}

// NoStore sets the Cache-Control directive to 'no-store'.
// NoStore indicates that the response must not be stored in any cache.
func (b *CacheControlBuilder) NoStore() *CacheControlBuilder {
	b.directives = append(b.directives, "no-store")
	return b
}

// MaxAge sets the Cache-Control directive to 'max-age=<seconds>'.
// MaxAge specifies the maximum amount of time a resource is considered fresh.
// The duration is written in whole seconds (a fraction of a second is dropped),
// for example MaxAge(time.Hour) writes "max-age=3600". Zero writes "max-age=0".
//
// It panics if d is negative, or above zero but below one second: a bare
// number such as MaxAge(3600) is 3600 nanoseconds, not an hour.
func (b *CacheControlBuilder) MaxAge(d time.Duration) *CacheControlBuilder {
	b.directives = append(b.directives, "max-age="+cacheControlSeconds("MaxAge", d))
	return b
}

// SMaxAge sets the Cache-Control directive to 's-maxage=<seconds>'.
// SMaxAge specifies the maximum amount of time a resource is considered fresh in shared caches.
// The duration is written in whole seconds, the same way as MaxAge,
// and SMaxAge panics on the same values.
func (b *CacheControlBuilder) SMaxAge(d time.Duration) *CacheControlBuilder {
	b.directives = append(b.directives, "s-maxage="+cacheControlSeconds("SMaxAge", d))
	return b
}

// cacheControlSeconds renders d as whole seconds for a Cache-Control directive.
// method is the builder method name, used in the panic message.
func cacheControlSeconds(method string, d time.Duration) string {
	if d < 0 {
		panic(fmt.Sprintf("cache: %s(%s) is negative: pass zero or a positive time.Duration, for example %s(time.Hour)", method, d, method))
	}
	if d > 0 && d < time.Second {
		panic(fmt.Sprintf("cache: %s(%s) is below one second: pass a time.Duration, for example %s(time.Hour)", method, d, method))
	}
	return strconv.FormatInt(int64(d/time.Second), 10)
}

// MustRevalidate sets the Cache-Control directive to 'must-revalidate'.
// MustRevalidate indicates that once a resource becomes stale, it must be validated with the origin server before being used.
func (b *CacheControlBuilder) MustRevalidate() *CacheControlBuilder {
	b.directives = append(b.directives, "must-revalidate")
	return b
}

// ProxyRevalidate sets the Cache-Control directive to 'proxy-revalidate'.
// ProxyRevalidate indicates that shared caches (e.g., proxies) must revalidate the response with the origin server once it becomes stale.
func (b *CacheControlBuilder) ProxyRevalidate() *CacheControlBuilder {
	b.directives = append(b.directives, "proxy-revalidate")
	return b
}

// Build constructs the Cache-Control header value from the added directives.
func (b *CacheControlBuilder) Build() string {
	return strings.Join(b.directives, ", ")
}
