# Built-in Controllers

The `github.com/kataras/iris/v14/controller` package tree ships ready-to-register router controllers. They implement the same `Routing(r iris.Router)` interface as user controllers and register the same way: `app.Router("/path", ctrl)` or the builder's `.API("/path", ctrl)` step.

## Controller Interface Recap

```go
// core/router
type Controller interface {
    Routing(child Router)
}
```

Registration:

```go
// Direct registration
app.Router("/static", fileserver.New("./assets", fileserver.DefaultOptions))

// Via the builder API step
iris.NewBuilder().
    // ... builder steps ...
    API("/static", fileserver.New("./assets", fileserver.DefaultOptions)).
    Build().
    Listen(":8080")
```

A plain function `func(r iris.Router)` and `router.Routing(func(r router.Router){...})` also satisfy the controller slot. See architecture-and-patterns.md for constructor injection and the full list of accepted controller forms.

## Packages

| Package | Import path | Purpose |
|---------|-------------|---------|
| File Server | `controller/fileserver` | Static files from a directory or embedded FS, directory listing, SPA, attachments, favicon, WebDAV |
| MVC | `controller/mvc` | Generic CRUD controllers with generated routes, DI, views, WebSocket controllers |
| API Graph | `controller/apigraph` | Interactive D3.js visualization of the application's routes |
| Sitemap | `controller/sitemap` | Auto-generated `/sitemap.xml` from registered static routes |

## File Server (`controller/fileserver`)

```go
func New(fileSystemOrDirectory any, options Options) router.Controller
```

`fileSystemOrDirectory` accepts: `string` (directory path), `http.FileSystem`, `embed.FS`, `fs.FS`.

```go
import "github.com/kataras/iris/v14/controller/fileserver"

app.Router("/static", fileserver.New("./assets", fileserver.Options{
    IndexName: "index.html",
    ShowList:  true,
}))
```

With an embedded file system:

```go
//go:embed assets
var embeddedFiles embed.FS

app.Router("/static", fileserver.New(embeddedFiles, fileserver.Options{
    IndexName: "index.html",
}))
```

A string directory goes through `http.Dir`, which follows a symbolic link inside it that points outside. `fileserver.New(root.FS(), opts)` with `root, err := os.OpenRoot(dir)` confines it (a link that leaves the root answers 404), at the cost of holding the directory open for the life of the process and of no longer serving a link that leaves the directory on purpose. See `security-hardening.md`.

### Options

| Field | Type | Description |
|-------|------|-------------|
| `IndexName` | `string` | Index file name, defaults to `/index.html`. Requests ending in it redirect to the directory path |
| `PushTargets` | `map[string][]string` | Files to HTTP/2-push per request path |
| `PushTargetsRegexp` | `map[string]*regexp.Regexp` | Like `PushTargets` but regexp-matched against all files under a directory. `MatchCommonAssets` is a ready regexp for js/css/ico/png/ttf/svg/webp/gif |
| `Cache` | `DirCacheOptions` | In-memory cache and pre-compression: `Enable`, `CompressMinSize`, `CompressIgnore` (regexp, see `MatchImagesAssets`), `Encodings` (gzip, deflate, br, snappy, s2, zstd), `Verbose` (0-2) |
| `ShowList` | `bool` | List directory contents when `IndexName` is missing |
| `DirList` | `DirListFunc` | Custom listing renderer when `ShowList` is true. Default is `DirListRich(...)`; `DirList` is the plain implementation |
| `ShowHidden` | `bool` | Include hidden files/dirs in listings |
| `Attachments` | `Attachments` | Serve as downloads: `Enable`, `Limit` (bytes/sec), `Burst`, `NameFunc` |
| `AssetValidator` | `func(ctx iris.Context, name string) bool` | Per-request resource filter |
| `SPA` | `bool` | Serve the index file for not-found files instead of 404. Requires `IndexName` |

`fileserver.DefaultOptions` is a ready Options value (index `/index.html`, cache disabled, rich dir listing renderer, no listing).

### Related helpers

```go
// Plain handler (no controller registration), same fs types and Options.
func Handler(fileSystemOrDirectory any, options Options) context.Handler

// Cached favicon at GET|HEAD /favicon.ico.
app.Router("/", fileserver.Favicon("./path/to/favicon.ico"))

// WebDAV file management (wraps a golang.org/x/net/webdav handler,
// registers all WebDAVMethods on /{p:path}).
app.Router("/", fileserver.WebDAV(webdavHandler))

// Strip a route prefix before passing to a file server Handler.
func StripPrefix(prefix string, h context.Handler) context.Handler
```

## MVC (`controller/mvc`)

A generic CRUD controller. Implement the `mvc.Controller` interface on a struct and register it with `mvc.UseController`:

```go
type Controller[Context handler.ContextType, Request any, Response any,
    ID iris.IDType, Filter any, Total iris.CountType] interface {
    Create(Context, Request) (Response, error)
    Get(Context, ID) (Response, error)
    Update(Context, ID, Request, ...string) (*Response, bool, error)
    Delete(Context, ID) (bool, error)
    List(ctx Context, filter Filter, limit int, offset int) ([]Response, Total, error)
}
```

`iris.IDType` is `string | int | int64 | uint64 | uuid.UUID`. `iris.CountType` is any integer or float.

### Generated routes

For `app.Router("/stores", mvc.UseController[*StoreController]())` with `Store` as the Response type, the path parameter is derived from the model name and its id json field (e.g. `store_id`):

| Method, path | Controller method | Success response |
|--------------|-------------------|------------------|
| `POST /stores` | `Create` | 201, JSON body |
| `GET /stores/{store_id}` | `Get` | 200, JSON body |
| `PUT /stores/{store_id}?columns=a&columns=b` | `Update` | 200 if `*Response` non-nil, else 204 (true) or 304 (false) |
| `DELETE /stores/{store_id}` | `Delete` | 204 (true) or 304 (false) |
| `POST /stores/list`, `QUERY /stores/list` | `List` | 200, `iris.Page` JSON |

Errors returned by any method are handled by the `rest/errors` pipeline (`errors.Handler` is applied with `UseOnce`).

The parameter's **name** comes from the Response model (`Store` becomes `store_id`), and its **macro type** from the `ID` type parameter:

| `ID` type param | Generated macro |
|-----------------|-----------------|
| `string` | `{store_id:string}` |
| `uuid.UUID` | `{store_id:uuid}` |
| `int`, `int64`, `uint64` | the matching numeric macro |

`string` maps to the string macro, so a string primary key of any shape (a slug, a ULID, a uuid stored as text) matches as it is; declare the `ID` as `uuid.UUID` when the router should reject a malformed uuid before the method runs. `mvc.GetPathParam(c)` returns a `*mvc.RawPathParam` carrying the `Name`, `Type` and route path.

### Registration, dependencies, custom routes

```go
import "github.com/kataras/iris/v14/controller/mvc"

app := iris.New()
app.RegisterDependency(NewStoreService) // injected into StoreController.Service

app.Router("/stores", mvc.UseController[*StoreController]())

type StoreController struct {
    Service *StoreService
}

// Optional: extra custom routes on the same child router.
func (c *StoreController) Routing(r iris.Router) {
    param := mvc.GetPathParam(c) // generated path parameter (name, type, route path)
    r.Get("/greet", func(ctx iris.Context) {
        ctx.Writef("%s (%s)", c.Service.Greet(), ctx.Params().GetString(param.Name))
    })
}
```

`Routing` runs **before** the five generated routes are registered, so an `r.Use` inside it guards Create, Get, Update, Delete and List as well as the routes it adds itself. A route captures the router's middleware when it is created, which is why the ordering decides this.

**A controller is a singleton.** It is resolved once, its exported fields are bound once, and its methods are captured into handlers at registration. There is no per-request instance and no pooling, so every field is shared across all concurrent requests. Use fields for dependencies only; per-request state belongs on the context or in a method parameter.

The `ID` type parameter drives the generated route macro: `int` gives `{store_id:int}`, `int64` and `uint64` likewise, `uuid.UUID` gives `{store_id:uuid}` and `string` gives `{store_id:string}`. The `{store_id:uuid}` macro stores a parsed `uuid.UUID`, and a path parameter is converted to the declared `ID` type, so a method that declares a `string` id on a uuid route receives the canonical lowercase text.

### Options (`UseController` and `UseRepository`)

Both entry points take a variadic `...mvc.Option`. Existing `UseController()` calls compile unchanged.

```go
app.Router("/stores", mvc.UseController[*StoreController](
    mvc.Guard(auth.RequireAdmin),
    mvc.Without(mvc.Delete),
    mvc.IDParam("id"),
    mvc.Pagination(iris.AllowSort("name", "created_at")),
))
```

| Option | Effect |
|--------|--------|
| `mvc.Guard(handlers...)` | Middleware on the **write** routes only: `Create`, `Update`, `Delete`. A router-level `Use` cannot express "everyone may read, only a signed-in caller may write" without splitting the resource in two |
| `mvc.Without(verbs...)` | Drops generated routes. Verbs: `mvc.Create`, `mvc.Get`, `mvc.Update`, `mvc.Delete`, `mvc.List`. A dropped route 404s (or 405s under `FireMethodNotAllowed` when another method of the path remains) |
| `mvc.IDParam(name)` | Overrides the derived parameter name; the macro still comes from the `ID` type |
| `mvc.BeforeWrite(fn)` | `func(ctx *context.Context, v *T) error`, run on `Create` and `Update` after the body is decoded and before the controller or repository sees it. Hooks run in registration order; a non-nil error stops the request through the `rest/errors` pipeline |
| `mvc.Pagination(opts...)` | The list route's sort policy: `iris.AllowSort(cols...)`, `iris.DenySort(msg)`. A disallowed sort is answered before anything runs, as a 400 by default or as whatever code `errors.<Code>.DenySort(msg)` names |

`BeforeWrite` is how a value the client must not choose is stamped on every body in one place:

```go
mvc.BeforeWrite(func(ctx iris.Context, v *Store) error {
    tenantID, ok := auth.ScopeID[string](ctx)
    if !ok {
        return errors.Unauthenticated
    }

    v.TenantID = tenantID
    return nil
})
```

The model type is a type parameter, so the hook receives `*T` directly and no reflection happens at serve time. Three of the options fail loudly at registration rather than quietly: a `BeforeWrite` whose `T` is not the type the resource writes **panics**, naming both types (what it used to skip silently is typically "stamp the tenant id", and the symptom was a cross-tenant write); `Without` panics on an unknown verb and `IDParam` on an empty name.

### UseRepository (routes without a controller)

When a resource's HTTP half adds nothing to its store, there is no controller type left to write.

```go
func UseRepository[T any, ID context.IDType, Filter any](
    repo Repository[T, ID, Filter], opts ...Option) router.Controller

type Repository[T any, ID context.IDType, Filter any] interface {
    Create(ctx stdContext.Context, v T) (ID, error)
    Get(ctx stdContext.Context, id ID) (T, error)
    Update(ctx stdContext.Context, id ID, v T, onlyColumns ...string) (updated bool, err error)
    Delete(ctx stdContext.Context, id ID) (deleted bool, err error)
    List(ctx stdContext.Context, filter Filter, limit, offset int) ([]T, int64, error)
}
```

```go
app.Router("/invoices", mvc.UseRepository(invoiceRepository))
```

The repository's context is the **standard** context: a repository never sees Iris. The six routes are the ones `UseController` generates (`POST /`, `GET /{model_id}`, `PUT`/`PATCH /{model_id}` with `?columns=`, `DELETE /{model_id}`, `POST` and `QUERY /list`), the parameter name and macro derive the same way, and every option above applies.

One deliberate difference: the update routes read body and columns through `ctx.BindUpdate`, so a column that is not a member of `T` answers 400 naming it and the members it could have been. The controller routes cannot do that, because their request type goes through a pluggable decoder. The names reaching the repository are still the client's: map them, never interpolate them into SQL.

Reach for a controller as soon as a route needs anything the repository does not do: a mail on create, a computed response, a second store to keep in sync.

### Nested controllers and parent path parameters

`UseController` returns a `*DeployedController`. Its `With(relPathWithoutParam, controllers...)` registers children under the parent's id segment:

```go
app.Router("/stores", mvc.UseController[*StoreController]().
    With("/books", mvc.UseController[*BookController]()))
// BookController serves under /stores/{store_id}/books
```

A child controller reads the parent's path parameter through an auto-bound `mvc.PathParam[T, V]` field (T is the parent Response type, V its ID type):

```go
type BookController struct {
    Service    *BookService
    StoreParam *mvc.PathParam[Store, string]
}

func (c *BookController) Get(ctx iris.Context, id string) (Book, error) {
    storeID := c.StoreParam.Get(ctx) // or GetString(ctx)
    // ...
}
```

### Views

Return `mvc.ViewModel[T]` from a method to render a template (requires a registered view engine, e.g. `app.UseRouter(view.Handler(viewEngine))`):

```go
func View[T any](status int, layout, name string, data T) ViewModel[T]

func (c *BookViewController) Get(ctx iris.Context, id string) (mvc.ViewModel[Book], error) {
    return mvc.View(iris.StatusOK, "main", "book.html", book), nil
}
```

`WithViews[IndexViewData, CreateViewData, ModelViewData, ListViewData]()` adds HTML endpoints for a controller that also implements the `ViewController` methods (`IndexView`, `CreateView`, `DetailsView`, `UpdateView`, `DeleteView`, `ListView`):

```go
app.Router("/books", mvc.UseController[*BookController]().WithViews[IndexViewData, ViewData, ViewData, ListViewData]())
```

| Route | Method | Template (default) |
|-------|--------|--------------------|
| `GET /resource` | `IndexView` | `<model>/index.html` |
| `GET /resource/create` | `CreateView` | `<model>/create.html` |
| `GET /resource/{id}/details` | `DetailsView` | `<model>/details.html` |
| `GET /resource/{id}/update` | `UpdateView` | `<model>/update.html` |
| `GET /resource/{id}/delete` | `DeleteView` | `<model>/delete.html` |
| `GET /resource/list` | `ListView` | `<model>/list.html` |

### The handler subpackage

`controller/mvc/handler` gives one route the same typed pipeline (decode, call, encode, route errors) without committing to the full five-method controller. Useful for the endpoints that do not fit CRUD.

```go
import "github.com/kataras/iris/v14/controller/mvc/handler"

// func(Context, Request) (Response, error) becomes an iris.Handler.
r.Post("/orders/{order_id:int64}/refund",
    handler.Handler(refund, handler.WithStatusCode(iris.StatusAccepted),
        handler.WithPathParameter[int64]("order_id")))

// No request body: func(Context) (Response, error).
r.Get("/summary", handler.IndexHandler(summary))
```

The default decoder already reads the **last** path parameter for single-value request types (`string`, `int`, `int64`, `uint64`, `uuid.UUID`), so `WithPathParameter` is only needed to name a different one. Also exported: `HandlerBy` and `HandlerByColumns` for the update-with-columns shape.

### gRPC controllers

```go
grpcServer := grpc.NewServer()
ctrl := mvc.GRPC(&myServiceServer{}).
    Server(grpcServer).
    ServiceName("pkg.MyService").
    Strict(false) // true: plain HTTP gets 404; required for streaming

app.Router("/grpc", ctrl)
```

Routes are POST-only at `/{ServiceName}/{Method}`. With `Strict(false)` the same methods also answer plain HTTP/JSON, which makes a gRPC service testable with curl.

### WebSocket controllers

`mvc.Websocket(controllers ...any) router.Controller` registers neffos WebSocket controllers (struct pointers or constructor functions, DI applies):

```go
app.RegisterDependency(app.Logger, newTestService)
app.Router("/websocket", mvc.Websocket(newWebsocketController))
```

A controller can implement `Namespace() string` plus standard neffos events (`OnNamespaceConnect`, `OnNamespaceConnected`, `OnNamespaceDisconnect`, `OnRoomJoin`, `OnRoomJoined`, `OnRoomLeave`, `OnRoomLeft`). Any other exported method with the `func(*neffos.NSConn, neffos.Message) error` shape becomes a custom event (e.g. `OnChat`).

## API Graph (`controller/apigraph`)

Serves an interactive D3.js tree of all registered routes (paths, methods, handlers) at `GET /` of its child router. Node links open the handler's file at its registration line in VS Code (`vscode://file/` scheme). The page is built once, after the application builds (`RegisterOnBuild`), so registration order does not matter.

```go
import "github.com/kataras/iris/v14/controller/apigraph"

app.Router("/apigraph", apigraph.New(apigraph.Options{Theme: "dark"}))
```

```go
type Options struct {
    Theme string // "light" (default) or "dark"
}
```

Custom themes: implement `Theme` (`GetName() string`, `GetStyle() string`) or fill a `DefaultTheme{Name, BackgroundColor, Color, ReactColor, StrokeColor}`, then `apigraph.RegisterTheme(theme)` and pass its name in `Options.Theme`.

## Sitemap (`controller/sitemap`)

Generates `sitemap.xml` from the registered routes at build time. Handlers are registered for GET, HEAD and OPTIONS. If more than 50,000 static routes exist, the sitemap splits and a sitemap index is served at `/sitemap.xml`.

```go
import "github.com/kataras/iris/v14/controller/sitemap"

app := iris.New()
app.Router("/", sitemap.New("https://www.example.com"))

app.Get("/", index)
app.Get("/about", about).
    SetLastMod(time.Now()).
    SetChangeFreq("monthly").
    SetPriority(0.8)
app.Get("/internal", internal).ExcludeSitemap()
// GET /sitemap.xml now lists / and /about.
```

`New(domain string) *Controller` takes the site's base URL. Routes are collected on build, so the controller can be registered before or after the routes.

Excluded automatically: dynamic routes, subdomain routes, offline routes, and routes where `ExcludeSitemap()` was called.

### Route metadata (`*Route` methods)

| Method | Sitemap element |
|--------|-----------------|
| `SetLastMod(t time.Time)` | `<lastmod>` |
| `SetChangeFreq(freq string)` | `<changefreq>`: always, hourly, daily, weekly, monthly, yearly, never |
| `SetPriority(prio float32)` | `<priority>` |
| `ExcludeSitemap()` | Removes the route from the sitemap |

### Localization

If the app uses the i18n middleware, chain `WithLocalization` to emit translated `<xhtml:link rel="alternate">` entries per language (path prefix, subdomain, or URL parameter mode, depending on the i18n setup):

```go
app.Router("/", sitemap.New("https://www.example.com").
    WithLocalization(i18nHandler, i18nRouter)) // either may be nil
```

For full control over the output, use the standalone `github.com/kataras/sitemap` package instead.

## Source Files

- `controller/fileserver/fileserver.go` - `New`, `Options`, `BlockHidden`, `Favicon`, `WebDAV`
- `controller/mvc/mvc.go`, `controller/mvc/controller.go` - The `Controller` interface, `UseController`, `DeployedController`
- `controller/mvc/param.go` - `PathParam[T, V]`, `GetPathParam`, `RawPathParam`
- `controller/mvc/options.go` - `Option`, `Verb`, `Guard`, `Without`, `IDParam`, `BeforeWrite`, `Pagination`
- `controller/mvc/repository.go` - `Repository`, `UseRepository`
- `controller/mvc/handler/` - `Handler`, `IndexHandler`, `WithStatusCode`, `WithPathParameter`
- `controller/apigraph/` - Route graph rendering, `Options`, `RegisterTheme`
- `controller/sitemap/` - Sitemap generation and localization
- `core/router/controller.go` - The `Controller` interface every one of these satisfies
