package handler_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"

	"uuid"
)

type User struct {
	ID        string `json:"id" url:"auth_user_id"` // these tags are not required, just for testing to pass inside the WithQueryObject.
	Firstname string `json:"firstname" url:"auth_user_firstname"`
	Lastname  string `json:"lastname" url:"auth_user_lastname"`
}

type Authenticated struct {
	User
}

func (p *Authenticated) DecodeRequest(ctx iris.Context) error { // overrides the default behavior for decoding struct requests such as URL (with the url tags).
	// For the shake of the example, we will use the URL query parameters as the authentication data,
	// you can use web tokens or sessions here.
	// If $your_no_access_logic {
	// 	return ErrUnauthorized
	// }
	p.User = User{
		ID:        ctx.Query().Get("auth_user_id"),
		Firstname: ctx.Query().Get("auth_user_firstname"),
		Lastname:  ctx.Query().Get("auth_user_lastname"),
	}

	// fmt.Printf("Authenticated.DecodeRequest with user: %#+v\n", p.User)
	return nil
}

type (
	BookStore struct {
		ID      string `json:"id"`
		OwnerID string `json:"owner_id"`
		Name    string `json:"name"`
	}

	BookStoreRequest struct {
		Name string `json:"name"`
	}

	BookStoreResponse struct {
		BookStore
		Books []Book `json:"books"`
		Owner User   `json:"owner"`
	}

	AppParam struct {
		Authenticated
	}

	BookStoreParam struct {
		AppParam

		StoreID string `param:"store_id,uuid"` // ,uuid is optional option.
	}
)

func (p *AppParam) DecodeRequest(ctx iris.Context) error {
	if err := p.Authenticated.DecodeRequest(ctx); err != nil {
		return err
	}

	return nil
}

func (p *BookStoreParam) DecodeRequest(ctx iris.Context) error {
	if err := p.AppParam.DecodeRequest(ctx); err != nil {
		return err
	}

	p.StoreID = ctx.Params().GetString("store_id")
	return nil
}

type (
	Book struct {
		ID       string `json:"id"`
		StoreID  string `json:"store_id"`  // the store's id.
		AuthorID string `json:"author_id"` // the current user id.
		Title    string `json:"title"`
	}

	BookRequest struct {
		Title string `json:"title"`
	}

	BookResponse struct {
		Book
		Author User `json:"author"`
	}

	BookParam struct {
		BookStoreParam

		Author User
		BookID string `param:"book_id"` // should be given in order for XXXBy methods to resolve the {book_id:uuid}, if missing from relative path.
	}
)

func (p *BookParam) DecodeRequest(ctx iris.Context) error {
	if err := p.BookStoreParam.DecodeRequest(ctx); err != nil {
		return err
	}

	p.Author = User{
		ID:        ctx.Query().Get("author_id"),
		Firstname: ctx.Query().Get("author_firstname"),
		Lastname:  ctx.Query().Get("author_lastname"),
	}

	p.BookID = ctx.Params().GetString("book_id")
	return nil
}

type BookService struct {
	stores map[string]BookStore // example of a repository for the shake of simplicity..
	books  map[string]Book      // example of a repository for the shake of simplicity.

	users       map[string]User        // example of a repository for the shake of simplicity. Contains pre-loaded users.
	ownerStores map[string][]BookStore // store owner id -> stores. Example of relation in a database, for the shake of simplicity.
	storeBooks  map[string][]Book      // store id -> books. Example of relation in a database, for the shake of simplicity.
}

func NewBookService() *BookService {
	service := &BookService{
		users:       make(map[string]User),
		stores:      make(map[string]BookStore),
		books:       make(map[string]Book),
		ownerStores: make(map[string][]BookStore),
		storeBooks:  make(map[string][]Book),
	}
	service.fillUsers()

	return service
}

func (c *BookService) fillUsers() {
	c.users["1"] = User{ID: "1", Firstname: "John", Lastname: "Doe"}
	c.users["2"] = User{ID: "2", Firstname: "Jane", Lastname: "Doe"}
}

func (s *BookService) GetUser(ctx context.Context, userID string) (User, error) {
	user, ok := s.users[userID]
	if !ok {
		return User{}, fmt.Errorf("%w: user_id: %q: user not found", ErrEntityNotFound, userID)
	}
	return user, nil
}

func (s *BookService) InsertBookStore(ctx context.Context, store BookStore) (string, error) {
	storeID := uuid.New().String()

	store.ID = storeID
	s.stores[storeID] = store

	s.ownerStores[store.OwnerID] = append(s.ownerStores[store.OwnerID], store)
	return storeID, nil
}

func (s *BookService) InsertBook(ctx context.Context, book Book) (string, error) {
	bookID := uuid.New().String()

	book.ID = bookID
	s.books[bookID] = book

	// fill the relation.
	s.storeBooks[book.StoreID] = append(s.storeBooks[book.StoreID], book)
	return bookID, nil
}

func (c *BookService) GetBookStore(ctx context.Context, storeID string) (BookStoreResponse, error) {
	store, ok := c.stores[storeID]
	if !ok {
		return BookStoreResponse{}, fmt.Errorf("%w: store_id: %q: store not found", ErrEntityNotFound, storeID)
	}

	owner, err := c.GetUser(ctx, store.OwnerID)
	if err != nil {
		return BookStoreResponse{}, err
	}

	books := c.storeBooks[storeID]

	// Convert a domain entity to an http response manually.
	resp := BookStoreResponse{
		BookStore: store,
		Books:     books,
		Owner:     owner,
	}
	return resp, nil
}

type BookController struct {
	Service *BookService
}

func NewBookController(service *BookService) *BookController {
	return &BookController{Service: service}
}

func (c *BookController) CreateBookStore(ctx context.Context, param AppParam, req BookStoreRequest) (string, error) {
	// Convert an http request to a domain entity manually.
	store := BookStore{
		OwnerID: param.Authenticated.ID,
		Name:    req.Name,
	}
	return c.Service.InsertBookStore(ctx, store)
}

func (c *BookController) CreateBook(ctx context.Context, param BookStoreParam, req BookRequest) (string, error) {
	book := Book{
		StoreID:  param.StoreID,
		AuthorID: param.Authenticated.ID,
		Title:    req.Title,
	}
	return c.Service.InsertBook(ctx, book)
}

func (c *BookController) GetBookStore(ctx context.Context, param BookStoreParam) (BookStoreResponse, error) {
	return c.Service.GetBookStore(ctx, param.StoreID) // all users have access to retrieve a book store.
}

func newParamAppWithGuide() *iris.Application {
	return iris.NewBuilder().
		Prefix("/api").
		AllowOrigin("*").
		// Timeout(iris.Timeout().WithServerTimeout(30*time.Second)).
		Compression(true).
		LogRequests(false).
		Health(true, "testing", "kataras").
		Errors(errors.NewOptions().
			MapErrors(errors.NotFound, ErrEntityNotFound).
			MapErrors(errors.InvalidArgument, ErrInvalidRecipe)).
		Services(NewBookService, NewBookController).
		Middlewares().
		API("/store", func(c *BookController) iris.Controller { // BookController is not registered as a dependency but it automatically registered by the container (It binds its exported fields and returns it).
			// paramDecoder := handler.WithParamDecoderFunc(func(ctx iris.Context, req *BookParam) error {
			// 	return nil
			// })
			// handler.CreateBy(c.CreateBookStore, paramDecoder),
			return iris.Join(
				handler.CreateBy(c.CreateBookStore),
				handler.Get(c.GetBookStore),
			)
		}).
		API("/store/{store_id:uuid}/books", func(c *BookController) iris.Controller {
			return iris.Join(
				handler.CreateBy(c.CreateBook),
			)
		}).
		Build()
}

func newParamApp() *iris.Application {
	app := iris.New()
	opts := errors.NewOptions()
	opts.MapErrors(errors.NotFound, ErrEntityNotFound)
	app.Use(opts.Handler())

	api := app.Router("/api")
	api.RegisterDependency(NewBookService, NewBookController)
	api.Router("/store", func(c *BookController) iris.Controller { // BookController is not registered as a dependency but it automatically registered by the container (It binds its exported fields and returns it).
		// paramDecoder := handler.WithParamDecoderFunc(func(ctx iris.Context, req *BookParam) error {
		// 	return nil
		// })
		// handler.CreateBy(c.CreateBookStore, paramDecoder),
		return iris.Join(
			handler.CreateBy(c.CreateBookStore),
			handler.Get(c.GetBookStore),
		)
	})
	api.Router("/store/{store_id:uuid}/books", func(c *BookController) iris.Controller {
		return iris.Join(
			handler.CreateBy(c.CreateBook),
		)
	})

	if err := app.Build(); err != nil {
		panic(err)
	}
	return app
}

func TestControllerMethodsBy(t *testing.T) {
	app := newParamApp()
	e := httptest.New(t, app)

	auth := User{
		ID:        "1",
		Firstname: "John",
		Lastname:  "Doe",
	}
	// Create a store.
	storeID := e.POST("/api/store").WithQueryObject(auth).WithJSON(BookStoreRequest{Name: "My Book Store"}).
		Expect().Status(iris.StatusCreated).JSON().Object().Value("id").IsString().String().NotEmpty().Raw()
	t.Logf("Store ID: %s", storeID)

	// Get the book store.
	e.GET("/api/store/" + storeID).WithQueryObject(auth).Expect().Status(iris.StatusOK).JSON().Object().Value("id").IsEqual(storeID)

	// Create a book.
	bookID := e.POST("/api/store/" + storeID + "/books").WithQueryObject(auth).WithJSON(BookRequest{Title: "My Book"}).
		Expect().Status(iris.StatusCreated).JSON().Object().Value("id").IsString().String().NotEmpty().Raw()
	t.Logf("Book ID: %s", bookID)
}
