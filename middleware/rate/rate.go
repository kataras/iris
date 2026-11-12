// Package rate implements a token-bucket rate limiter for incoming requests,
// one bucket per client.
//
// Its entry point is New(limit, burst, Options), which returns a *Limiter
// rather than a Handler: the limiter owns a background purge goroutine and must
// be stopped with Stop when it is no longer needed, and it exposes the clients
// map to `Get`. Register `Limiter.Handler()` on a Router or a Route, or use the
// `Limit` one-liner when the limiter lives for the whole process.
//
// Example can be found at: _examples/06-middleware/rate-limit/main.go.
package rate

import (
	"math"
	"sync"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"

	"golang.org/x/time/rate"
)

const middlewareName = "iris.ratelimit"

func init() {
	// Tolerant of the method-value wrapper suffix ("-fm"), whose exact form
	// is compiler-generated and not stable across Go releases. The previous
	// pattern was an invalid regexp (unescaped "(*Limiter)") and silently
	// degraded to an exact-literal match.
	context.SetHandlerName(`iris/middleware/rate\.\(\*Limiter\)\.serveHTTP.*`, middlewareName)
}

// RegisterExceedHandler sets a middleware which sets the exceed "handler" for the current Router.
// The "handler"  will be executed every time a client tries to reach a page/resource
// which is not accessible for that moment.
func RegisterExceedHandler(router router.Router, handler context.Handler) {
	router.GetRoot().Handlers().SubscribePrependBefore(func(ctx *context.Context) {
		setExceedHandler(ctx, handler)
		ctx.Next()
	}, middlewareName)
	// if router.Handlers().FuncExists(handler) { // TODO: this is wrong. Think of it, in that state the rate middleware was not existed and this should be prepended right before the rate middleware in the chain..hmm.
}

var exceedHandlerContextKey = "iris.ratelimit.exceed_handler"

func setExceedHandler(ctx *context.Context, handler context.Handler) {
	ctx.Values().Set(exceedHandlerContextKey, handler)
}

// getExceedHandler returns the handler for the current request.
func getExceedHandler(ctx *context.Context) context.Handler {
	if v := ctx.Values().Get(exceedHandlerContextKey); v != nil {
		if h, ok := v.(context.Handler); ok {
			return h
		}
	}

	return nil
}

// Options holds the optional configuration of a Limiter.
// The zero value is valid and uses the documented defaults.
type Options struct {
	// ExceedHandler is executed every time a client tries to reach a
	// page/resource which is not accessible for that moment.
	// This sets the handler for all requests. To override it for the requests of
	// a specific Router, use the `RegisterExceedHandler` package-level function.
	//
	// Defaults to DefaultExceedHandler, which stops with 429 (Too Many Requests).
	ExceedHandler context.Handler

	// ClientData provides custom data that will be stored on the Client and can
	// be retrieved as `Get(ctx).Data` later on.
	//
	// Defaults to nil.
	ClientData func(ctx *context.Context) any

	// PurgeEvery is how often old client entries are checked and removed,
	// e.g. PurgeEvery: time.Minute and PurgeMaxLifetime: 5*time.Minute to check
	// every minute whether a client's last visit was 5 minutes ago and drop it
	// from the memory.
	//
	// Defaults to one minute. There is always a purger, because a limiter
	// without one grows forever - see the note in `New`.
	//
	// PurgeEvery and PurgeMaxLifetime are independent: setting one alone leaves
	// the other at its default.
	PurgeEvery time.Duration
	// PurgeMaxLifetime is how long a client entry with no request is kept.
	//
	// Defaults to a value derived from the rate, long enough that dropping an
	// entry can never hand a client a burst it had not already earned.
	// See PurgeEvery.
	PurgeMaxLifetime time.Duration
}

// DefaultExceedHandler is the default handler which is being used when Options.ExceedHandler is not set.
// It stops the execution with status code 429 (Too Many Requests).
var DefaultExceedHandler = func(ctx *context.Context) {
	ctx.StopWithStatus(429) // Too Many Requests.
}

// Every converts a minimum time interval between events to a limit.
// Usage: Limit(Every(1*time.Minute), 3, Options{})
func Every(interval time.Duration) float64 {
	if interval <= 0 {
		return Inf
	}
	return 1 / interval.Seconds()
}

type (
	// Limiter is featured with the necessary functions to limit requests per second.
	// It is created through the `New` package-level function (or, indirectly,
	// through `Limit`). Register its `Handler` and call `Stop` when the limiter
	// is no longer needed, to release its background purge goroutine.
	Limiter struct {
		clientDataFunc func(ctx *context.Context) any // fill the Client's Data field.
		exceedHandler  context.Handler                // when too many requests.

		limit     rate.Limit
		burstSize int

		clients map[string]*Client
		mu      sync.RWMutex // mutex for clients.

		// The purge configuration, recorded by the PurgeEvery option
		// (or defaulted by New); the single purge goroutine is started by New.
		purgeInterval  time.Duration
		purgeCondition func(*Client) bool

		// done ends the purge goroutine; closed by Stop, once.
		done     chan struct{}
		stopOnce sync.Once
		// stopped is closed when the purge goroutine exits;
		// Stop blocks on it so that, when it returns, no purge is or will be running.
		stopped chan struct{}
	}

	// Client holds some request information and the rate limiter itself.
	// It can be retrieved by the `Get` package-level function.
	// It can be used to manually add RateLimit response headers.
	Client struct {
		ID      string
		Data    any
		Limiter *rate.Limiter

		lastSeen time.Time
		// Delay indicates, when the limit was exceeded, how long the client must
		// wait before the resource becomes available again.
		//
		// Read it through RetryDelay: this value is written under mu by whichever
		// request hit the limit, and a Client is shared by every concurrent
		// request from the same identifier.
		Delay time.Duration
		mu    sync.RWMutex // mutex for lastSeen and Delay.
	}
)

// Inf is the infinite rate limit; it allows all events (even if burst is zero).
const Inf = math.MaxFloat64

// Limit returns a new rate limiter handler that allows requests up to rate "limit" and permits
// bursts of at most "burst" tokens. See `rate.SetIdentifier(ctx, key string)` and `rate.Get` too.
//
// E.g. Limit(1, 5, rate.Options{}) to allow 1 request per second, with a maximum
// burst size of 5.
//
// It is shorthand for New(limit, burst, opts).Handler(): the Limiter
// itself is discarded and its purge goroutine runs for the lifetime of the
// process, which is correct for process-lifetime middleware. Use `New` when
// the limiter must be stopped, e.g. on tests or reloadable configurations.
func Limit(limit float64, burst int, opts Options) context.Handler {
	return New(limit, burst, opts).Handler()
}

// New returns a new Limiter which allows requests up to rate "limit" and
// permits bursts of at most "burst" tokens. Register its `Handler` and call
// `Stop` when the limiter is no longer needed. See `Limit` for the one-liner
// version and `Options` for the optional configuration.
func New(limit float64, burst int, opts Options) *Limiter {
	exceedHandler := opts.ExceedHandler
	if exceedHandler == nil {
		exceedHandler = DefaultExceedHandler
	}

	l := &Limiter{
		clients:        make(map[string]*Client),
		limit:          rate.Limit(limit),
		burstSize:      burst,
		exceedHandler:  exceedHandler,
		clientDataFunc: opts.ClientData,
		done:           make(chan struct{}),
		stopped:        make(chan struct{}),
	}

	// Note(@kataras): a limiter with no purger grows forever.
	//
	// Every distinct identifier allocates a Client plus a rate.Limiter that is
	// never reclaimed, so the documented one-liner Limit(1, 5) turns into a
	// memory-exhaustion vector: an attacker rotating source addresses (an IPv6
	// /64 gives them 2^64, and a spoofable X-Forwarded-For gives them all of
	// them) mints map entries at line rate until the process dies. That is a DoS
	// through the middleware whose entire job is DoS protection.
	//
	// There is therefore ALWAYS a purger. Options.PurgeEvery and
	// Options.PurgeMaxLifetime each override one half of it and are independent:
	// setting one alone leaves the other at its default, rather than silently
	// discarding the one that was set.
	//
	// The default only reclaims memory, it never changes whether a request is
	// allowed - an entry is only dropped once the client's bucket has had time to
	// refill completely.
	purgeInterval := opts.PurgeEvery
	if purgeInterval <= 0 {
		purgeInterval = defaultPurgeInterval
	}

	maxLifetime := opts.PurgeMaxLifetime
	if maxLifetime <= 0 {
		maxLifetime = l.defaultMaxLifetime()
	}

	l.purgeInterval = purgeInterval
	// for a custom purger the end-developer may use the c.Data filled from Options.ClientData.
	l.purgeCondition = func(c *Client) bool {
		return time.Since(c.LastSeen()) > maxLifetime
	}

	l.startPurger(l.purgeInterval, l.purgeCondition)

	return l
}

// Handler returns the limiter's middleware handler,
// ready to be registered on a Router or a Route.
func (l *Limiter) Handler() context.Handler {
	return l.serveHTTP
}

// Stop terminates the limiter's background purge goroutine and waits for
// it to exit; after Stop returns no purge runs anymore (already-stored
// clients remain until garbage collected with the Limiter itself).
// It is safe to call Stop more than once and from concurrent goroutines.
func (l *Limiter) Stop() {
	l.stopOnce.Do(func() {
		close(l.done)
	})

	<-l.stopped
}

const (
	// defaultPurgeInterval is how often idle clients are reclaimed when the
	// caller did not pass PurgeEvery.
	defaultPurgeInterval = time.Minute
	// minPurgeLifetime bounds the computed idle lifetime from below.
	minPurgeLifetime = time.Minute
	// maxPurgeLifetime bounds it from above.
	maxPurgeLifetime = time.Hour
)

// defaultMaxLifetime returns how long an idle client is kept.
//
// It is derived from the configured rate so that dropping an entry can never
// hand a client a fresh burst it had not already earned: after this long without
// a request, the client's bucket would have refilled to full anyway.
func (l *Limiter) defaultMaxLifetime() time.Duration {
	if l.limit <= 0 || math.IsInf(float64(l.limit), 1) || float64(l.limit) >= math.MaxFloat64 {
		return minPurgeLifetime
	}

	burst := max(l.burstSize, 1)

	// Seconds needed to refill the whole burst, with a safety factor.
	refill := time.Duration(float64(burst)/float64(l.limit)*float64(time.Second)) * 2

	return min(max(refill, minPurgeLifetime), maxPurgeLifetime)
}

// startPurger starts the single background reclaim loop.
// It runs until Stop is called and it is only started by New,
// after the options are applied.
func (l *Limiter) startPurger(every time.Duration, condition func(*Client) bool) {
	go func() {
		defer close(l.stopped)

		ticker := time.NewTicker(every)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				l.Purge(condition)
			case <-l.done:
				return
			}
		}
	}()
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
		// Re-check under the write lock.
		//
		// Note(@kataras): two concurrent first requests from the same client both
		// miss the read above and both construct a Client with a FULL burst; the
		// second then overwrites the first. Under a burst-open attack that
		// multiplies the effective burst by the concurrency level, which is
		// exactly the case the limiter exists to bound.
		if existing, found := l.clients[id]; found {
			client = existing
		} else {
			l.clients[id] = client
		}
		l.mu.Unlock()
	}

	client.mu.Lock()
	client.lastSeen = time.Now()
	client.mu.Unlock()

	ctx.Values().Set(clientContextKey, client)

	// Reserve instead of Allow so the retry delay stays available through
	// the `Get` package-level function. A reservation that is not OK can
	// never be satisfied (needs more than the burst); one with a positive
	// delay is not available NOW — both must be rejected: admitting delayed
	// reservations would let every request through immediately, disabling
	// the limiter entirely.
	reservation := client.Limiter.Reserve()
	if delay := reservation.Delay(); !reservation.OK() || delay > 0 {
		reservation.Cancel() // release the future token we will not use.

		client.mu.Lock()
		client.Delay = delay // it's available by the `Get` package-level function.
		client.mu.Unlock()

		exceedHandler := getExceedHandler(ctx) // request-scope exceed handler overrides the rest.
		if exceedHandler == nil {
			exceedHandler = l.exceedHandler

			if exceedHandler == nil {
				exceedHandler = DefaultExceedHandler
			}
		}

		if exceedHandler != nil {
			exceedHandler(ctx)
		} else {
			// for any case the default was set to nil and no other exceed handler was registered.
			// so we have to stop the execution with status code 429 (Too Many Requests).
			ctx.StopWithStatus(429)
		}

		return
	}

	ctx.Next()
}

const identifierContextKey = "iris.ratelimit.identifier"

// SetIdentifier can be called manually from a handler or a middleare
// to change the identifier per client. The default key for a client is its Remote IP.
func SetIdentifier(ctx *context.Context, key string) {
	ctx.Values().Set(identifierContextKey, key)
}

func getIdentifier(ctx *context.Context) string {
	// if entry, ok := ctx.Values().GetEntry(identifierContextKey); ok {
	// 	return entry.MustString()
	// }
	if entry, ok := ctx.Values().Lookup(identifierContextKey); ok {
		return entry.(string)
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

// RetryDelay returns how long the client must wait before the resource becomes
// available again, or zero when the last request was not rate limited.
//
// Use this instead of reading the Delay field directly: a Client is shared by
// every concurrent request from the same identifier, and Delay is written under
// the client's mutex. Reading it unlocked - which is what building
// "X-RateLimit-*" headers from rate.Get(ctx).Delay does - is a data race.
func (c *Client) RetryDelay() time.Duration {
	c.mu.RLock()
	d := c.Delay
	c.mu.RUnlock()
	return d
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
