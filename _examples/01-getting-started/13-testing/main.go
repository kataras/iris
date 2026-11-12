// Step 13 of 13: the whole service tested in memory.
//
// main.go is step 12 unchanged; main_test.go is new. newApp(cfg, repo) takes
// its storage as a parameter, so a test builds the real application (real
// router, real middleware, real error map) around any repository it likes and
// httptest.New drives it without a port, a goroutine or a sleep. Every test
// constructs its own application and its own store, so nothing is shared and
// nothing has to be reset.
//
// The test to internalize is the last one: failingRepo embeds the Repository
// interface and overrides a single method. One type, a few lines, and you have
// proven that a database outage renders as a generic 500 with the connection
// string kept server-side.
//
// Learn: test an Iris application end to end in milliseconds, replace storage
// with a fake, and assert on status, JSON and headers with the httptest chain.
// Before: ../12-newbuilder
//
// Run:
//
//	go test -v
//	ADMIN_PASSWORD=change-me go run main.go     # still a working server
//
// Try:
//
//	go test -run TestInternalErrorsDoNotLeak -v        # PASS, and nothing about 10.0.0.3 in the output
//	go test -run TestBooksPaginationAndGuards -v       # PASS
//
// Next: TUTORIAL.md in the Iris repository splits this file into packages
// (library, api, main) and ../../02-routing/routing-basics starts the topic
// sections. Every section of the README names the book chapter it matches.
package main

import (
	stdContext "context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"uuid"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/basicauth"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/rest/errors/validation"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	store := NewMemStore()
	seed(store)

	app := newApp(cfg, store)
	app.Listen(cfg.Addr())
}

// Config is the whole application configuration: the fields every service
// shares come from iris.BaseConfiguration, inlined, and the Bookshelf's own
// settings sit beside them in the same YAML document. AdminPassword is
// yaml:"-": secrets come from the environment, never from a committed file.
type Config struct {
	iris.BaseConfiguration `yaml:",inline"`

	Env           string `yaml:"Env"`
	Developer     string `yaml:"Developer"`
	AdminUser     string `yaml:"AdminUser"`
	AdminPassword string `yaml:"-"`
}

// LoadConfig reads the document from the BOOKSHELF_CONFIG environment variable
// or the config.yml file, lets HOST, PORT and ALLOW_ORIGIN override it, then
// reads the secret from the environment.
func LoadConfig() (Config, error) {
	cfg, err := iris.LoadConfiguration[Config]("BOOKSHELF_CONFIG", "config.yml")
	if err != nil {
		return cfg, err
	}

	cfg.AdminPassword = os.Getenv("ADMIN_PASSWORD")
	if cfg.AdminPassword == "" {
		return cfg, fmt.Errorf("ADMIN_PASSWORD environment variable is required")
	}
	return cfg, nil
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

// Init makes MemStore an iris.Initializer: Build calls it once, after
// injection and before serving. A real store pings its database here.
func (s *MemStore) Init(stdContext.Context) error {
	log.Println("bookshelf: store ready")
	return nil
}

// Close registers the store as a shutdown closer, because Services notices
// the method. A real store releases its connections here.
func (s *MemStore) Close() error {
	log.Println("bookshelf: store closed")
	return nil
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

func (s *MemStore) Counts(_ stdContext.Context) (int64, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return int64(len(s.shelves)), int64(len(s.books)), nil
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

	Counts(ctx stdContext.Context) (shelves, books int64, err error)
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

// Stats is the admin dashboard summary.
type Stats struct {
	Shelves int64 `json:"shelves"`
	Books   int64 `json:"books"`
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

// ForceDeleteShelf bypasses the lock. Only the admin API exposes it.
func (s *Service) ForceDeleteShelf(ctx stdContext.Context, id uuid.UUID) error {
	return s.repo.DeleteShelf(ctx, id)
}

func (s *Service) Stats(ctx stdContext.Context) (Stats, error) {
	shelves, books, err := s.repo.Counts(ctx)
	return Stats{Shelves: shelves, Books: books}, err
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

// AdminAPI is mounted behind basic auth. It contains no authentication code:
// the check is a middleware on the group it is mounted on.
type AdminAPI struct {
	svc *Service
}

func NewAdminAPI(svc *Service) *AdminAPI {
	return &AdminAPI{svc: svc}
}

func (api *AdminAPI) Routing(r iris.Router) {
	r.Get("/stats", api.stats)
	r.Delete("/shelves/{id:uuid}", api.forceDeleteShelf)
}

func (api *AdminAPI) stats(ctx iris.Context) {
	stats, err := api.svc.Stats(ctx)
	ctx.OK(stats, err)
}

// forceDeleteShelf bypasses the shelf lock: an admin-only capability.
func (api *AdminAPI) forceDeleteShelf(ctx iris.Context) {
	id := ctx.Params().GetUUID("id")

	err := api.svc.ForceDeleteShelf(ctx, id)
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

// newApp assembles the whole application. The repository arrives as a
// parameter: main passes the real store, tests pass fakes.
func newApp(cfg Config, repo Repository) *iris.Application {
	return iris.NewBuilder(cfg.Iris).
		Prefix("/api").
		AllowOrigin(cfg.AllowOrigin).
		Compression(cfg.EnableCompression).
		LogRequests(cfg.RequestLog).
		Health(true, cfg.Env, cfg.Developer).
		Errors(errors.NewOptions().
			MapErrors(errors.NotFound, ErrNotFound).
			MapErrors(errors.PermissionDenied, ErrShelfLocked)).
		Deferrables(func() { log.Println("bookshelf: goodbye") }).
		Services(repo, NewService).
		RouterMiddlewares(logRequests).
		Middlewares().
		API("/shelves", NewShelvesAPI).
		API("/admin", basicauth.Default(map[string]string{
			cfg.AdminUser: cfg.AdminPassword,
		}), NewAdminAPI).
		Build()
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
