package view_test

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/kataras/iris/v14"
	irishttptest "github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/view"
)

// dataEngine is a view engine that prints the data it is given as
// "key=value;" pairs, sorted by key. It refuses anything that is not a plain
// map[string]any, which is the type an engine asserts on.
type dataEngine struct{}

func (dataEngine) ExecuteTemplate(w io.Writer, _, _ string, data any) error {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("data is %T, want map[string]any", data)
	}

	for _, key := range slices.Sorted(maps.Keys(m)) {
		if _, err := fmt.Fprintf(w, "%s=%v;", key, m[key]); err != nil {
			return err
		}
	}

	return nil
}

// sharedDataApp registers the map with WithData, so every request holds the
// same one, and a route that adds the "user" query parameter to its view data.
func sharedDataApp(t *testing.T, shared iris.Map) *iris.Application {
	t.Helper()

	app := iris.New()
	app.Use(view.Handler(dataEngine{}))
	app.Use(view.WithData(shared))
	app.Get("/", func(ctx iris.Context) {
		if user := ctx.Query().Get("user"); user != "" {
			if err := view.KeyValue(ctx, "User", user); err != nil {
				ctx.StopWithError(iris.StatusInternalServerError, err)
				return
			}

			// The second call of a request reuses what the first one made.
			if err := view.KeyValue(ctx, "Visits", 1); err != nil {
				ctx.StopWithError(iris.StatusInternalServerError, err)
				return
			}
		}

		if err := view.Render(ctx, "page", "", nil); err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
		}
	})

	return app
}

// A key that one request adds belongs to that request. The map given to
// WithData is the same for all of them and must stay as it was.
func TestKeyValueLeavesSharedDataAlone(t *testing.T) {
	shared := iris.Map{"AppName": "My Iris App"}
	e := irishttptest.New(t, sharedDataApp(t, shared))

	e.GET("/").WithQuery("user", "makis").Expect().Status(irishttptest.StatusOK).
		Body().IsEqual("AppName=My Iris App;User=makis;Visits=1;")

	// This one adds nothing and sees nothing of the request before it.
	e.GET("/").Expect().Status(irishttptest.StatusOK).
		Body().IsEqual("AppName=My Iris App;")

	if want := (iris.Map{"AppName": "My Iris App"}); !maps.Equal(shared, want) {
		t.Fatalf("the map given to WithData changed: %v", shared)
	}
}

// Requests that run together each write their own key. Run it with -race.
func TestKeyValueConcurrentRequests(t *testing.T) {
	shared := iris.Map{"AppName": "My Iris App"}

	app := sharedDataApp(t, shared)
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

				want := "AppName=My Iris App;User=" + user + ";Visits=1;"
				if got := rec.Body.String(); got != want {
					t.Errorf("got %q, want %q", got, want)
					return
				}
			}
		})
	}
	wg.Wait()

	if len(shared) != 1 {
		t.Fatalf("the map given to WithData has %d keys, want 1: %v", len(shared), shared)
	}
}

func TestKeyValue(t *testing.T) {
	// inRequest runs fn as the handler of one request.
	inRequest := func(t *testing.T, fn func(ctx iris.Context)) {
		t.Helper()

		app := iris.New()
		app.Get("/", fn)
		if err := app.Build(); err != nil {
			t.Fatal(err)
		}

		app.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}

	// requestData returns the view data of the request, which is always a
	// plain map[string]any once KeyValue ran.
	requestData := func(t *testing.T, ctx iris.Context) map[string]any {
		t.Helper()

		data, ok := view.GetData(ctx).(map[string]any)
		if !ok {
			t.Fatalf("GetData is %T, want map[string]any", view.GetData(ctx))
		}

		return data
	}

	t.Run("creates the map", func(t *testing.T) {
		inRequest(t, func(ctx iris.Context) {
			if err := view.KeyValue(ctx, "a", 1); err != nil {
				t.Fatal(err)
			}

			if got, want := requestData(t, ctx), (iris.Map{"a": 1}); !maps.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	})

	t.Run("copies a map set with Data", func(t *testing.T) {
		inRequest(t, func(ctx iris.Context) {
			given := iris.Map{"a": 1}
			view.Data(ctx, given)

			for key, value := range map[string]int{"b": 2, "c": 3} {
				if err := view.KeyValue(ctx, key, value); err != nil {
					t.Fatal(err)
				}
			}

			if got, want := requestData(t, ctx), (iris.Map{"a": 1, "b": 2, "c": 3}); !maps.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}

			if want := (iris.Map{"a": 1}); !maps.Equal(given, want) {
				t.Fatalf("the map given to Data changed: %v", given)
			}
		})
	})

	t.Run("Data replaces what KeyValue built", func(t *testing.T) {
		inRequest(t, func(ctx iris.Context) {
			if err := view.KeyValue(ctx, "a", 1); err != nil {
				t.Fatal(err)
			}

			given := iris.Map{"x": 0}
			view.Data(ctx, given)

			if err := view.KeyValue(ctx, "b", 2); err != nil {
				t.Fatal(err)
			}

			if got, want := requestData(t, ctx), (iris.Map{"x": 0, "b": 2}); !maps.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}

			if want := (iris.Map{"x": 0}); !maps.Equal(given, want) {
				t.Fatalf("the map given to Data changed: %v", given)
			}
		})
	})

	t.Run("a nil map is replaced", func(t *testing.T) {
		inRequest(t, func(ctx iris.Context) {
			view.Data(ctx, iris.Map(nil))

			if err := view.KeyValue(ctx, "a", 1); err != nil {
				t.Fatal(err)
			}

			if got, want := requestData(t, ctx), (iris.Map{"a": 1}); !maps.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	})

	t.Run("data that is not a map", func(t *testing.T) {
		inRequest(t, func(ctx iris.Context) {
			type page struct{ Title string }
			view.Data(ctx, page{Title: "Home"})

			if err := view.KeyValue(ctx, "a", 1); err == nil {
				t.Fatal("expected an error")
			}

			if _, ok := view.GetData(ctx).(page); !ok {
				t.Fatalf("GetData is %T, want the value given to Data", view.GetData(ctx))
			}
		})
	})
}
