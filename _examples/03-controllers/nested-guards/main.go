// Nested controllers and the two guard idioms: a group guard for every route
// and a positional guard for single routes.
//
// ShelvesAPI mounts ItemsAPI with r.Router("/{shelf_id:uuid}/items",
// NewItemsAPI): a child router whose controller is resolved by the same
// container, so the nested API declares its own dependencies and owns its own
// route table.
//
// Inside the child, guards are methods: they see the injected dependencies
// like any other handler. A group guard registered with r.Use runs before
// every route of the controller: requireShelf loads the shelf once, answers
// 404 when it does not exist and caches it on the request with
// ctx.Values().Set. A positional guard listed between path and handler
// protects individual routes: requireUnlocked runs on the writes only.
// Handlers read the cached shelf back with ctx.ValueAs, a typed lookup with
// no interface assertion at the call site.
//
// A guard either calls ctx.Next() to continue or writes a response; the
// rest/errors senders stop the request for you.
//
// Learn: mount a controller inside another one, guard all of its routes with
// r.Use and single routes with a positional handler, and cache a loaded value
// on the request for the handlers.
// Before: ../basic
//
// Run:
//
//	go run main.go
//
// Try:
//
//	OPEN=http://localhost:8080/api/shelves/11111111-1111-4111-8111-111111111111
//	LOCKED=http://localhost:8080/api/shelves/22222222-2222-4222-8222-222222222222
//	GONE=http://localhost:8080/api/shelves/33333333-3333-4333-8333-333333333333
//
//	curl http://localhost:8080/api/shelves                # 200, both shelves
//	curl $OPEN/items                                      # 200, one seeded item
//	curl -X POST $OPEN/items -d '{"title":"Go in Depth"}'  # 201, {"id":"..."}
//	curl -i -X POST $LOCKED/items -d '{"t":1}'            # 403, the positional guard
//	curl -i $GONE/items                                   # 404, the group guard
//
// Next: ../mvc-generics
package main

import (
	stdContext "context"
	"fmt"
	"slices"
	"sync"

	"uuid"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	// The builder's API step mounts a controller exactly like r.Router does:
	// it takes the constructor and resolves it through the container.
	app := iris.NewBuilder().
		Prefix("/api").
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Errors(errors.NewOptions()).
		Services(NewLibrary).
		RouterMiddlewares().
		Middlewares().
		API("/shelves", NewShelvesAPI).
		Build()

	app.Listen(":8080")
}

// Library is the single service both controllers depend on.
type Library struct {
	mu      sync.RWMutex
	shelves []Shelf
	items   map[uuid.UUID][]Item
}

func NewLibrary() *Library {
	return &Library{
		shelves: []Shelf{
			{ID: openShelfID, Name: "reading"},
			{ID: lockedShelfID, Name: "archive", Locked: true},
		},
		items: map[uuid.UUID][]Item{
			openShelfID: {{ID: uuid.New(), ShelfID: openShelfID, Title: "Iris in Action"}},
		},
	}
}

func (l *Library) Shelves(stdContext.Context) ([]Shelf, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return slices.Clone(l.shelves), nil
}

func (l *Library) Shelf(_ stdContext.Context, id uuid.UUID) (Shelf, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, s := range l.shelves {
		if s.ID == id {
			return s, nil
		}
	}

	// The error code name is what turns this into a 404 response.
	return Shelf{}, errors.NotFound.Wrap(fmt.Errorf("shelf %s not found", id))
}

func (l *Library) Items(_ stdContext.Context, shelfID uuid.UUID) ([]Item, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return slices.Clone(l.items[shelfID]), nil
}

func (l *Library) CreateItem(_ stdContext.Context, shelfID uuid.UUID, in ItemInput) (uuid.UUID, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	item := Item{ID: uuid.New(), ShelfID: shelfID, Title: in.Title}
	l.items[shelfID] = append(l.items[shelfID], item)
	return item.ID, nil
}

// ShelvesAPI serves /shelves and mounts the nested items controller.
type ShelvesAPI struct {
	lib *Library
}

func NewShelvesAPI(lib *Library) *ShelvesAPI {
	return &ShelvesAPI{lib: lib}
}

func (api *ShelvesAPI) Routing(r iris.Router) {
	r.Get("/", api.list)

	// Everything about a shelf's items lives in its own nested router.
	r.Router("/{shelf_id:uuid}/items", NewItemsAPI)
}

func (api *ShelvesAPI) list(ctx iris.Context) {
	shelves, err := api.lib.Shelves(ctx)
	ctx.OK(shelves, err)
}

// ItemsAPI serves /shelves/{shelf_id:uuid}/items.
type ItemsAPI struct {
	lib *Library
}

func NewItemsAPI(lib *Library) *ItemsAPI {
	return &ItemsAPI{lib: lib}
}

func (api *ItemsAPI) Routing(r iris.Router) {
	// Group guard: every item route needs an existing shelf.
	r.Use(api.requireShelf)

	r.Get("/", api.list)
	// Positional guard: writes additionally need an unlocked shelf.
	r.Post("/", api.requireUnlocked, api.create)
}

// requireShelf loads the shelf of the path parameter, replies 404 when it does
// not exist and caches it in the request values for the handlers.
func (api *ItemsAPI) requireShelf(ctx iris.Context) {
	// The {shelf_id:uuid} macro rejected anything that is not a UUID before
	// this ran, and the framework parsed it once, so this reads the typed value.
	shelfID := ctx.Params().GetUUID("shelf_id")

	shelf, err := api.lib.Shelf(ctx, shelfID)
	if errors.HandleError(ctx, err) {
		return // mapped to 404, request stopped.
	}

	ctx.Values().Set(shelfContextKey, shelf)
	ctx.Next()
}

// requireUnlocked rejects writes against a locked shelf. The shelf is already
// on the request: the group guard above put it there.
func (api *ItemsAPI) requireUnlocked(ctx iris.Context) {
	if currentShelf(ctx).Locked {
		errors.PermissionDenied.Detail(ctx, "shelf is locked",
			"unlock the shelf before modifying its items")
		return // the errors senders stop the request.
	}

	ctx.Next()
}

func (api *ItemsAPI) list(ctx iris.Context) {
	items, err := api.lib.Items(ctx, currentShelf(ctx).ID)
	ctx.OK(items, err)
}

func (api *ItemsAPI) create(ctx iris.Context) {
	input, ok := ctx.BindJSON[ItemInput]()
	if !ok {
		return
	}

	id, err := api.lib.CreateItem(ctx, currentShelf(ctx).ID, input)
	ctx.Created(id, err) // 201 with {"id": "..."}.
}

type Shelf struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Locked bool      `json:"locked"`
}

type Item struct {
	ID      uuid.UUID `json:"id"`
	ShelfID uuid.UUID `json:"shelf_id"`
	Title   string    `json:"title"`
}

type ItemInput struct {
	Title string `json:"title"`
}

var (
	openShelfID   = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	lockedShelfID = uuid.MustParse("22222222-2222-4222-8222-222222222222")
)

// shelfContextKey names the guard-loaded shelf on the request, so the handlers
// never load it twice.
const shelfContextKey = "example.shelf"

// currentShelf reads the value cached by requireShelf. ValueAs is
// generic, so the result is a Shelf and not an any.
func currentShelf(ctx iris.Context) Shelf {
	shelf, _ := ctx.ValueAs[Shelf](shelfContextKey)
	return shelf
}
