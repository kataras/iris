package httpcache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"maps"

	"github.com/redis/go-redis/v9"
)

// Options defines the configuration options for the DragonflyDB/Redis cache.
// It includes the address, password, TTL for cache entries, command timeout,
// and options for excluding request body and query parameters from cache key generation.
type Options struct {
	// If true, the cache is disabled, even if the fields are filled, and no operations will be performed.
	Disable bool `json:"disable" yaml:"Disable"`
	// If true, it will delete all keys in the cache on startup.
	StartupCleanup bool `json:"startup_cleanup" yaml:"StartupCleanup"`
	// DragonflyDB/Redis address.
	Addr string `json:"addr" yaml:"Addr"`
	// DragonflyDB/Redis optional password.
	Password string `json:"-" yaml:"Password"`
	// Time to live for cache entries. Defaults to 10 minutes.
	TTL time.Duration `json:"ttl" yaml:"TTL"`
	// Timeout for cache commands. Defaults to 15 seconds.
	CommandTimeout time.Duration `json:"command_timeout" yaml:"CommandTimeout"`

	// If true, do not include request body in cache key generation.
	NoBody bool `json:"no_body" yaml:"NoBody"`
	// If true, do not include query parameters in cache key generation.
	NoQuery bool `json:"no_query" yaml:"NoQuery"`
	// Headers to include in cache key generation. E.g. Authorization.
	Headers []string `json:"headers" yaml:"Headers"`
	// Cookies to include in cache key generation. E.g. session_id.
	Cookies []string `json:"cookies" yaml:"Cookies"`
}

// Cache represents a DragonflyDB/Redis cache for HTTP responses.
// It provides methods to set and get cached responses, generate cache keys,
// and invalidate cache entries based on request URLs or prefixes.
type Cache struct {
	Options *Options

	// The underlying Redis/DragonflyDB client.
	db *redis.Client

	// warnOnce keeps the "credentialed requests are not cached" warning to a
	// single log line per instance.
	warnOnce sync.Once
}

// Must is New but it panics instead of reporting the error.
func Must(options Options) *Cache {
	cache, err := New(options)
	if err != nil {
		panic(err)
	}

	return cache
}

// New creates and returns a new Cache instance using the provided options.
// Returns an error if the connection to DragonflyDB/Redis fails.
// See the `Must` companion.
func New(options Options) (*Cache, error) {
	if options.Disable {
		// Return an empty instance, so the API and any of its dependencies dont't break.
		return &Cache{Options: &options}, nil
	}

	if options.Addr == "" {
		return nil, fmt.Errorf("cache address must be provided")
	}

	if options.TTL == 0 {
		options.TTL = 10 * time.Minute // Default TTL if not set.
	}

	if options.CommandTimeout == 0 {
		options.CommandTimeout = 15 * time.Second // Default command timeout if not set.
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     options.Addr,
		Password: options.Password, // set password here
	})

	ctx, cancel := context.WithTimeout(context.Background(), options.CommandTimeout)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	if options.StartupCleanup {
		// Perform a cleanup of the cache on startup if configured.
		if err := rdb.FlushDB(ctx).Err(); err != nil {
			return nil, fmt.Errorf("failed to flush cache on startup: %w", err)
		}
	}

	// Initialize the cache with the provided options.
	// This is a placeholder for actual cache initialization logic.
	return &Cache{Options: &options, db: rdb}, nil
}

// Close closes the underlying DragonflyDB/Redis connection.
func (c *Cache) Close() error {
	if c.db == nil {
		return nil // No-op if cache is disabled.
	}

	if err := c.db.Close(); err != nil {
		return fmt.Errorf("failed to close cache: %w", err)
	}

	return nil
}

// RequestKey generates a unique cache key for the given HTTP request.
// The key is derived from the request's URL path, method, query parameters, and body.
// The key format is:
// httpcache:<url_path_only>:<method>:<query_hash>:<body_hash>[:<headers_hash>][:<cookies_hash>]
func (c *Cache) RequestKey(r *http.Request) string {
	if c.db == nil {
		return "" // No-op if cache is disabled.
	}

	urlPath := r.URL.Path // Only the path, no query params.
	key := fmt.Sprintf("httpcache:%s:%s", urlPath, r.Method)
	key = c.appendKeySuffix(key, r) // Append query, body, headers, and cookies hashes.
	return key
}

// maxKeyBodySize bounds how much of a request body is buffered to build a cache
// key. Requests larger than this are simply not keyed by body.
const maxKeyBodySize int64 = 1 << 20 // 1MB

func (c *Cache) appendKeySuffix(key string, r *http.Request) string {
	if !c.Options.NoQuery { // Hash the query parameters.
		queryHash := sha256.Sum256([]byte(r.URL.RawQuery))
		key += fmt.Sprintf(":%x", queryHash) // Append the query hash to the key.
	}

	if !c.Options.NoBody { // Hash the request body.
		// Note(@kataras): the read is BOUNDED and the error is not discarded.
		//
		// POST is in resourceGetMethods, so this ran for POST bodies too: an
		// unbounded io.ReadAll here allocated the whole body, then bytes.NewBuffer
		// allocated it a second time, before any handler had a chance to reject
		// the request. A 500MB POST cost a gigabyte, repeated concurrently it is
		// an OOM. Beyond the limit the key falls back to a marker, which simply
		// makes those requests share a key and miss - never a wrong hit, because
		// the response is only stored under the same key it was looked up with.
		b, err := io.ReadAll(io.LimitReader(r.Body, maxKeyBodySize+1))
		r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(b), r.Body))

		switch {
		case err != nil:
			key += ":body-unreadable"
		case int64(len(b)) > maxKeyBodySize:
			key += ":body-too-large"
		default:
			bodyHash := sha256.Sum256(b)
			key += fmt.Sprintf(":%x", bodyHash)
		}
	}

	if len(c.Options.Headers) > 0 {
		// Hash selected headers from options.
		var headersData []byte
		for _, h := range c.Options.Headers {
			headersData = append(headersData, []byte(h)...)
			headersData = append(headersData, []byte(":")...)
			headersData = append(headersData, []byte(r.Header.Get(h))...)
			headersData = append(headersData, []byte(";")...)
		}
		headersHash := sha256.Sum256(headersData)

		// Don't compare the hash length, as it's always a 32-byte slice.
		// if len(headersHash) > 0 {
		key += fmt.Sprintf(":%x", headersHash)
	}

	if len(c.Options.Cookies) > 0 {
		// Hash selected cookies from options.
		var cookiesData []byte
		for _, cname := range c.Options.Cookies {
			c, err := r.Cookie(cname)
			if err == nil && c.Value != "" {
				cookiesData = append(cookiesData, []byte(cname)...)
				cookiesData = append(cookiesData, []byte(":")...)
				cookiesData = append(cookiesData, []byte(c.Value)...)
				cookiesData = append(cookiesData, []byte(";")...)
			}
		}
		cookiesHash := sha256.Sum256(cookiesData)

		key += fmt.Sprintf(":%x", cookiesHash)
	}

	return key
}

// Response represents a cached HTTP response.
// It includes the response body, headers, and expiration time.
type Response struct {
	ExpiresAt time.Time `json:"expired_at"` // Expiration time for the cached response.

	Body    []byte      `json:"body"`
	Headers http.Header `json:"headers"` // Including content-type and other headers.
}

// WriteTo writes the cached response to the provided http.ResponseWriter.
func (r *Response) WriteTo(w http.ResponseWriter) error {
	// Write the headers, including content-type at any case.
	maps.Copy(w.Header(), r.Headers)

	// Set Cache-Control: max-age header if TTL is positive.
	if !r.ExpiresAt.IsZero() {
		maxAge := max(int(time.Until(r.ExpiresAt).Seconds()), 0)
		// Set Cache-Control header to public with max-age.
		// This allows the response to be cached by any cache.
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	}

	// Write the body.
	if len(r.Body) > 0 {
		if _, err := w.Write(r.Body); err != nil {
			// http.Error(w, "failed to write response body", http.StatusInternalServerError)
			return err
		}
	}

	return nil
}

// Set stores the response in the cache under the given key, as it is.
// Use RequestKey to build a key out of a request, or SetResponse to let it derive one.
// Key format (as produced by RequestKey):
// httpcache:<url_path_only>:<method>:<query_hash>:<body_hash>
// Example:
// httpcache:/users/list:POST:abcdef:123456
func (c *Cache) Set(ctx context.Context, key string, response Response) error {
	if c.db == nil {
		return nil // No-op if cache is disabled.
	}

	// Set expiration time for the cached response.
	// So we can track it for any case.
	response.ExpiresAt = time.Now().Add(c.Options.TTL)

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}
	return c.db.Set(ctx, key, data, c.Options.TTL).Err()
}

// SetResponse stores the response for a specific HTTP request in the cache.
// We cache the response body and its headers, not the status code.
// The built-in handler strips the per-client headers (Set-Cookie, hop-by-hop and so on)
// before it reaches this method, see cacheableHeaders.
// As the cache is designed for GET & POST, 200s requests.
func (c *Cache) SetResponse(ctx context.Context, r *http.Request, response Response) error {
	if c.db == nil {
		return nil // No-op if cache is disabled.
	}

	// Set expiration time for the cached response.
	response.ExpiresAt = time.Now().Add(c.Options.TTL)

	key := c.RequestKey(r)
	if err := c.Set(ctx, key, response); err != nil {
		return fmt.Errorf("failed to set cache for request %s: %w", key, err)
	}
	return nil
}

// Get retrieves the cached response for a given key.
// Returns the cached response, a boolean indicating if it was found,
// and any error encountered.
func (c *Cache) Get(ctx context.Context, key string) (Response, bool, error) {
	if c.db == nil {
		return Response{}, false, nil // No-op if cache is disabled.
	}

	val, err := c.db.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return Response{}, false, nil // Cache miss.
		}
		return Response{}, false, fmt.Errorf("failed to get cache: %w", err)
	}

	var response Response
	if err := json.Unmarshal(val, &response); err != nil {
		return Response{}, false, fmt.Errorf("failed to unmarshal cached response: %w", err)
	}
	return response, true, nil
}

// GetResponse retrieves the cached response for a specific HTTP request.
// It uses a cache key derived from the request's URL path, method, query parameters,
// and body.
// Returns the cached response, a boolean indicating if it was found,
// and any error encountered.
func (c *Cache) GetResponse(ctx context.Context, r *http.Request) (Response, bool, error) {
	if c.db == nil {
		return Response{}, false, nil // No-op if cache is disabled.
	}

	key := c.RequestKey(r)
	response, ok, err := c.Get(ctx, key)
	if err != nil {
		return Response{}, false, fmt.Errorf("failed to get cached response for request %s: %w", key, err)
	}
	return response, ok, nil
}

// CountAll returns the total number of cache entries.
func (c *Cache) CountAll(ctx context.Context) (int, error) {
	if c.db == nil {
		return 0, nil // No-op if cache is disabled.
	}

	// Count all cache entries by using a wildcard prefix.
	return c.CountPrefix(ctx, "httpcache:")
}

// PathPrefix generates a cache key prefix based on the request URL path.
func PathPrefix(r *http.Request) string {
	// Generate a cache key prefix based on the request URL path and method.
	// This is used to invalidate all cache entries for a specific endpoint.
	return fmt.Sprintf("httpcache:%s:", r.URL.Path)
}

// CountPrefix returns the number of cache keys that match the given prefix.
func (c *Cache) CountPrefix(ctx context.Context, prefix string) (int, error) {
	if c.db == nil {
		return 0, nil // No-op if cache is disabled.
	}

	const batchSize = 100
	var (
		cursor uint64
		total  int
	)

	for {
		keys, nextCursor, err := c.db.Scan(ctx, cursor, prefix+"*", int64(batchSize)).Result()
		if err != nil {
			return total, fmt.Errorf("failed to scan cache keys: %w", err)
		}
		cursor = nextCursor

		total += len(keys)

		if cursor == 0 {
			break
		}
	}

	return total, nil
}

// InvalidatePrefix deletes all cache entries with the given prefix as fast as possible.
// For Redis >= 6.0, UNLINK is used for asynchronous deletion (faster than DEL).
// Falls back to DEL if UNLINK is not available.
// Returns the number of deleted keys and any error encountered.
func (c *Cache) InvalidatePrefix(ctx context.Context, prefix string) (int, error) {
	if c.db == nil {
		return 0, nil // No-op if cache is disabled.
	}

	const batchSize = 100
	var (
		cursor uint64
		total  int
	)

	for {
		keys, nextCursor, err := c.db.Scan(ctx, cursor, prefix+"*", int64(batchSize)).Result()
		if err != nil {
			return total, fmt.Errorf("failed to scan cache keys: %w", err)
		}
		cursor = nextCursor

		if len(keys) > 0 {
			// Use UNLINK for faster, non-blocking deletion if available.
			unlinkErr := c.db.Unlink(ctx, keys...).Err()
			if unlinkErr != nil {
				// Fallback to DEL if UNLINK is not available.
				if delErr := c.db.Del(ctx, keys...).Err(); delErr != nil {
					return total, fmt.Errorf("failed to delete cache keys: %w", delErr)
				}
			}
			total += len(keys)
		}

		if cursor == 0 {
			break
		}
	}

	return total, nil
}

// InvalidateAll deletes all cache entries by using a wildcard prefix.
// This is useful for clearing the entire cache.
// Returns the number of deleted keys and any error encountered.
// Note: This operation can be expensive if the cache contains a large number of entries.
// It is recommended to use this with caution in production environments.
func (c *Cache) InvalidateAll(ctx context.Context) (int, error) {
	if c.db == nil {
		return 0, nil // No-op if cache is disabled.
	}

	// Invalidate all cache entries by using a wildcard prefix.
	return c.InvalidatePrefix(ctx, "httpcache:")
}

// Invalidate deletes a specific cache entry by its key.
// Returns an error if the deletion fails.
func (c *Cache) Invalidate(ctx context.Context, key string) error {
	if c.db == nil {
		return nil // No-op if cache is disabled.
	}

	if err := c.db.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to invalidate cache key %s: %w", key, err)
	}

	return nil
}

// InvalidateRequest deletes the cache entry for a specific HTTP request.
// It uses a cache key derived from the request's URL path, method, query parameters,
// and body.
// Returns an error if the deletion fails.
func (c *Cache) InvalidateRequest(ctx context.Context, r *http.Request) error {
	if c.db == nil {
		return nil // No-op if cache is disabled.
	}

	key := c.RequestKey(r)
	return c.Invalidate(ctx, key)
}
