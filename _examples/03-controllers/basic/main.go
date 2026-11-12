// Controllers: a constructor and a Routing method, mounted with app.Router.
//
// A controller is a plain struct with exactly two things: NewXxxAPI, the
// constructor that declares its dependencies, and Routing(r iris.Router), the
// route table the router calls once at startup. Mounting resolves the
// constructor through the container: app.Router("/notes", NewNotesAPI) on any
// router, or the builder's API("/notes", NewNotesAPI) step on the production
// stack; the same constructor either way.
//
// Handlers are methods, so they share the struct's dependencies instead of
// reaching for a package-level variable, and the route table stays in one
// readable place.
//
// Learn: write a controller as a NewXxxAPI constructor plus a Routing method,
// register its dependency once and mount it with app.Router.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/notes                                # 200, one seeded note
//	curl -X POST http://localhost:8080/notes -d '{"title":"buy milk"}'  # 201, {"id":2}
//	curl -X POST http://localhost:8080/notes -d '{}'                # 400, title is required
//	curl http://localhost:8080/notes/1                              # 200, the note
//	curl -i -X DELETE http://localhost:8080/notes/1                 # 204
//	curl -i http://localhost:8080/notes/404                         # 404, from errors.NotFound
//
// Next: ../nested-guards
package main

import (
	stdContext "context"
	"fmt"
	"log"
	"sync"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	// The container calls NewNoteService once and keeps the result.
	if err := app.RegisterDependency(NewNoteService); err != nil {
		log.Fatal(err)
	}

	// Mount: the child router at /notes is handed to NotesAPI.Routing.
	app.Router("/notes", NewNotesAPI)

	app.Listen(":8080")
}

// Note is the stored entity and the JSON response shape.
type Note struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// NoteInput is the create payload. ctx.BindJSON calls Validate after
// decoding; a plain error comes back to the client as 400 INVALID_ARGUMENT.
type NoteInput struct {
	Title string `json:"title"`
}

func (in NoteInput) Validate() error {
	if in.Title == "" {
		return fmt.Errorf("title is required")
	}

	return nil
}

// NoteService is the business layer: one singleton shared by every request.
type NoteService struct {
	mu     sync.RWMutex
	notes  []Note
	nextID int
}

func NewNoteService() *NoteService {
	return &NoteService{
		notes:  []Note{{ID: 1, Title: "read the docs"}},
		nextID: 2,
	}
}

func (s *NoteService) List(stdContext.Context) ([]Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Note, len(s.notes))
	copy(list, s.notes)
	return list, nil
}

func (s *NoteService) ByID(_ stdContext.Context, id int) (Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, n := range s.notes {
		if n.ID == id {
			return n, nil
		}
	}

	// The error code name is what turns this into a 404 response.
	return Note{}, errors.NotFound.Wrap(fmt.Errorf("note %d not found", id))
}

func (s *NoteService) Create(_ stdContext.Context, in NoteInput) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	note := Note{ID: s.nextID, Title: in.Title}
	s.notes = append(s.notes, note)
	s.nextID++
	return note.ID, nil
}

func (s *NoteService) Delete(_ stdContext.Context, id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, n := range s.notes {
		if n.ID == id {
			s.notes = append(s.notes[:i], s.notes[i+1:]...)
			return nil
		}
	}

	return errors.NotFound.Wrap(fmt.Errorf("note %d not found", id))
}

// NotesAPI is the controller: the dependencies it needs, and nothing else.
type NotesAPI struct {
	svc *NoteService
}

// NewNotesAPI is called by the container with the registered *NoteService.
func NewNotesAPI(svc *NoteService) *NotesAPI {
	return &NotesAPI{svc: svc}
}

// Routing receives the child router mounted at /notes. Paths are relative to
// it, so "/" is /notes and "/{id:int}" is /notes/42.
func (api *NotesAPI) Routing(r iris.Router) {
	r.Get("/", api.list)
	r.Post("/", api.create)
	r.Get("/{id:int}", api.get)
	r.Delete("/{id:int}", api.delete)
}

func (api *NotesAPI) list(ctx iris.Context) {
	notes, err := api.svc.List(ctx)
	ctx.OK(notes, err)
}

func (api *NotesAPI) create(ctx iris.Context) {
	input, ok := ctx.BindJSON[NoteInput]()
	if !ok {
		return // 400 already written: malformed JSON or failed Validate.
	}

	id, err := api.svc.Create(ctx, input)
	ctx.Created(id, err) // 201 with {"id": 2}.
}

func (api *NotesAPI) get(ctx iris.Context) {
	// The :int macro already rejected anything that is not a number.
	note, err := api.svc.ByID(ctx, ctx.Params().GetInt("id"))
	ctx.OK(note, err)
}

func (api *NotesAPI) delete(ctx iris.Context) {
	err := api.svc.Delete(ctx, ctx.Params().GetInt("id"))
	// A found note is always removed, so a success is a 204.
	ctx.NoContent(true, err) // 204, or 404 for an unknown id.
}
