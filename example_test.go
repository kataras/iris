package iris_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http/httptest"
	"strings"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

// Serve a request without binding a port, which is also how the httptest
// package runs your tests.
func Example() {
	app := iris.New()

	app.Get("/greet/{name:string}", func(ctx iris.Context) {
		ctx.WriteString("Hello, " + ctx.Params().GetString("name"))
	})

	if err := app.Build(); err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	app.Handler.ServeHTTP(w, httptest.NewRequest("GET", "/greet/gopher", nil))

	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
	// Output:
	// 200
	// Hello, gopher
}

// A path parameter carries a type. A request whose value does not match is
// answered before the handler runs, so the handler never sees a bad id.
func ExampleNew() {
	app := iris.New()

	app.Get("/products/{id:uuid}", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"id": ctx.Params().GetString("id")})
	})

	if err := app.Build(); err != nil {
		panic(err)
	}

	// The rejection status depends on how errors are configured, so this only
	// reports whether the request reached the handler at all.
	for _, path := range []string{
		"/products/6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"/products/not-a-uuid",
	} {
		w := httptest.NewRecorder()
		app.Handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		fmt.Println(w.Code == 200)
	}
	// Output:
	// true
	// false
}

// Default pre-registers a debug logger, permissive CORS, panic recovery and
// request IDs. Reach for it while exploring; configure the middleware yourself
// before you ship.
//
// The application it returns is not built yet, so routes are registered on it
// exactly as on New.
func ExampleDefault() {
	app := iris.Default()

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"message": "Hello, World!"})
	})

	// In a program this is app.Listen(":8080"), which builds and serves.
	if err := app.Build(); err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	app.Handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
	// Output:
	// 200
	// {"message":"Hello, World!"}
}

// greetAPI is a controller: the builder resolves its constructor through the
// dependency container and calls Routing to register the group's routes.
type greetAPI struct{}

func newGreetAPI() *greetAPI { return &greetAPI{} }

func (api *greetAPI) Routing(r iris.Router) {
	r.Get("/{name:string}", func(ctx iris.Context) {
		ctx.WriteString("Hello, " + ctx.Params().GetString("name"))
	})
}

// The builder returns a narrower interface at every step, so a chain written in
// the wrong order does not compile. Health comes before Services, Services
// before Middlewares, and API last.
//
// Build closes the chain. Routes registered on the returned Application after
// that point are not picked up, so register them through API.
func ExampleNewBuilder() {
	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("*").
		Compression(true).
		LogRequests(false).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/greet", newGreetAPI).
		Build()

	w := httptest.NewRecorder()
	app.Handler.ServeHTTP(w, httptest.NewRequest("GET", "/greet/gopher", nil))

	fmt.Println(w.Body.String())
	// Output: Hello, gopher
}

// The pieces a real API is wired from: a resource with a Close, a repository
// over it, a service over that, and a controller that the container builds
// from the service.
type exampleDB struct{ closed bool }

func openExampleDB() *exampleDB { return &exampleDB{} }

func (db *exampleDB) Close() error {
	db.closed = true
	return nil
}

type noteRepository struct{ db *exampleDB }

func newNoteRepository(db *exampleDB) *noteRepository {
	return &noteRepository{db: db}
}

func (r *noteRepository) list() []string { return []string{"first", "second"} }

type noteService struct{ repo *noteRepository }

func newNoteService(repo *noteRepository) *noteService {
	return &noteService{repo: repo}
}

func (s *noteService) list() []string { return s.repo.list() }

type noteAPI struct{ service *noteService }

func newNoteAPI(service *noteService) *noteAPI {
	return &noteAPI{service: service}
}

func (api *noteAPI) Routing(r iris.Router) {
	r.Get("/", func(ctx iris.Context) {
		ctx.JSON(api.service.list())
	})
}

// A full chain: the container resolves openExampleDB into the repository, the
// repository into the service and the service into the controller, and the
// database's Close is registered as a shutdown closer because Services
// recognises a value with a Close() error method.
func ExampleNewBuilder_full() {
	app := iris.NewBuilder().
		Prefix("/api").
		AllowOrigin("https://example.com").
		Compression(true).
		LogRequests(false).
		Health(true, "development", "you@example.com").
		Errors(errors.NewOptions().
			MapErrors(errors.NotFound, sql.ErrNoRows)).
		Deferrables(func() { /* stop background workers */ }).
		Services(
			openExampleDB(),
			newNoteRepository,
			newNoteService,
		).
		RouterMiddlewares(func(ctx iris.Context) {
			ctx.SetHeader("X-Api", "notes")
			ctx.Next()
		}).
		Middlewares(func(service *noteService) iris.Handler {
			return func(ctx iris.Context) {
				ctx.Values().Set("notes", len(service.list()))
				ctx.Next()
			}
		}).
		API("/notes", newNoteAPI).
		Build()

	w := httptest.NewRecorder()
	app.Handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/notes", nil))

	fmt.Println(w.Header().Get("X-Api"))
	fmt.Println(w.Body.String())

	if err := app.Shutdown(context.Background()); err != nil {
		panic(err)
	}
	// Output:
	// notes
	// ["first","second"]
}

// BindJSON decodes the request body into the type you name. When it returns
// false the client error (400 for malformed JSON, 413 for a body over
// BodyMaxMemory) has already been written, so the handler only returns.
func ExampleContext_BindJSON() {
	type createUser struct {
		Name string `json:"name"`
	}

	app := iris.New()
	app.Post("/users", func(ctx iris.Context) {
		in, ok := ctx.BindJSON[createUser]()
		if !ok {
			return
		}
		ctx.Writef("creating %s", in.Name)
	})

	if err := app.Build(); err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	app.Handler.ServeHTTP(w, httptest.NewRequest("POST", "/users", strings.NewReader(`{"name":"gopher"}`)))
	fmt.Println(w.Code, w.Body.String())

	// The 400 body is a rest/errors envelope with a fresh id, so only the
	// status is printed here.
	w = httptest.NewRecorder()
	app.Handler.ServeHTTP(w, httptest.NewRequest("POST", "/users", strings.NewReader(`{"name":`)))
	fmt.Println(w.Code)
	// Output:
	// 200 creating gopher
	// 400
}

// BindParams binds the typed path parameters to a struct through the `param`
// field tag, so a route with several parameters reads them in one call.
func ExampleContext_BindParams() {
	type bookPath struct {
		StoreID int    `param:"store_id"`
		Book    string `param:"book"`
	}

	app := iris.New()
	app.Get("/stores/{store_id:int}/books/{book:string}", func(ctx iris.Context) {
		p, ok := ctx.BindParams[bookPath]()
		if !ok {
			return
		}
		ctx.Writef("store %d, book %s", p.StoreID, p.Book)
	})

	if err := app.Build(); err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	app.Handler.ServeHTTP(w, httptest.NewRequest("GET", "/stores/7/books/iris", nil))
	fmt.Println(w.Body.String())
	// Output: store 7, book iris
}

// Paginated reads the page and size from the URL query and the filter from the
// JSON body, calls your collector with them and writes the Page envelope.
// The collector receives the offset and limit already computed; page_size in
// the envelope is the number of items on this page.
func ExampleContext_Paginated() {
	type bookFilter struct {
		Author string `json:"author"`
	}
	books := []string{"first", "second", "third"}

	app := iris.New()
	app.Post("/books/list", func(ctx iris.Context) {
		ctx.Paginated(func(_ context.Context, opts iris.PageOptions, f bookFilter) ([]string, int64, error) {
			start := min(opts.GetOffset(), len(books))
			end := min(start+opts.GetLimit(), len(books))
			return books[start:end], int64(len(books)), nil
		})
	})

	if err := app.Build(); err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	app.Handler.ServeHTTP(w, httptest.NewRequest("POST", "/books/list?page=2&size=2", strings.NewReader(`{"author":"any"}`)))
	fmt.Println(w.Body.String())
	// Output: {"current_page":2,"page_size":1,"total_pages":2,"total_items":3,"has_next_page":false,"filter":{"author":"any"},"items":["third"]}
}

// Count answers a "how many" call with {"count": n}. Zero is an ordinary 200;
// a non-nil error is rendered through rest/errors instead.
func ExampleContext_Count() {
	app := iris.New()
	app.Delete("/sessions", func(ctx iris.Context) {
		var removed int64 = 3 // what the store reported.
		ctx.Count(removed, nil)
	})

	if err := app.Build(); err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	app.Handler.ServeHTTP(w, httptest.NewRequest("DELETE", "/sessions", nil))
	fmt.Println(w.Code, w.Body.String())
	// Output: 200 {"count":3}
}

// OK takes a service call's value and error together: the value renders as
// JSON, the error is rendered through rest/errors (here tagged NotFound with
// errors.NotFound.Wrap; an application-wide map does the same for a bare
// sql.ErrNoRows) and OK reports false.
func ExampleContext_OK() {
	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	find := func(id int) (user, error) {
		if id != 1 {
			return user{}, errors.NotFound.Wrap(sql.ErrNoRows)
		}
		return user{ID: 1, Name: "gopher"}, nil
	}

	app := iris.New()
	app.Get("/users/{id:int}", func(ctx iris.Context) {
		ctx.OK(find(ctx.Params().GetInt("id")))
	})

	if err := app.Build(); err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	app.Handler.ServeHTTP(w, httptest.NewRequest("GET", "/users/1", nil))
	fmt.Println(w.Code, strings.TrimSpace(w.Body.String()))

	w = httptest.NewRecorder()
	app.Handler.ServeHTTP(w, httptest.NewRequest("GET", "/users/2", nil))
	fmt.Println(w.Code)
	// Output:
	// 200 {"id":1,"name":"gopher"}
	// 404
}

// Created answers 201. A bare identifier (string, int, int64, uint64 or
// uuid.UUID) is wrapped as {"id": ...}; any other value renders as it is.
func ExampleContext_Created() {
	app := iris.New()
	app.Post("/users", func(ctx iris.Context) {
		var newID int64 = 42 // what the insert returned.
		ctx.Created(newID, nil)
	})

	if err := app.Build(); err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	app.Handler.ServeHTTP(w, httptest.NewRequest("POST", "/users", nil))
	fmt.Println(w.Code, w.Body.String())
	// Output: 201 {"id":42}
}

// NoContent is the empty-body answer to a delete or an update: 204 when the
// call changed something, 304 when it succeeded and nothing changed. A non-nil
// error is rendered through rest/errors instead.
func ExampleContext_NoContent() {
	users := map[int]bool{1: true}

	app := iris.New()
	app.Delete("/users/{id:int}", func(ctx iris.Context) {
		id := ctx.Params().GetInt("id")
		_, deleted := users[id]
		delete(users, id)
		ctx.NoContent(deleted, nil)
	})

	if err := app.Build(); err != nil {
		panic(err)
	}

	for range 2 {
		w := httptest.NewRecorder()
		app.Handler.ServeHTTP(w, httptest.NewRequest("DELETE", "/users/1", nil))
		fmt.Println(w.Code)
	}
	// Output:
	// 204
	// 304
}
