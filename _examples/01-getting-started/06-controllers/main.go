// Step 6 of 13: a controller instead of closures.
//
// An Iris controller in this style is a plain struct with two things: a
// constructor, which is its dependency declaration, and a Routing(r iris.Router)
// method, which is its route table. The handlers are methods, so they all see
// the same injected service and the closures of step 5 disappear.
//
// You never call the constructor yourself. app.RegisterDependency(svc) tells
// the application's container about the service, and
// app.Router("/api/shelves", NewShelvesAPI) hands the constructor to the
// container: it sees a parameter of type *Service, finds the registered value,
// calls NewShelvesAPI(svc) and then calls Routing once with the child router
// mounted at /api/shelves. Paths inside Routing are relative to the mount.
//
// Learn: write a controller with a constructor and a Routing method, and let
// the container wire its dependencies.
// Before: ../05-storage-behind-an-interface
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i localhost:8080/api/shelves                                                  # 200
//	curl -i -X POST localhost:8080/api/shelves -d '{"name":"history"}'                  # 201
//	curl -i localhost:8080/api/shelves/11111111-1111-1111-1111-111111111111             # 200
//	curl -i -X DELETE localhost:8080/api/shelves/11111111-1111-1111-1111-111111111111   # 204
//
// Next: step 7 nests books under shelves and guards the routes (../07-nested-routes-and-guards).
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

	app := iris.New()

	errorOptions := errors.NewOptions().
		MapErrors(errors.NotFound, ErrNotFound).
		MapErrors(errors.PermissionDenied, ErrShelfLocked)
	app.Use(errorOptions.Handler())

	// Register the value the constructor needs, then mount the constructor.
	// Step 12 does both through the builder's Services and API steps.
	if err := app.RegisterDependency(NewService(store)); err != nil {
		log.Fatal(err)
	}
	app.Router("/api/shelves", NewShelvesAPI)

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

type Repository interface {
	CreateShelf(ctx stdContext.Context, s Shelf) error
	Shelves(ctx stdContext.Context) ([]Shelf, error)
	Shelf(ctx stdContext.Context, id uuid.UUID) (Shelf, error) // ErrNotFound
	DeleteShelf(ctx stdContext.Context, id uuid.UUID) error    // ErrNotFound
}

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

// ShelvesAPI groups the /shelves handlers around their one dependency.
type ShelvesAPI struct {
	svc *Service
}

// NewShelvesAPI is the dependency declaration: the container calls it with the
// registered *Service.
func NewShelvesAPI(svc *Service) *ShelvesAPI {
	return &ShelvesAPI{svc: svc}
}

// Routing is the route table. The router calls it once at startup with the
// child router this controller is mounted on; paths are relative to it.
func (api *ShelvesAPI) Routing(r iris.Router) {
	r.Get("/", api.list)
	r.Post("/", api.create)
	r.Get("/{id:uuid}", api.get)
	r.Delete("/{id:uuid}", api.delete)
}

func (api *ShelvesAPI) list(ctx iris.Context) {
	shelves, err := api.svc.Shelves(ctx)
	ctx.OK(shelves, err)
}

func (api *ShelvesAPI) create(ctx iris.Context) {
	input, ok := ctx.BindJSON[ShelfInput]()
	if !ok {
		return
	}

	id, err := api.svc.CreateShelf(ctx, input)
	ctx.Created(id, err)
}

func (api *ShelvesAPI) get(ctx iris.Context) {
	id := ctx.Params().GetUUID("id")

	shelf, err := api.svc.Shelf(ctx, id)
	ctx.OK(shelf, err)
}

func (api *ShelvesAPI) delete(ctx iris.Context) {
	id := ctx.Params().GetUUID("id")

	err := api.svc.DeleteShelf(ctx, id)
	ctx.NoContent(true, err)
}

var (
	ErrNotFound    = fmt.Errorf("library: not found")
	ErrShelfLocked = fmt.Errorf("library: shelf is locked")
)

var seedShelfID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

func seed(repo Repository) {
	shelf := Shelf{ID: seedShelfID, Name: "sci-fi", CreatedAt: time.Now().UTC()}
	if err := repo.CreateShelf(stdContext.Background(), shelf); err != nil {
		log.Fatal(err)
	}
}
