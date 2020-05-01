// TODO: godoc and add tests.
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

type Option func(*Limiter)

func ExceedHandler(handler context.Handler) Option {
	return func(l *Limiter) {
		l.exceedHandler = handler
	}
}

func ClientData(clientDataFunc func(ctx context.Context) interface{}) Option {
	return func(l *Limiter) {
		l.clientDataFunc = clientDataFunc
	}
}

func PurgeEvery(every time.Duration, maxLifetime time.Duration) Option {
	condition := func(c *Client) bool {
		// for a custom purger the end-developer may use the c.Data filled from a `ClientData` option.
		return time.Since(c.LastSeen) > maxLifetime
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

type (
	Limiter struct {
		clientDataFunc func(ctx context.Context) interface{} // fill the Client's Data field.
		exceedHandler  context.Handler                       // when too many requests.

		clients map[string]*Client
		mu      sync.RWMutex // mutex for clients.
		pool    *sync.Pool   // object pool for clients.
	}

	Client struct {
		limiter  *rate.Limiter
		LastSeen time.Time
		IP       string
		Data     interface{}
	}
)

// Inf is the infinite rate limit; it allows all events (even if burst is zero).
const Inf = math.MaxFloat64

func Limit(limit float64, burst int, options ...Option) context.Handler {
	rateLimit := rate.Limit(limit)

	l := &Limiter{
		clients: make(map[string]*Client),
		pool: &sync.Pool{New: func() interface{} {
			return &Client{limiter: rate.NewLimiter(rateLimit, burst)}
		}},

		exceedHandler: func(ctx context.Context) {
			ctx.StopWithStatus(429) // Too Many Requests.
		},
	}

	for _, opt := range options {
		opt(l)
	}

	return l.serveHTTP
}

func (l *Limiter) acquire() *Client {
	v := l.pool.Get().(*Client)
	v.LastSeen = time.Now()
	return v
}

func (l *Limiter) release(client *Client) {
	client.IP = ""
	client.Data = nil
	l.pool.Put(client)
}

func (l *Limiter) Purge(condition func(*Client) bool) {
	l.mu.Lock()
	for ip, client := range l.clients {
		if condition(client) {
			l.release(client)
			delete(l.clients, ip)
		}
	}
	l.mu.Unlock()
}

func (l *Limiter) serveHTTP(ctx context.Context) {
	ip := ctx.RemoteAddr()
	l.mu.RLock()
	client, ok := l.clients[ip]
	l.mu.RUnlock()

	if !ok {
		client = l.acquire()
		client.IP = ip

		if l.clientDataFunc != nil {
			client.Data = l.clientDataFunc(ctx)
		}
		//  if l.store(ctx, client) {
		// ^ no, let's keep it simple.
		l.mu.Lock()
		l.clients[ip] = client
		l.mu.Unlock()
	}

	client.LastSeen = time.Now()
	ctx.Values().Set(clientContextKey, client)

	if client.limiter.Allow() {
		ctx.Next()
		return
	}

	if l.exceedHandler != nil {
		l.exceedHandler(ctx)
	}
}

const clientContextKey = "iris.ratelimit.client"

func Get(ctx context.Context) *Client {
	if v := ctx.Values().Get(clientContextKey); v != nil {
		if c, ok := v.(*Client); ok {
			return c
		}
	}

	return nil
}
