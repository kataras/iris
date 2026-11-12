# REST Helpers

Typed request binding and typed responses remove handler boilerplate: parse the request, call a service, send JSON or a structured error. Every helper returns a `bool` where `true` means success and `false` means an error response was already written to the client (the handler should `return`).

Since v14 the binding and response helpers are **methods on `iris.Context`**: `ctx.BindJSON[T]()`, `ctx.BindParams[T]()`, `ctx.BindParam[T](name)`, `ctx.ValueAs[T](key)`, `ctx.OK(v, err)`, `ctx.Created(v, err)`, `ctx.NoContent(modified, err)`, `ctx.HandleError(err)`, `ctx.JSONOrError(v)`, `ctx.Count(n, err)`, `ctx.Paginated(collect)`, `ctx.BindPagination[F]()` and friends. There is no `github.com/kataras/iris/v14/rest` root package: the moved types are aliases on `iris.` (`iris.PageOptions`, `iris.Page[T]`, `iris.CountResponse[N]`, `iris.IDType`, `iris.IDPayload[T]`, `iris.Validatable`) and the error handlers live in `rest/errors`.

```go
import "github.com/kataras/iris/v14/rest/errors"
```

Structured error handling (canonical codes, `errors.NotFound.Message`, error mapping) lives in `rest/errors` and is documented separately. See [error-handling.md](./error-handling.md).

## Handler Pattern: Before and After

Manual context work:

```go
func (api *FoodBookmarksAPI) createFoodBookmark(ctx iris.Context) {
    var entry customer.CustomerFoodBookmark
    if err := ctx.ReadJSON(&entry); err != nil {
        ctx.StopWithJSON(iris.StatusBadRequest, iris.Map{
            "message": "unable to parse body",
            "detail":  err.Error(),
        })
        return
    }
    entry.CustomerID = getCustomerID(ctx) // a helper every handler calls.
    id, err := api.customers.CreateCustomerFoodBookmark(ctx, entry)
    if err != nil {
        if entity.IsErrDuplicate(err) {
            ctx.StopWithJSON(iris.StatusConflict, iris.Map{"message": "already exists"})
        } else {
            ctx.StopWithJSON(iris.StatusInternalServerError, iris.Map{"message": "server error"})
            ctx.Application().Logger().Error(err)
        }
        return
    }
    ctx.StatusCode(iris.StatusCreated)
    ctx.JSON(iris.Map{"id": id})
}
```

With the Context helpers (error branches handled by the `rest/errors` mapping):

```go
func (api *FoodBookmarksAPI) createFoodBookmark(ctx iris.Context) {
    entry, ok := ctx.BindJSON[customer.CustomerFoodBookmark]()
    if !ok {
        return
    }
    customerID, ok := auth.ScopeID[int64](ctx)
    if !ok {
        return // Scope did not run: the route is not behind VerifyHandler.
    }
    entry.CustomerID = customerID

    id, err := api.customers.CreateCustomerFoodBookmark(ctx, entry)
    ctx.Created(id, err)
}
```

Two things changed there, not one. The error branches moved into the central `rest/errors` mapping, and the tenant id now comes from `auth.ScopeID`, which reports whether `auth.Scope` ran, rather than from a `getCustomerID` helper each handler calls and nobody checks; see [authentication-sdk.md](./authentication-sdk.md). Stamping the id on every decoded body is `mvc.BeforeWrite`, one hook instead of one line per write handler.

## Response Helpers

| Function | Status | Behavior |
|----------|--------|----------|
| `ctx.OK(resp, err) bool` | 200 (untouched) | Errors go through `HandleError`; otherwise sends `resp` as JSON. `nil` resp sends nothing. |
| `ctx.Created(resp, err) bool` | 201 | ID scalars (`string`, `int`, `int64`, `uint64`, the standard library's `uuid.UUID`) are wrapped as `{"id": ...}`; a `[]string` of UUIDs becomes a list of id objects; other values are sent as-is. |
| `ctx.NoContent(modified, err) bool` | 204 or 304 | `modified=true` sends 204 No Content, `false` sends 304 Not Modified. |
| `ctx.Count[N context.CountType](rowsAffected, err) bool` | 200 | Sends `{"count": N}`. Zero is a valid answer, `{"count":0}`; a negative count renders as zero. |
| `ctx.List(items, err, tolerate ...error) bool` | 200 | Renders `items` as a JSON array. A nil or empty slice renders `[]`, never `null`; an error matching one of `tolerate` (`errors.Is`) also renders `[]`. Any other error goes through the error renderer and it returns false. A non-slice panics. |
| `ctx.JSONOrError(resp) bool` | current | `ctx.JSON` plus automatic JSON error handling. |
| `ctx.Paginated[Filter, Item, TotalCount](collect) bool` | 200 | Full pagination flow, see below. |

```go
func getFoodBookmark(ctx iris.Context) {
    id, ok := ctx.BindParam[int64]("id")
    if !ok {
        return
    }
    entry, err := api.presenters.GetCustomerFoodBookmark(ctx, id)
    ctx.OK(entry, err)
}

func deleteFoodBookmark(ctx iris.Context) {
    deleted, err := api.customers.DeleteFoodBookmarkByID(ctx, id)
    ctx.NoContent(deleted, err)
}
```

The `Created` ID wrapping types:

```go
type IDType interface {
    string | int | int64 | uint64 | uuid.UUID
}

type IDPayload[T IDType] struct {
    ID T `json:"id"`
}
```

## Request Reading

| Function | Returns |
|----------|---------|
| `ctx.BindJSON[Request]() (Request, bool)` | Parsed body. On parse failure a detailed 4xx JSON error is written and `false` is returned. If `Request` implements `context.Validatable` (`Validate() error`), the method runs after decoding and a non-nil result also rejects with a 4xx. |
| `ctx.BindJSONOptional[Request]() (Request, bool)` | Like BindJSON, but an empty body returns the zero `Request` with `true` and skips validation. |
| `ctx.BindJSONOneOrMany[Request]() (*Request, []Request, bool)` | Accepts either a single object or an array (each element validated). Exactly one of the first two results is non-nil. |
| `ctx.BindJSONList[Request]() ([]Request, bool)` | Same, but a single object is wrapped into a one-element slice. |

```go
func createFoodBookmark(ctx iris.Context) {
    entry, ok := ctx.BindJSON[customer.CustomerFoodBookmark]()
    if !ok {
        return // error response already sent.
    }
    // ...
}
```

## Error Shortcuts

`rest/errors` handles the error branches:

```go
errors.HandleError(ctx, err) bool        // unknown errors render as 500 Internal.
errors.HandleRequestError(ctx, err) bool // unknown errors render as 400 InvalidArgument.
errors.HandleRequestJSONError(ctx, err) bool  // a body that would not decode: 400.
errors.HandleResponseJSONError(ctx, err) bool // a value that would not encode: 500.
errors.HandleErrorf(ctx, err, format, args...) bool // wraps with %w (mappings still
                                       // apply), renders the format as message.
errors.HandleRequestErrorf(ctx, err, format, args...) bool // client-error variant.
```

All report `true` when an error was written, so the common guard is:

```go
if errors.HandleError(ctx, err) {
    return
}
```

`errors.NotFoundID(ctx, resource string, id any) bool` renders `NOT_FOUND` with the detail `"<resource> <id> not found"`, replacing the hand-written `NotFound.Detailf` that every store lookup repeats. The id is formatted with `%v`.

```go
plan, err := svc.GetPlan(ctx, id)
if errors.HandleError(ctx, err) {
    return
}

if plan == nil {
    errors.NotFoundID(ctx, "plan", id)
    return
}

ctx.OK(plan, nil)
```

Its result follows the `Context.OK` convention (true means the handler succeeded), so it is always false, the opposite of `HandleError` above it. Do not use it as a guard.

See [error-handling.md](./error-handling.md) for error mapping, canonical codes and validation errors.

## Type-Safe Accessors

The typed accessors are generic methods on the Iris context. A standard `context.Context` (as passed to a service layer) is often the Iris context itself, so a type assertion to `iris.Context` bridges back to it.

| Function | Description |
|----------|-------------|
| `auth.GetUser[T any](ctx *context.Context) T` | The authenticated user stored by the `auth` package's `VerifyHandler`. Zero value if none. |
| `ctx.BindParams[Struct any]() (Struct, bool)` | Binds all dynamic path parameters to a struct (field tag `param`) via `ctx.ReadParams`. Sends a client error and returns `false` on bind failure. |
| `ctx.BindParam[T any](name string) (T, bool)` | A single path parameter. Sends 404 if missing, 400 if not of type `T`. |
| `ctx.ValueAs[Request any](key string) (Request, bool)` | A typed value from the Iris context store. |
| `ctx.RequestValue[Request any]() (Request, bool)` | Shortcut for the `context.RequestValueContextKey` (`"iris.rest.request"`) cached request value. |

`BindParam` type-asserts the stored parameter value, so `T` must match the route macro type: `{id:int64}` stores `int64`, `{name:string}` stores `string`, and `{id:uuid}` stores a parsed `uuid.UUID`. Read a uuid parameter with `ctx.BindParam[uuid.UUID]("id")` or `ctx.Params().GetUUID("id")`; `ctx.BindParam[string]("id")` answers 400 on a `:uuid` route. `ctx.Params().GetString("id")` still returns the text, in canonical lowercase form.

```go
app.Get("/bookmarks/{id:int64}", func(ctx iris.Context) {
    id, ok := ctx.BindParam[int64]("id")
    if !ok {
        return
    }
    // ...
})

// In a service that only holds a standard context ("context" is imported
// as stdContext here, so iris.Context is unambiguous):
func (s *Service) audit(ctx stdContext.Context) {
    irisCtx, ok := ctx.(iris.Context) // iris.Context is *context.Context.
    if !ok {
        return // not an Iris request context.
    }
    user := auth.GetUser[*MyUserClaims](irisCtx)
    _ = user
}
```

## Query Date

```go
func (e encoding.StringSliceEntries) GetSimpleDate(keys ...string) (types.SimpleDate, bool, error)
```

For a reader that renders the 400 itself, use `ctx.QueryDate` (see [context-api.md](./context-api.md)). `ctx.Query().GetSimpleDate` reads a `types.SimpleDate` (`2006-01-02`, or `2006-1-2`) from the URL query. Keys are tried in order and default to `date` when none is given; a blank value counts as absent. It renders nothing: a missing key returns `(zero, false, nil)` so the caller picks the default, and an unparsable value returns `(zero, true, err)` so a client typo is not mistaken for an absent parameter.

```go
date, found, err := ctx.Query().GetSimpleDate("date", "day")
if err != nil {
    errors.InvalidArgument.Detailf(ctx, "invalid date query parameter", "%v", err)
    return
}
if !found {
    date = types.SimpleDateFromTime(time.Now().UTC())
}
plan, err := api.plans.ForDate(ctx, date)
ctx.OK(plan, err)
```

## Pagination

### PageOptions (client input)

Read from the URL query (`?page=1&size=20&details=true`):

```go
type PageOptions struct {
    Page    int  `json:"page" url:"page"`       // current page, 1-based.
    Size    int  `json:"size" url:"size"`       // page size (LIMIT). Zero means DefaultPageSize.
    Details bool `json:"details" url:"details"` // ask the server for nested objects.
}
```

Methods: `GetLimit()`, `GetOffset()`, `GetCurrentPage()`, `GetNextPage()`, and `Bind(r *http.Request) error` (usable as a `client.RequestOption`: it sets `page`, `size` and `details` on the outgoing request's query). Package variables `pagination.DefaultPageSize = 100` and `pagination.MaxPageSize = 100000` (package `core/pkg/pagination`) bound the size; set them once at startup, directly or through `iris.Patches().Context().SetDefaultPageSize(n)` and `SetMaxPageSize(n)`. There is no `context.DefaultPageSize` or `context.MaxPageSize` any more. The canonical type is `pagination.PageOptions`; `context.PageOptions` and `iris.PageOptions` are aliases of it.

The page types (`PageOptions`, `Page[T]`, `NewPage`, `CountType`, `CountResponse[N]`, `CountToInt64`, `ListFilter`, `ErrSortNotAllowed`, `SortRequester`) live in `core/pkg/pagination`, which imports only the standard library, so a client or a service that does not import Iris can use the same types. `context` and `iris` alias all of them except `SortRequester`, and the `client` package aliases `PageOptions`, `Page`, `CountResponse` and `ListFilter` (see [http-client.md](./http-client.md): `client.List[T]` and `client.Items[T]`).

### Which method a list uses

A list with only `?page=&size=` is a `GET`. A list with a filter body should be a `QUERY` (RFC 10008): safe, idempotent and cacheable like `GET`, with a body like `POST`. Register it with `app.Query(path, handler)`; the same handler serves both:

```go
app.Get("/books", listBooks)   // ?page=&size= only.
app.Query("/books", listBooks) // ?page=&size= plus a JSON filter body.
```

The generated list routes (`controller/mvc` `List`, `handler.List`/`ListBy`, identity `/u/list` and `/u/groups/list`) answer both `POST` and `QUERY` on the same path. Send the QUERY filter as JSON: on Go 1.27 net/http parses a url-encoded body only for POST, PUT and PATCH, so it never reaches `ctx.Form*`.

### Page[T] (server response)

```go
type Page[T any] struct {
    CurrentPage int   `json:"current_page"`
    PageSize    int   `json:"page_size"`
    TotalPages  int   `json:"total_pages"`
    TotalItems  int64 `json:"total_items"`
    HasNextPage bool  `json:"has_next_page"`
    Filter      any   `json:"filter"`
    Items       []T   `json:"items"`
}
```

Constructors and helpers:

```go
func NewPage[T any](opts PageOptions, filter any, items []T, totalCount int64) *Page[T]
func (list *Page[T]) Transform[V any](transform func(T) (V, error)) (*Page[V], error)
func (ctx *Context) BindPagination[Filter any]() (PageOptions, Filter, bool)
```

`ctx.BindPagination` reads `PageOptions` from the URL query and an optional `Filter` from the JSON request body (an empty body yields the zero `Filter`). It checks the cached `context.RequestValueContextKey` value first, so an options-decoder middleware can pre-populate the filter.

### ctx.Paginated

The one-call flow: read options and filter, collect, respond.

```go
type ItemsCollectorFunc[Filter any, Item any, TotalCount CountType] func(
    context.Context, PageOptions, Filter) ([]Item, TotalCount, error)

func (ctx *Context) Paginated[Filter any, Item any, TotalCount CountType](
    collect ItemsCollectorFunc[Filter, Item, TotalCount],
    opts ...PaginationOption) bool

func (ctx *Context) BindPagination[Filter any](opts ...PaginationOption) (
    PageOptions, Filter, bool)
```

```go
func listCustomerFoodBookmarks(ctx iris.Context) {
    customerID, ok := auth.ScopeID[int64](ctx)
    if !ok {
        return
    }

    ctx.Paginated(func(_ context.Context, pageOpts iris.PageOptions, filter customer.Filter) ([]customer.CustomerFoodBookmark, int64, error) {
        return api.presenters.ListCustomerFoodBookmarks(ctx, pageOpts, filter, customerID)
    })
}
```

Response shape:

```json
{
    "current_page": 1,
    "page_size": 20,
    "total_pages": 1,
    "total_items": 2,
    "has_next_page": false,
    "filter": null,
    "items": [
        {"id": 1, "name": "Bookmark 1"},
        {"id": 2, "name": "Bookmark 2"}
    ]
}
```

## ListFilter and the Sort Policy

`iris.ListFilter` (alias of `context.ListFilter`) is the request half every list endpoint shares. Embed it and add only what the service adds.

```go
type ListFilter struct {
    Term           string   `json:"term,omitzero" url:"term"`
    Names          []string `json:"names,omitzero" url:"names"`
    Substring      bool     `json:"substring,omitzero" url:"substring"`
    Sort           string   `json:"sort,omitzero" url:"sort"`
    SortDescending bool     `json:"sort_descending,omitzero" url:"sort_descending"`
}

func (f ListFilter) SortColumn(allowed ...string) (string, error) // wraps iris.ErrSortNotAllowed
func (f ListFilter) SortOrder() string                            // "ASC" or "DESC"
func (f ListFilter) SortRequest() (column string, descending bool, requested bool)
```

It carries both tag sets on purpose: the `json` names for a `QUERY` or `POST` list body and the `url` names for a `GET` list route read through `ctx.ReadQuery`.

A sort column reaches an `ORDER BY` clause, so the set of columns an endpoint offers belongs to the endpoint and never to the request. Two options declare it, and both answer 400 before the collector runs:

```go
ctx.Paginated(collect, iris.AllowSort("created_at", "name"))
ctx.Paginated(collect, iris.DenySort("this list is ordered by rank"))
```

- `iris.AllowSort(columns...)`: a sort outside the list is a 400 naming the column and the allowed ones.
- `iris.DenySort(message)`: any sort request is a 400 with that message.
- Both work on `ctx.BindPagination` too, and on the MVC list route through `mvc.Pagination(...)` (see [builtin-controllers.md](./builtin-controllers.md)).
- Both read the decoded filter through its `SortRequest` method (the `pagination.SortRequester` interface), so a filter that does not provide one is unaffected rather than silently mis-read.

Sorting is often a permission rather than a shape: a list a client may read but not order should answer 403, not 400. Any error code carries the rejection:

```go
list, filter, ok := ctx.BindPagination[customer.CheatFoodFilterOptions](
    errors.PermissionDenied.DenySort("sort feature is not available for customers"))
if !ok {
    return
}
```

`errors.ErrorCodeName.DenySort` (package `rest/errors`) is `iris.DenySort` answered with that code; the option is otherwise identical.

A collector that takes its filter by pointer - the shape a repository method already has - can be passed as a bare method value, and never receives nil:

```go
ctx.Paginated(api.exercises.ListExercisesPaginated)  // func(context.Context, iris.PageOptions, *FilterOptions) ([]T, int64, error)
```

A request with no body decodes to the filter's zero value rather than a nil pointer, so the repository reads an empty filter and the sort policy still sees a real `SortRequest`.

## Scalar Types (rest/types)

`github.com/kataras/iris/v14/rest/types` holds JSON-ready field types for request and response payloads. They marshal to the wire formats below. Several also work as SQL column types: `SimpleDate` and `ISO8601` implement both `sql.Scanner` and `driver.Valuer`, `TimeNotationDuration` implements `driver.Valuer`, `DayTime` and `KitchenTime` implement `Scan`, `Duration` implements `Value`.

```go
import "github.com/kataras/iris/v14/rest/types"
```

| Type | Underlying | Wire format | Parse |
|------|-----------|-------------|-------|
| `types.SimpleDate` | `time.Time` | `"2006-01-02"` | `ParseSimpleDate(s)`, `SimpleDateFromTime(t)` |
| `types.DayTime` | `time.Time` | `"15:04:05"` | `ParseDayTime(s)` |
| `types.KitchenTime` | `time.Time` | `"3:04 PM"` | `ParseKitchenTime(s)` |
| `types.ISO8601` | `time.Time` | `"2006-01-02T15:04:05"` (JS ISO) | `ParseISO8601(s)` |
| `types.Duration` | `time.Duration` | `"1h30m0s"` | `ParseDuration(s)` |
| `types.TimeNotationDuration` | `time.Duration` | `"01:30:00"` (hh:mm:ss) | `ParseTimeNotationDuration(s)` |
| `types.Season` | `int` bitmask | `3` (`Winter \| Spring`) | constants `Winter`, `Spring`, `Summer`, `Autumn`, `AllSeasons` |
| `types.StringOrArray` | `string` | accepts `"a"` or `["a","b"]`, marshals comma-joined | `UnmarshalJSON` |
| `types.Number` | `float64` | a JSON number, or `null` for `NaN` and the infinities | `NullNumber()`, `IsNull()`, `Float64()` |
| `types.Weekday` | `time.Weekday` | `"monday"` (always lowercase) | `ParseWeekday(s)`, `WeekdayOf(d)`, `d.Std()` |

```go
type CreateEventInput struct {
    Date     types.SimpleDate `json:"date"`     // "2026-06-12"
    StartsAt types.DayTime    `json:"starts_at"` // "18:30:00"
    Duration types.Duration   `json:"duration"`  // "1h30m"
}

func createEvent(ctx iris.Context) {
    input, ok := ctx.BindJSON[CreateEventInput]()
    if !ok {
        return
    }
    id, err := api.events.Create(ctx, input)
    ctx.Created(id, err)
}
```

The parse functions treat `""` and `"null"` as the zero value, not an error, so optional fields round-trip cleanly.

`types.Number` and `types.Weekday` both implement `sql.Scanner` and `driver.Valuer` as well:

- `Number` exists because `encoding/json` refuses to encode `NaN` or an infinity, so one computed average takes the whole document down. It marshals all three as `null` and reads a JSON `null` back as `NaN`; a `NULL` column scans to `NaN` and a `NaN` is written as `NULL`, so a nullable `float8` needs no `sql.NullFloat64`. The zero value is the number zero, not null.
- `Weekday` numbers as `time.Weekday` does, Sunday being zero, so the zero value marshals as `"sunday"`. `ParseWeekday` accepts any case and the three-letter form (`"Monday"`, `"monday"`, `"MON"`, `"mon"`); the seven days are exported as constants and the column is text.

## Source Files

- `context/context_bind.go` - `BindJSON`/`BindJSONOptional`, the `Validatable` hook, `BindJSONOneOrMany`/`BindJSONList`, `BindParams`/`BindParam`, `RequestValue`
- `context/rest_helpers.go` - `OK`, `Created`, `NoContent`, `List`, `JSONOrError`, `HandleError`
- `core/pkg/pagination/pagination.go` - `PageOptions`, `Page[T]`, `NewPage`, `Page.Transform`, `CountType`, `CountResponse[N]`, `CountToInt64`, `MaxPageSize`, `DefaultPageSize`
- `core/pkg/pagination/list_filter.go` - `ListFilter`, `ErrSortNotAllowed`, `SortRequester`
- `context/page.go` - the aliases, `Count`, `BindPagination`, `Paginated`
- `context/list_filter.go` - `PaginationOption`, `AllowSort`, `DenySort`
- `context/context_bind_update.go` - `BindUpdate`, `UpdateColumnsQueryKey`
- `context/query_strict.go` - `QueryDate`, `QueryInt`, `QueryBool`, `QueryRequired`, `QueryDefault`, `QueryZeroValues`
- `rest/errors/pagination.go` - `ErrorCodeName.DenySort`
- `rest/errors/not_found_id.go` - `NotFoundID`
- `context/context_request.go` - `ValueAs`
- `context/encoding/entries_string_slice.go` - `GetSimpleDate`
- `context/rest_error_renderer.go` - the `RestErrorRenderer` seam the methods render errors through (the canonical implementation installs itself from `rest/errors` at init)
- `aliases.go` - `iris.PageOptions`, `iris.Page[T]`, `iris.CountResponse[N]`, `iris.IDType`, `iris.IDPayload[T]`, `iris.Validatable`, `iris.NewPage`, `ContextPatches.SetMaxPageSize`/`SetDefaultPageSize`
- `rest/types/` - Scalar payload types
- `rest/errors/` - Structured error handling (see [error-handling.md](./error-handling.md))
