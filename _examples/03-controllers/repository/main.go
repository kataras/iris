// mvc.UseRepository: the six resource routes over a repository, with no
// controller in between.
//
// When a resource's HTTP half adds nothing to its persistence half there is no
// type left to write, so UseRepository generates the routes straight from a
// mvc.Repository[T, ID, Filter]:
//
//	POST   /notes              -> 201 {"id": ...}
//	GET    /notes/{note_id}    -> 200 the model
//	PUT    /notes/{note_id}    -> 204, or 304 when nothing changed
//	PATCH  /notes/{note_id}    -> the same, honouring ?columns=a&columns=b
//	DELETE /notes/{note_id}    -> 204, or 304 when nothing was deleted
//	POST   /notes/list         -> 200 a page of models
//
// The path parameter's name comes from the model (Note becomes note_id) and its
// macro from the ID type, so a uuid.UUID id would register {note_id:uuid} and
// validate before any handler runs; a string id stays {note_id:string}.
//
// The four options below are the ones worth knowing:
//
//	mvc.Guard        middleware for the write routes only, so reads stay public
//	mvc.Pagination   the list route's sort policy, checked before the repository
//	mvc.BeforeWrite  a hook over the decoded body, for values the client must
//	                 not choose (here the owner, taken from the API key)
//	mvc.Without      drops a generated route entirely (not used here)
//
// The update routes read their columns through ctx.BindUpdate, so a column that
// is not a member of Note answers 400 naming it and the members it could have
// been, instead of updating nothing and reporting success. The names that reach
// the repository are still the client's: a repository building SQL from them
// maps them to its own columns and never interpolates them, which is what the
// switch in Update does.
//
// Learn: serve a resource straight from an mvc.Repository with
// mvc.UseRepository, guard the writes, bound the sort columns and stamp
// server-owned fields before they are stored.
// Before: ../mvc-generics
//
// Run:
//
//	go run main.go
//
// Try:
//
//	K='X-API-Key: editor-key'
//
//	curl -i -X POST localhost:8080/notes -H "$K" -d '{"title":"read the mvc chapter","body":"start at UseController"}'
//	                                                       # 201 {"id":"note-1"}
//	curl -i -X POST localhost:8080/notes -d '{"title":"anonymous"}'
//	                                                       # 401 UNAUTHENTICATED, writes are guarded
//	curl -i localhost:8080/notes/note-1                    # 200, owner "editor"
//	curl -i localhost:8080/notes/note-9                    # 404 NOT_FOUND
//	curl -i -X PUT 'localhost:8080/notes/note-1?columns=title' -H "$K" -d '{"title":"renamed"}'
//	                                                       # 204 No Content
//	curl -i -X PUT 'localhost:8080/notes/note-1?columns=title' -H "$K" -d '{"title":"renamed"}'
//	                                                       # 304 Not Modified, the same value again
//	curl -i -X PUT 'localhost:8080/notes/note-1?columns=titel' -H "$K" -d '{"title":"typo"}'
//	                                                       # 400 INVALID_ARGUMENT, unknown column
//	curl -i -X POST localhost:8080/notes/list -d '{"sort":"title"}'
//	                                                       # 200, the Page envelope
//	curl -i -X POST localhost:8080/notes/list -d '{"sort":"body"}'
//	                                                       # 400 INVALID_ARGUMENT, sort not allowed
//	curl -i -X DELETE localhost:8080/notes/note-1 -H "$K"  # 204 No Content
//	curl -i -X DELETE localhost:8080/notes/note-1 -H "$K"  # 304 Not Modified, already gone
//
// Next: ../../04-request/query-and-forms
package main

import (
	stdContext "context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/middleware/apikey"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	// The guard runs on Create, Update and Delete only. Reads stay public,
	// which a router-level Use cannot express without splitting the resource
	// into two routers.
	guard := apikey.New(apikey.Options{
		Header: "X-API-Key",
		Scheme: apikey.NoScheme,
		Keys:   map[string]string{"editor": "editor-key"},
	})

	app.Router("/notes", mvc.UseRepository(NewNoteRepository(),
		mvc.Guard(guard),
		mvc.Pagination(iris.AllowSort(sortableColumns...)),
		mvc.BeforeWrite[Note](stampOwner),
	))

	app.Listen(":8080")
}

// NoteRepository is an in-memory mvc.Repository[Note, string, NoteFilter]. A
// real one talks to a database; the five methods are the same five.
type NoteRepository struct {
	mu    sync.RWMutex
	notes map[string]Note
	next  int
}

func NewNoteRepository() *NoteRepository {
	return &NoteRepository{notes: make(map[string]Note)}
}

// Create stores v and returns its new identifier, which ctx.Created renders as
// 201 {"id": "note-1"}.
func (r *NoteRepository) Create(_ stdContext.Context, v Note) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.next++
	v.ID = fmt.Sprintf("note-%d", r.next)
	v.CreatedAt = time.Now()
	r.notes[v.ID] = v

	return v.ID, nil
}

func (r *NoteRepository) Get(_ stdContext.Context, id string) (Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	note, ok := r.notes[id]
	if !ok {
		return Note{}, errors.NotFound.Wrap(fmt.Errorf("note %s not found", id))
	}

	return note, nil
}

// Update reports whether anything changed, and that boolean is the whole
// difference between the 204 and the 304 the generated route answers.
func (r *NoteRepository) Update(_ stdContext.Context, id string, v Note,
	onlyColumns ...string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stored, ok := r.notes[id]
	if !ok {
		return false, errors.NotFound.Wrap(fmt.Errorf("note %s not found", id))
	}

	// The client's column names, mapped to this repository's own fields. An
	// empty list means a full update. The id, the owner and the timestamp are
	// the server's: naming one of them here changes nothing.
	updated := stored
	for _, column := range columnsOrAll(onlyColumns) {
		switch column {
		case "title":
			updated.Title = v.Title
		case "body":
			updated.Body = v.Body
		}
	}

	if updated == stored {
		return false, nil // 304 Not Modified.
	}

	r.notes[id] = updated
	return true, nil // 204 No Content.
}

func (r *NoteRepository) Delete(_ stdContext.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.notes[id]; !ok {
		return false, nil // 304: there was nothing to delete.
	}

	delete(r.notes, id)
	return true, nil // 204.
}

// List returns one page and the total count of the matches, ignoring limit and
// offset, which is what the Page envelope needs for total_pages.
func (r *NoteRepository) List(_ stdContext.Context, filter NoteFilter,
	limit, offset int) ([]Note, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	matched := make([]Note, 0, len(r.notes))
	for _, note := range r.notes {
		if filter.Owner != "" && note.Owner != filter.Owner {
			continue
		}

		if filter.Term != "" && !strings.Contains(
			strings.ToLower(note.Title), strings.ToLower(filter.Term)) {
			continue
		}

		matched = append(matched, note)
	}

	// mvc.Pagination already rejected a column outside sortableColumns with a
	// 400, so this is the same list read from the inside: SortColumn answers
	// ("", nil) when no sort was asked for, and the default order stands.
	column, err := filter.SortColumn(sortableColumns...)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %w", errors.ErrRequest, err)
	}

	sortNotes(matched, column, filter.SortOrder() == "DESC")

	total := int64(len(matched))
	if offset >= len(matched) {
		return nil, total, nil // past the last page.
	}

	return matched[offset:min(offset+limit, len(matched))], total, nil
}

// Note is the model. Its JSON member names are the column names the update
// routes accept, so "title" and "created_at" are valid ?columns= values and
// anything else is a 400.
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Owner     string    `json:"owner"`
	CreatedAt time.Time `json:"created_at"`
}

// NoteFilter is the POST /notes/list body. Embedding iris.ListFilter brings the
// fields every list endpoint shares (term, names, substring, sort,
// sort_descending) and, with them, the SortRequest method the sort policy reads.
type NoteFilter struct {
	iris.ListFilter

	Owner string `json:"owner,omitzero"`
}

// sortableColumns is stated once, by the endpoint. A sort column reaches an
// ORDER BY clause, so the set a client may name never comes from the request.
var sortableColumns = []string{"title", "created_at"}

// stampOwner is the mvc.BeforeWrite hook: it runs on Create and on Update,
// after the body is decoded and before the repository sees it. It is how a
// value the client must not choose gets onto every write.
func stampOwner(ctx iris.Context, v *Note) error {
	principal, ok := apikey.Get(ctx)
	if !ok {
		// Unreachable below the guard, but a hook that fails open is a hook
		// that writes somebody else's rows the day the guard moves.
		return errors.Unauthenticated.Wrap(fmt.Errorf("no authenticated principal"))
	}

	v.Owner = principal
	return nil
}

func columnsOrAll(columns []string) []string {
	if len(columns) == 0 {
		return []string{"title", "body"}
	}

	return columns
}

func sortNotes(notes []Note, column string, descending bool) {
	if column == "" {
		column = "created_at"
	}

	slices.SortStableFunc(notes, func(a, b Note) int {
		order := 0
		switch column {
		case "title":
			order = strings.Compare(a.Title, b.Title)
		case "created_at":
			order = a.CreatedAt.Compare(b.CreatedAt)
		}

		if descending {
			return -order
		}

		return order
	})
}
