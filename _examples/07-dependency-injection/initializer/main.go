// Service lifecycle: iris.Initializer at startup, closers at shutdown.
//
// A service passed to Services gets a full lifecycle. Its constructor runs at
// registration; Build() then calls Init on every singleton that implements
// iris.Initializer, in registration order, and a non-nil error stops the build.
// Because *UserCache is passed as a value and has a Close() error method,
// Services registers it as a dependency AND as a shutdown closer - no extra
// call needed. Deferrables adds cleanup that owns no such method.
//
// Both run when app.Shutdown is called, and Ctrl+C alone does not call it: the
// interrupt handler stops the HTTP server and exits. Hand app.Shutdown to
// iris.RegisterOnInterrupt, as main does below, and the signal runs the
// closers too.
//
// Learn: warm a service with iris.Initializer during Build, and run closers and
// Deferrables when app.Shutdown is called.
// Before: ../request-scoped
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/users          # 200, served from the warmed cache
//	curl http://localhost:8080/users/status   # 200, cached count and warmed_at
//
// Press Ctrl+C: the log shows the cache Close and the deferrable running.
//
// Next: ../../08-errors/central-error-map
package main

import (
	stdContext "context"
	"log"
	"sync"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	// The auto-closer applies to values you pass in, not to what constructors
	// create internally: build the cache here so Services sees its Close.
	repo := NewMemUsers()
	cache := NewUserCache(repo)

	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Errors(errors.NewOptions()).
		Deferrables(func() { log.Println("deferrable: goodbye") }).
		Services(repo, cache).
		RouterMiddlewares().
		Middlewares().
		API("/users", NewUsersAPI).
		Build() // Init runs here.

	// The interrupt handler stops the server and exits; app.Shutdown is what
	// runs the closers collected above, so hand it to the handler.
	iris.RegisterOnInterrupt(app.Shutdown)

	app.Listen(":8080")

	// Listen returns the moment the server stops, while the interrupt handler
	// is still working through the closers - and it exits the process when
	// they are done. Block, so main does not race it to the exit.
	select {}
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UserRepository is the slow source the cache sits in front of.
type UserRepository interface {
	List(ctx stdContext.Context) ([]User, error)
}

type MemUsers struct{}

func NewMemUsers() *MemUsers { return &MemUsers{} }

func (*MemUsers) List(stdContext.Context) ([]User, error) {
	return []User{{ID: "1", Name: "Ada"}, {ID: "2", Name: "Linus"}}, nil
}

// UserCache serves reads from memory after a one-time warm-up.
type UserCache struct {
	repo UserRepository

	mu       sync.RWMutex
	users    []User
	warmedAt time.Time
}

func NewUserCache(repo UserRepository) *UserCache {
	return &UserCache{repo: repo}
}

// Init implements iris.Initializer: it runs once inside Build(), after
// injection and before the server starts, so no request ever sees a cold
// cache. Returning an error aborts the build.
func (c *UserCache) Init(ctx stdContext.Context) error {
	users, err := c.repo.List(ctx)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.users, c.warmedAt = users, time.Now()
	c.mu.Unlock()

	log.Printf("cache: warmed %d users", len(users))
	return nil
}

// Close makes the value a shutdown closer as far as Services is concerned.
func (c *UserCache) Close() error {
	c.mu.Lock()
	c.users = nil
	c.mu.Unlock()

	log.Println("cache: closed")
	return nil
}

func (c *UserCache) All() []User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.users
}

func (c *UserCache) WarmedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.warmedAt
}

// UsersAPI reads the cache instead of the repository.
type UsersAPI struct {
	cache *UserCache
}

func NewUsersAPI(cache *UserCache) *UsersAPI {
	return &UsersAPI{cache: cache}
}

func (api *UsersAPI) Routing(r iris.Router) {
	r.Get("/", api.list)
	r.Get("/status", api.status)
}

func (api *UsersAPI) list(ctx iris.Context) {
	ctx.OK(api.cache.All(), nil)
}

func (api *UsersAPI) status(ctx iris.Context) {
	ctx.OK(iris.Map{
		"cached":    len(api.cache.All()),
		"warmed_at": api.cache.WarmedAt(),
	}, nil)
}
