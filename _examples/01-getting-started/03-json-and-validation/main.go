// Step 3 of 13: a JSON body, validated before the handler sees it.
//
// POST /api/shelves creates a shelf. The payload is its own type, ShelfInput,
// and because the type has a Validate() error method, ctx.BindJSON runs it
// right after decoding. A malformed body and a failed validation both end as a
// 400 that is already written when BindJSON returns ok == false, so the handler
// has exactly one branch: return.
//
// The validation builders in rest/errors/validation produce field-level
// errors, and the 400 lists them one by one:
//
//	{"http_error_code":{"canonical_name":"INVALID_ARGUMENT","status":400},
//	 "message":"validation failure","detail":"fields were invalid",
//	 "validation":[{"field":"name","value":"","reason":"must not be empty, must be between 1 and 100 characters"}]}
//
// ctx.Created follows the same helper contract as ctx.OK and renders 201; an
// id value becomes {"id": "..."}. Handlers now write, so the data moves into a
// map behind a mutex: Iris serves requests concurrently.
//
// Learn: decode and validate a request body in one call, and know the exact
// 400 clients receive.
// Before: ../02-routes-and-params
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i -X POST localhost:8080/api/shelves -d '{"name":"history"}'   # 201 {"id":"..."}
//	curl -i -X POST localhost:8080/api/shelves -d '{"name":""}'          # 400, one field error
//	curl -i -X POST localhost:8080/api/shelves -d '{not json'            # 400, malformed body
//	curl localhost:8080/api/shelves                                      # 200, two shelves
//
// Next: step 4 maps domain errors to status codes in one place (../04-errors-mapped-once).
package main

import (
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

	api := app.Router("/api")
	api.Get("/shelves", listShelves)
	api.Post("/shelves", createShelf)
	api.Get("/shelves/{id:uuid}", getShelf)

	app.Listen(":8080")
}

type Shelf struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Locked    bool      `json:"locked"`
	CreatedAt time.Time `json:"created_at"`
}

// ShelfInput is the create payload. Implementing Validate makes ctx.BindJSON
// run it after decoding, so a bad body never reaches a handler.
type ShelfInput struct {
	Name   string `json:"name"`
	Locked bool   `json:"locked"`
}

func (in ShelfInput) Validate() error {
	return validation.Join(
		validation.String("name", in.Name).NotEmpty().Length(1, 100),
	)
}

var seedShelfID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

// shelves is still the whole database, now a map behind a read-write mutex
// because createShelf writes to it while other requests read.
var (
	mu      sync.RWMutex
	shelves = map[uuid.UUID]Shelf{
		seedShelfID: {ID: seedShelfID, Name: "sci-fi", CreatedAt: time.Now().UTC()},
	}
)

func listShelves(ctx iris.Context) {
	mu.RLock()
	list := make([]Shelf, 0, len(shelves))
	for _, shelf := range shelves {
		list = append(list, shelf)
	}
	mu.RUnlock()

	// Map iteration order is random; clients deserve a stable listing.
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
	ctx.OK(list, nil)
}

func createShelf(ctx iris.Context) {
	input, ok := ctx.BindJSON[ShelfInput]()
	if !ok {
		return // 400 already written: malformed JSON or failed Validate.
	}

	shelf := Shelf{ID: uuid.New(), Name: input.Name, Locked: input.Locked, CreatedAt: time.Now().UTC()}

	mu.Lock()
	shelves[shelf.ID] = shelf
	mu.Unlock()

	ctx.Created(shelf.ID, nil) // 201 with {"id": "..."}.
}

func getShelf(ctx iris.Context) {
	id := ctx.Params().GetUUID("id")

	mu.RLock()
	shelf, ok := shelves[id]
	mu.RUnlock()
	if !ok {
		errors.NotFoundID(ctx, "shelf", id)
		return
	}

	ctx.OK(shelf, nil)
}
