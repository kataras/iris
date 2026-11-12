# Writing Good Web Backends with Iris

This tutorial takes you from `go mod init` to a tested, production-shaped
REST API with the Iris web framework (`github.com/kataras/iris/v14`).
"Good" here means something specific: handlers that stay six lines long,
errors that are mapped in one place, dependencies that arrive through
constructors so tests can replace them, and an HTTP test suite that runs in
milliseconds. Every code block below belongs to a real file of one small
program; you can build the whole thing by copying them in order.

Requirements: Go 1.27 or newer. Upgrading from Iris v12 instead of starting
fresh? Read [MIGRATION.md](MIGRATION.md) first.

## What You Will Build

The Bookshelf service: shelves that hold books. A shelf can be locked,
which freezes its books, a small rule, but enough to exercise guards,
error mapping and an admin override.

| Method | Path | Purpose | Protection |
| --- | --- | --- | --- |
| GET | /api/shelves | list shelves | — |
| POST | /api/shelves | create a shelf | validation |
| GET | /api/shelves/{id} | one shelf | — |
| DELETE | /api/shelves/{id} | delete (refused when locked) | — |
| GET | /api/shelves/{sid}/books | list books, paginated | shelf exists |
| POST | /api/shelves/{sid}/books | add a book | shelf unlocked |
| GET | /api/shelves/{sid}/books/{id} | one book | shelf exists |
| PUT | /api/shelves/{sid}/books/{id} | update a book | shelf unlocked |
| DELETE | /api/shelves/{sid}/books/{id} | remove a book | shelf unlocked |
| GET | /api/admin/stats | totals | basic auth |
| DELETE | /api/admin/shelves/{id} | force-delete, ignores lock | basic auth |
| GET | /health | health check | — |

The project layout, three layers with one direction of dependency
(api depends on library; library depends on nothing of HTTP):

```
bookshelf/
├── go.mod
├── config.yml             # committed settings; secrets stay in the env
├── main.go                # configuration, wiring, startup
├── main_test.go           # HTTP tests with fake repositories
├── api/
│   ├── shelves.go         # ShelvesAPI: CRUD + nested router
│   ├── books.go           # BooksAPI: guards, CRUD, pagination
│   └── admin.go           # AdminAPI: stats, force-delete
└── library/
    ├── library.go         # entities, inputs, sentinel errors
    ├── repository.go      # the storage interface (the test seam)
    ├── memstore.go        # in-memory implementation
    └── service.go         # business rules
```

Four habits carry this tutorial, and they scale from toys to production:

1. Handlers use the Context's response helpers; the error branches live elsewhere.
2. Domain errors are sentinels, mapped to HTTP codes exactly once.
3. Constructors declare dependencies; `Services(...)` provides them.
4. Every feature lands with an `httptest` test that runs in memory.

## Setup

```sh
mkdir bookshelf && cd bookshelf
go mod init bookshelf
go get github.com/kataras/iris/v14@latest

# The program loads config.yml at startup and fails without it. These are
# the settings it needs to run; the full document, and what every key
# means, is in "Configuration: Env First, Secrets Never in YAML" below.
cat > config.yml <<'YAML'
Port: 8080
AllowOrigin: none
Env: development
Developer: bookshelf
AdminUser: admin
YAML
```

Run `go mod tidy` after pasting the files below.

## The Six-Line Handler

Here is a create-endpoint the way most Go web code starts out, with every
handler carrying its own error branches:

```go
func createShelf(ctx iris.Context) {
    var input library.ShelfInput
    if err := ctx.ReadJSON(&input); err != nil {
        ctx.StopWithJSON(iris.StatusBadRequest, iris.Map{
            "error": "unable to parse body", "detail": err.Error(),
        })
        return
    }

    if err := input.Validate(); err != nil {
        ctx.StopWithJSON(iris.StatusBadRequest, iris.Map{
            "error": err.Error(),
        })
        return
    }

    id, err := svc.CreateShelf(ctx, input)
    if err != nil {
        log.Printf("create shelf: %v", err)
        ctx.StopWithJSON(iris.StatusInternalServerError, iris.Map{
            "error": "internal server error",
        })
        return
    }

    ctx.StatusCode(iris.StatusCreated)
    ctx.JSON(iris.Map{"id": id})
}
```

The same handler with the typed Context methods:

```go
func (api *ShelvesAPI) create(ctx iris.Context) {
    input, ok := ctx.BindJSON[library.ShelfInput]()
    if !ok {
        return // 400 already written: malformed JSON or failed Validate.
    }

    id, err := api.svc.CreateShelf(ctx, input)
    ctx.Created(id, err) // 201 with {"id": "..."}.
}
```

The error branches did not disappear. Decoding and validation failures are
rendered by `ctx.BindJSON` (it runs the input's `Validate() error` method
after decoding); service errors flow through one error map you configure at
startup. The rest of this tutorial builds that one place, and everything
around it.

## Domain First: library/library.go

Start with the domain, not the router. The `library` package defines the
entities, the request inputs and two sentinel errors, and imports nothing
from HTTP.

```go
// Package library is the domain of the Bookshelf service: entities, inputs
// and business errors. It imports no HTTP types.
package library

import (
    "errors"
    "time"
    "uuid"

    "github.com/kataras/iris/v14/rest/errors/validation"
)

// Sentinel errors: the service returns these and the HTTP layer maps them
// to status codes exactly once, at startup.
var (
    ErrNotFound    = errors.New("library: not found")
    ErrShelfLocked = errors.New("library: shelf is locked")
)

// Shelf is a named collection of books. A locked shelf freezes its books.
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

// ShelfInput is the request payload for creating a shelf.
// Implementing Validate makes ctx.BindJSON run it after decoding.
type ShelfInput struct {
    Name   string `json:"name"`
    Locked bool   `json:"locked"`
}

func (in ShelfInput) Validate() error {
    return validation.Join(
        validation.String("name", in.Name).NotEmpty().Length(1, 100),
    )
}

// BookInput is the request payload for creating or updating a book.
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

// BookFilter narrows book listings; it arrives as an optional JSON body
// next to the ?page=&size= query (see ctx.Paginated).
type BookFilter struct {
    Author string `json:"author"`
}

// Stats is the admin dashboard summary.
type Stats struct {
    Shelves int64 `json:"shelves"`
    Books   int64 `json:"books"`
}
```

Two decisions worth naming. Sentinel errors (`ErrNotFound`,
`ErrShelfLocked`) let the domain speak its own language: no status codes
here; the mapping to 404 and 403 happens once, in `main.go`. And the
`Validate() error` methods use the `rest/errors/validation` builders, which
produce field-level error payloads that render as structured JSON. You
will see the wire format in the error section.

> `ctx.BindJSON` calls `Validate` automatically when the type implements
> it, so handlers never call it by hand.

## Storage Behind an Interface

The service talks to storage through an interface. This is the seam that
makes the whole application testable: production wires a database
implementation, tests wire the in-memory store or a one-method stub.

```go
// library/repository.go
package library

import (
    "context"
    "uuid"

    "github.com/kataras/iris/v14"
)

// Repository is the storage seam: the service depends on this interface,
// production wires a database implementation, tests wire MemStore or a stub.
// Every method takes a context.Context first — iris.Context implements it,
// so request cancellation reaches the database for free.
type Repository interface {
    CreateShelf(ctx context.Context, s Shelf) error
    Shelves(ctx context.Context) ([]Shelf, error)
    Shelf(ctx context.Context, id uuid.UUID) (Shelf, error) // ErrNotFound
    DeleteShelf(ctx context.Context, id uuid.UUID) error // ErrNotFound

    CreateBook(ctx context.Context, b Book) error
    Books(ctx context.Context, shelfID uuid.UUID, opts iris.PageOptions,
        filter BookFilter) ([]Book, int64, error)
    Book(ctx context.Context, shelfID, id uuid.UUID) (Book, error) // ErrNotFound
    UpdateBook(ctx context.Context, b Book) error                 // ErrNotFound
    DeleteBook(ctx context.Context, shelfID, id uuid.UUID) error // ErrNotFound

    Counts(ctx context.Context) (shelves, books int64, err error)
}
```

Note the first parameter of every method: a plain `context.Context`.
`iris.Context` implements the standard context interface, so handlers pass
`ctx` straight through and client disconnects cancel your database queries
with zero extra code.

The in-memory implementation is ordinary Go: a mutex and two maps. It is
the tutorial's storage and, later, the test suite's fake:

```go
// library/memstore.go
package library

import (
    "context"
    "sort"
    "strings"
    "sync"
    "uuid"

    "github.com/kataras/iris/v14"
)

// MemStore is an in-memory Repository: the tutorial's storage and the test
// suite's fake. Swapping it for Postgres later changes one Services entry.
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

func (s *MemStore) CreateShelf(_ context.Context, shelf Shelf) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.shelves[shelf.ID] = shelf
    return nil
}

func (s *MemStore) Shelves(_ context.Context) ([]Shelf, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    list := make([]Shelf, 0, len(s.shelves))
    for _, shelf := range s.shelves {
        list = append(list, shelf)
    }
    sort.Slice(list, func(i, j int) bool {
        return list[i].CreatedAt.Before(list[j].CreatedAt)
    })
    return list, nil
}

func (s *MemStore) Shelf(_ context.Context, id uuid.UUID) (Shelf, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    shelf, ok := s.shelves[id]
    if !ok {
        return Shelf{}, ErrNotFound
    }
    return shelf, nil
}

func (s *MemStore) DeleteShelf(_ context.Context, id uuid.UUID) error {
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

func (s *MemStore) CreateBook(_ context.Context, b Book) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.books[b.ID] = b
    return nil
}

func (s *MemStore) Books(_ context.Context, shelfID uuid.UUID,
    opts iris.PageOptions, filter BookFilter) ([]Book, int64, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    matched := make([]Book, 0)
    for _, b := range s.books {
        if b.ShelfID != shelfID {
            continue
        }
        if filter.Author != "" &&
            !strings.EqualFold(b.Author, filter.Author) {
            continue
        }
        matched = append(matched, b)
    }
    sort.Slice(matched, func(i, j int) bool {
        return matched[i].CreatedAt.Before(matched[j].CreatedAt)
    })

    total := int64(len(matched))

    // Apply the page window (a SQL implementation uses LIMIT/OFFSET).
    offset, limit := opts.GetOffset(), opts.GetLimit()
    if offset >= len(matched) {
        return []Book{}, total, nil
    }
    end := offset + limit
    if limit == 0 || end > len(matched) {
        end = len(matched)
    }
    return matched[offset:end], total, nil
}

func (s *MemStore) Book(_ context.Context, shelfID, id uuid.UUID) (Book, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    b, ok := s.books[id]
    if !ok || b.ShelfID != shelfID {
        return Book{}, ErrNotFound
    }
    return b, nil
}

func (s *MemStore) UpdateBook(_ context.Context, b Book) error {
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

func (s *MemStore) DeleteBook(_ context.Context, shelfID, id uuid.UUID) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    b, ok := s.books[id]
    if !ok || b.ShelfID != shelfID {
        return ErrNotFound
    }
    delete(s.books, id)
    return nil
}

func (s *MemStore) Counts(_ context.Context) (int64, int64, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    return int64(len(s.shelves)), int64(len(s.books)), nil
}
```

## The Service Layer

Business rules live in the service, not in handlers. The one rule of this
domain (a locked shelf refuses deletion) is a perfect example: the
handler never checks `Locked`, it just calls `DeleteShelf` and forwards
whatever comes back.

The constructor is the important line. `NewService(repo Repository)`
declares what the service needs; you never call it yourself. The
dependency container does, with the repository you registered.

```go
// library/service.go
package library

import (
    "context"
    "time"
    "uuid"

    "github.com/kataras/iris/v14"
)

// Service holds the business rules. Its constructor declares what it needs;
// the iris dependency container calls it with the registered Repository.
type Service struct {
    repo Repository
}

func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}

func (s *Service) CreateShelf(ctx context.Context, in ShelfInput) (uuid.UUID, error) {
    shelf := Shelf{
        ID:        uuid.New(),
        Name:      in.Name,
        Locked:    in.Locked,
        CreatedAt: time.Now().UTC(),
    }
    return shelf.ID, s.repo.CreateShelf(ctx, shelf)
}

func (s *Service) Shelves(ctx context.Context) ([]Shelf, error) {
    return s.repo.Shelves(ctx)
}

func (s *Service) Shelf(ctx context.Context, id uuid.UUID) (Shelf, error) {
    return s.repo.Shelf(ctx, id)
}

// DeleteShelf refuses to delete a locked shelf; that rule lives here,
// not in a handler.
func (s *Service) DeleteShelf(ctx context.Context, id uuid.UUID) error {
    shelf, err := s.repo.Shelf(ctx, id)
    if err != nil {
        return err
    }
    if shelf.Locked {
        return ErrShelfLocked
    }
    return s.repo.DeleteShelf(ctx, id)
}

// ForceDeleteShelf bypasses the lock; only the admin API exposes it.
func (s *Service) ForceDeleteShelf(ctx context.Context, id uuid.UUID) error {
    return s.repo.DeleteShelf(ctx, id)
}

func (s *Service) CreateBook(ctx context.Context, shelfID uuid.UUID,
    in BookInput) (uuid.UUID, error) {
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

func (s *Service) Books(ctx context.Context, shelfID uuid.UUID,
    opts iris.PageOptions, filter BookFilter) ([]Book, int64, error) {
    return s.repo.Books(ctx, shelfID, opts, filter)
}

func (s *Service) Book(ctx context.Context, shelfID, id uuid.UUID) (Book, error) {
    return s.repo.Book(ctx, shelfID, id)
}

func (s *Service) UpdateBook(ctx context.Context, shelfID, id uuid.UUID,
    in BookInput) error {
    return s.repo.UpdateBook(ctx, Book{
        ID:      id,
        ShelfID: shelfID,
        Title:   in.Title,
        Author:  in.Author,
        Pages:   in.Pages,
    })
}

func (s *Service) DeleteBook(ctx context.Context, shelfID, id uuid.UUID) error {
    return s.repo.DeleteBook(ctx, shelfID, id)
}

func (s *Service) Stats(ctx context.Context) (Stats, error) {
    shelves, books, err := s.repo.Counts(ctx)
    return Stats{Shelves: shelves, Books: books}, err
}
```

## Controllers: api/shelves.go

An Iris controller in this style is a plain struct with two things: a
constructor (its dependency declaration) and a `Routing(r iris.Router)`
method (its route table). The router calls `Routing` once at startup.

```go
// Package api is the HTTP layer of the Bookshelf service: thin handlers
// that read input, call the service and hand the result to the Context's
// response helpers.
package api

import (
    "github.com/kataras/iris/v14"

    "bookshelf/library"
)

// ShelvesAPI groups the /shelves handlers around their dependencies.
// NewShelvesAPI is registered in the builder's API step; the container
// calls it with the *library.Service registered in Services.
type ShelvesAPI struct {
    svc *library.Service
}

func NewShelvesAPI(svc *library.Service) *ShelvesAPI {
    return &ShelvesAPI{svc: svc}
}

// Routing implements the iris controller convention: the router calls it
// once at startup.
func (api *ShelvesAPI) Routing(r iris.Router) {
    r.Get("/", api.list)
    r.Post("/", api.create)
    r.Get("/{id:uuid}", api.get)
    r.Delete("/{id:uuid}", api.delete)

    // Everything about a shelf's books lives in its own nested router.
    r.Router("/{shelf_id:uuid}/books", NewBooksAPI)
}

func (api *ShelvesAPI) list(ctx iris.Context) {
    shelves, err := api.svc.Shelves(ctx)
    ctx.OK(shelves, err)
}

func (api *ShelvesAPI) create(ctx iris.Context) {
    input, ok := ctx.BindJSON[library.ShelfInput]()
    if !ok {
        return // 400 already written: malformed JSON or failed Validate.
    }

    id, err := api.svc.CreateShelf(ctx, input)
    ctx.Created(id, err) // 201 with {"id": "..."}.
}

func (api *ShelvesAPI) get(ctx iris.Context) {
    id := ctx.Params().GetUUID("id")

    shelf, err := api.svc.Shelf(ctx, id)
    ctx.OK(shelf, err) // ErrNotFound is mapped to 404 centrally.
}

func (api *ShelvesAPI) delete(ctx iris.Context) {
    id := ctx.Params().GetUUID("id")

    err := api.svc.DeleteShelf(ctx, id)
    // The shelf was there and is gone now, so a success is a 204.
    ctx.NoContent(true, err) // 204; 403 for a locked shelf, 404 for a missing one.
}
```

The route paths carry typed macros: `{id:uuid}` rejects anything that is
not a UUID before your handler runs: the macro error handler that
`RouterMiddlewares()` registers renders a 400 `INVALID_ARGUMENT`
("invalid path parameter") response, so the parse error never reaches
your code. The framework parses the value once, so the handlers below
read it with `GetUUID` and have no error branch to write. The most useful macros:

| Macro | Example | Matches |
| --- | --- | --- |
| `{name:string}` | /users/{name:string} | any single path segment |
| `{id:int}` | /orders/{id:int} | integers (also int64, uint64) |
| `{id:uuid}` | /shelves/{id:uuid} | UUIDs (any version), stored as a `uuid.UUID` |
| `{ok:bool}` | /flags/{ok:bool} | true/false, 1/0 |
| `{email:email}` | /verify/{email:email} | e-mail addresses |
| `{path:path}` | /files/{path:path} | the rest of the path, any depth |

Inside handlers, parameters come back through typed accessors with safe
defaults, with no `(value, err)` pairs: `ctx.Params().GetString("id")` returns
`""` when absent, `ctx.Query().GetInt("page")` returns `-1`, and
`ctx.Params().GetUUID("id")` hands back the `uuid.UUID` the `{id:uuid}`
macro already parsed (the zero UUID when the key is absent or does not parse;
`GetDuration`, `GetTime` and `GetIP` follow the same rule). Response
helpers follow one contract, `ctx.Helper(value, err)`: `ctx.OK` renders
200, `ctx.Created` renders 201 (an id value becomes `{"id": ...}`),
`ctx.NoContent(modified, err)` renders 204, or 304 when the call changed
nothing. Every one of them hands a non-nil `err`
to the central error pipeline first, which is why a missing resource is a
404 and a locked shelf a 403 without a branch in the handler.

## Nesting and Guards: api/books.go

Books live under a shelf, so they get their own router mounted at
`/{shelf_id:uuid}/books`; you saw the mounting line in `ShelvesAPI`. The
constructor is resolved by the container, exactly like the parent's.

Two middleware idioms appear here. A group guard (`r.Use`) runs before
every book route: it loads the shelf once, replies 404 when it does not
exist and caches it on the request. A positional guard sits between path
and handler on the write routes only.

```go
// api/books.go
package api

import (
    stdContext "context"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/rest/errors"

    "bookshelf/library"
)

// shelfContextKey caches the guard-loaded shelf on the request so the
// handlers below never load it twice.
const shelfContextKey = "bookshelf.shelf"

// BooksAPI serves /shelves/{shelf_id:uuid}/books.
type BooksAPI struct {
    svc *library.Service
}

func NewBooksAPI(svc *library.Service) *BooksAPI {
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

// requireShelf loads the shelf of the path parameter, replies 404 when it
// does not exist and caches it in the request values for later handlers.
func (api *BooksAPI) requireShelf(ctx iris.Context) {
    shelfID := ctx.Params().GetUUID("shelf_id")

    shelf, err := api.svc.Shelf(ctx, shelfID)
    if errors.HandleError(ctx, err) {
        return // mapped 404, request stopped.
    }

    ctx.Values().Set(shelfContextKey, shelf)
    ctx.Next()
}

// requireUnlocked rejects writes against a locked shelf.
func (api *BooksAPI) requireUnlocked(ctx iris.Context) {
    shelf, _ := ctx.ValueAs[library.Shelf](shelfContextKey)
    if shelf.Locked {
        errors.PermissionDenied.Detail(ctx, "shelf is locked",
            "unlock the shelf before modifying its books")
        return
    }
    ctx.Next()
}

func currentShelf(ctx iris.Context) library.Shelf {
    shelf, _ := ctx.ValueAs[library.Shelf](shelfContextKey)
    return shelf
}

func (api *BooksAPI) list(ctx iris.Context) {
    shelf := currentShelf(ctx)

    ctx.Paginated(func(reqCtx stdContext.Context,
        opts iris.PageOptions, filter library.BookFilter) ([]library.Book, int64, error) {
        return api.svc.Books(reqCtx, shelf.ID, opts, filter)
    })
}

func (api *BooksAPI) create(ctx iris.Context) {
    input, ok := ctx.BindJSON[library.BookInput]()
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

    input, ok := ctx.BindJSON[library.BookInput]()
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
```

Guards are methods on the API struct, so they see the same injected
service as the handlers. The middleware contract is plain: call
`ctx.Next()` to continue, or write a response (the `errors` senders stop
the request) to end it.

## Pagination: One Call

The `list` handler above is complete; `ctx.Paginated` does everything:
it reads `?page=` and `?size=` from the query and an optional JSON filter
body, calls your collector, and renders the standard page envelope. Your
collector receives the parsed options and returns the window plus the
total count.

```sh
curl 'http://localhost:8080/api/shelves/<id>/books?page=2&size=2'
```

```json
{
    "current_page": 2,
    "page_size": 2,
    "total_pages": 3,
    "total_items": 5,
    "has_next_page": true,
    "filter": {"author": ""},
    "items": [{"id": "...", "title": "Volume 3", "...": "..."}]
}
```

When you need the pieces separately (for example to pass options deeper
into a repository that returns `*iris.Page[T]` itself), decompose it:
`ctx.BindPagination[F]()` reads options and filter, and
`context.NewPage(opts, filter, items, total)` builds the envelope. Page sizes
are clamped by `pagination.DefaultPageSize` and `pagination.MaxPageSize`
(package `core/pkg/pagination`); set them at start-up with
`iris.Patches().Context().SetDefaultPageSize(n)` and `SetMaxPageSize(n)`.

A filter body belongs on a QUERY request (RFC 10008): safe and cacheable
like GET, but with a body. Register the list for it with
`r.Handle("GET QUERY", "/", api.list)` and send the filter as JSON:

```sh
curl -X QUERY 'http://localhost:8080/api/shelves/<id>/books?page=1&size=2' \
  -H 'Content-Type: application/json' -d '{"author":"Kurt Vonnegut"}'
```

On the client side, `client.List[T]` fetches one page over QUERY and
`client.Items[T]` walks every page.

## One Place for Errors

Every helper you have seen (`ctx.OK`, `ctx.Created`, `ctx.NoContent`,
`errors.HandleError`) sends its non-nil error through the same pipeline. The
pipeline is configured once, with `errors.NewOptions()` handed to the
builder's `Errors` step (you will see it in `main.go`):

```go
errors.NewOptions().
    MapErrors(errors.NotFound, library.ErrNotFound).
    MapErrors(errors.PermissionDenied, library.ErrShelfLocked)
```

From then on, classification is automatic: `errors.Is` decides whether a
returned error (however deeply wrapped) is one of the mapped sentinels.
Unmapped errors become a generic 500: the message and detail sent to the
client are constants, the real error is logged server-side. Clients never
see your SQL.

The canonical codes and their status lines:

| Code | HTTP status |
| --- | --- |
| `errors.InvalidArgument` | 400 |
| `errors.Unauthenticated` | 401 |
| `errors.PermissionDenied` | 403 |
| `errors.NotFound` | 404 |
| `errors.AlreadyExists` | 409 |
| `errors.ResourceExhausted` | 429 |
| `errors.Internal` | 500 |

A mapped 404 renders like this on the wire:

```json
{
    "http_error_code": {"canonical_name": "NOT_FOUND", "status": 404},
    "message": "library: not found"
}
```

And a failed `Validate` renders the field errors individually; this is
what the `validation` builders in `library.go` buy you:

```json
{
    "http_error_code": {"canonical_name": "INVALID_ARGUMENT", "status": 400},
    "message": "validation failure",
    "detail": "fields were invalid",
    "validation": [
        {"field": "title", "value": "",
         "reason": "must not be empty, must be between 1 and 200 characters"},
        {"field": "pages", "value": 0, "reason": "must be positive"}
    ]
}
```

For direct use inside handlers and guards, each code is also a sender:
`errors.InvalidArgument.Err(ctx, err)`, `.Message(ctx, "text")`,
`.Detail(ctx, "message", "detail")`, `.LogErr(ctx, err)` (respond
generically, log the cause). And when a handler wants to add call-site
context to a mapped error, wrap-and-handle in one call:

```go
// Renders the mapped 404 with message "plan with id: 42".
errors.HandleErrorf(ctx, err, "plan with id: %s", id)
```

Errors from your database layer can join the map too. If you use
`github.com/kataras/pg`, one blank import restores the built-in
PostgreSQL mapping (no-rows to 404, duplicate key to 409):

```go
import _ "github.com/kataras/iris/v14/rest/errors/pgerrors"
```

## The Admin Party: api/admin.go

The admin API is a third controller. Nothing in it knows about
authentication: protection is applied where it is mounted.

```go
// api/admin.go
package api

import (
    "github.com/kataras/iris/v14"

    "bookshelf/library"
)

// AdminAPI is mounted behind basic auth (see the builder's API step).
type AdminAPI struct {
    svc *library.Service
}

func NewAdminAPI(svc *library.Service) *AdminAPI {
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
```

### JWT in one look

Basic auth fits an internal admin surface. For user-facing APIs, the
`auth` package provides a generic JWT SDK; this is its shape (not wired
into Bookshelf; full setup lives in the auth reference):

```go
// Configuration (keys, cookie, headers) usually loads from YAML:
sso := auth.Must[Customer](cfg.Auth)

// Registered as a service, applied per router:
r.Use(sso.VerifyHandler())

// Claims check variant:
r.Use(sso.VerifyHandler(func(ctx context.Context, u Customer) error {
    if !u.Active {
        return fmt.Errorf("account disabled")
    }
    return nil
}))

// Inside handlers:
user := auth.GetUser[Customer](ctx)
```

See the [Authentication and Authorization](https://iris-go.com/docs/authentication-authorization/)
chapter for the complete walkthrough.

## Configuration: Env First, Secrets Never in YAML

One struct holds the whole configuration. Iris's own options embed inline,
so a single YAML document configures both the framework and your app.
The admin password is deliberately excluded from YAML (`yaml:"-"`): the
file is committed, the environment is not.

```go
// main.go (part 1 of 2)
// Bookshelf is the Iris getting-started service: shelves and books behind
// a production-shaped API. See TUTORIAL.md in the Iris repository.
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/basicauth"
    "github.com/kataras/iris/v14/rest/errors"

    "bookshelf/api"
    "bookshelf/library"
)

// Config is the whole application configuration: the fields every service
// shares come from iris.BaseConfiguration, inlined, and the bookshelf's own
// settings sit beside them in the same YAML document. AdminPassword is
// deliberately yaml:"-": secrets come from the environment, never from a
// committed file.
type Config struct {
    iris.BaseConfiguration `yaml:",inline"`

    Env           string `yaml:"Env"`
    Developer     string `yaml:"Developer"`
    AdminUser     string `yaml:"AdminUser"`
    AdminPassword string `yaml:"-"`
}

// LoadConfig reads the document from the BOOKSHELF_CONFIG environment
// variable or the config.yml file, lets HOST, PORT and ALLOW_ORIGIN override
// it, then reads the secrets from the environment.
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
```

The committed `config.yml`:

```yaml
# Bookshelf configuration. Committed settings only — no secrets here:
# the admin password comes from the ADMIN_PASSWORD environment variable.
Host: ""            # empty means every interface
Port: 8080
AllowOrigin: none
EnableCompression: true
RequestLog: false
Env: development
Developer: bookshelf
AdminUser: admin
Iris:
  Name: bookshelf
  LogLevel: info
```

`iris.LoadConfiguration[T]` is environment-first: it reads
`BOOKSHELF_CONFIG` when that is set and `config.yml` otherwise, so one
container variable can carry the entire configuration while a local file
serves development. The variable may hold either the YAML document *or* a
path to it, a value naming an existing file being read as a file, so the
same binary works whether the deployment mounts the configuration or
passes it inline.

`iris.BaseConfiguration` is the half every service shares: `Host`, `Port`,
`AllowOrigin`, `EnableCompression`, `RequestLog` and the framework `Iris`
options. Embedding it inline puts those keys at the top level of the same
document as the bookshelf's own, and it brings a `BindEnv` method that
`LoadConfiguration` calls through the pointer *after* decoding, so `HOST`,
`PORT`, `HTTP_PLATFORM_PORT` and `ALLOW_ORIGIN` override the file rather
than the other way round. `HTTP_PLATFORM_PORT` wins over `PORT`, because a
hosting platform that picks the port passes it there. `cfg.Addr()` joins
host and port for `app.Listen` and brackets an IPv6 host correctly, which
is why there is no `Addr` field of our own any more. Passing a partial
`iris.Options` under `Iris:` is safe; unset fields inherit the framework
defaults.

One consequence to know before you run it: the document is required. A
deployment with neither `BOOKSHELF_CONFIG` nor `config.yml` fails at
startup instead of serving on defaults nobody chose.

> Never commit credentials, key material or connection strings, even in
> private repositories. The environment (or a secret manager that feeds
> it) is their only home.

## Wiring It Together: iris.NewBuilder

Everything assembles in one expression. The builder is a type-state chain:
each step returns the interface of the next, so the compiler enforces the
order: you cannot register APIs before services, or CORS after logging.

```go
// main.go (part 2 of 2)

func main() {
    cfg, err := LoadConfig()
    if err != nil {
        log.Fatal(err)
    }

    app := newApp(cfg, library.NewMemStore())
    app.Listen(cfg.Addr())
}

// newApp assembles the whole application. The repository arrives as a
// parameter: main passes the real store, tests pass fakes.
func newApp(cfg Config, repo library.Repository) *iris.Application {
    return iris.NewBuilder(cfg.Iris).
        Prefix("/api").
        AllowOrigin(cfg.AllowOrigin).
        Compression(cfg.EnableCompression).
        LogRequests(cfg.RequestLog).
        Health(true, cfg.Env, cfg.Developer).
        Errors(errors.NewOptions().
            MapErrors(errors.NotFound, library.ErrNotFound).
            MapErrors(errors.PermissionDenied, library.ErrShelfLocked)).
        Deferrables(func() { log.Println("bookshelf: goodbye") }).
        Services(repo, library.NewService).
        RouterMiddlewares().
        Middlewares().
        API("/shelves", api.NewShelvesAPI).
        API("/admin", basicauth.Default(map[string]string{
            cfg.AdminUser: cfg.AdminPassword,
        }), api.NewAdminAPI).
        Build()
}
```

Step by step:

| Step | What it does |
| --- | --- |
| `NewBuilder(cfg.Iris)` | applies your options over the defaults |
| `Prefix("/api")` | mounts everything below under /api |
| `AllowOrigin` | CORS; `"none"` disables, `"*"` or a list enables |
| `Compression(bool)` | response/request compression negotiation |
| `LogRequests(bool)` | tuned access log to access_log.json + stdout |
| `Health(true, env, dev)` | GET /health at the root (not under the prefix) |
| `Errors(opts)` | installs the central error map per request |
| `Deferrables(fns...)` | closers that run on application shutdown |
| `Services(deps...)` | registers values and constructors for injection |
| `RouterMiddlewares()` | recovery, macro errors, Server header, and yours |
| `Middlewares()` | matched-route middlewares (yours) |
| `API(path, ctors...)` | mounts a controller; extra args are middleware |
| `Build()` | resolves everything; panics on misconfiguration |

Three details are load-bearing. `Services(repo, library.NewService)` mixes
a value and a constructor: the container calls `NewService` with the
registered repository because the parameter types match, the same
resolution that later calls `NewShelvesAPI(svc)`. Anything registered with
a `Close` method becomes a shutdown closer automatically. And a service
implementing `iris.Initializer` (`Init(ctx context.Context) error`) is
initialized inside `Build()`, after injection and before serving, the
place for cache warm-up, with `Deferrables` as its mirror image at
shutdown. Note the two middleware steps passed empty: the interface chain
walks through them even when you have nothing to add.

The access log defaults are opinionated (JSON lines, request/response
bodies included). When they do not fit, build your own and pass it
through `LogRequestsWith(ac)` at the same position; add `access_log.json`
to `.gitignore` either way.

## Shutdown Without Tears

On Ctrl+C (SIGINT/SIGTERM) the server stops accepting connections and
drains in-flight requests gracefully. Closers (your `Deferrables`, plus
every registered service with a `Close` method) run when the application
shuts down through `app.Shutdown(ctx)`. For extra work at interrupt time,
register it explicitly:

```go
iris.RegisterOnInterrupt(func() {
    log.Println("interrupt received")
})
```

## Run It

```sh
ADMIN_PASSWORD=change-me go run .
```

Create, read, and trip the rules:

```sh
# 201 {"id":"6f36b06d-..."}
curl -X POST localhost:8080/api/shelves \
  -H 'Content-Type: application/json' -d '{"name":"sci-fi"}'

# 201 — add a book (use the shelf id from above)
curl -X POST localhost:8080/api/shelves/$SHELF/books \
  -H 'Content-Type: application/json' \
  -d '{"title":"Dune","author":"Frank Herbert","pages":412}'

# 200 — paginated listing
curl "localhost:8080/api/shelves/$SHELF/books?page=1&size=10"

# 400 — validation, field-level payload
curl -X POST localhost:8080/api/shelves \
  -H 'Content-Type: application/json' -d '{"name":""}'

# 404 — the central map at work
curl localhost:8080/api/shelves/00000000-0000-0000-0000-000000000000

# 401 then 200
curl localhost:8080/api/admin/stats
curl -u admin:change-me localhost:8080/api/admin/stats

# 200 — health, at the root
curl localhost:8080/health
```

Create a locked shelf (`{"name":"archive","locked":true}`) and POST a book
to it: 403 with `"canonical_name": "PERMISSION_DENIED"` and the detail you
wrote in the guard.

## Test It in Milliseconds

This is where the architecture pays off. `newApp(cfg, repo)` takes its
storage as a parameter, so a test builds the real application (real
router, real middleware, real error map) with any repository it likes,
and `httptest.New` drives it in memory. No ports, no network, no sleeps.

```go
// main_test.go
package main

import (
    stdContext "context"
    "fmt"
    "testing"
    "uuid"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/httptest"

    "bookshelf/library"
)

func testConfig() Config {
    return Config{
        BaseConfiguration: iris.BaseConfiguration{
            Iris:        iris.DefaultOptions,
            AllowOrigin: "none",
        },
        Env:           "test",
        Developer:     "tests",
        AdminUser:     "admin",
        AdminPassword: "test-password",
    }
}

func TestShelvesCRUD(t *testing.T) {
    app := newApp(testConfig(), library.NewMemStore())
    e := httptest.New(t, app)

    // Create.
    id := e.POST("/api/shelves").
        WithJSON(library.ShelfInput{Name: "sci-fi"}).
        Expect().Status(httptest.StatusCreated).
        JSON().Object().Value("id").String().NotEmpty().Raw()

    // Read back.
    e.GET("/api/shelves/"+id).Expect().Status(httptest.StatusOK).
        JSON().Object().HasValue("name", "sci-fi")

    // Validation: ctx.BindJSON runs ShelfInput.Validate -> 400.
    e.POST("/api/shelves").
        WithJSON(library.ShelfInput{Name: ""}).
        Expect().Status(httptest.StatusBadRequest)

    // The central error map turns library.ErrNotFound into a 404.
    e.GET("/api/shelves/" + uuid.New().String()).
        Expect().Status(httptest.StatusNotFound)

    // Delete.
    e.DELETE("/api/shelves/" + id).Expect().Status(httptest.StatusNoContent)
}

func TestBooksPaginationAndGuards(t *testing.T) {
    store := library.NewMemStore()
    app := newApp(testConfig(), store)
    e := httptest.New(t, app)

    shelfID := e.POST("/api/shelves").
        WithJSON(library.ShelfInput{Name: "history"}).
        Expect().Status(httptest.StatusCreated).
        JSON().Object().Value("id").String().NotEmpty().Raw()

    for i := 1; i <= 5; i++ {
        e.POST("/api/shelves/"+shelfID+"/books").
            WithJSON(library.BookInput{
                Title:  fmt.Sprintf("Volume %d", i),
                Author: "Herodotus",
                Pages:  100 * i,
            }).
            Expect().Status(httptest.StatusCreated)
    }

    // Page 2 with 2 items per page: items 3 and 4 of 5.
    page := e.GET("/api/shelves/"+shelfID+"/books").
        WithQuery("page", 2).WithQuery("size", 2).
        Expect().Status(httptest.StatusOK).JSON().Object()
    page.HasValue("current_page", 2)
    page.HasValue("total_items", 5)
    page.Value("items").Array().Length().IsEqual(2)

    // A locked shelf rejects writes with 403 but keeps serving reads.
    lockedID := e.POST("/api/shelves").
        WithJSON(library.ShelfInput{Name: "archive", Locked: true}).
        Expect().Status(httptest.StatusCreated).
        JSON().Object().Value("id").String().NotEmpty().Raw()

    e.POST("/api/shelves/"+lockedID+"/books").
        WithJSON(library.BookInput{Title: "x", Author: "y", Pages: 1}).
        Expect().Status(httptest.StatusForbidden)
    e.GET("/api/shelves/" + lockedID + "/books").
        Expect().Status(httptest.StatusOK)

    // Unknown shelf: the group guard replies 404 before any handler runs.
    e.GET("/api/shelves/" + uuid.New().String() + "/books").
        Expect().Status(httptest.StatusNotFound)
}

func TestAdminRequiresCredentials(t *testing.T) {
    app := newApp(testConfig(), library.NewMemStore())
    e := httptest.New(t, app)

    e.GET("/api/admin/stats").Expect().Status(httptest.StatusUnauthorized)

    e.GET("/api/admin/stats").WithBasicAuth("admin", "test-password").
        Expect().Status(httptest.StatusOK).
        JSON().Object().HasValue("shelves", 0)
}

// failingRepo overrides one method of a working repository; everything else
// is inherited. Interface embedding makes one-method fakes one-liners.
type failingRepo struct {
    library.Repository
}

func (failingRepo) Shelves(stdContext.Context) ([]library.Shelf, error) {
    return nil, fmt.Errorf("pg: connection refused on 10.0.0.3:5432")
}

// Internal error details must never reach the client.
func TestInternalErrorsDoNotLeak(t *testing.T) {
    app := newApp(testConfig(), failingRepo{library.NewMemStore()})
    e := httptest.New(t, app)

    body := e.GET("/api/shelves").
        Expect().Status(httptest.StatusInternalServerError).
        JSON().Object()
    body.HasValue("message", "server error")
    body.NotContainsValue("10.0.0.3") // the connection string stays server-side.
}
```

```sh
go test ./...
```

The last test is the one to internalize: `failingRepo` embeds the
`Repository` interface and overrides a single method. One type, five
lines, and you have proven that a database outage renders as a generic
500 with the connection string kept server-side. `httptest.New` silences
application logs by default; pass `httptest.Debug(true)` when you want to
see the traffic.

## Production Checklist

Everything below is one or two lines away from the code you already have.

| Concern | What to do |
| --- | --- |
| Request body limits | set `Context.BodyMaxMemory` in `iris.Options` (e.g. `1 * iris.MB`), or per route with `bodylimit.New(1 * iris.MB)` |
| Rate limiting | `app.Use(rate.Limit(rate.Every(time.Second), 10, rate.Options{}))`; the builder pre-wires the JSON 429 response |
| CORS | replace `"none"` with your exact origins, never `"*"` with credentials |
| Access log | `LogRequests(true)` (or `LogRequestsWith`); rotate and gitignore `access_log.json` |
| Recovery | already installed by the builder; do not add a second one |
| Secure cookies | pass `iris.CookieSecure` and `iris.CookieHTTPOnly` options wherever you set cookies |
| Log level | `LogLevel: info` in production; `debug` locally |
| Health endpoint | `/health` reports env and developer; mount your own instead if that is too much exposure |
| TLS | `app.Run(iris.ServerOptions{Addr: ":443", CertFile: "...", KeyFile: "..."})` |

## Where to Go Next

- [The live AI wiki](https://codewiki.google/github.com/kataras/iris):
  free, generated by Google from the Iris source and kept in sync with
  it; browse the architecture or ask its chat anything about the
  codebase in plain language.
- [Plexon AI](https://plexon.ai) is a desktop assistant built by the
  author of Iris. It ships the `skill/iris/` material listed below built in,
  and its Software Developer persona turns that skill on.
- [MIGRATION.md](MIGRATION.md): upgrading an Iris v12 application.

The book chapters are free to read at
[iris-go.com/docs](https://iris-go.com/docs/):

- [Response Handling](https://iris-go.com/docs/response-handling/):
  every `rest` helper, including the read variants not shown here.
- [Error Handling](https://iris-go.com/docs/error-handling/).
  The full error pipeline: renderers, collectors, RFC 7807 output.
- [Core Concepts](https://iris-go.com/docs/core-concepts/):
  the builder and controller patterns at application scale.
- [Authentication and Authorization](https://iris-go.com/docs/authentication-authorization/):
  the JWT auth SDK end to end.
- [Testing](https://iris-go.com/docs/testing/).
  More `httptest` patterns: sessions, auth, websockets.
- [The Iris book](https://iris-go.com/book/): 24 chapters,
  a preface and an epilogue.
