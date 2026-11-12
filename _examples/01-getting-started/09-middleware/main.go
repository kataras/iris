// Step 9 of 13: a middleware, and where it runs.
//
// A middleware is an ordinary handler that calls ctx.Next() to run the rest of
// the chain. Work before Next sees the request, work after Next sees the
// response. logRequests below times every request and logs the method, path
// and status.
//
// Where you register it decides what it sees:
//
//	app.UseRouter(h)   before routing: every request, including 404s and CORS preflights
//	app.Use(h)         after routing, before the matched handler: matched routes only
//	r.Use(h)           the same, scoped to one router and its children (requireShelf in step 7)
//	r.Post("/", g, h)  positional: one route (requireUnlocked in step 7)
//
// The error map from step 4 sits in app.Use, because it only has work to do
// on matched routes. The logger goes in app.UseRouter, so a typo in a path is
// logged like everything else. Step 12 shows the same two slots on the
// builder: RouterMiddlewares(...) and Middlewares(...).
//
// Learn: write a middleware, and pick between UseRouter, Use and a positional
// guard on purpose.
// Before: ../08-pagination
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl localhost:8080/api/shelves                        # 200, and a log line with the duration
//	curl -i localhost:8080/nothing-here                    # 404, still logged: UseRouter runs before routing
//	curl -i -X POST localhost:8080/api/shelves -d '{}'     # 400, logged with its status
//
// Next: step 10 adds an admin group behind basic authentication (../10-admin-and-basicauth).
package main

import (
	stdContext "context"
	"fmt"
	"log"
	"sort"
	"strings"
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

	// Before routing: sees every request, matched or not.
	app.UseRouter(logRequests)

	// After routing: the error map only has work on matched routes.
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

// Books returns one page of a shelf's books and the total number of matches.
func (s *MemStore) Books(_ stdContext.Context, shelfID uuid.UUID, opts iris.PageOptions, filter BookFilter) ([]Book, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := make([]Book, 0)
	for _, b := range s.books {
		if b.ShelfID != shelfID {
			continue
		}
		if filter.Author != "" && !strings.EqualFold(b.Author, filter.Author) {
			continue
		}
		matched = append(matched, b)
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].CreatedAt.Before(matched[j].CreatedAt) })

	total := int64(len(matched))

	// The page window. A SQL implementation passes the same two numbers to
	// LIMIT and OFFSET.
	offset := opts.GetOffset()
	if offset >= len(matched) {
		return []Book{}, total, nil // past the last page.
	}
	end := min(offset+opts.GetLimit(), len(matched))
	return matched[offset:end], total, nil
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

type Repository interface {
	CreateShelf(ctx stdContext.Context, s Shelf) error
	Shelves(ctx stdContext.Context) ([]Shelf, error)
	Shelf(ctx stdContext.Context, id uuid.UUID) (Shelf, error) // ErrNotFound
	DeleteShelf(ctx stdContext.Context, id uuid.UUID) error    // ErrNotFound

	CreateBook(ctx stdContext.Context, b Book) error
	Books(ctx stdContext.Context, shelfID uuid.UUID, opts iris.PageOptions, filter BookFilter) ([]Book, int64, error)
	Book(ctx stdContext.Context, shelfID, id uuid.UUID) (Book, error) // ErrNotFound
	UpdateBook(ctx stdContext.Context, b Book) error                  // ErrNotFound
	DeleteBook(ctx stdContext.Context, shelfID, id uuid.UUID) error   // ErrNotFound
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

// BookFilter narrows book listings. It arrives as an optional JSON body next
// to the ?page=&size= query; an empty filter matches everything.
type BookFilter struct {
	Author string `json:"author"`
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

func (s *Service) Books(ctx stdContext.Context, shelfID uuid.UUID, opts iris.PageOptions, filter BookFilter) ([]Book, int64, error) {
	return s.repo.Books(ctx, shelfID, opts, filter)
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
	// The {id:uuid} macro already rejected anything that is not a UUID and
	// parsed the value once, so the handler reads the typed value.
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

	// The list answers GET (no filter) and QUERY (a JSON filter in the body).
	r.Get("/", api.list)
	r.Query("/", api.list)
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
	shelf := currentShelf(ctx)

	// Paginated binds ?page=&size= and the optional filter body, calls the
	// collector and renders the envelope. A bad page or size is a 400 before
	// the collector runs.
	ctx.Paginated(func(reqCtx stdContext.Context, opts iris.PageOptions, filter BookFilter) ([]Book, int64, error) {
		return api.svc.Books(reqCtx, shelf.ID, opts, filter)
	})
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

// seed stores one shelf with a fixed id and five books, so the pages above
// have something to show.
func seed(repo Repository) {
	ctx := stdContext.Background()
	now := time.Now().UTC()

	shelf := Shelf{ID: seedShelfID, Name: "sci-fi", CreatedAt: now}
	if err := repo.CreateShelf(ctx, shelf); err != nil {
		log.Fatal(err)
	}

	books := []Book{
		{Title: "Dune", Author: "Frank Herbert", Pages: 412},
		{Title: "Dune Messiah", Author: "Frank Herbert", Pages: 256},
		{Title: "Neuromancer", Author: "William Gibson", Pages: 271},
		{Title: "Solaris", Author: "Stanislaw Lem", Pages: 204},
		{Title: "A Wizard of Earthsea", Author: "Ursula K. Le Guin", Pages: 183},
	}
	for i, b := range books {
		b.ID = uuid.New()
		b.ShelfID = shelf.ID
		b.CreatedAt = now.Add(time.Duration(i) * time.Second) // a stable order.
		if err := repo.CreateBook(ctx, b); err != nil {
			log.Fatal(err)
		}
	}
}

// logRequests is a middleware: it runs the rest of the chain with ctx.Next()
// and logs what came out of it. ctx.GetStatusCode reads the status the
// handler wrote.
func logRequests(ctx iris.Context) {
	start := time.Now()
	ctx.Next()
	log.Printf("%s %s -> %d in %s", ctx.Method(), ctx.Path(), ctx.GetStatusCode(), time.Since(start))
}

// currentShelf reads the value requireShelf cached. ValueAs is typed, so no
// type assertion at the call site.
func currentShelf(ctx iris.Context) Shelf {
	shelf, _ := ctx.ValueAs[Shelf](shelfContextKey)
	return shelf
}
