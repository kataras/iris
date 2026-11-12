package mvc_test

import (
	"strings"
	"testing"
	"uuid"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/httptest"
)

// The CRUD routes are generated from the controller's ID type parameter.
// Every test in this tree used ID = string, which hid two defects:
//
//  1. the Go type -> route macro mapping always answered "uuid", so an int id
//     registered "{id:uuid}" and every request to it 404'd, and
//  2. the path parameter decoder only accepted a value whose runtime type
//     matched ID exactly, while the "uuid" macro stored the raw string form at
//     the time (it stores a uuid.UUID now) - so a uuid.UUID id answered 400 on
//     every read.

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ItemFilter struct {
	// Embedded so that the list route can take a sort policy; see
	// TestUseControllerOptions.
	iris.ListFilter

	Name string `json:"name"`
}

type ItemController struct{}

func (c *ItemController) Create(ctx iris.Context, item Item) (Item, error) {
	return item, nil
}

func (c *ItemController) Get(ctx iris.Context, id int) (Item, error) {
	return Item{ID: id, Name: "item"}, nil
}

func (c *ItemController) Update(ctx iris.Context, id int, item Item, onlyColumns ...string) (*Item, bool, error) {
	item.ID = id
	if len(onlyColumns) > 0 {
		// Echoed so that a test can pin how ?columns= was parsed; the routes
		// this controller serves otherwise ignore it.
		item.Name = strings.Join(onlyColumns, "|")
	}

	return &item, true, nil
}

func (c *ItemController) Delete(ctx iris.Context, id int) (bool, error) {
	return true, nil
}

func (c *ItemController) List(ctx iris.Context, filter ItemFilter, limit, offset int) ([]Item, int, error) {
	return []Item{{ID: 1, Name: "item"}}, 1, nil
}

func TestControllerIntID(t *testing.T) {
	app := iris.New()
	app.Router("/items", mvc.UseController[*ItemController]())

	e := httptest.New(t, app)

	e.GET("/items/42").Expect().Status(httptest.StatusOK).
		JSON().Object().Value("id").IsEqual(42)

	// A non-numeric id must not match the route at all.
	e.GET("/items/not-a-number").Expect().Status(httptest.StatusNotFound)

	e.DELETE("/items/42").Expect().Status(httptest.StatusNoContent)
}

type Thing struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ThingFilter struct {
	Name string `json:"name"`
}

type ThingController struct{}

func (c *ThingController) Create(ctx iris.Context, thing Thing) (Thing, error) {
	return thing, nil
}

func (c *ThingController) Get(ctx iris.Context, id uuid.UUID) (Thing, error) {
	return Thing{ID: id, Name: "thing"}, nil
}

func (c *ThingController) Update(ctx iris.Context, id uuid.UUID, thing Thing, onlyColumns ...string) (*Thing, bool, error) {
	thing.ID = id
	return &thing, true, nil
}

func (c *ThingController) Delete(ctx iris.Context, id uuid.UUID) (bool, error) {
	return true, nil
}

func (c *ThingController) List(ctx iris.Context, filter ThingFilter, limit, offset int) ([]Thing, int, error) {
	return []Thing{}, 0, nil
}

func TestControllerUUIDID(t *testing.T) {
	app := iris.New()
	app.Router("/things", mvc.UseController[*ThingController]())

	e := httptest.New(t, app)

	const id = "a6ce7330-d584-4675-af9e-2c2d2c40de07"

	e.GET("/things/" + id).Expect().Status(httptest.StatusOK).
		JSON().Object().Value("id").IsEqual(id)

	e.GET("/things/not-a-uuid").Expect().Status(httptest.StatusNotFound)

	e.DELETE("/things/" + id).Expect().Status(httptest.StatusNoContent)
}

type Note struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type NoteFilter struct {
	Name string `json:"name"`
}

type NoteController struct{}

func (c *NoteController) Create(ctx iris.Context, note Note) (Note, error) {
	return note, nil
}

func (c *NoteController) Get(ctx iris.Context, id string) (Note, error) {
	return Note{ID: id, Name: "note"}, nil
}

func (c *NoteController) Update(ctx iris.Context, id string, note Note, onlyColumns ...string) (*Note, bool, error) {
	note.ID = id
	return &note, true, nil
}

func (c *NoteController) Delete(ctx iris.Context, id string) (bool, error) {
	return true, nil
}

func (c *NoteController) List(ctx iris.Context, filter NoteFilter, limit, offset int) ([]Note, int, error) {
	return []Note{}, 0, nil
}

// A string ID is a string, not a UUID: only a controller declared with
// uuid.UUID gets the uuid macro and its validation.
func TestControllerStringID(t *testing.T) {
	app := iris.New()
	app.Router("/notes", mvc.UseController[*NoteController]())

	e := httptest.New(t, app)

	e.GET("/notes/shopping-list").Expect().Status(httptest.StatusOK).
		JSON().Object().Value("id").IsEqual("shopping-list")

	e.DELETE("/notes/shopping-list").Expect().Status(httptest.StatusNoContent)
}
