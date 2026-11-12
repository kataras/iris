package context_test

import (
	stdContext "context"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// pointerFilter is a service's own filter, taken by pointer the way a
// repository method takes it.
type pointerFilter struct {
	iris.ListFilter
	GroupID string `json:"group_id,omitzero" url:"group_id"`
}

// describe reads a field through the pointer, so a nil filter panics here
// exactly as a repository would.
func (f *pointerFilter) describe() string {
	return f.GroupID + "|" + f.Sort
}

// collectPointerFiltered is the shape Paginated infers Filter as *pointerFilter
// from: a repository method passed as a bare method value.
func collectPointerFiltered(_ stdContext.Context, _ iris.PageOptions, filter *pointerFilter) ([]string, int64, error) {
	return []string{filter.describe()}, 1, nil
}

// A collector taking its filter by pointer never receives nil: an absent body
// decodes to a nil *Filter, which would be a nil dereference the first time
// the repository read a field.
func TestPaginatedPointerFilterIsAllocated(t *testing.T) {
	app := iris.New()
	app.Post("/list", func(ctx iris.Context) { ctx.Paginated(collectPointerFiltered) })
	app.Get("/list", func(ctx iris.Context) { ctx.Paginated(collectPointerFiltered) })
	app.Get("/bind", func(ctx iris.Context) {
		_, filter, ok := ctx.BindPagination[*pointerFilter]()
		if !ok {
			return
		}
		ctx.WriteString(filter.describe())
	})

	e := httptest.New(t, app)

	// A body still decodes into the pointer.
	body := e.POST("/list").WithText(`{"group_id":"g1","sort":"name"}`).
		WithHeader("Content-Type", "application/json").Expect().Status(httptest.StatusOK)
	body.JSON().Object().Value("items").Array().IsEqual([]string{"g1|name"})
	body.JSON().Object().Value("filter").Object().Value("group_id").IsEqual("g1")

	// No body: the collector gets a zero filter, not nil, and the envelope
	// renders it as {} rather than null.
	empty := e.GET("/list").Expect().Status(httptest.StatusOK)
	empty.JSON().Object().Value("items").Array().IsEqual([]string{"|"})
	empty.JSON().Object().Value("filter").Object().IsEmpty()

	// BindPagination answers the same on its own.
	e.GET("/bind").Expect().Status(httptest.StatusOK).Body().IsEqual("|")
}

// The sort policy still fires for a pointer filter: a nil one has nothing to
// answer SortRequest with, so a silently allowed sort would be a bypass.
func TestPaginatedPointerFilterSortPolicy(t *testing.T) {
	app := iris.New()
	app.Post("/deny", func(ctx iris.Context) {
		ctx.Paginated(collectPointerFiltered, iris.DenySort("this list cannot be sorted"))
	})
	app.Post("/allow", func(ctx iris.Context) {
		ctx.Paginated(collectPointerFiltered, iris.AllowSort("name"))
	})

	e := httptest.New(t, app)
	post := func(path, body string) *httptest.Request {
		return e.POST(path).WithText(body).WithHeader("Content-Type", "application/json")
	}

	post("/deny", `{"sort":"name"}`).Expect().
		Status(httptest.StatusBadRequest).Body().Contains("this list cannot be sorted")

	// An empty body requests no sort, so the policy has nothing to reject.
	post("/deny", `{}`).Expect().Status(httptest.StatusOK)
	e.POST("/deny").Expect().Status(httptest.StatusOK)

	post("/allow", `{"sort":"name"}`).Expect().Status(httptest.StatusOK)
	post("/allow", `{"sort":"password"}`).Expect().
		Status(httptest.StatusBadRequest).Body().Contains("password")
}
