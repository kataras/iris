package types_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/client"
	"github.com/kataras/iris/v14/rest/types"
)

type profile struct {
	ID   int      `json:"id"`
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type audit struct {
	Name    string `json:"name"`
	Updated string `json:"updated"`
}

// A Merge response with overlapping keys round-trips through the HTTP client,
// whose decoder keeps the v2 default of rejecting duplicate object names.
func TestMergeRoundTripsThroughClient(t *testing.T) {
	app := iris.New()
	app.Get("/merged", func(ctx iris.Context) {
		ctx.JSON(types.Merge(profile{ID: 1, Name: "old"}, audit{Name: "new", Updated: "today"}))
	})
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(app)
	t.Cleanup(srv.Close)

	c := client.New(client.BaseURL(srv.URL))
	got, err := c.BindJSON[map[string]any](context.Background(), http.MethodGet, "/merged", nil)
	if err != nil {
		t.Fatal(err)
	}

	if got["name"] != "new" || got["updated"] != "today" || got["id"] != float64(1) {
		t.Fatalf("unexpected merged document: %v", got)
	}
	if tags, ok := got["tags"].([]any); !ok || len(tags) != 0 {
		t.Fatalf("nil slice must render as an empty array, got: %v", got["tags"])
	}
}
