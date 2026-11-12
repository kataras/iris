// Package httpcache provides a middleware for caching HTTP responses in
// DragonflyDB or Redis.
// It allows caching of GET and POST requests to improve performance by reducing the need
// for repeated processing of the same requests.
//
// Its entry point is New(Options) (*Cache, error), which reports a failed
// connection to the store; `Must` is the panicking companion. It returns a
// *Cache rather than a Handler because the store is also used directly, to
// invalidate entries; register its `Handler` method.
package httpcache
