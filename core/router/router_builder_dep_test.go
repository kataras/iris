package router_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

type Repository[T any] interface {
	Get(string) T
}

type (
	user struct {
		Firstname string
	}

	UserRepository struct{}
)

var _ Repository[user] = (*UserRepository)(nil)

func (r *UserRepository) Get(id string) user {
	firstname := fmt.Sprintf("UserRepository.Get: %s", id)
	return user{Firstname: firstname}
}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

type AppService struct {
	UserRepo Repository[user]
}

func NewAppService(userRepo Repository[user]) *AppService {
	return &AppService{
		UserRepo: userRepo,
	}
}

func (s *AppService) Print() string {
	user := s.UserRepo.Get("1")
	return user.Firstname
}

func TestRouterBuilder_HandlerFunc(t *testing.T) {
	app := iris.New()
	app.RegisterDependency(NewUserRepository, NewAppService)

	fn := func(service *AppService) iris.Handler {
		return func(ctx iris.Context) {
			ctx.WriteString(service.Print())
			ctx.Next()
		}
	}

	app.Get("/", app.HandlerFunc(fn))

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("UserRepository.Get: 1")
}

type userControllerBindStructExportedFields struct {
	Repository Repository[user] // this should be automatically binded (no a function constructor).
}

func (u *userControllerBindStructExportedFields) Get(ctx iris.Context) {
	user := u.Repository.Get("1")
	ctx.WriteString(user.Firstname)
}

func (u *userControllerBindStructExportedFields) Routing(r iris.Router) {
	r.Get("/", u.Get)
}

func TestRouterBuilder_Router_StructPtr(t *testing.T) {
	app := iris.New()
	app.RegisterDependency(NewUserRepository)

	app.Router("/", new(userControllerBindStructExportedFields))

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("UserRepository.Get: 1")
}
