// Step 4 of 13: domain errors, mapped to status codes exactly once.
//
// The data functions now return errors in the domain's own language:
// ErrNotFound and ErrShelfLocked are sentinel values that know nothing about
// HTTP. One errors.NewOptions() value maps each sentinel to a canonical code
// (NotFound is 404, PermissionDenied is 403) and app.Use(opts.Handler())
// attaches that map to every request. From then on the handlers only forward:
// ctx.OK(shelf, err) renders the shelf or the mapped error, and
// ctx.NoContent(true, err) renders 204 or the mapped error. Matching uses
// errors.Is, so wrapped errors still map. Anything unmapped is a generic 500
// whose cause is logged server-side and never sent to the client.
//
// The first business rule arrives with the DELETE route: a locked shelf
// refuses deletion. The rule lives next to the data, not in the handler, and
// the handler does not know it exists.
//
// Learn: keep status codes out of your domain, map sentinel errors once, and
// write handlers that forward errors instead of branching on them.
// Before: ../03-json-and-validation
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i -X POST localhost:8080/api/shelves -d '{"name":"archive","locked":true}'    # 201, copy the id
//	curl -i -X DELETE localhost:8080/api/shelves/THE_ID_ABOVE                           # 403 PERMISSION_DENIED, "library: shelf is locked"
//	curl -i -X DELETE localhost:8080/api/shelves/11111111-1111-1111-1111-111111111111   # 204
//	curl -i localhost:8080/api/shelves/11111111-1111-1111-1111-111111111111             # 404 NOT_FOUND, "library: not found"
//
// Next: step 5 puts the data behind an interface so it can be swapped and faked (../05-storage-behind-an-interface).
package main

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"uuid"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/rest/errors/validation"
)

func main() {
	app := iris.New()

	// The one place where domain errors become status codes. Step 12 hands
	// the same value to the builder's Errors step.
	errorOptions := errors.NewOptions().
		MapErrors(errors.NotFound, ErrNotFound).
		MapErrors(errors.PermissionDenied, ErrShelfLocked)
	app.Use(errorOptions.Handler())

	api := app.Router("/api")
	api.Get("/shelves", listShelves)
	api.Post("/shelves", createShelf)
	api.Get("/shelves/{id:uuid}", getShelf)
	api.Delete("/shelves/{id:uuid}", deleteShelf)

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

// Sentinel errors: the domain speaks its own language and the HTTP layer maps
// them to status codes exactly once, in main.
var (
	ErrNotFound    = fmt.Errorf("library: not found")
	ErrShelfLocked = fmt.Errorf("library: shelf is locked")
)

var seedShelfID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

var (
	mu      sync.RWMutex
	shelves = map[uuid.UUID]Shelf{
		seedShelfID: {ID: seedShelfID, Name: "sci-fi", CreatedAt: time.Now().UTC()},
	}
)

func listShelves(ctx iris.Context) {
	list, err := allShelves()
	ctx.OK(list, err)
}

func createShelf(ctx iris.Context) {
	input, ok := ctx.BindJSON[ShelfInput]()
	if !ok {
		return
	}

	id, err := storeShelf(input)
	ctx.Created(id, err)
}

func getShelf(ctx iris.Context) {
	id := ctx.Params().GetUUID("id")

	shelf, err := findShelf(id)
	ctx.OK(shelf, err) // ErrNotFound renders as the mapped 404.
}

func deleteShelf(ctx iris.Context) {
	id := ctx.Params().GetUUID("id")

	err := removeShelf(id)
	// The shelf was there and is gone now, so a success is a 204.
	ctx.NoContent(true, err) // 204; 403 for a locked shelf, 404 for a missing one.
}

// The data functions return (value, error) and never touch the Context. The
// handlers above are now one call each.

func allShelves() ([]Shelf, error) {
	mu.RLock()
	defer mu.RUnlock()

	list := make([]Shelf, 0, len(shelves))
	for _, shelf := range shelves {
		list = append(list, shelf)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
	return list, nil
}

func storeShelf(in ShelfInput) (uuid.UUID, error) {
	shelf := Shelf{ID: uuid.New(), Name: in.Name, Locked: in.Locked, CreatedAt: time.Now().UTC()}

	mu.Lock()
	defer mu.Unlock()
	shelves[shelf.ID] = shelf
	return shelf.ID, nil
}

func findShelf(id uuid.UUID) (Shelf, error) {
	mu.RLock()
	defer mu.RUnlock()

	shelf, ok := shelves[id]
	if !ok {
		return Shelf{}, ErrNotFound
	}
	return shelf, nil
}

// removeShelf refuses a locked shelf. The rule lives here, next to the data,
// and its error is mapped to 403 once, in main.
func removeShelf(id uuid.UUID) error {
	mu.Lock()
	defer mu.Unlock()

	shelf, ok := shelves[id]
	if !ok {
		return ErrNotFound
	}
	if shelf.Locked {
		return ErrShelfLocked
	}
	delete(shelves, id)
	return nil
}
