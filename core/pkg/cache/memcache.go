package cache

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"
)

// cacheItem represents a cached item with its value and creation time.
type cacheItem[T any] struct {
	value     *T
	createdAt time.Time
}

// memoryCache is an in-memory cache implementation for information.
type memoryCache[T any] struct {
	items         map[string]*cacheItem[T]
	mu            sync.RWMutex
	maxCount      int
	itemsToRemove int
}

// NewMemoryCache creates a new MemoryCache with the specified maximum count.
// If maxCount is zero or negative, a default value is calculated based on the available memory.
func NewMemoryCache[T any](maxCount int) CacheIterator[T] {
	if maxCount <= 0 {
		maxCount = calculateDefaultMaxCount()
	}

	cache := &memoryCache[T]{
		items:         make(map[string]*cacheItem[T]),
		maxCount:      maxCount,
		itemsToRemove: calculateItemsToRemove(maxCount),
	}

	return cache
}

// Get retrieves the information for the given key address from the cache.
func (c *memoryCache[T]) Get(ctx context.Context, key string) (*T, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false, nil
	}

	return item.value, true, nil
}

// Set stores the information for the given key address in the cache.
func (c *memoryCache[T]) Set(ctx context.Context, key string, info *T) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxCount {
		c.cleanup()
	}

	c.items[key] = &cacheItem[T]{
		value:     info,
		createdAt: time.Now(),
	}

	return nil
}

// Delete removes the information for the given key address from the cache.
func (c *memoryCache[T]) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}

// ErrWalkStop is used to stop walking over the cache items.
var ErrWalkStop = errors.New("stop walking")

// Walk iterates over all items in the cache and calls the given function for each item.
// If the function returns an error, the iteration stops.
// If the error is ErrWalkStop, the iteration stops without an error.
func (c *memoryCache[T]) Walk(_ context.Context, fn func(key string, value *T) error) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for key, item := range c.items {
		if err := fn(key, item.value); err != nil {
			if errors.Is(err, ErrWalkStop) {
				return nil
			}
			return err
		}
	}

	return nil
}

// ErrNotFound is returned when an item is not found in the cache.
// var ErrNotFound = errors.New("not found")

// First finds the first item in the cache that satisfies the given function.
func (c *memoryCache[T]) First(_ context.Context, fn func(value *T) bool) (string, *T, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for key, item := range c.items {
		if fn(item.value) {
			return key, item.value, true, nil
		}
	}

	return "", nil, false, nil
}

// cleanup removes the oldest items if the cache exceeds the maximum count.
func (c *memoryCache[T]) cleanup() {
	if len(c.items) > c.maxCount {
		// Ensure we do not remove more items than are necessary to get below maxCount.
		// We need this functionality to avoid hitting the limit on every future write
		// when the maxCount is reached. For example, if the maxCount is 100 and we
		// always have 100 entries, any future writes would trigger the cleanup method.
		// By calculating and removing more items in one go, we ensure that the cache
		// stays below the maxCount for a longer period, improving performance and
		// reducing the frequency of cleanup operations.
		itemsToRemove := min(c.itemsToRemove, len(c.items)-c.maxCount)

		// Remove the oldest items
		items := make([]string, 0, len(c.items))
		for item := range c.items {
			items = append(items, item)
		}

		// Sort items by creation time.
		slices.SortFunc(items, func(a, b string) int {
			return c.items[a].createdAt.Compare(c.items[b].createdAt)
		})

		// Remove the oldest items.
		for i := 0; i < itemsToRemove && len(c.items) > c.maxCount; i++ {
			delete(c.items, items[i])
		}
	}
}

// calculateDefaultMaxCount calculates the default maxCount based on the available memory.
func calculateDefaultMaxCount() int {
	totalMemory, err := getTotalMemory()
	if err != nil {
		// Fallback to a reasonable default if we can't get the total memory.
		return 100000
	}

	// Allocate a portion of the available memory for caching, e.g. 512MB.
	allocatedMemory := uint64(512 * 1024 * 1024) // 512MB.
	if allocatedMemory > totalMemory {
		allocatedMemory = totalMemory / 16 // Use 1/16th of the total memory if 512MB is too much.
	}

	// Assume each cached item takes approximately 1KB of memory.
	memoryPerItem := uint64(1024) // 1KB.

	return int(allocatedMemory / memoryPerItem)
}

// calculateItemsToRemove calculates the number of items to remove based on the maxCount and available memory.
func calculateItemsToRemove(maxCount int) int {
	totalMemory, err := getTotalMemory()
	if err != nil {
		// Fallback to a reasonable default if we can't get the total memory.
		totalMemory = 512 * 1024 * 1024 // 512MB.
	}

	// Allocate a portion of the available memory for caching, e.g., 512MB.
	allocatedMemory := uint64(512 * 1024 * 1024) // 512MB.
	if allocatedMemory > totalMemory {
		allocatedMemory = totalMemory / 16 // Use 1/16th of the total memory if 512MB is too much.
	}

	// Assume each cached item takes approximately 1KB of memory.
	memoryPerItem := uint64(1024) // 1KB.

	// Calculate the number of items to remove based on memory limit.
	itemsToRemove := int(allocatedMemory / memoryPerItem)

	// Ensure we do not remove more items than necessary to get below maxCount.
	// Ensure itemsToRemove does not exceed 10% of maxCount.
	maxItemsToRemove := maxCount / 10
	if itemsToRemove > maxItemsToRemove {
		itemsToRemove = maxItemsToRemove
	}

	if itemsToRemove < 1 {
		itemsToRemove = 1
	}

	return itemsToRemove
}
