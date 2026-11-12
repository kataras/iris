// Step 5 of 13: storage behind an interface, rules in a service.
//
// The package-level map and mutex become MemStore, an implementation of the
// Repository interface. Handlers never talk to it directly: a Service holds
// the business rules (the lock check moved here) and depends on the interface,
// not on the map. This is the seam that makes the whole application testable:
// production wires a database implementation, tests wire MemStore or a
// one-method stub. Step 13 does exactly that.
//
// Every Repository method takes a context.Context first. iris.Context
// implements it, so handlers pass ctx straight through and a client that
// disconnects cancels your database query with no extra code.
//
// Handlers need the service, so for now each is a function that takes the
// service and returns an iris.Handler. That gets noisy fast; step 6 replaces
// the closures with a controller.
//
// Learn: separate storage, rules and HTTP into three layers that depend in one
// direction, and pass the request context down to storage.
// Before: ../04-errors-mapped-once
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i -X POST localhost:8080/api/shelves -d '{"name":"archive","locked":true}'    # 201, copy the id
//	curl -i -X DELETE localhost:8080/api/shelves/THE_ID_ABOVE                           # 403, the rule now lives in Service.DeleteShelf
//	curl -i -X DELETE localhost:8080/api/shelves/11111111-1111-1111-1111-111111111111   # 204
//	curl -i localhost:8080/api/shelves                                                  # 200, the locked shelf only
//
// Next: step 6 groups the handlers into a controller (../06-controllers).
package main

import (
	stdContext "context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"uuid"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/rest/errors/validation"
)

func main() {
	store := NewMemStore()
	seed(store)
	svc := NewService(store)

	app := iris.New()

	errorOptions := errors.NewOptions().
		MapErrors(errors.NotFound, ErrNotFound).
		MapErrors(errors.PermissionDenied, ErrShelfLocked)
	app.Use(errorOptions.Handler())

	api := app.Router("/api")
	api.Get("/shelves", listShelves(svc))
	api.Post("/shelves", createShelf(svc))
	api.Get("/shelves/{id:uuid}", getShelf(svc))
	api.Delete("/shelves/{id:uuid}", deleteShelf(svc))

	app.Listen(":8080")
}

type Shelf struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Locked    bool      `json:"locked"`
	CreatedAt time.Time `json:"created_at"`
}

type ShelfInput struct {
	Name   string `json:"name"`
	Locked bool   `json:"locked"`
}

func (in ShelfInput) Validate() error {
	return validation.Join(
		validation.String("name", in.Name).NotEmpty().Length(1, 100),
	)
}

// Repository is the storage seam: the service depends on this interface,
// production wires a database implementation, tests wire MemStore or a stub.
type Repository interface {
	CreateShelf(ctx stdContext.Context, s Shelf) error
	Shelves(ctx stdContext.Context) ([]Shelf, error)
	Shelf(ctx stdContext.Context, id uuid.UUID) (Shelf, error) // ErrNotFound
	DeleteShelf(ctx stdContext.Context, id uuid.UUID) error    // ErrNotFound
}

// MemStore is an in-memory Repository: a mutex and a map, ordinary Go.
type MemStore struct {
	mu      sync.RWMutex
	shelves map[uuid.UUID]Shelf
}

var _ Repository = (*MemStore)(nil)

func NewMemStore() *MemStore {
	return &MemStore{shelves: make(map[uuid.UUID]Shelf)}
}

func (s *MemStore) CreateShelf(_ stdContext.Context, shelf Shelf) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.shelves[shelf.ID] = shelf
	return nil
}

func (s *MemStore) Shelves(_ stdContext.Context) ([]Shelf, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Shelf, 0, len(s.shelves))
	for _, shelf := range s.shelves {
		list = append(list, shelf)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
	return list, nil
}

func (s *MemStore) Shelf(_ stdContext.Context, id uuid.UUID) (Shelf, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	shelf, ok := s.shelves[id]
	if !ok {
		return Shelf{}, ErrNotFound
	}
	return shelf, nil
}

func (s *MemStore) DeleteShelf(_ stdContext.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.shelves[id]; !ok {
		return ErrNotFound
	}
	delete(s.shelves, id)
	return nil
}

// Service holds the business rules and depends on the Repository interface,
// never on MemStore.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateShelf(ctx stdContext.Context, in ShelfInput) (uuid.UUID, error) {
	shelf := Shelf{ID: uuid.New(), Name: in.Name, Locked: in.Locked, CreatedAt: time.Now().UTC()}
	return shelf.ID, s.repo.CreateShelf(ctx, shelf)
}

func (s *Service) Shelves(ctx stdContext.Context) ([]Shelf, error) {
	return s.repo.Shelves(ctx)
}

func (s *Service) Shelf(ctx stdContext.Context, id uuid.UUID) (Shelf, error) {
	return s.repo.Shelf(ctx, id)
}

// DeleteShelf refuses to delete a locked shelf. The storage no longer knows
// the rule; the service does.
func (s *Service) DeleteShelf(ctx stdContext.Context, id uuid.UUID) error {
	shelf, err := s.repo.Shelf(ctx, id)
	if err != nil {
		return err
	}
	if shelf.Locked {
		return ErrShelfLocked
	}
	return s.repo.DeleteShelf(ctx, id)
}

var (
	ErrNotFound    = fmt.Errorf("library: not found")
	ErrShelfLocked = fmt.Errorf("library: shelf is locked")
)

var seedShelfID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

// seed stores one shelf with a fixed id so the Try lines work as written.
func seed(repo Repository) {
	shelf := Shelf{ID: seedShelfID, Name: "sci-fi", CreatedAt: time.Now().UTC()}
	if err := repo.CreateShelf(stdContext.Background(), shelf); err != nil {
		log.Fatal(err)
	}
}

// The handlers close over the service. Each one reads input, calls the
// service and hands the result to a response helper. Nothing else.

func listShelves(svc *Service) iris.Handler {
	return func(ctx iris.Context) {
		shelves, err := svc.Shelves(ctx) // ctx is a context.Context too.
		ctx.OK(shelves, err)
	}
}

func createShelf(svc *Service) iris.Handler {
	return func(ctx iris.Context) {
		input, ok := ctx.BindJSON[ShelfInput]()
		if !ok {
			return
		}

		id, err := svc.CreateShelf(ctx, input)
		ctx.Created(id, err)
	}
}

func getShelf(svc *Service) iris.Handler {
	return func(ctx iris.Context) {
		id := ctx.Params().GetUUID("id")

		shelf, err := svc.Shelf(ctx, id)
		ctx.OK(shelf, err)
	}
}

func deleteShelf(svc *Service) iris.Handler {
	return func(ctx iris.Context) {
		id := ctx.Params().GetUUID("id")

		err := svc.DeleteShelf(ctx, id)
		ctx.NoContent(true, err)
	}
}
