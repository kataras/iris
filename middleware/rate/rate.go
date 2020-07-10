// Package rate implements rate limiter for Iris client requests.
// Example can be found at: _examples/request-ratelimit/main.go.
package rate

import (
	"math"
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"

	"golang.org/x/time/rate"
)

func init() {
	context.SetHandlerName("iris/middleware/rate.(*Limiter).serveHTTP-fm", "iris.ratelimit")
}

// Option declares a function which can be passed on `Limit` package-level
// to modify its internal fields. Available Options are:
// * ExceedHandler
// * ClientData
// * PurgeEvery
type Option func(*Limiter)

// ExceedHandler is an `Option` that can be passed at the `Limit` package-level function.
// It accepts a handler that will be executed every time a client tries to reach a page/resource
// which is not accessible for that moment.
func ExceedHandler(handler context.Handler) Option {
	return func(l *Limiter) {
		l.exceedHandler = handler
	}
}

// ClientData is an `Option` that can be passed at the `Limit` package-level function.
// It accepts a function which provides the Iris Context and should return custom data
// that will be stored to the Client and be retrieved as `Get(ctx).Client.Data` later on.
func ClientData(clientDataFunc func(ctx *context.Context) interface{}) Option {
	return func(l *Limiter) {
		l.clientDataFunc = clientDataFunc
	}
}

// PurgeEvery is an `Option` that can be passed at the `Limit` package-level function.
// This function will check for old entries and remove them.
//
// E.g. Limit(..., PurgeEvery(time.Minute, 5*time.Minute)) to
// check every 1 minute if a client's last visit was 5 minutes ago ("old" entry)
// and remove it from the memory.
func PurgeEvery(every time.Duration, maxLifetime time.Duration) Option {
	condition := func(c *Client) bool {
		// for a custom purger the end-developer may use the c.Data filled from a `ClientData` option.
		return time.Since(c.LastSeen()) > maxLifetime
	}

	return func(l *Limiter) {
		go func() {
			for {
				time.Sleep(every)

				l.Purge(condition)
			}
		}()
	}
}

// Every converts a minimum time interval between events to a limit.
// Usage: Limit(Every(1*time.Minute), 3, options...)
func Every(interval time.Duration) float64 {
	if interval <= 0 {
		return Inf
	}
	return 1 / interval.Seconds()
}

type (
	// Limiter is featured with the necessary functions to limit requests per second.
	// It has a single exported method `Purge` which helps to manually remove
	// old clients from the memory. Limiter is not exposed by a function,
	// callers should use it inside an `Option` for the `Limit` package-level function.
	Limiter struct {
		clientDataFunc func(ctx *context.Context) interface{} // fill the Client's Data field.
		exceedHandler  context.Handler                        // when too many requests.

		limit     rate.Limit
		burstSize int

		clients map[string]*Client
		mu      sync.RWMutex // mutex for clients.
	}

	// Client holds some request information and the rate limiter itself.
	// It can be retrieved by the `Get` package-level function.
	// It can be used to manually add RateLimit response headers.
	Client struct {
		ID      string
		Data    interface{}
		Limiter *rate.Limiter

		lastSeen time.Time
		mu       sync.RWMutex // mutex for lastSeen.
	}
)

// Inf is the infinite rate limit; it allows all events (even if burst is zero).
const Inf = math.MaxFloat64

// Limit returns a new rate limiter handler that allows requests up to rate "limit" and permits
// bursts of at most "burst" tokens. See `rate.SetKey(ctx, key string)` and `rate.Get` too.
//
// E.g. Limit(1, 5) to allow 1 request per second, with a maximum burst size of 5.
//
// See `ExceedHandler`, `ClientData` and `PurgeEvery` for the available "options".
func Limit(limit float64, burst int, options ...Option) context.Handler {
	l := &Limiter{
		clients:   make(map[string]*Client),
		limit:     rate.Limit(limit),
		burstSize: burst,
		exceedHandler: func(ctx *context.Context) {
			ctx.StopWithStatus(429) // Too Many Requests.
		},
	}

	for _, opt := range options {
		opt(l)
	}

	return l.serveHTTP
}

// Purge removes client entries from the memory based on the given "condition".
func (l *Limiter) Purge(condition func(*Client) bool) {
	l.mu.Lock()
	for id, client := range l.clients {
		if condition(client) {
			delete(l.clients, id)
		}
	}
	l.mu.Unlock()
}

func (l *Limiter) serveHTTP(ctx *context.Context) {
	id := getIdentifier(ctx)
	l.mu.RLock()
	client, ok := l.clients[id]
	l.mu.RUnlock()

	if !ok {
		client = &Client{
			ID:      id,
			Limiter: rate.NewLimiter(l.limit, l.burstSize),
		}

		if l.clientDataFunc != nil {
			client.Data = l.clientDataFunc(ctx)
		}

		//  if l.store(ctx, client) {
		// ^ no, let's keep it simple.
		l.mu.Lock()
		l.clients[id] = client
		l.mu.Unlock()
	}

	client.mu.Lock()
	client.lastSeen = time.Now()
	client.mu.Unlock()

	ctx.Values().Set(clientContextKey, client)

	if client.Limiter.Allow() {
		ctx.Next()
		return
	}

	if l.exceedHandler != nil {
		l.exceedHandler(ctx)
	}
}

const identifierContextKey = "iris.ratelimit.identifier"

// SetIdentifier can be called manually from a handler or a middleare
// to change the identifier per client. The default key for a client is its Remote IP.
func SetIdentifier(ctx *context.Context, key string) {
	ctx.Values().Set(identifierContextKey, key)
}

func getIdentifier(ctx *context.Context) string {
	if entry, ok := ctx.Values().GetEntry(identifierContextKey); ok {
		return entry.ValueRaw.(string)
	}

	return ctx.RemoteAddr()
}

const clientContextKey = "iris.ratelimit.client"

// Get returns the current rate limited `Client`.
// Use it when you want to log or add response headers based on the current request limitation.
//
// You can read more about X-RateLimit response headers at:
// https://tools.ietf.org/id/draft-polli-ratelimit-headers-00.html.
// A good example of that is the GitHub API itself: https://developer.github.com/v3/#rate-limiting
func Get(ctx *context.Context) *Client {
	if v := ctx.Values().Get(clientContextKey); v != nil {
		if c, ok := v.(*Client); ok {
			return c
		}
	}

	return nil
}

// LastSeen reports the last Client's visit.
func (c *Client) LastSeen() time.Time {
	c.mu.RLock()
	t := c.lastSeen
	c.mu.RUnlock()
	return t
}

// TokensFromDuration is a unit conversion function from a time duration to the number of tokens
// which could be accumulated during that duration at a rate of limit tokens per second.
func (c *Client) TokensFromDuration(d time.Duration) float64 {
	// rate.go#tokensFromDuration
	limit := float64(c.Limiter.Limit())
	sec := float64(d/time.Second) * limit
	nsec := float64(d%time.Second) * limit
	return sec + nsec/1e9
}

// DurationFromTokens is a unit conversion function from the number of tokens to the duration
// of time it takes to accumulate them at a rate of limit tokens per second.
func (c *Client) DurationFromTokens(tokens float64) time.Duration {
	// rate.go#durationFromTokens
	seconds := tokens / float64(c.Limiter.Limit())
	return time.Nanosecond * time.Duration(1e9*seconds)
}
