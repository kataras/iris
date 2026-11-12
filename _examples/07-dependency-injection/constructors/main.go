// Constructor injection: a value, constructor functions and an interface
// parameter matched by method set.
//
// Services(store, NewUserService) registers the *MemUsers value under its
// concrete type, then calls NewUserService immediately - its UserRepository
// parameter is an interface, so the container picks the registration whose
// method set satisfies it. API("/users", NewUsersAPI) resolves the controller
// the same way, with the *UserService built a moment earlier. Registration is
// eager and ordered: list a dependency before the constructors that consume it.
//
// Learn: register values and constructors with Services and let the container
// resolve interface parameters by method set.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/users          # 200, two users
//	curl http://localhost:8080/users/1        # 200, Ada
//	curl -i http://localhost:8080/users/404   # 404 NOT_FOUND from errors.NotFound
//
// Next: ../handler-func
package main

import (
	stdContext "context"
	"fmt"
	"sync"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Errors(errors.NewOptions()).
		Services(NewMemUsers(), NewUserService).
		RouterMiddlewares().
		Middlewares().
		API("/users", NewUsersAPI).
		Build()

	app.Listen(":8080")
}

// User is the resource served by this example.
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UserRepository is the seam between logic and storage: constructors depend on
// this interface, main decides what implements it.
type UserRepository interface {
	List(ctx stdContext.Context) ([]User, error)
	ByID(ctx stdContext.Context, id string) (User, error)
}

// MemUsers is an in-memory UserRepository, safe for concurrent use because
// every registration is a singleton shared by all requests.
type MemUsers struct {
	mu    sync.RWMutex
	users []User
}

var _ UserRepository = (*MemUsers)(nil)

func NewMemUsers() *MemUsers {
	return &MemUsers{users: []User{{ID: "1", Name: "Ada"}, {ID: "2", Name: "Linus"}}}
}

func (m *MemUsers) List(stdContext.Context) ([]User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]User, len(m.users))
	copy(list, m.users)
	return list, nil
}

func (m *MemUsers) ByID(_ stdContext.Context, id string) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}

	// Wrapping with an error code name is what makes ctx.OK answer 404.
	return User{}, errors.NotFound.Wrap(fmt.Errorf("user %q not found", id))
}

// UserService holds the business rules. It receives its repository through the
// constructor; it never builds one.
type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) List(ctx stdContext.Context) ([]User, error) {
	return s.repo.List(ctx)
}

func (s *UserService) ByID(ctx stdContext.Context, id string) (User, error) {
	return s.repo.ByID(ctx, id)
}

// UsersAPI is the HTTP layer: a constructor that declares what it needs and a
// Routing method that declares the routes.
type UsersAPI struct {
	svc *UserService
}

func NewUsersAPI(svc *UserService) *UsersAPI {
	return &UsersAPI{svc: svc}
}

func (api *UsersAPI) Routing(r iris.Router) {
	r.Get("/", api.list)
	r.Get("/{id}", api.get)
}

func (api *UsersAPI) list(ctx iris.Context) {
	users, err := api.svc.List(ctx)
	ctx.OK(users, err)
}

func (api *UsersAPI) get(ctx iris.Context) {
	user, err := api.svc.ByID(ctx, ctx.Params().GetString("id"))
	ctx.OK(user, err)
}
