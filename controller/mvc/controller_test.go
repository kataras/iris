package mvc_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/view"

	"github.com/kataras/blocks"
)

// Model.
type Store struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

// Service.
type StoreService struct{}

// Service constructor.
func NewStoreService() *StoreService {
	return &StoreService{}
}

// A Service method.
func (s *StoreService) Greet() string {
	return "Hello, Store!"
}

// Controller.
type StoreController struct {
	Service *StoreService
}

func (c *StoreController) Create(ctx iris.Context, store Store) (Store, error) {
	return store, nil
}

func (c *StoreController) Get(ctx iris.Context, id string) (Store, error) {
	return Store{ID: id, Name: "Bookstore", Location: "New York"}, nil
}

func (c *StoreController) Update(ctx iris.Context, id string, store Store, onlyColumns ...string) (*Store, bool, error) {
	return &store, true, nil
}

func (c *StoreController) Delete(ctx iris.Context, id string) (bool, error) {
	return true, nil
}

// Filter Model.
type StoreFilter struct {
	Location string `json:"location"`
}

func (c *StoreController) List(ctx iris.Context, filter StoreFilter, limit int, offset int) ([]Store, int, error) {
	return []Store{{ID: "a6ce7330-d584-4675-af9e-2c2d2c40de07", Name: "Bookstore", Location: "New York"}}, 1, nil
}

// Optional extra custom routing.
func (c *StoreController) Routing(r iris.Router) {
	param := mvc.GetPathParam(c) // you cam get the generated path parameter's key and type from the controller itself.
	r.Get("/greet", func(ctx iris.Context) {
		ctx.Writef("%s (%s)", c.Service.Greet(), param.Name)
	})
}

// Model.
type Book struct {
	ID      string `json:"id"`
	StoreID string `json:"store_id"`
	Title   string `json:"title"`
	Author  string `json:"author"`
}

// Service.
type BookService struct{}

// Service constructor.
func NewBookService() *BookService {
	return &BookService{}
}

// A Service method.
func (s *BookService) Greet() string {
	return "Hello, Book!"
}

// Controller.
type BookController struct {
	Service *BookService

	StoreParam *mvc.PathParam[Store, string]
}

func (c *BookController) Create(ctx iris.Context, book Book) (Book, error) {
	// fmt.Printf("BookController.Create.storeParam: %#+v\n", c.StoreParam)
	book.StoreID = c.StoreParam.Get(ctx)
	return book, nil
}

func (c *BookController) Get(ctx iris.Context, id string) (Book, error) {
	fetchedBook := Book{
		ID:      id,
		StoreID: c.StoreParam.Get(ctx),
		Title:   "The Catcher in the Rye",
		Author:  "J.D. Salinger",
	}

	return fetchedBook, nil
}

func (c *BookController) Update(ctx iris.Context, id string, book Book, onlyColumns ...string) (*Book, bool, error) {
	return &book, true, nil
}

func (c *BookController) Delete(ctx iris.Context, id string) (bool, error) {
	return true, nil
}

// Filter Model.
type BookFilter struct {
	Author string `json:"author"`
}

func (c *BookController) List(ctx iris.Context, filter BookFilter, limit int, offset int) ([]Book, int, error) {
	return []Book{{ID: "d56cbd6e-6a76-4ba3-bdbb-a93204fbcde8", Title: "The Catcher in the Rye", Author: "J.D. Salinger"}}, 1, nil
}

// Optional extra custom routing.
func (c *BookController) Routing(r iris.Router) { // and custom routes.
	r.Get("/greet", func(ctx iris.Context) {
		ctx.WriteString(c.Service.Greet())
	})
}

// Optionall views.
type (
	IndexViewData struct {
		Name string
	}

	ViewData struct { // can be the same for all views, but we separate them for the shake of the example
		Message string
	}

	ListViewData struct {
		RootMessage string

		Data []ViewData
	}
)

// Every view method below reads an injected dependency through the receiver on
// purpose. DeployedController.Controller used to be captured as the zero (nil)
// C, and a method value on a nil pointer receiver is legal Go that only fails
// when it is called - so views that returned plain literals passed while any
// realistic view method nil-dereferenced on every request.
//
// GET: /resource
func (c *BookController) IndexView(ctx iris.Context) (IndexViewData, error) {
	_ = c.Service.Greet()
	return IndexViewData{Name: "Index!"}, nil
}

// GET: /resource/create
func (c *BookController) CreateView(ctx iris.Context) (ViewData, error) {
	_ = c.Service.Greet()
	return ViewData{Message: "Create, View!"}, nil
}

// GET: /resource/{id}/details
func (c *BookController) DetailsView(ctx iris.Context, id string) (ViewData, error) {
	_ = c.Service.Greet()
	return ViewData{Message: "Get, Details View!"}, nil
}

// GET: /resource/{id}/update
func (c *BookController) UpdateView(ctx iris.Context, id string) (ViewData, error) {
	_ = c.Service.Greet()
	return ViewData{Message: "Update Message!"}, nil
}

// GET: /resource/{id}/delete
func (c *BookController) DeleteView(ctx iris.Context, id string) (ViewData, error) {
	_ = c.Service.Greet()
	return ViewData{Message: "Delete Message!"}, nil
}

// GET: /resource/list
func (c *BookController) ListView(ctx iris.Context) (ListViewData, error) {
	_ = c.Service.Greet()
	return ListViewData{RootMessage: "Hello, World from Root List View Data Object!", Data: []ViewData{{Message: "Hello, World!"}}}, nil
}

//

func TestController(t *testing.T) {
	app := iris.New()
	app.RegisterDependency(
		NewStoreService, // Register the StoreService, so the StoreController can depend on it.
		NewBookService,  // Register the BookService, so the BookController can depend on it.
	)

	// GET: 	/stores/{store_id}
	// GET: 	/stores/greet
	// POST: 	/stores
	// POST, QUERY: 	/stores/list
	// PUT: 	/stores/{store_id}
	// DELETE: 	/stores/{store_id}
	app.Router("/stores", mvc.UseController[*StoreController]().
		// GET: 	/stores/{store_id}/books/{book_id}
		// GET: 	/stores/{store_id}/books/greet
		// POST: 	/stores/{store_id}/books
		// POST: 	/stores/{store_id}/books/list
		// PUT: 	/stores/{store_id}/books/{book_id}
		// DELETE: 	/stores/{store_id}/books/{book_id}
		With("/books", mvc.UseController[*BookController]()),
	) // so simple.

	// The test.
	e := httptest.New(t, app)

	e.POST("/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books").WithJSON(Book{Title: "The Catcher in the Rye", Author: "J.D. Salinger"}).Expect().Status(iris.StatusCreated).
		JSON().Object().ContainsKey("id")
	e.GET("/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books/d56cbd6e-6a76-4ba3-bdbb-a93204fbcde8").Expect().Status(iris.StatusOK).
		JSON().Object().ContainsKey("title").HasValue("title", "The Catcher in the Rye")
	e.PUT("/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books/d56cbd6e-6a76-4ba3-bdbb-a93204fbcde8").WithJSON(Book{Title: "The Catcher in the Rye", Author: "J.D. Salinger"}).Expect().Status(iris.StatusOK).
		JSON().Object().ContainsKey("title").HasValue("title", "The Catcher in the Rye")
	e.DELETE("/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books/d56cbd6e-6a76-4ba3-bdbb-a93204fbcde8").Expect().Status(iris.StatusNoContent)
	e.POST("/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books/list").WithJSON(map[string]any{"author": "J.D. Salinger"}).Expect().Status(iris.StatusOK).
		JSON().Object().IsEqual(context.NewPage(
		iris.PageOptions{Page: 1, Size: 1},
		BookFilter{Author: "J.D. Salinger"},
		[]Book{
			{ID: "d56cbd6e-6a76-4ba3-bdbb-a93204fbcde8", Title: "The Catcher in the Rye", Author: "J.D. Salinger"},
		},
		1))
	// The list answers QUERY (RFC 10008) with the same page as POST.
	e.Request(iris.MethodQuery, "/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books/list").WithJSON(map[string]any{"author": "J.D. Salinger"}).Expect().Status(iris.StatusOK).
		JSON().Object().IsEqual(context.NewPage(
		iris.PageOptions{Page: 1, Size: 1},
		BookFilter{Author: "J.D. Salinger"},
		[]Book{
			{ID: "d56cbd6e-6a76-4ba3-bdbb-a93204fbcde8", Title: "The Catcher in the Rye", Author: "J.D. Salinger"},
		},
		1))
	e.GET("/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books/greet").Expect().Status(iris.StatusOK).Body().IsEqual("Hello, Book!")
	e.GET("/stores/greet").Expect().Status(iris.StatusOK).Body().IsEqual("Hello, Store! (store_id)")
}

func TestViewController(t *testing.T) {
	// register the views, here we register them as part of the code of the shake of the example
	// but you can use the `block.New`'s first input argument to load them from the disk.
	mfs := blocks.NewMemoryFileSystem()
	// define a book layout.
	err := mfs.ParseTemplate("book/layout.html", []byte(`<html>
<head>
<title>Book Layout</title>

</head>
<body>
	<h1>[layout] Body content is below...</h1>
	{{ yield . }}
</body>
</html>
`), nil)
	if err != nil {
		t.Fatal(err)
	}

	// define the book index page.
	err = mfs.ParseTemplate("book/index.html", []byte(`<h1>Hello, {{.Name}}!</h1>`), nil)
	if err != nil {
		t.Fatal(err)
	}

	err = mfs.ParseTemplate("book/create.html", []byte(`<h1>Create, {{.Message}}!</h1>`), nil)
	if err != nil {
		t.Fatal(err)
	}

	err = mfs.ParseTemplate("book/details.html", []byte(`<h1>Get, {{.Message}}!</h1>`), nil)
	if err != nil {
		t.Fatal(err)
	}

	err = mfs.ParseTemplate("book/update.html", []byte(`<h1>{{.Message}}</h1>`), nil)
	if err != nil {
		t.Fatal(err)
	}

	err = mfs.ParseTemplate("book/delete.html", []byte(`<h1>{{.Message}}</h1>`), nil)
	if err != nil {
		t.Fatal(err)
	}

	err = mfs.ParseTemplate("book/list.html", []byte(`<h1>{{.RootMessage}}</h1>
{{ range .Data }}
	<h2>{{.Message}}</h2>
{{ end }}
`), nil)
	if err != nil {
		t.Fatal(err)
	}

	viewEngine := blocks.New(mfs)
	if err = viewEngine.Load(); err != nil {
		t.Fatal(err)
	}
	//

	app := iris.New()
	app.UseRouter(view.Handler(viewEngine))

	app.RegisterDependency(
		NewStoreService, // Register the StoreService, so the StoreController can depend on it.
		NewBookService,  // Register the BookService, so the BookController can depend on it.
	)

	// GET: 	/stores/{store_id}
	// GET: 	/stores/greet
	// POST: 	/stores
	// POST: 	/stores/list
	// PUT: 	/stores/{store_id}
	// DELETE: 	/stores/{store_id}
	app.Router("/stores", mvc.UseController[*StoreController]().
		// View: See WithViews[IndexViewData, ViewData, ViewData, ListViewData]() to enable automatically template resolving and views rendering below.
		// GET: 	/stores/{store_id}/books (View)
		// GET: 	/stores/{store_id}/books/create (View)
		// GET: 	/stores/{store_id}/books/{book_id}/details (View)
		// GET: 	/stores/{store_id}/books/{book_id}/update (View)
		// GET: 	/stores/{store_id}/books/{book_id}/delete (View)
		// GET: 	/stores/{store_id}/books/list (View)
		//
		// GET: 	/stores/{store_id}/books/{book_id}
		// GET: 	/stores/{store_id}/books/greet (Custom, see Routing method)
		// POST: 	/stores/{store_id}/books
		// POST: 	/stores/{store_id}/books/list
		// PUT: 	/stores/{store_id}/books/{book_id}
		// DELETE: 	/stores/{store_id}/books/{book_id}
		With("/books", mvc.UseController[*BookController]().WithViews[IndexViewData, ViewData, ViewData, ListViewData]()),
	) // so simple.

	// View tests.
	e := httptest.New(t, app, httptest.Debug(false), httptest.LogLevel("disable"))
	// GET: /resource
	e.GET("/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books").Expect().Status(iris.StatusOK).
		Body().
		Contains("<title>Book Layout</title>").
		Contains("Hello, Index!")

	// GET: /resource/create
	e.GET("/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books/create").Expect().Status(iris.StatusOK).
		Body().
		Contains("Create, View!")

	// GET: /resource/{id}/details
	e.GET("/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books/d56cbd6e-6a76-4ba3-bdbb-a93204fbcde8/details").Expect().Status(iris.StatusOK).
		Body().
		Contains("Get, Details View!")

	// GET: /resource/{id}/update
	e.GET("/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books/d56cbd6e-6a76-4ba3-bdbb-a93204fbcde8/update").Expect().Status(iris.StatusOK).
		Body().
		Contains("Update Message!")

	// GET: /resource/{id}/delete
	e.GET("/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books/d56cbd6e-6a76-4ba3-bdbb-a93204fbcde8/delete").Expect().Status(iris.StatusOK).
		Body().
		Contains("Delete Message!")

	// GET: /resource/list
	e.GET("/stores/a6ce7330-d584-4675-af9e-2c2d2c40de07/books/list").Expect().Status(iris.StatusOK).
		Body().Contains("Hello, World from Root List View Data Object!").
		Contains("Hello, World!")
}
