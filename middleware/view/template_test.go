package view_test

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/kataras/iris/v14"
	irishttptest "github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/view"
)

// userTemplate is a Template whose data come from the request: the "user"
// query parameter. Its Data field is what a request with no DataFunc renders.
func userTemplate() *view.Template {
	return &view.Template{
		Name: "profile",
		Data: iris.Map{"User": "nobody"},
		DataFunc: func(ctx iris.Context) (any, error) {
			user := ctx.Query().Get("user")
			if user == "" {
				return nil, errors.New("no user")
			}

			return iris.Map{"User": user}, nil
		},
	}
}

// WithTemplate gives one Template to every request of the route, so a render
// must leave it as it was. The result of DataFunc belongs to the request.
func TestTemplateDataFuncLeavesTemplateAlone(t *testing.T) {
	tmpl := userTemplate()

	app := iris.New()
	app.Use(view.Handler(dataEngine{}))
	app.Get("/", view.WithTemplate(tmpl))

	e := irishttptest.New(t, app)
	e.GET("/").WithQuery("user", "makis").Expect().Status(irishttptest.StatusOK).
		Body().IsEqual("User=makis;")

	if want := (iris.Map{"User": "nobody"}); !maps.Equal(tmpl.Data.(iris.Map), want) {
		t.Fatalf("the render changed Template.Data to %v", tmpl.Data)
	}

	// A DataFunc that fails stops the request and renders nothing.
	e.GET("/").Expect().Status(irishttptest.StatusInternalServerError).Body().IsEmpty()

	if want := (iris.Map{"User": "nobody"}); !maps.Equal(tmpl.Data.(iris.Map), want) {
		t.Fatalf("the failed render changed Template.Data to %v", tmpl.Data)
	}
}

// Requests that run together are each rendered with their own data. Run it
// with -race.
func TestTemplateDataFuncConcurrentRequests(t *testing.T) {
	app := iris.New()
	app.Use(view.Handler(dataEngine{}))
	app.Get("/", view.WithTemplate(userTemplate()))
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	const (
		workers  = 8
		requests = 200
	)

	var wg sync.WaitGroup
	for worker := range workers {
		wg.Go(func() {
			for i := range requests {
				user := fmt.Sprintf("user-%d-%d", worker, i)

				rec := httptest.NewRecorder()
				app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?user="+user, nil))

				if got, want := rec.Body.String(), "User="+user+";"; got != want {
					t.Errorf("got %q, want %q", got, want)
					return
				}
			}
		})
	}
	wg.Wait()
}
