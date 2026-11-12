package mvc_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/httptest"
)

// The options are shared by UseController and UseRepository: a controller
// gets the same Guard, Without, IDParam and BeforeWrite without a line of
// its own. ItemController is declared in controller_idtypes_test.go.
func TestUseControllerOptions(t *testing.T) {
	guard := func(ctx iris.Context) {
		if ctx.Header("X-Token") != "letmein" {
			ctx.StopWithStatus(iris.StatusUnauthorized)
			return
		}
		ctx.Next()
	}

	app := iris.New()
	app.Router("/items", mvc.UseController[*ItemController](
		mvc.IDParam("id"),
		mvc.Without(mvc.Delete),
		mvc.Guard(guard),
		mvc.BeforeWrite(func(ctx iris.Context, v *Item) error {
			v.Name = "stamped"
			return nil
		}),
	))

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	if route := app.Routes().GetByPath("/items/{id:int}"); route == nil {
		t.Fatalf("IDParam did not override the derived parameter name, routes: %s", app.Routes().List())
	}

	e := httptest.New(t, app)

	// A read is open, a write is guarded.
	e.GET("/items/42").Expect().Status(httptest.StatusOK)
	e.POST("/items").WithJSON(Item{Name: "given"}).Expect().Status(httptest.StatusUnauthorized)

	// BeforeWrite stamps the decoded body before the controller sees it.
	e.POST("/items").WithHeader("X-Token", "letmein").WithJSON(Item{Name: "given"}).
		Expect().Status(httptest.StatusCreated).
		JSON().Object().Value("name").IsEqual("stamped")

	// The hook runs on the update route too, which is a different handler
	// (HandlerByColumns, not Handler).
	e.PUT("/items/42").WithHeader("X-Token", "letmein").WithJSON(Item{Name: "given"}).
		Expect().Status(httptest.StatusOK).
		JSON().Object().Value("name").IsEqual("stamped")

	// Both spellings of ?columns= parse the same way here as they do on the
	// repository routes.
	for _, query := range []string{"name,extra", "name, extra"} {
		e.PUT("/items/42").WithHeader("X-Token", "letmein").WithQuery("columns", query).
			WithJSON(Item{Name: "given"}).Expect().Status(httptest.StatusOK).
			JSON().Object().Value("name").IsEqual("name|extra")
	}

	e.PUT("/items/42").WithHeader("X-Token", "letmein").
		WithQuery("columns", "name").WithQuery("columns", "extra").
		WithJSON(Item{Name: "given"}).Expect().Status(httptest.StatusOK).
		JSON().Object().Value("name").IsEqual("name|extra")

	// Without dropped the Delete route.
	e.DELETE("/items/42").WithHeader("X-Token", "letmein").
		Expect().Status(httptest.StatusNotFound)
}

// The same registration-time panic UseRepository raises: a hook that cannot
// run is a mistake in the program.
func TestUseControllerBeforeWriteWrongModel(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("a BeforeWrite hook of another model did not panic")
		}

		if msg := fmt.Sprint(r); !strings.Contains(msg, "mvc_test.Memo") || !strings.Contains(msg, "mvc_test.Item") {
			t.Fatalf("the panic message names neither type: %s", msg)
		}
	}()

	app := iris.New()
	app.Router("/items", mvc.UseController[*ItemController](
		mvc.BeforeWrite(func(ctx iris.Context, v *Memo) error { return nil })))

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}
}

// The List route takes the same sort policy the repository routes take.
func TestUseControllerPagination(t *testing.T) {
	app := iris.New()
	app.Router("/items", mvc.UseController[*ItemController](
		mvc.Pagination(iris.AllowSort("name"))))

	e := httptest.New(t, app)
	e.POST("/items/list").WithJSON(ItemFilter{}).Expect().Status(httptest.StatusOK)
	e.POST("/items/list").WithJSON(ItemFilter{ListFilter: iris.ListFilter{Sort: "name"}}).
		Expect().Status(httptest.StatusOK)
	e.POST("/items/list").WithJSON(ItemFilter{ListFilter: iris.ListFilter{Sort: "secret"}}).
		Expect().Status(httptest.StatusBadRequest)

	denied := iris.New()
	denied.Router("/items", mvc.UseController[*ItemController](
		mvc.Pagination(iris.DenySort("this list is ordered by rank"))))

	httptest.New(t, denied).POST("/items/list").
		WithJSON(ItemFilter{ListFilter: iris.ListFilter{Sort: "name"}}).
		Expect().Status(httptest.StatusBadRequest)
}

// Without and IDParam reject an argument that can only be a typo.
func TestOptionArgumentValidation(t *testing.T) {
	for name, fn := range map[string]func(){
		"unknown verb":   func() { mvc.Without(mvc.Verb(9)) },
		"empty id param": func() { mvc.IDParam("") },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("did not panic")
				}
			}()

			fn()
		})
	}
}
