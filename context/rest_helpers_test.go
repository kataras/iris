package context_test

import (
	stdContext "context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"
	"uuid"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
)

func TestOK(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		resp := map[string]string{"message": "success"}
		ctx.OK(resp, nil)
	})
	app.Get("/nil", func(ctx iris.Context) {
		if ctx.OK(nil, nil) {
			ctx.WriteString("no-body")
		}
	})
	app.Get("/err", func(ctx iris.Context) {
		if ctx.OK(map[string]string{"never": "written"}, fmt.Errorf("boom")) {
			t.Error("OK must report false on error")
		}
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).JSON().Object().HasValue("message", "success")
	e.GET("/nil").Expect().Status(httptest.StatusOK).Body().IsEqual("no-body")
	e.GET("/err").Expect().Status(httptest.StatusInternalServerError)
}

// Created wraps a bare identifier (a type in IDType) as {"id": ...} and
// renders everything else as given: a slice of strings is a JSON array, not a
// list of id payloads.
func TestCreated(t *testing.T) {
	expectedID := uuid.New()
	type payload struct {
		Name string `json:"name"`
	}

	app := iris.New()
	app.Post("/uuid", func(ctx iris.Context) {
		ctx.Created(expectedID, nil)
	})
	app.Post("/int", func(ctx iris.Context) {
		ctx.Created(42, nil)
	})
	app.Post("/string", func(ctx iris.Context) {
		ctx.Created("abc", nil)
	})
	app.Post("/struct", func(ctx iris.Context) {
		ctx.Created(payload{Name: "n"}, nil)
	})
	app.Post("/strings", func(ctx iris.Context) {
		ctx.Created([]string{expectedID.String(), uuid.New().String()}, nil)
	})
	app.Post("/nil", func(ctx iris.Context) {
		ctx.Created(nil, nil)
	})
	app.Post("/err", func(ctx iris.Context) {
		if ctx.Created(expectedID, fmt.Errorf("boom")) {
			t.Error("Created must report false on error")
		}
	})

	e := httptest.New(t, app)
	e.POST("/uuid").Expect().Status(httptest.StatusCreated).JSON().Object().HasValue("id", expectedID.String())
	e.POST("/int").Expect().Status(httptest.StatusCreated).JSON().Object().HasValue("id", 42)
	e.POST("/string").Expect().Status(httptest.StatusCreated).JSON().Object().HasValue("id", "abc")
	e.POST("/struct").Expect().Status(httptest.StatusCreated).JSON().Object().HasValue("name", "n")
	e.POST("/strings").Expect().Status(httptest.StatusCreated).JSON().Array().Length().IsEqual(2)
	e.POST("/nil").Expect().Status(httptest.StatusCreated).Body().IsEmpty()
	e.POST("/err").Expect().Status(httptest.StatusInternalServerError)
}

// NoContent renders 204 No Content when the call changed something and 304
// Not Modified, with no body on the wire, when it did not.
func TestNoContent(t *testing.T) {
	app := iris.New()
	app.Delete("/", func(ctx iris.Context) {
		if !ctx.NoContent(true, nil) {
			t.Error("NoContent must report true without an error")
		}
	})
	app.Delete("/unmodified", func(ctx iris.Context) {
		if !ctx.NoContent(false, nil) {
			t.Error("NoContent must report true without an error")
		}
	})
	app.Delete("/err", func(ctx iris.Context) {
		if ctx.NoContent(true, fmt.Errorf("boom")) {
			t.Error("NoContent must report false on error")
		}
	})

	// A real net/http server, so the empty bodies are the wire truth.
	srv := serve(t, app)
	for _, tt := range []struct {
		path string
		code int
	}{
		{"/", http.StatusNoContent},
		{"/unmodified", http.StatusNotModified},
		{"/err", http.StatusInternalServerError},
	} {
		req, err := http.NewRequest(http.MethodDelete, srv.URL+tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != tt.code {
			t.Fatalf("%s: expected %d, got %d", tt.path, tt.code, resp.StatusCode)
		}
		if tt.code != http.StatusInternalServerError && len(body) != 0 {
			t.Fatalf("%s: expected an empty body for the %d, got %q", tt.path, tt.code, body)
		}
	}
}

type unmarshalableValue struct {
	Ch chan int `json:"ch"`
}

// JSONOrError renders the value and routes an encode failure through the
// error renderer as a 500, reporting false.
func TestJSONOrError(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		if !ctx.JSONOrError(map[string]int{"n": 1}) {
			t.Error("JSONOrError must report true on success")
		}
	})
	app.Get("/bad", func(ctx iris.Context) {
		if ctx.JSONOrError(unmarshalableValue{}) {
			t.Error("JSONOrError must report false on an encode failure")
		}
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).JSON().Object().HasValue("n", 1)
	e.GET("/bad").Expect().Status(httptest.StatusInternalServerError)
}

func TestCountRendersCount(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.Count(10, nil)
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).JSON().Object().Value("count").IsEqual(10)
}

func TestPaginated(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.Paginated(func(_ stdContext.Context, pageOpts iris.PageOptions, filter any) ([]string, int64, error) {
			items := []string{"item1", "item2"}
			totalCount := int64(2)
			return items, totalCount, nil
		})
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).JSON().IsEqual(context.NewPage(
		iris.PageOptions{
			Page: 1,
			Size: 2,
		},
		nil,
		[]string{
			"item1",
			"item2",
		},
		2,
	),
	)
}

func TestPageOptionsBind(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "http://localhost/u/list", nil)
	if err != nil {
		t.Fatal(err)
	}

	opts := iris.PageOptions{Page: 2, Size: 50, Details: true}
	if err = opts.Bind(r); err != nil {
		t.Fatal(err)
	}

	// Bind used to mutate the copy that URL.Query() returns
	// and the request's RawQuery stayed empty.
	q := r.URL.Query()
	tests := []struct {
		key      string
		expected string
	}{
		{"page", "2"},
		{"size", "50"},
		{"details", "true"},
	}

	for _, tt := range tests {
		if got := q.Get(tt.key); got != tt.expected {
			t.Fatalf("expected RawQuery %q to contain %s=%s but got %q", r.URL.RawQuery, tt.key, tt.expected, got)
		}
	}
}

// iris.NewPage is the root-package wrapper of context.NewPage: a generic
// function cannot be a type alias, so it forwards.
func TestNewPageAlias(t *testing.T) {
	opts := iris.PageOptions{Page: 1, Size: 2}
	items := []string{"item1", "item2"}

	got := iris.NewPage(opts, nil, items, 2)
	expected := context.NewPage(opts, nil, items, 2)

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %#+v but got %#+v", expected, got)
	}

	var _ *iris.Page[string] = got
}

// The iris aliases of the rest types are the context types.
var (
	_ iris.PageOptions          = context.PageOptions{}
	_ *iris.Page[string]        = (*context.Page[string])(nil)
	_ iris.CountResponse[int64] = context.CountResponse[int64]{}
	_ iris.IDPayload[string]    = context.IDPayload[string]{}
	_ iris.Validatable          = context.Validatable(nil)
)

// errNoRowsInResultSet stands for the not-found a repository returns for an
// empty set, the error List is asked to tolerate.
var errNoRowsInResultSet = errors.New("no rows in result set")

// List renders a JSON array and never a null, and it tolerates the errors the
// caller names.
func TestList(t *testing.T) {
	type item struct {
		Name string `json:"name"`
	}

	app := iris.New()
	app.Get("/nil", func(ctx iris.Context) {
		var items []item
		if !ctx.List(items, nil) {
			t.Error("List must report true for a nil slice")
		}
	})
	app.Get("/empty", func(ctx iris.Context) {
		ctx.List([]item{}, nil)
	})
	app.Get("/items", func(ctx iris.Context) {
		ctx.List([]item{{Name: "makis"}}, nil)
	})
	app.Get("/tolerated", func(ctx iris.Context) {
		err := fmt.Errorf("list users: %w", errNoRowsInResultSet)
		if !ctx.List([]item(nil), err, io.EOF, errNoRowsInResultSet) {
			t.Error("List must report true for a tolerated error")
		}
	})
	app.Get("/err", func(ctx iris.Context) {
		if ctx.List([]item(nil), errors.New("boom"), errNoRowsInResultSet) {
			t.Error("List must report false for any other error")
		}
	})
	app.Get("/not-a-list", func(ctx iris.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				ctx.Writef("%v", recovered)
			}
		}()

		ctx.List(map[string]string{"a": "b"}, nil)
	})

	e := httptest.New(t, app)
	// The bytes, not just "an empty array": the point is that null never
	// reaches the client. The trailing newline is the JSON writer's.
	e.GET("/nil").Expect().Status(httptest.StatusOK).Body().IsEqual("[]\n")
	e.GET("/empty").Expect().Status(httptest.StatusOK).Body().IsEqual("[]\n")
	e.GET("/items").Expect().Status(httptest.StatusOK).JSON().Array().Length().IsEqual(1)
	e.GET("/tolerated").Expect().Status(httptest.StatusOK).Body().IsEqual("[]\n")
	e.GET("/err").Expect().Status(httptest.StatusInternalServerError).
		JSON().Object().ContainsKey("message")
	notAList := e.GET("/not-a-list").Expect().Status(httptest.StatusOK).Body()
	notAList.Contains("map[string]string")
	notAList.Contains("is not a slice or an array")
}
