# Database and Persistence

Iris has no ORM and no opinion about your driver. What it does provide is the wiring: a dependency container that builds your store, a startup hook that migrates or warms it, a shutdown hook that closes it, and an error pipeline that turns driver errors into correct HTTP responses without your handlers knowing the driver exists.

## The Repository Seam

Define persistence as an interface owned by the domain, not by the database package. Two conventions matter:

```go
type OrderStore interface {
    Create(ctx context.Context, o Order) (string, error) // store generates and returns the ID
    ByID(ctx context.Context, id string) (Order, error)
    Page(ctx context.Context, opts iris.PageOptions, f OrderFilter) ([]Order, int64, error)
    Delete(ctx context.Context, id string) (bool, error)
}
```

1. **`context.Context` is the first parameter of every method.** `iris.Context` satisfies `context.Context`, so a handler passes `ctx` straight through and cancellation, deadlines and request values reach the driver with no adapter.
2. **The store generates IDs and returns them.** Handlers never invent primary keys.

Domain errors are sentinels declared next to the interface. They are what the rest of the application matches on:

```go
var (
    ErrOrderNotFound = errors.New("order not found")
    ErrSKUTaken      = errors.New("sku already exists")
)
```

## Wiring a Store Through Services

Register the store with the builder and its whole lifecycle is handled:

```go
iris.NewBuilder().
    Prefix("/api").
    Errors(errors.NewOptions().
        MapErrors(errors.NotFound, order.ErrOrderNotFound).
        MapErrors(errors.AlreadyExists, order.ErrSKUTaken)).
    Services(NewPostgresStore, order.NewService).
    RouterMiddlewares().
    Middlewares().
    API("/orders", NewOrdersAPI).
    Build().
    Listen(":8080")
```

Three builder behaviors do the work:

- **`Services` routes values by shape.** A `func()` or `func() error` becomes a shutdown closer rather than a dependency. A value with `Close()` or `Close() error` is registered as **both** a dependency and a closer. `Configurator` values are applied to the app; nil entries are skipped. A registration error panics immediately, so a misconfigured app fails at startup instead of misbehaving later.
- **`iris.Initializer` runs inside `Build()`.** A store implementing `Init(ctx context.Context) error` is initialized exactly once after dependency injection and before serving, in registration order. An error aborts the build. This is where schema creation and connection priming belong, not in `Routing`.
- **`Deferrables` is the manual counterpart** for closers that are not methods on a registered value.

So a store that implements both `Init` and `Close` needs no lifecycle code anywhere else:

```go
type PostgresStore struct {
    db   *pg.DB
    repo *pg.Repository[Order]
}

func NewPostgresStore() (*PostgresStore, error) {
    schema := pg.NewSchema()
    schema.MustRegister("orders", Order{})
    db, err := pg.Open(context.Background(), schema, os.Getenv("DATABASE_URL"))
    if err != nil {
        return nil, err
    }
    return &PostgresStore{db: db, repo: pg.NewRepository[Order](db)}, nil
}

func (s *PostgresStore) Init(ctx context.Context) error { return s.db.CreateSchema(ctx) }
func (s *PostgresStore) Close()                        { s.db.Close() }
```

## PostgreSQL with kataras/pg

`github.com/kataras/pg` is a separate module by the same author: a generics-based repository over pgx that maps structs to tables from struct tags.

```sh
go get github.com/kataras/pg@latest
```

```go
schema := pg.NewSchema()
schema.MustRegister("orders", Order{})
db, err := pg.Open(ctx, schema, connString)
repo := pg.NewRepository[Order](db)
```

Repository methods used most:

| Method | Purpose |
|--------|---------|
| `InsertSingle(ctx, value, &id)` | Insert one row, write the generated ID into the pointer |
| `SelectByID(ctx, id) (T, error)` | Single row by primary key |
| `Select(ctx, query, args...) ([]T, error)` | Arbitrary query into `[]T` |
| `Update(ctx, values...) (int64, error)` | Full-row update, returns rows affected |
| `DeleteByID(ctx, id) (bool, error)` | Delete, reports whether a row existed |
| `InTransaction(ctx, fn func(*Repository[T]) error) error` | Atomic unit |

Also available: `Insert`, `SelectSingle`, `Upsert`, `UpdateOnlyColumns`, `Exists`, `Query`, `QueryRow`.

Pool tuning is connection-string parameters, handled by pgx rather than by an Iris API: `pool_max_conns`, `pool_min_conns`, `pool_max_conn_lifetime`, `pool_max_conn_idle_time`, `pool_health_check_period`.

## Translating Driver Errors at the Boundary

The store converts driver errors into domain sentinels. Nothing above the store ever sees a pgx or `database/sql` error:

```go
func (s *PostgresStore) ByID(ctx context.Context, id string) (Order, error) {
    o, err := s.repo.SelectByID(ctx, id)
    if err != nil {
        if errors.Is(err, pg.ErrNoRows) {
            return Order{}, ErrOrderNotFound
        }
        return Order{}, err
    }
    return o, nil
}

func (s *PostgresStore) Create(ctx context.Context, o Order) (string, error) {
    var id string
    if err := s.repo.InsertSingle(ctx, o, &id); err != nil {
        if _, dup := pg.IsErrDuplicate(err); dup {
            return "", ErrSKUTaken
        }
        return "", err
    }
    return id, nil
}
```

This is what keeps handlers portable: swapping Postgres for something else changes one file.

## Handlers Stay Thin

Read, call, hand `(value, error)` to the context. The response helpers write the success body or route the error into the central pipeline, and return `false` when they already responded.

```go
func (api *OrdersAPI) Routing(r iris.Router) {
    r.Post("/", api.create)
    r.Get("/{id:uuid}", api.get)
    r.Delete("/{id:uuid}", api.delete)
}

func (api *OrdersAPI) create(ctx iris.Context) {
    in, ok := ctx.BindJSON[CreateOrder]()
    if !ok {
        return
    }
    id, err := api.orders.Create(ctx, in)
    ctx.Created(id, err)
}

func (api *OrdersAPI) get(ctx iris.Context) {
    o, err := api.orders.ByID(ctx, ctx.Params().GetString("id"))
    ctx.OK(o, err)
}

func (api *OrdersAPI) delete(ctx iris.Context) {
    deleted, err := api.orders.Delete(ctx, ctx.Params().GetString("id"))
    ctx.NoContent(deleted, err) // 204 when true, 304 when false
}
```

## Mapping Database Errors to HTTP

Three rings, outermost last. Each catches what the ring inside it did not.

**1. Domain sentinels, mapped once at startup.** This is the ring you own:

```go
errors.NewOptions().
    MapErrors(errors.NotFound, order.ErrOrderNotFound).
    MapErrors(errors.AlreadyExists, order.ErrSKUTaken)
```

**2. Driver defaults.** `errors.NewOptions()` already maps `database/sql.ErrNoRows` to 404. For `kataras/pg`, one blank import restores the PostgreSQL mapping (no-rows → 404, duplicate key → 409 with a generic client detail and the raw error logged, SQLSTATE noise trimmed on invalid input syntax):

```go
import _ "github.com/kataras/iris/v14/rest/errors/pgerrors"
```

That subpackage exists so the pgx driver is not linked into every Iris binary. It is built on `errors.RegisterErrorResolver`, the extension point for any other driver: resolvers run after the per-route code map and before built-in classification, and the first one that reports `true` wins.

**3. Call-site enrichment.** When the mapped status is right but the message needs request detail, wrap rather than re-classify. `HandleErrorf` wraps with `%w`, so the mapping from ring 1 still applies:

```go
if err != nil {
    errors.HandleErrorf(ctx, err, "order %s", id) // still the mapped 404
    return
}
```

`MapErrorFuncCtx` is the startup-time equivalent when the transformation needs the request context.

See [error-handling.md](./error-handling.md) for the full pipeline.

## Transactions Below the Handler

Atomic units belong in the store. A handler that opens a transaction has leaked the database into the HTTP layer.

```go
func (s *PostgresStore) TransferStock(ctx context.Context, from, to string, qty int) error {
    return s.repo.InTransaction(ctx, func(tx *pg.Repository[Order]) error {
        if err := tx.UpdateOnlyColumns(ctx, []string{"stock"}, decrement(from, qty)); err != nil {
            return err
        }
        return tx.UpdateOnlyColumns(ctx, []string{"stock"}, increment(to, qty))
    })
}
```

`InTransaction` is re-entrant: a nested call joins the outer transaction instead of opening a second one, so store methods can compose without knowing whether they were called inside one.

## Other Stores

MongoDB, Redis, GORM, ent, sqlc and plain `database/sql` need nothing special. Two steps make any of them fit:

1. Register the handle (client, pool, `*sql.DB`) through `Services(...)`. If it has a `Close()` method it becomes a shutdown closer automatically; if it needs migration or a ping at startup, give the wrapping store an `Init(ctx) error`.
2. Add its sentinel errors to `MapErrors`, or register a resolver with `errors.RegisterErrorResolver` for driver-specific classification.

## Source Files

- `iris_builder.go` - `Services`, `Deferrables`, `Initializer`, `Build`
- `core/dep/dep.go` - The dependency container, `VisitSingletons`
- `rest/errors/options.go` - `NewOptions`, the default error code map, `MapErrors`, `MapErrorFuncCtx`
- `rest/errors/resolver.go`, `rest/errors/pgerrors/` - `RegisterErrorResolver`, the PostgreSQL resolver
- `rest/ok.go`, `rest/created.go`, `rest/no_content.go`, `rest/paginated.go` - Response helpers
- [book/13-database-integration.md](https://github.com/kataras/iris-book/blob/main/book/13-database-integration.md) - The long-form chapter this condenses
