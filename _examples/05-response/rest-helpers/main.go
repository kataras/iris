// The typed response helpers: one contract, ctx.Helper(value, err).
//
// Every helper - ctx.OK(value, err), ctx.Created(value, err),
// ctx.NoContent(modified, err) and ctx.Count(n, err) - takes the value and the
// error a service returned. On a non-nil error it writes the structured error
// response of rest/errors and reports false; otherwise it renders the
// success response. Handlers never branch on the error themselves.
//
// Learn: hand a service's (value, err) pair to one helper and get the right
// 2xx, or the mapped error, without an if in the handler.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/books                                                                   # 200, the JSON array
//	curl http://localhost:8080/books/count                                                             # 200, {"count":1}
//	curl -i -X POST http://localhost:8080/books -d '{"title":"Go in Depth","author":"Ada"}'            # 201, {"id":2}
//	curl -i -X PUT http://localhost:8080/books/1 -d '{"title":"Iris in Action","author":"Jane Doe"}'   # 304, the stored values already
//	curl -i -X PUT http://localhost:8080/books/1 -d '{"title":"Iris in Depth","author":"Jane Doe"}'    # 204, changed
//	curl -i -X DELETE http://localhost:8080/books/1                                                    # 204
//	curl -i http://localhost:8080/books/999                                                            # 404 from errors.NotFound
//
// Next: ../writers
package main

import (
	"fmt"
	"sync"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	store := newStore()

	app := iris.New()

	books := app.Router("/books")
	{
		// 200 with the JSON array. A nil value would send nothing.
		books.Get("/", func(ctx iris.Context) {
			list, err := store.List()
			ctx.OK(list, err)
		})

		// 200 with {"count": N}, zero included.
		books.Get("/count", func(ctx iris.Context) {
			n, err := store.Count()
			ctx.Count(n, err)
		})

		// 200 with the entity, or the 404 error body when the service reports
		// a wrapped errors.NotFound.
		books.Get("/{id:int64}", func(ctx iris.Context) {
			id, ok := ctx.BindParam[int64]("id")
			if !ok {
				return // 400 or 404 already written.
			}

			book, err := store.ByID(id)
			ctx.OK(book, err)
		})

		// 201 with {"id": 3}: Created wraps string, int, int64, uint64 and
		// the standard library's uuid.UUID results in a JSON id object. Any other value, such as the
		// full entity, is sent as-is.
		books.Post("/", func(ctx iris.Context) {
			input, ok := ctx.BindJSON[BookInput]()
			if !ok {
				return // 400 already written: malformed JSON.
			}

			id, err := store.Create(input)
			ctx.Created(id, err)
		})

		// An empty body: 204 when the row changed, 304 when the values were
		// already the ones sent. An unknown id is the mapped error (404).
		books.Put("/{id:int64}", func(ctx iris.Context) {
			id, ok := ctx.BindParam[int64]("id")
			if !ok {
				return
			}

			input, ok := ctx.BindJSON[BookInput]()
			if !ok {
				return
			}

			modified, err := store.Update(id, input)
			ctx.NoContent(modified, err)
		})

		books.Delete("/{id:int64}", func(ctx iris.Context) {
			id, ok := ctx.BindParam[int64]("id")
			if !ok {
				return
			}

			// A delete that found the row always changed something: 204.
			// An unknown id is a 404 through errors.NotFound.
			err := store.Delete(id)
			ctx.NoContent(true, err)
		})
	}

	app.Listen(":8080")
}

// Book is the stored entity and the JSON response shape.
type Book struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

// BookInput is the request payload of create and update.
type BookInput struct {
	Title  string `json:"title"`
	Author string `json:"author"`
}

// store is a stand-in for the application's service layer: it returns
// (value, error) pairs and never touches the Iris context.
type store struct {
	mu     sync.RWMutex
	books  map[int64]Book
	lastID int64
}

func newStore() *store {
	return &store{
		books:  map[int64]Book{1: {ID: 1, Title: "Iris in Action", Author: "Jane Doe"}},
		lastID: 1,
	}
}

func (s *store) List() ([]Book, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Book, 0, len(s.books))
	for _, b := range s.books {
		list = append(list, b)
	}

	return list, nil
}

func (s *store) Count() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return int64(len(s.books)), nil
}

func (s *store) ByID(id int64) (Book, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	book, ok := s.books[id]
	if !ok {
		// Wrapping with an error code name is what makes the helper answer 404
		// instead of the default 500 for an unrecognized error.
		return Book{}, errors.NotFound.Wrap(fmt.Errorf("book %d not found", id))
	}

	return book, nil
}

func (s *store) Create(input BookInput) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastID++
	s.books[s.lastID] = Book{ID: s.lastID, Title: input.Title, Author: input.Author}

	return s.lastID, nil
}

// Update reports whether the stored book actually changed, so the handler can
// answer 204 or 304.
func (s *store) Update(id int64, input BookInput) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	book, ok := s.books[id]
	if !ok {
		return false, errors.NotFound.Wrap(fmt.Errorf("book %d not found", id))
	}

	if book.Title == input.Title && book.Author == input.Author {
		return false, nil // nothing to write.
	}

	book.Title, book.Author = input.Title, input.Author
	s.books[id] = book

	return true, nil
}

func (s *store) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.books[id]; !ok {
		return errors.NotFound.Wrap(fmt.Errorf("book %d not found", id))
	}

	delete(s.books, id)

	return nil
}
