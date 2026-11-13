// The service's own configuration file, and a background worker the framework
// starts and stops for you.
//
// iris.LoadConfiguration[T](envKey, filename) reads one YAML document into your
// own type. The environment variable wins over the file, and it may hold either
// the document itself or a path to it, so a container can pass the whole
// configuration inline or mount it without this program knowing which.
//
// iris.BaseConfiguration is the part every service shares: Host, Port,
// AllowOrigin, EnableCompression, RequestLog and the framework's own Iris
// options. Embed it inline and add your own keys next to it. Its BindEnv reads
// HOST, PORT, HTTP_PLATFORM_PORT and ALLOW_ORIGIN over what the document said,
// which is why "PORT=9090 go run ." moves the server without touching the file;
// LoadConfiguration calls it for you, and cfg.BindEnv() below covers the path
// where no document was found and the defaults were built in code.
// cfg.Addr() joins Host and Port into the string app.Listen wants, defaulting
// to every interface on port 8080.
//
// An iris.Worker is a long-running service with Run and Stop. Passed to
// Services it is registered as an injectable dependency AND started after every
// Initializer has returned, then stopped when app.Shutdown runs its closers: it
// lives inside the startup and shutdown chain the builder already enforces,
// instead of in a goroutine next to it. Shutdown cancels Run's context, calls
// Stop and waits for Run to return, so the worker really has finished
// refreshing when Shutdown returns.
//
// Learn: load a service's YAML into a type that embeds iris.BaseConfiguration,
// and run a background job as an iris.Worker the builder starts and stops.
// Before: ../configuration
//
// The program reads server.yml from the working directory, so run it from this
// directory. Ctrl+C drains the server and then stops the worker, which logs
// "rates: worker stopping after N refreshes" before the process exits. The
// signal alone would not stop it: main hands app.Shutdown to
// iris.RegisterOnInterrupt, which is what ties the closers to the signal
// (../../07-dependency-injection/initializer goes through it).
//
// Run:
//
//	go run .                      # reads server.yml: port 8080, refresh every 10s
//	PORT=9090 go run .            # the file's Port is overridden by the environment
//	SERVER_CONFIG='Port: 9091
//	RefreshEvery: 2s
//	Currencies: [CHF]' go run .   # the document inline, no file read at all
//
// Try:
//
//	curl -i localhost:8080/rates           # 200, the cache and how often it refreshed
//	curl -i localhost:8080/rates/EUR       # 200
//	curl -i localhost:8080/rates/XYZ       # 404 NOT_FOUND
//	curl -i localhost:8080/health          # 200, from the builder's Health step
//
// Next: ../../11-views/templates
package main

import (
	stdContext "context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"sync"
	"time"

	"github.com/kataras/iris/v14"
	resterrors "github.com/kataras/iris/v14/rest/errors"
)

func main() {
	cfg, err := loadConfiguration()
	if err != nil {
		log.Fatal(err)
	}

	// The one place where a domain error becomes a status code.
	errorOptions := resterrors.NewOptions().
		MapErrors(resterrors.NotFound, ErrUnknownCurrency)

	app := iris.NewBuilder(cfg.Iris).
		NoPrefix(). // routes mount at the root; Prefix("/api") would move them.
		AllowOrigin(cfg.AllowOrigin).
		Compression(cfg.EnableCompression).
		LogRequests(cfg.RequestLog).
		Health(true, "production", "rates-team").
		Errors(errorOptions).
		Services(NewRatesCache(cfg)). // injected, started, and stopped by app.Shutdown.
		RouterMiddlewares().
		Middlewares().
		API("/rates", NewRatesAPI).
		Build()

	// Ctrl+C stops the server by itself. Handing app.Shutdown to the signal
	// makes it stop the worker too, within the 15 seconds Listen allows.
	iris.RegisterOnInterrupt(app.Shutdown)

	// Addr is "host:port" from the configuration, ":8080" when both are unset.
	app.Listen(cfg.Addr())
}

// Configuration is the whole service's configuration: the shared half inlined,
// then this service's own keys. Secrets stay out of it; they come from the
// environment, because this file is committed.
type Configuration struct {
	iris.BaseConfiguration `yaml:",inline"`

	// RefreshEvery is how often the worker refreshes the cache.
	RefreshEvery time.Duration `yaml:"RefreshEvery"`
	// Currencies is what the cache holds.
	Currencies []string `yaml:"Currencies"`
}

// RatesCache is the iris.Worker: a ticker-driven refresher of an in-memory
// cache. It is an ordinary service that happens to have Run and Stop, so the
// controller below takes it as a plain dependency.
type RatesCache struct {
	interval   time.Duration
	currencies []string

	mu        sync.RWMutex
	rates     map[string]float64
	refreshes int
}

func NewRatesCache(cfg Configuration) *RatesCache {
	return &RatesCache{
		interval:   cfg.RefreshEvery,
		currencies: cfg.Currencies,
		rates:      make(map[string]float64, len(cfg.Currencies)),
	}
}

// Run blocks until the context is cancelled, which happens at shutdown just
// before Stop. An error returned before that is logged and does not take the
// HTTP server with it, so a rate feed that is down degrades the service
// instead of stopping it.
func (c *RatesCache) Run(ctx stdContext.Context) error {
	c.refresh() // fill the cache once, so the first request is not empty.

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			c.refresh()
		}
	}
}

// Stop releases what Run acquired. It is called once, with a bounded context,
// and Shutdown then waits for Run to return within the same budget.
func (c *RatesCache) Stop(stdContext.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	log.Printf("rates: worker stopping after %d refreshes", c.refreshes)
	return nil
}

// refresh stands in for the call to the upstream feed.
func (c *RatesCache) refresh() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, currency := range c.currencies {
		c.rates[currency] = 1 + rand.Float64()
	}

	c.refreshes++
}

func (c *RatesCache) All() (map[string]float64, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rates := make(map[string]float64, len(c.rates))
	for currency, rate := range c.rates {
		rates[currency] = rate
	}

	return rates, c.refreshes
}

func (c *RatesCache) Get(currency string) (float64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rate, ok := c.rates[currency]
	if !ok {
		return 0, fmt.Errorf("%s: %w", currency, ErrUnknownCurrency)
	}

	return rate, nil
}

// RatesAPI is the controller. The cache arrives injected, exactly as any other
// registered service would.
type RatesAPI struct {
	cache *RatesCache
}

func NewRatesAPI(cache *RatesCache) *RatesAPI { return &RatesAPI{cache: cache} }

func (api *RatesAPI) Routing(r iris.Router) {
	r.Get("/", api.list)
	r.Get("/{currency:string}", api.get)
}

func (api *RatesAPI) list(ctx iris.Context) {
	rates, refreshes := api.cache.All()
	ctx.OK(iris.Map{"rates": rates, "refreshes": refreshes}, nil)
}

func (api *RatesAPI) get(ctx iris.Context) {
	rate, err := api.cache.Get(ctx.Params().GetString("currency"))
	ctx.OK(iris.Map{"rate": rate}, err)
}

// ErrUnknownCurrency is the domain sentinel, mapped to 404 in the builder.
var ErrUnknownCurrency = errors.New("rates: unknown currency")

// loadConfiguration reads server.yml, or the SERVER_CONFIG variable when it is
// set. A missing file is not fatal here: the defaults below are a working
// development configuration, so the example runs from a clone.
func loadConfiguration() (Configuration, error) {
	cfg, err := iris.LoadConfiguration[Configuration]("SERVER_CONFIG", "server.yml")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return cfg, err
		}

		cfg = Configuration{Currencies: []string{"EUR", "USD"}}
		cfg.Iris = iris.Options{Name: "rates", LogLevel: "info"}
		// LoadConfiguration runs BindEnv after decoding. Nothing was decoded
		// on this path, so the environment overrides are applied by hand.
		cfg.BindEnv()
	}

	if cfg.RefreshEvery <= 0 {
		cfg.RefreshEvery = 10 * time.Second
	}

	return cfg, nil
}
