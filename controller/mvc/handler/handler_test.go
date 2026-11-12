package handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/basicauth"
	"github.com/kataras/iris/v14/rest/errors"

	"uuid"
)

type UserAPIParam struct {
	Username string
}

func (p *UserAPIParam) DecodeRequest(ctx iris.Context) error {
	username, err := ctx.User().GetUsername()
	if err != nil {
		return errors.Unauthenticated.Wrap(err)
	}

	p.Username = username
	return nil
}

type ProfileResponse struct {
	Message string `json:"message"`
}

type PostRequest struct {
	Content string `json:"content"`
}

type PostResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Author    string    `json:"author"`

	Post PostRequest `json:"post"`
}

func TestHandlerAndHandlerByWithMiddlewareAndParam(t *testing.T) {
	app := iris.New()

	userAPI := app.Router("/api/user", basicauth.Default(map[string]string{"makis": "123456", "giwrgos": "1234567"}))
	{
		userAPI.Get("/profile", handler.Handler(func(ctx context.Context, in UserAPIParam) (ProfileResponse, error) {
			resp := ProfileResponse{
				Message: fmt.Sprintf("Hello, %s!", in.Username),
			}
			return resp, nil
		}))

		userAPI.Post("/post", handler.HandlerBy(func(ctx context.Context, param UserAPIParam, req PostRequest) (PostResponse, error) {
			resp := PostResponse{
				ID:        uuid.New().String(),
				CreatedAt: time.Now(),
				Author:    param.Username,
				Post:      req,
			}

			return resp, nil
		}))
	}

	e := httptest.New(t, app)

	e.GET("/api/user/profile").Expect().Status(httptest.StatusUnauthorized)

	e.GET("/api/user/profile").WithBasicAuth("makis", "123456").Expect().Status(httptest.StatusOK).
		JSON().IsEqual(iris.Map{"message": "Hello, makis!"})

	e.POST("/api/user/post").WithBasicAuth("makis", "123456").WithJSON(iris.Map{"content": "Hello, World!"}).Expect().Status(httptest.StatusOK).
		JSON().Object().ContainsKey("id").ContainsKey("created_at").HasValue("author", "makis").Value("post").Object().HasValue("content", "Hello, World!")
}

func TestGetHandlerByID(t *testing.T) {
	app := iris.New()

	getUser := func(ctx context.Context, id string) (User, error) {
		return User{ID: id, Firstname: "John", Lastname: "Doe"}, nil
	}
	app.Get("/user/{id:uuid}", handler.Handler(getUser))

	e := httptest.New(t, app)
	userID := uuid.New().String()
	e.GET("/user/" + userID).Expect().Status(httptest.StatusOK).JSON().Object().Value("firstname").IsEqual("John")
}

func TestGetHandlerByParamStruct(t *testing.T) {
	app := iris.New()

	type UserParam struct {
		ID string `param:"id"`
	}

	getUser := func(ctx context.Context, req UserParam) (User, error) {
		return User{ID: req.ID, Firstname: "John", Lastname: "Doe"}, nil
	}
	app.Get("/user/{id:uuid}", handler.Handler(getUser))

	e := httptest.New(t, app)
	userID := uuid.New().String()
	e.GET("/user/" + userID).Expect().Status(httptest.StatusOK).JSON().Object().ContainsSubset(iris.Map{"id": userID, "firstname": "John"})
}

func TestGetHandlerByRequestDecoderFunc(t *testing.T) {
	app := iris.New()

	type UserParam struct {
		ID string
	}

	requestDecoder := func(ctx iris.Context, req *UserParam) error {
		req.ID = ctx.Params().GetString("id")
		return nil
	}

	getUser := func(ctx context.Context, req UserParam) (User, error) {
		return User{ID: req.ID, Firstname: "John", Lastname: "Doe"}, nil
	}
	app.Get("/user/{id:uuid}", handler.Handler(getUser, handler.WithRequestDecoderFunc(requestDecoder)))

	e := httptest.New(t, app)
	userID := uuid.New().String()
	e.GET("/user/" + userID).Expect().Status(httptest.StatusOK).JSON().Object().ContainsSubset(iris.Map{"id": userID, "firstname": "John"})
}

func TestHandlerByRequestAndParamDecoderFunc(t *testing.T) {
	app := iris.New()

	type UserParam struct {
		ID string
	}

	paramDecoder := func(ctx iris.Context, req *UserParam) error {
		req.ID = ctx.Params().GetString("id")
		return nil
	}

	// requestDecoder := func(ctx iris.Context, req *User) error {
	// 	return ctx.ReadJSON(req)
	// 	// OR
	// 	// req.Firstname = ctx.Query().Get("firstname")
	// 	// req.Lastname = ctx.Query().Get("lastname")
	// 	// return nil
	// }

	createUser := func(ctx context.Context, param UserParam, req User) (User, error) {
		newUserID := uuid.New().String()
		user := User{ID: newUserID, Firstname: req.Firstname, Lastname: req.Lastname}
		return user, nil
	}

	// app.Post("/user", handler.HandlerBy(createUser, handler.WithStatusCode(iris.StatusCreated), handler.WithParamDecoderFunc(paramDecoder), handler.WithRequestDecoderFunc(requestDecoder)))
	app.Post("/user2", handler.HandlerBy(createUser, handler.WithStatusCode(iris.StatusCreated), handler.WithParamDecoderFunc(paramDecoder))) // set only param decoder, keep the default request decoder, which reads from JSON too, smae effect.

	e := httptest.New(t, app)
	// e.POST("/user").WithJSON(iris.Map{"firstname": "John", "lastname": "Doe"}).Expect().Status(httptest.StatusCreated).JSON().Object().
	// 	ContainsKey("id").ContainsKey("firstname").ContainsKey("lastname").Value("firstname").IsEqual("John")
	e.POST("/user2").WithJSON(iris.Map{"firstname": "John", "lastname": "Doe"}).Expect().Status(httptest.StatusCreated).JSON().Object().
		ContainsKey("id").ContainsKey("firstname").ContainsKey("lastname").Value("firstname").IsEqual("John")
}

// A fractional total count can only come from an aggregate, and truncating it
// hides the last page. ListHandler used to convert with a plain int64() cast
// while ctx.Paginated rounded half up, so the same 2.5 answered 2 through one
// door and 3 through the other.
func TestListHandlerRoundsFloatTotal(t *testing.T) {
	type filter struct{}
	type response struct {
		Name string `json:"name"`
	}

	app := iris.New()
	app.Post("/list", handler.ListHandler(
		func(_ context.Context, _ filter, _ int, _ int) ([]response, float64, error) {
			return []response{{Name: "one"}}, 2.5, nil
		}))

	e := httptest.New(t, app)
	e.POST("/list").WithJSON(filter{}).Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("total_items", 3)
}

func TestListByHandlerRoundsFloatTotal(t *testing.T) {
	type filter struct{}
	type response struct {
		Name string `json:"name"`
	}

	app := iris.New()
	app.Post("/list/{group}", handler.ListByHandler(
		func(_ context.Context, _ string, _ filter, _ int, _ int) ([]response, float64, error) {
			return []response{{Name: "one"}}, 2.5, nil
		}))

	e := httptest.New(t, app)
	e.POST("/list/a").WithJSON(filter{}).Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("total_items", 3)
}

// The "uuid" macro stores a uuid.UUID: a handler may declare the id as
// uuid.UUID and bind it straight into a struct field.
func TestGetHandlerByParamStructUUID(t *testing.T) {
	app := iris.New()

	type UserParam struct {
		ID uuid.UUID `param:"id"`
	}

	getUser := func(ctx context.Context, req UserParam) (User, error) {
		return User{ID: req.ID.String(), Firstname: "John", Lastname: "Doe"}, nil
	}
	app.Get("/user/{id:uuid}", handler.Handler(getUser))

	e := httptest.New(t, app)
	userID := uuid.New().String()
	e.GET("/user/" + userID).Expect().Status(httptest.StatusOK).JSON().Object().ContainsSubset(iris.Map{"id": userID, "firstname": "John"})
}

// A string field tagged with the uuid macro option binds through the default
// field setter, without a custom DecodeRequest, on a route that stores a
// uuid.UUID: it receives the canonical text.
func TestGetHandlerByParamStructStringOnUUIDRoute(t *testing.T) {
	app := iris.New()

	type StoreParam struct {
		StoreID string `param:"store_id,uuid"`
	}

	getStore := func(ctx context.Context, req StoreParam) (User, error) {
		return User{ID: req.StoreID, Firstname: "Store"}, nil
	}
	app.Get("/store/{store_id:uuid}", handler.Handler(getStore))

	e := httptest.New(t, app)
	storeID := uuid.New().String()
	e.GET("/store/"+storeID).Expect().Status(httptest.StatusOK).JSON().Object().HasValue("id", storeID)
}
