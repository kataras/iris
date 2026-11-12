// A real database behind the repository seam: kataras/pg, with its errors
// mapped to HTTP.
//
// The schema is the struct: pg tags describe the columns, pg.NewSchema()
// registers the table and db.CreateSchema creates it when it is missing.
// pg.NewRepository[Book] gives typed Insert/Select/Delete calls, and every one
// of them takes a context.Context, which iris.Context satisfies, so a client
// that disconnects cancels its query.
//
// The blank import of rest/errors/pgerrors is what connects the two worlds:
// pg.ErrNoRows renders as 404 and a unique-key violation as 409 through the
// same ctx.OK/Created/NoContent helpers as everywhere else. No handler here
// inspects a database error.
//
// The Store owns the connection. It opens it in Init, which the builder calls
// inside Build after injection and before serving, and releases it in Close,
// which runs at shutdown because Services notices the method. DATABASE_URL is
// read in main, never written in a file: connection strings are secrets.
//
// Learn: wire a Postgres repository through pg, create the schema from struct
// tags, and get the database's errors mapped to status codes without code.
// Before: ../postgres-mapping
//
// Run (a throwaway Postgres in Docker, then the example against it):
//
//	docker run --rm -e POSTGRES_PASSWORD=iris -p 5432:5432 postgres
//	DATABASE_URL='postgres://postgres:iris@localhost:5432/postgres?sslmode=disable' go run main.go
//
// Try:
//
//	curl -i -X POST localhost:8080/api/books -d '{"title":"Dune","author":"Frank Herbert","pages":412}'   # 201 {"id":"..."}, copy the id
//	curl localhost:8080/api/books                                                                       # 200, newest first
//	curl -i localhost:8080/api/books/THE_ID                                                             # 200
//	curl -i localhost:8080/api/books/00000000-0000-0000-0000-000000000000                               # 404: pg.ErrNoRows through pgerrors
//	curl -i -X POST localhost:8080/api/books -d '{"title":"Dune","author":"Frank Herbert","pages":412}'   # 409: the unique title through pgerrors
//	curl -i -X DELETE localhost:8080/api/books/THE_ID                                                   # 204
//	curl -i -X DELETE localhost:8080/api/books/THE_ID                                                   # 304: the row was already gone, nothing changed
//
// Next: ../../13-websockets/chat
package main

import (
	stdContext "context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
	_ "github.com/kataras/iris/v14/rest/errors/pgerrors" // pg.ErrNoRows to 404, duplicates to 409.
	"github.com/kataras/iris/v14/rest/errors/validation"
	"github.com/kataras/pg"
)

func main() {
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		log.Fatal("DATABASE_URL environment variable is required, for example postgres://postgres:iris@localhost:5432/postgres?sslmode=disable")
	}

	app := iris.NewBuilder().
		Prefix("/api").
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Errors(errors.NewOptions()). // nothing of our own to map; pgerrors registered itself.
		Services(NewStore(connString)).
		RouterMiddlewares().
		Middlewares().
		API("/books", NewBooksAPI).
		Build()

	app.Listen(":8080")
}

// Store owns the database connection and the typed repository. Init and Close
// give it a lifetime the builder manages.
type Store struct {
	connString string
	db         *pg.DB
	books      *pg.Repository[Book]
}

func NewStore(connString string) *Store {
	return &Store{connString: connString}
}

// Init runs inside Build, after injection and before serving: open, create
// the missing tables, verify the schema matches the structs.
func (s *Store) Init(ctx stdContext.Context) error {
	schema := pg.NewSchema().MustRegister("books", Book{})

	db, err := pg.Open(ctx, schema, s.connString)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	if err := db.CreateSchema(ctx); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if err := db.CheckSchema(ctx); err != nil {
		return fmt.Errorf("check schema: %w", err)
	}

	s.db = db
	s.books = pg.NewRepository[Book](db)
	log.Println("books: database ready")
	return nil
}

// Close runs at shutdown.
func (s *Store) Close() error {
	if s.db != nil {
		s.db.Close()
	}
	log.Println("books: database closed")
	return nil
}

// BooksAPI is thin on purpose: read input, call the repository, hand the
// (value, err) pair to a response helper. The database errors are mapped on
// the way out by the pgerrors resolver.
type BooksAPI struct {
	store *Store
}

func NewBooksAPI(store *Store) *BooksAPI {
	return &BooksAPI{store: store}
}

func (api *BooksAPI) Routing(r iris.Router) {
	r.Get("/", api.list)
	r.Post("/", api.create)
	r.Get("/{id:uuid}", api.get)
	r.Delete("/{id:uuid}", api.delete)
}

func (api *BooksAPI) list(ctx iris.Context) {
	books, err := api.store.books.Select(ctx, `SELECT * FROM books ORDER BY created_at DESC`)
	ctx.OK(books, err)
}

func (api *BooksAPI) create(ctx iris.Context) {
	input, ok := ctx.BindJSON[BookInput]()
	if !ok {
		return
	}

	book := Book{Title: input.Title, Author: input.Author, Pages: input.Pages}
	// InsertSingle writes the generated id back through the pointer.
	err := api.store.books.InsertSingle(ctx, book, &book.ID)
	ctx.Created(book.ID, err) // 201 {"id": "..."}; 409 on a duplicate title.
}

func (api *BooksAPI) get(ctx iris.Context) {
	book, err := api.store.books.SelectByID(ctx, ctx.Params().GetString("id"))
	ctx.OK(book, err) // 404 when pg.ErrNoRows comes back.
}

func (api *BooksAPI) delete(ctx iris.Context) {
	deleted, err := api.store.books.DeleteByID(ctx, ctx.Params().GetString("id"))
	ctx.NoContent(deleted, err) // 204 when a row went away, 304 when nothing changed.
}

// Book is the table. The pg tags are the DDL: a uuid primary key that the
// database generates, a creation timestamp with a default, and a unique title
// so the 409 above has something to trip on.
type Book struct {
	ID        string    `pg:"type=uuid,primary" json:"id"`
	CreatedAt time.Time `pg:"type=timestamp,default=clock_timestamp()" json:"created_at"`
	Title     string    `pg:"type=varchar(255),unique" json:"title"`
	Author    string    `pg:"type=varchar(255)" json:"author"`
	Pages     int       `pg:"type=int" json:"pages"`
}

// BookInput is the create payload; ctx.BindJSON runs Validate after decoding.
type BookInput struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Pages  int    `json:"pages"`
}

func (in BookInput) Validate() error {
	return validation.Join(
		validation.String("title", in.Title).NotEmpty().Length(1, 255),
		validation.String("author", in.Author).NotEmpty(),
		validation.Number("pages", in.Pages).Positive(),
	)
}
