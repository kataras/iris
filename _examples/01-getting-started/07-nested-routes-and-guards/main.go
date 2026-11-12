// Step 7 of 13: books under shelves, with guards.
//
// Books belong to a shelf, so they get their own controller mounted inside
// ShelvesAPI.Routing at /{shelf_id:uuid}/books. The constructor is resolved
// by the container exactly like the parent's: r.Router(path, NewBooksAPI).
//
// Two middleware idioms appear here. A group guard, r.Use(api.requireShelf),
// runs before every book route: it loads the shelf once, replies 404 when it
// does not exist and caches it in the request values so no handler loads it
// again. A positional guard sits between the path and the handler on the write
// routes only: r.Post("/", api.requireUnlocked, api.create). Guards are methods
// on the controller, so they see the same injected service as the handlers.
// The middleware contract is plain: call ctx.Next() to continue, or write a
// response to stop. The errors senders write and stop in one call.
//
// Learn: nest a controller inside another, guard a whole group, guard single
// routes, and cache a loaded entity on the request with ctx.Values.
// Before: ../06-controllers
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i -X POST localhost:8080/api/shelves/11111111-1111-1111-1111-111111111111/books -d '{"title":"Dune","author":"Frank Herbert","pages":412}'   # 201
//	curl localhost:8080/api/shelves/11111111-1111-1111-1111-111111111111/books                                                                    # 200, one book
//	curl -i localhost:8080/api/shelves/00000000-0000-0000-0000-000000000000/books                                                                 # 404 from requireShelf
//	curl -i -X POST localhost:8080/api/shelves -d '{"name":"archive","locked":true}'                                                              # 201, copy the id
//	curl -i -X POST localhost:8080/api/shelves/LOCKED_ID/books -d '{"title":"x","author":"y","pages":1}'                                          # 403 from requireUnlocked
//	curl -i localhost:8080/api/shelves/LOCKED_ID/books                                                                                            # 200, reads still work
//
// Next: step 8 paginates the book list (../08-pagination).
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

// Book belongs to exactly one shelf.
type Book struct {
	ID        uuid.UUID `json:"id"`
	ShelfID   uuid.UUID `json:"shelf_id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Pages     int       `json:"pages"`
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

// BookInput is the payload for creating or updating a book.
type BookInput struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Pages  int    `json:"pages"`
}

func (in BookInput) Validate() error {
	return validation.Join(
		validation.String("title", in.Title).NotEmpty().Length(1, 200),
		validation.String("author", in.Author).NotEmpty(),
		validation.Number("pages", in.Pages).Positive(),
	)
}

type Repository interface {
	CreateShelf(ctx stdContext.Context, s Shelf) error
	Shelves(ctx stdContext.Context) ([]Shelf, error)
	Shelf(ctx stdContext.Context, id uuid.UUID) (Shelf, error) // ErrNotFound
	DeleteShelf(ctx stdContext.Context, id uuid.UUID) error    // ErrNotFound

	CreateBook(ctx stdContext.Context, b Book) error
	Books(ctx stdContext.Context, shelfID uuid.UUID) ([]Book, error)
	Book(ctx stdContext.Context, shelfID, id uuid.UUID) (Book, error) // ErrNotFound
	UpdateBook(ctx stdContext.Context, b Book) error                  // ErrNotFound
	DeleteBook(ctx stdContext.Context, shelfID, id uuid.UUID) error   // ErrNotFound
}

type MemStore struct {
	mu      sync.RWMutex
	shelves map[uuid.UUID]Shelf
	books   map[uuid.UUID]Book
}

var _ Repository = (*MemStore)(nil)

func NewMemStore() *MemStore {
	return &MemStore{
		shelves: make(map[uuid.UUID]Shelf),
		books:   make(map[uuid.UUID]Book),
	}
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

// DeleteShelf removes the shelf and its books.
func (s *MemStore) DeleteShelf(_ stdContext.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.shelves[id]; !ok {
		return ErrNotFound
	}

	delete(s.shelves, id)
	for bookID, b := range s.books {
		if b.ShelfID == id {
			delete(s.books, bookID)
		}
	}
	return nil
}

func (s *MemStore) CreateBook(_ stdContext.Context, b Book) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.books[b.ID] = b
	return nil
}

func (s *MemStore) Books(_ stdContext.Context, shelfID uuid.UUID) ([]Book, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Book, 0)
	for _, b := range s.books {
		if b.ShelfID == shelfID {
			list = append(list, b)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
	return list, nil
}

func (s *MemStore) Book(_ stdContext.Context, shelfID, id uuid.UUID) (Book, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, ok := s.books[id]
	if !ok || b.ShelfID != shelfID {
		return Book{}, ErrNotFound
	}
	return b, nil
}

func (s *MemStore) UpdateBook(_ stdContext.Context, b Book) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.books[b.ID]
	if !ok || existing.ShelfID != b.ShelfID {
		return ErrNotFound
	}
	b.CreatedAt = existing.CreatedAt
	s.books[b.ID] = b
	return nil
}

func (s *MemStore) DeleteBook(_ stdContext.Context, shelfID, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.books[id]
	if !ok || b.ShelfID != shelfID {
		return ErrNotFound
	}
	delete(s.books, id)
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

func (s *Service) CreateBook(ctx stdContext.Context, shelfID uuid.UUID, in BookInput) (uuid.UUID, error) {
	book := Book{
		ID:        uuid.New(),
		ShelfID:   shelfID,
		Title:     in.Title,
		Author:    in.Author,
		Pages:     in.Pages,
		CreatedAt: time.Now().UTC(),
	}
	return book.ID, s.repo.CreateBook(ctx, book)
}

func (s *Service) Books(ctx stdContext.Context, shelfID uuid.UUID) ([]Book, error) {
	return s.repo.Books(ctx, shelfID)
}

func (s *Service) Book(ctx stdContext.Context, shelfID, id uuid.UUID) (Book, error) {
	return s.repo.Book(ctx, shelfID, id)
}

func (s *Service) UpdateBook(ctx stdContext.Context, shelfID, id uuid.UUID, in BookInput) error {
	return s.repo.UpdateBook(ctx, Book{
		ID:      id,
		ShelfID: shelfID,
		Title:   in.Title,
		Author:  in.Author,
		Pages:   in.Pages,
	})
}

func (s *Service) DeleteBook(ctx stdContext.Context, shelfID, id uuid.UUID) error {
	return s.repo.DeleteBook(ctx, shelfID, id)
}

type ShelvesAPI struct {
	svc *Service
}

func NewShelvesAPI(svc *Service) *ShelvesAPI {
	return &ShelvesAPI{svc: svc}
}

func (api *ShelvesAPI) Routing(r iris.Router) {
	r.Get("/", api.list)
	r.Post("/", api.create)
	r.Get("/{id:uuid}", api.get)
	r.Delete("/{id:uuid}", api.delete)

	// Everything about a shelf's books lives in its own nested controller. The
	// container resolves NewBooksAPI the same way it resolved NewShelvesAPI.
	r.Router("/{shelf_id:uuid}/books", NewBooksAPI)
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

// BooksAPI serves /api/shelves/{shelf_id:uuid}/books.
type BooksAPI struct {
	svc *Service
}

func NewBooksAPI(svc *Service) *BooksAPI {
	return &BooksAPI{svc: svc}
}

func (api *BooksAPI) Routing(r iris.Router) {
	// Group guard: every book route needs an existing shelf.
	r.Use(api.requireShelf)

	r.Get("/", api.list)
	// Positional guard: writes additionally need an unlocked shelf.
	r.Post("/", api.requireUnlocked, api.create)
	r.Get("/{id:uuid}", api.get)
	r.Put("/{id:uuid}", api.requireUnlocked, api.update)
	r.Delete("/{id:uuid}", api.requireUnlocked, api.delete)
}

// requireShelf loads the shelf named in the path, replies 404 when it does
// not exist and caches it in the request values for the handlers.
func (api *BooksAPI) requireShelf(ctx iris.Context) {
	shelfID := ctx.Params().GetUUID("shelf_id")

	shelf, err := api.svc.Shelf(ctx, shelfID)
	if errors.HandleError(ctx, err) {
		return // mapped 404 written, request stopped.
	}

	ctx.Values().Set(shelfContextKey, shelf)
	ctx.Next()
}

func (api *BooksAPI) list(ctx iris.Context) {
	books, err := api.svc.Books(ctx, currentShelf(ctx).ID)
	ctx.OK(books, err)
}

// requireUnlocked rejects writes against a locked shelf.
func (api *BooksAPI) requireUnlocked(ctx iris.Context) {
	if currentShelf(ctx).Locked {
		errors.PermissionDenied.Detail(ctx, "shelf is locked", "unlock the shelf before modifying its books")
		return
	}
	ctx.Next()
}

func (api *BooksAPI) create(ctx iris.Context) {
	input, ok := ctx.BindJSON[BookInput]()
	if !ok {
		return
	}

	id, err := api.svc.CreateBook(ctx, currentShelf(ctx).ID, input)
	ctx.Created(id, err)
}

func (api *BooksAPI) get(ctx iris.Context) {
	id := ctx.Params().GetUUID("id")

	book, err := api.svc.Book(ctx, currentShelf(ctx).ID, id)
	ctx.OK(book, err)
}

func (api *BooksAPI) update(ctx iris.Context) {
	id := ctx.Params().GetUUID("id")

	input, ok := ctx.BindJSON[BookInput]()
	if !ok {
		return
	}

	err := api.svc.UpdateBook(ctx, currentShelf(ctx).ID, id, input)
	ctx.NoContent(true, err)
}

func (api *BooksAPI) delete(ctx iris.Context) {
	id := ctx.Params().GetUUID("id")

	err := api.svc.DeleteBook(ctx, currentShelf(ctx).ID, id)
	ctx.NoContent(true, err)
}

var (
	ErrNotFound    = fmt.Errorf("library: not found")
	ErrShelfLocked = fmt.Errorf("library: shelf is locked")
)

// shelfContextKey caches the guard-loaded shelf on the request so the book
// handlers never load it twice.
const shelfContextKey = "bookshelf.shelf"

var seedShelfID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

func seed(repo Repository) {
	shelf := Shelf{ID: seedShelfID, Name: "sci-fi", CreatedAt: time.Now().UTC()}
	if err := repo.CreateShelf(stdContext.Background(), shelf); err != nil {
		log.Fatal(err)
	}
}

// currentShelf reads the value requireShelf cached. ValueAs is typed, so no
// type assertion at the call site.
func currentShelf(ctx iris.Context) Shelf {
	shelf, _ := ctx.ValueAs[Shelf](shelfContextKey)
	return shelf
}
