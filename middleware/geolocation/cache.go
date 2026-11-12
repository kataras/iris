package geolocation

import "github.com/kataras/iris/v14/core/pkg/cache"

// NewMemoryCache creates a new MemoryCache with the specified maximum count.
// if maxCount is zero or negative, a default value is calculated based on the available memory.
func NewMemoryCache(maxCount int) cache.Cache[Info] {
	return cache.NewMemoryCache[Info](maxCount)
}
