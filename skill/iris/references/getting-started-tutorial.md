# Getting Started Tutorial

The condensed form of the framework's
[TUTORIAL.md](https://github.com/kataras/iris/blob/main/TUTORIAL.md):
the recommended v14 stack as one vertical slice: builder, constructor-
injected controllers, rest helpers, central error mapping, configuration
and in-memory HTTP tests. The full tutorial contains the complete runnable
Bookshelf program (shelves hold books; a locked shelf freezes its books);
this reference keeps the load-bearing patterns.

Project shape (one direction of dependency; the domain imports no HTTP):

```
bookshelf/
├── main.go                # config, newApp (builder chain), main
├── main_test.go           # httptest suite with fake repositories
├── api/                   # controllers: thin handlers
└── library/               # entities, inputs, sentinel errors,
                           # Repository interface, Service
```

## The Handler Shape

Every handler follows one contract: read, call the service, hand value
and error to a rest helper:

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

`ctx.BindJSON[T]` runs the type's `Validate() error` method after
decoding (implementing it is the opt-in; use `ctx.BindJSONOptional[T]`
when an empty body is acceptable). `ctx.OK`/`Created`/`NoContent` route
non-nil errors through the central pipeline.

## Domain Inputs and Sentinels

```go
var (
    ErrNotFound    = errors.New("library: not found")
    ErrShelfLocked = errors.New("library: shelf is locked")
)

type ShelfInput struct {
    Name   string `json:"name"`
    Locked bool   `json:"locked"`
}

func (in ShelfInput) Validate() error {
    return validation.Join(
        validation.String("name", in.Name).NotEmpty().Length(1, 100),
    )
}
```

Sentinels keep status codes out of the domain; `validation.String/Number`
build field-level error payloads rendered as structured JSON.

## The Storage Seam

```go
type Repository interface {
    Shelf(ctx context.Context, id uuid.UUID) (Shelf, error) // ErrNotFound
    Books(ctx context.Context, shelfID uuid.UUID, opts iris.PageOptions,
        filter BookFilter) ([]Book, int64, error)
    // ...
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }
```

Services take plain `context.Context`; `iris.Context` implements it, so
handlers pass `ctx` straight through and cancellation reaches the
database. The interface is the test seam; see Testing below.

## Controllers, Nesting, Guards

```go
func NewBooksAPI(svc *library.Service) *BooksAPI { return &BooksAPI{svc: svc} }

func (api *BooksAPI) Routing(r iris.Router) {
    r.Use(api.requireShelf) // group guard: 404 for unknown shelves.

    r.Get("/", api.list)
    r.Post("/", api.requireUnlocked, api.create) // positional guard.
    r.Get("/{id:uuid}", api.get)
}

func (api *BooksAPI) requireShelf(ctx iris.Context) {
    shelfID := ctx.Params().GetUUID("shelf_id")

    shelf, err := api.svc.Shelf(ctx, shelfID)
    if errors.HandleError(ctx, err) {
        return // mapped 404, request stopped.
    }

    ctx.Values().Set(shelfContextKey, shelf)
    ctx.Next()
}
```

Nested mounting inside a parent's `Routing`:
`r.Router("/{shelf_id:uuid}/books", NewBooksAPI)`, where the constructor
is resolved by the dependency container. Guards are API-struct methods, so
they see the injected services. Cached request state travels via
`ctx.Values().Set` and `ctx.ValueAs[T]`.

## Pagination

```go
ctx.Paginated(func(reqCtx stdContext.Context,
    opts iris.PageOptions, filter library.BookFilter) ([]library.Book, int64, error) {
    return api.svc.Books(reqCtx, shelf.ID, opts, filter)
})
```

Reads `?page=&size=` and an optional JSON filter body, renders the
`iris.Page[T]` envelope (`current_page`, `page_size`, `total_pages`,
`total_items`, `has_next_page`, `filter`, `items`). Manual decomposition:
`ctx.BindPagination[F]` + `iris.NewPage`.

## Central Error Mapping

Configured once, in the builder's `Errors` step:

```go
errors.NewOptions().
    MapErrors(errors.NotFound, library.ErrNotFound).
    MapErrors(errors.PermissionDenied, library.ErrShelfLocked)
```

Classification uses `errors.Is` over wrapped chains. Unmapped errors
render a generic 500 (real error logged server-side, never sent).
Call-site enrichment: `errors.HandleErrorf(ctx, err, "plan %s", id)`.
Direct senders: `errors.InvalidArgument.Err/.Message/.Detail/.LogErr`.
PostgreSQL (kataras/pg) mapping is opt-in:
`import _ "github.com/kataras/iris/v14/rest/errors/pgerrors"`.

## The Builder

```go
func newApp(cfg Config, repo library.Repository) *iris.Application {
    return iris.NewBuilder(cfg.Iris).
        Prefix("/api").
        AllowOrigin(cfg.AllowOrigin). // "none" disables CORS.
        Compression(cfg.EnableCompression).
        LogRequests(cfg.RequestLog). // or LogRequestsWith(customAccessLog)
        Health(true, cfg.Env, cfg.Developer). // GET /health at the ROOT.
        Errors(errors.NewOptions().
            MapErrors(errors.NotFound, library.ErrNotFound).
            MapErrors(errors.PermissionDenied, library.ErrShelfLocked)).
        Deferrables(func() { log.Println("bookshelf: goodbye") }).
        Services(repo, library.NewService). // values + constructors mix.
        RouterMiddlewares().  // recovery etc. pre-wired; add global ones.
        Middlewares().        // matched-route middlewares.
        API("/shelves", api.NewShelvesAPI).
        API("/admin", basicauth.Default(map[string]string{
            cfg.AdminUser: cfg.AdminPassword,
        }), api.NewAdminAPI). // extra args before the ctor = middleware.
        Build()
}
```

Constraints worth remembering: the chain order is compiler-enforced;
`Services` values with a `Close` method auto-register as shutdown
closers; services implementing `iris.Initializer`
(`Init(ctx context.Context) error`) run once inside `Build()` after
injection, the startup mirror of `Deferrables`. `RouterMiddlewares()`
and `Middlewares()` must appear even when empty.

## Configuration

```go
type Config struct {
    iris.BaseConfiguration `yaml:",inline"` // Host, Port, AllowOrigin,
    // EnableCompression, RequestLog and the framework Iris options.

    Env           string `yaml:"Env"`
    Developer     string `yaml:"Developer"`
    AdminUser     string `yaml:"AdminUser"`
    AdminPassword string `yaml:"-"` // secrets come from the env only.
}

cfg, err := iris.LoadConfiguration[Config]("BOOKSHELF_CONFIG", "config.yml")
// ...then cfg.AdminPassword = os.Getenv("ADMIN_PASSWORD"), fail fast when empty.
app.Listen(cfg.Addr())
```

Load order: `BOOKSHELF_CONFIG` when set (the whole YAML document, **or** a
path to it) → `config.yml` → `BindEnv`, promoted from the embedded struct and
called by `LoadConfiguration` through the pointer, so `HOST`, `PORT`,
`HTTP_PLATFORM_PORT` and `ALLOW_ORIGIN` win over the document
(`HTTP_PLATFORM_PORT` over `PORT`) → secrets from the environment, fail fast
when missing. There is no `Addr` field: `cfg.Addr()` joins `Host` and `Port`
and brackets an IPv6 host. Partial `iris.Options` under the `Iris:` key merge
with the defaults. The document is required: with neither the variable nor the
file present, startup fails instead of falling back to silent defaults. Full
treatment in
[configuration-and-deployment.md](./configuration-and-deployment.md#loadconfiguration-the-whole-service-not-just-the-framework).

## Testing

`newApp(cfg, repo)` takes the seam as a parameter, so tests build the
real application with any repository and drive it in memory:

```go
func TestShelvesCRUD(t *testing.T) {
    app := newApp(testConfig(), library.NewMemStore())
    e := httptest.New(t, app)

    id := e.POST("/api/shelves").
        WithJSON(library.ShelfInput{Name: "sci-fi"}).
        Expect().Status(httptest.StatusCreated).
        JSON().Object().Value("id").String().NotEmpty().Raw()

    e.GET("/api/shelves/"+id).Expect().Status(httptest.StatusOK).
        JSON().Object().HasValue("name", "sci-fi")

    e.GET("/api/shelves/" + uuid.New().String()).
        Expect().Status(httptest.StatusNotFound) // central map at work.
}

// One-method fake via interface embedding: everything else inherited.
type failingRepo struct{ library.Repository }

func (failingRepo) Shelves(stdContext.Context) ([]library.Shelf, error) {
    return nil, fmt.Errorf("pg: connection refused on 10.0.0.3:5432")
}

func TestInternalErrorsDoNotLeak(t *testing.T) {
    app := newApp(testConfig(), failingRepo{library.NewMemStore()})
    e := httptest.New(t, app)

    body := e.GET("/api/shelves").
        Expect().Status(httptest.StatusInternalServerError).
        JSON().Object()
    body.HasValue("message", "server error")
    body.NotContainsValue("10.0.0.3")
}
```

`httptest.New` runs in memory (no ports), silences app logs by default;
options: `httptest.Debug(true)`, `httptest.Strict(true)`,
`httptest.LogLevel("debug")`.

## Production Checklist

Body limits (`Options.Context.BodyMaxMemory`, e.g. `1 * iris.MB`, or
`bodylimit.New`) · rate limiting (`rate.Limit(rate.Every(time.Second), 10, rate.Options{})`)
· exact CORS origins · access log rotation (gitignore `access_log.json`)
· recovery pre-installed by the builder (do not double-register) · cookie
flags (`iris.CookieSecure`, `iris.CookieHTTPOnly`) · TLS via
`app.Run(iris.ServerOptions{Addr: ":443", CertFile: ..., KeyFile: ...})`.

See also: [rest-helpers.md](./rest-helpers.md),
[error-handling.md](./error-handling.md),
[architecture-and-patterns.md](./architecture-and-patterns.md),
[authentication-sdk.md](./authentication-sdk.md),
[testing-guide.md](./testing-guide.md).

## Source Files

- `TUTORIAL.md` (repository root) - the complete runnable Bookshelf program
- `iris_builder.go` - NewBuilder step interfaces: Prefix through Build, Initializer, LogRequestsWith
- `context/context_bind.go`, `context/page.go` - BindJSON/BindJSONOptional, Validatable, PageOptions, Page[T], Paginated, BindPagination
- `rest/json.go`, `rest/ok.go`, `rest/created.go`, `rest/no_content.go` - response helpers and type aliases
- `rest/errors/errors.go`, `rest/errors/options.go` - canonical codes, HandleError(f), NewOptions, MapErrors
- `rest/errors/pgerrors/pgerrors.go` - opt-in PostgreSQL error mapping
- `rest/errors/validation/` - String/Number/Slice field validation builders
- `core/router/router.go`, `core/router/router_builder_dep.go` - Router interface, DI controller registration
- `core/router/macro/macros.go` - path parameter macro types
- `context/context_request.go`, `context/encoding/entries.go` - Params(), Query(), typed accessors
- `options.go`, `context/context_options.go` - iris.Options, Bind, defaults merge, BodyMaxMemory
- `middleware/basicauth/basicauth.go`, `middleware/bodylimit/`, `middleware/rate/` - checklist middlewares
- `httptest/httptest.go`, `httptest/status.go` - in-memory test client and options
