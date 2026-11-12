// A tasks API tested in main_test.go with the framework's own github.com/kataras/iris/v14/httptest package.
//
// The sibling ../std-httptest directory holds the SAME application with tests
// written against net/http/httptest and encoding/json only. Read the two test
// files side by side to pick the style you prefer.
//
// What makes this application testable is the newApp(cfg, repo) seam: it takes
// its configuration and its storage as arguments and returns an application
// that has not started listening. main passes production values, a test passes
// its own, and both drive the same router, middleware and error map.
//
// Learn: test an Iris application without opening a port, assert on JSON, headers and cookies, and swap storage for a fake through the newApp seam.
//
// Run:
//
//	go test -v
//	ADMIN_PASSWORD=change-me go run main.go
//
// Try:
//
//	curl -i -X POST localhost:8080/api/tasks -d '{"title":"buy milk"}'   # 201 {"id":"..."}
//	curl localhost:8080/api/tasks                                        # 200, the task list
//	curl -i localhost:8080/api/admin/stats                               # 401, no credentials
//	curl -u admin:change-me localhost:8080/api/admin/stats               # 200 {"open":1,"admin":"admin"}
//
// Next: ../std-httptest
package main

import (
	stdContext "context"
	"log"
	"os"
	"sync"

	"uuid"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/basicauth"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/rest/errors/validation"
)

func main() {
	cfg := Config{AdminUser: "admin", AdminPassword: os.Getenv("ADMIN_PASSWORD")}
	if cfg.AdminPassword == "" {
		log.Fatal("ADMIN_PASSWORD environment variable is required")
	}

	newApp(cfg, NewMemStore()).Listen(":8080")
}

// Config carries what a test wants to control. Production fills it from the
// environment, the tests fill it inline.
type Config struct {
	AdminUser     string
	AdminPassword string
}

// MemStore is the in-memory Repository: this example's storage and the test
// suite's default fake. A slice, not a map, so List has a stable order.
type MemStore struct {
	mu    sync.RWMutex
	tasks []Task
}

var _ Repository = (*MemStore)(nil)

func NewMemStore() *MemStore { return new(MemStore) }

func (s *MemStore) Add(_ stdContext.Context, t Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks = append(s.tasks, t)

	return nil
}

func (s *MemStore) List(_ stdContext.Context, onlyOpen bool) ([]Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		if onlyOpen && t.Done {
			continue
		}

		list = append(list, t)
	}

	return list, nil
}

func (s *MemStore) Get(_ stdContext.Context, id uuid.UUID) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.tasks {
		if t.ID == id {
			return t, nil
		}
	}

	return Task{}, ErrTaskNotFound
}

func (s *MemStore) Complete(_ stdContext.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks[i].Done = true
			return nil
		}
	}

	return ErrTaskNotFound
}

// Repository is the seam that makes the handlers testable: production wires a
// database implementation, a test wires MemStore or a stub that fails on
// purpose. Because it is an interface, a test double is an ordinary struct.
type Repository interface {
	Add(ctx stdContext.Context, t Task) error
	List(ctx stdContext.Context, onlyOpen bool) ([]Task, error)
	Get(ctx stdContext.Context, id uuid.UUID) (Task, error) // ErrTaskNotFound
	Complete(ctx stdContext.Context, id uuid.UUID) error    // ErrTaskNotFound
}

// Task is the stored entity and the JSON response shape.
type Task struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	Done  bool      `json:"done"`
}

// TaskInput is the request payload. Implementing Validate makes ctx.BindJSON
// run it right after decoding, so an invalid body never reaches a handler.
type TaskInput struct {
	Title string `json:"title"`
}

// Validate reports every broken rule at once; nil means the payload is good.
func (in TaskInput) Validate() error {
	return validation.Join(
		validation.String("title", in.Title).NotEmpty().Length(3, 120),
	)
}

// tasksAPI groups the handlers; the repository arrives once, at construction.
type tasksAPI struct {
	repo Repository
}

func (api *tasksAPI) list(ctx iris.Context) {
	tasks, err := api.repo.List(ctx, ctx.Query().GetBool("open"))
	ctx.OK(tasks, err)
}

func (api *tasksAPI) create(ctx iris.Context) {
	input, ok := ctx.BindJSON[TaskInput]()
	if !ok {
		return // 400 already written: malformed JSON or a failed Validate.
	}

	t := Task{ID: uuid.New(), Title: input.Title}
	ctx.Created(t.ID, api.repo.Add(ctx, t)) // 201 {"id":"..."}.
}

func (api *tasksAPI) get(ctx iris.Context) {
	t, err := api.repo.Get(ctx, taskID(ctx))
	ctx.OK(t, err) // 200, or 404 through the central error map.
}

func (api *tasksAPI) complete(ctx iris.Context) {
	ctx.NoContent(true, api.repo.Complete(ctx, taskID(ctx))) // 204 or 404.
}

func (api *tasksAPI) stats(ctx iris.Context) {
	open, err := api.repo.List(ctx, true)
	if err != nil {
		errors.HandleError(ctx, err)
		return
	}

	// ctx.User() is non-nil for every request that got past basicauth.
	username, _ := ctx.User().GetUsername()
	ctx.OK(iris.Map{"open": len(open), "admin": username}, nil)
}

// ErrTaskNotFound is a domain sentinel: it knows nothing about HTTP. newApp
// classifies it once, so every handler that forwards it answers the same 404.
var ErrTaskNotFound = errors.New("task not found")

// newApp is the composition root and the test seam. Note what it does NOT do:
// no os.Getenv, no file reads, no database dial, no Listen. Those belong to
// main, which is the only part of the program a test cannot reach.
func newApp(cfg Config, repo Repository) *iris.Application {
	app := iris.New()
	api := &tasksAPI{repo: repo}

	// The central error map: one place decides that ErrTaskNotFound is a 404
	// and that anything unmapped is a generic 500. UseRouter (over Use) also
	// covers unmatched paths, so 404s carry the same JSON shape.
	app.UseRouter(errors.NewOptions().
		MapErrors(errors.NotFound, ErrTaskNotFound).
		Handler())
	app.UseRouter(stamp)

	tasks := app.Router("/api/tasks")
	tasks.Get("/", api.list)
	tasks.Post("/", api.create)
	tasks.Get("/{id:uuid}", api.get)
	tasks.Patch("/{id:uuid}/done", api.complete)

	// The admin group is the only one behind credentials.
	admin := app.Router("/api/admin", basicauth.Default(map[string]string{
		cfg.AdminUser: cfg.AdminPassword,
	}))
	admin.Get("/stats", api.stats)

	return app
}

// stamp marks every response with the API version and hands a first-time
// client a visitor cookie: a header and a cookie for the tests to assert on.
func stamp(ctx iris.Context) {
	ctx.SetHeader("X-Api-Version", "v1")

	if ctx.CookieValue("visitor") == "" {
		ctx.AddCookie("visitor", uuid.New().String())
	}

	ctx.Next()
}

// taskID reads the {id:uuid} path parameter. The macro rejected anything that
// is not a UUID before the handler ran and parsed the value once, so the
// handler reads the typed value.
func taskID(ctx iris.Context) uuid.UUID {
	return ctx.Params().GetUUID("id")
}
