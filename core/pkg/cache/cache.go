package cache

import "context"

// Cache is an interface for caching information.
type Cache[T any] interface {
	Get(ctx context.Context, key string) (*T, bool, error)
	Set(ctx context.Context, key string, info *T) error
	Delete(ctx context.Context, key string) error
}

// CacheIterator is an interface for iterating over a cache.
type CacheIterator[T any] interface {
	Cache[T]

	Walk(ctx context.Context, fn func(key string, value *T) error) error
	First(ctx context.Context, fn func(value *T) bool) (string, *T, bool, error)
}
