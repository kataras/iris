// Step 2 of 13: routes, a router prefix and a typed path parameter.
//
// The Bookshelf gets its first entity, Shelf, and two read-only routes under
// /api. The data is a slice for now; the point of this step is the routing.
//
// app.Router("/api") creates a child router: every path registered on it is
// relative to the prefix. {id:uuid} is a typed path parameter, a macro: the
// router matches the segment only when it is a well-formed UUID, so the handler
// never sees garbage, and it reads the parsed value with
// ctx.Params().GetUUID("id"). Response helpers such as ctx.OK follow one contract,
// ctx.Helper(value, err): render the value on a nil error, otherwise hand the
// error to the error pipeline. The error is always nil in this step; from step
// 4 on, real errors flow through the same call.
//
// Learn: group routes under a prefix, read a typed path parameter and answer
// 200 or a 404 in the standard error envelope.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i localhost:8080/api/shelves                                         # 200, one shelf
//	curl -i localhost:8080/api/shelves/11111111-1111-1111-1111-111111111111    # 200
//	curl -i localhost:8080/api/shelves/00000000-0000-0000-0000-000000000000    # 404 NOT_FOUND, "shelf ... not found"
//	curl -i localhost:8080/api/shelves/not-a-uuid                              # 404: {id:uuid} did not match (step 12 turns this into a 400)
//
// Next: step 3 accepts a JSON body and validates it (../03-json-and-validation).
package main

import (
	"time"

	"uuid"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	// Routes registered on api live under /api.
	api := app.Router("/api")
	api.Get("/shelves", listShelves)
	api.Get("/shelves/{id:uuid}", getShelf)

	app.Listen(":8080")
}

// Shelf is the first entity of the Bookshelf. The json tags are the response
// shape; nothing else about the type is HTTP.
type Shelf struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Locked    bool      `json:"locked"`
	CreatedAt time.Time `json:"created_at"`
}

// seedShelfID is a fixed id so the Try lines above work as written.
var seedShelfID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

// shelves is the whole "database" for now. It is only read, so no lock yet.
var shelves = []Shelf{
	{ID: seedShelfID, Name: "sci-fi", CreatedAt: time.Now().UTC()},
}

func listShelves(ctx iris.Context) {
	ctx.OK(shelves, nil) // 200 with the slice as a JSON array.
}

func getShelf(ctx iris.Context) {
	// {id:uuid} rejected anything that is not a UUID before this ran, and the
	// router parsed the value once, so the handler reads the typed uuid.UUID.
	id := ctx.Params().GetUUID("id")

	for _, shelf := range shelves {
		if shelf.ID == id {
			ctx.OK(shelf, nil)
			return
		}
	}

	// A 404 whose body is the same envelope every Iris error uses:
	// {"http_error_code":{"canonical_name":"NOT_FOUND","status":404},"message":"resource not found","detail":"shelf <id> not found"}
	errors.NotFoundID(ctx, "shelf", id)
}
