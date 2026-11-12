package context_test

import (
	stdContext "context"
	"errors"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// userFilter is what a service embeds the shared request half into.
type userFilter struct {
	iris.ListFilter
	GroupID string `json:"group_id,omitzero" url:"group_id"`
}

func TestListFilterSortColumn(t *testing.T) {
	var empty iris.ListFilter
	column, err := empty.SortColumn("name", "created_at")
	if column != "" || err != nil {
		t.Fatalf("an empty sort must answer (\"\", nil), got (%q, %v)", column, err)
	}

	allowed := iris.ListFilter{Sort: "name"}
	column, err = allowed.SortColumn("name", "created_at")
	if column != "name" || err != nil {
		t.Fatalf("expected (name, nil), got (%q, %v)", column, err)
	}

	denied := iris.ListFilter{Sort: "password"}
	column, err = denied.SortColumn("name", "created_at")
	if column != "" {
		t.Fatalf("a denied sort must answer an empty column, got %q", column)
	}
	if !errors.Is(err, iris.ErrSortNotAllowed) {
		t.Fatalf("expected an ErrSortNotAllowed, got %v", err)
	}
	if !strings.Contains(err.Error(), "password") {
		t.Fatalf("the error must name the column, got %v", err)
	}

	// No allowed column at all denies every sort.
	if _, err = denied.SortColumn(); !errors.Is(err, iris.ErrSortNotAllowed) {
		t.Fatalf("expected an ErrSortNotAllowed, got %v", err)
	}
}

func TestListFilterSortOrderAndRequest(t *testing.T) {
	ascending := iris.ListFilter{Sort: "name"}
	if got := ascending.SortOrder(); got != "ASC" {
		t.Fatalf("expected ASC, got %s", got)
	}

	descending := iris.ListFilter{Sort: "name", SortDescending: true}
	if got := descending.SortOrder(); got != "DESC" {
		t.Fatalf("expected DESC, got %s", got)
	}

	column, desc, requested := descending.SortRequest()
	if column != "name" || !desc || !requested {
		t.Fatalf("expected (name, true, true), got (%q, %v, %v)", column, desc, requested)
	}

	var empty iris.ListFilter
	if _, _, requested = empty.SortRequest(); requested {
		t.Fatal("an empty sort is not a sort request")
	}
}

// The same ListFilter decodes from the JSON body of the POST /list route and
// from the URL query of a GET one.
func TestListFilterBothDecodePaths(t *testing.T) {
	app := iris.New()
	echo := func(ctx iris.Context, filter userFilter) {
		ctx.JSON(filter)
	}

	app.Post("/list", func(ctx iris.Context) {
		_, filter, ok := ctx.BindPagination[userFilter]()
		if !ok {
			return
		}
		echo(ctx, filter)
	})
	app.Get("/list", func(ctx iris.Context) {
		var filter userFilter
		if err := ctx.ReadQuery(&filter); err != nil {
			ctx.StopWithError(httptest.StatusBadRequest, err)
			return
		}
		echo(ctx, filter)
	})

	e := httptest.New(t, app)

	body := `{"term":"mak","names":["a","b"],"substring":true,"sort":"name","sort_descending":true,"group_id":"g1"}`
	obj := e.POST("/list").WithText(body).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusOK).JSON().Object()
	obj.HasValue("term", "mak")
	obj.HasValue("names", []string{"a", "b"})
	obj.HasValue("substring", true)
	obj.HasValue("sort", "name")
	obj.HasValue("sort_descending", true)
	obj.HasValue("group_id", "g1")

	obj = e.GET("/list").WithQuery("term", "mak").WithQuery("names", "a").WithQuery("names", "b").
		WithQuery("substring", "true").WithQuery("sort", "name").
		WithQuery("sort_descending", "true").WithQuery("group_id", "g1").
		Expect().Status(httptest.StatusOK).JSON().Object()
	obj.HasValue("term", "mak")
	obj.HasValue("names", []string{"a", "b"})
	obj.HasValue("substring", true)
	obj.HasValue("sort", "name")
	obj.HasValue("sort_descending", true)
	obj.HasValue("group_id", "g1")
}

// AllowSort rejects a sort column outside the list, DenySort rejects any sort,
// and a filter without a sort request is unaffected.
func TestBindPaginationSortPolicy(t *testing.T) {
	app := iris.New()
	app.Post("/allow", func(ctx iris.Context) {
		_, filter, ok := ctx.BindPagination[userFilter](iris.AllowSort("name", "created_at"))
		if !ok {
			return
		}
		ctx.WriteString(filter.Sort)
	})
	app.Post("/deny", func(ctx iris.Context) {
		_, _, ok := ctx.BindPagination[userFilter](iris.DenySort("this list cannot be sorted"))
		if !ok {
			return
		}
		ctx.WriteString("ok")
	})
	app.Post("/plain", func(ctx iris.Context) {
		_, filter, ok := ctx.BindPagination[bindItem](iris.AllowSort("name"))
		if !ok {
			return
		}
		ctx.WriteString(filter.Name)
	})

	e := httptest.New(t, app)
	json := func(body string) *httptest.Request {
		return e.POST("/allow").WithText(body).WithHeader("Content-Type", "application/json")
	}

	json(`{"sort":"name"}`).Expect().Status(httptest.StatusOK).Body().IsEqual("name")
	json(`{}`).Expect().Status(httptest.StatusOK).Body().IsEmpty()

	resp := json(`{"sort":"password"}`).Expect().Status(httptest.StatusBadRequest)
	resp.Body().Contains("password")
	resp.Body().Contains("created_at")

	e.POST("/deny").WithText(`{"sort":"name"}`).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusBadRequest).Body().Contains("this list cannot be sorted")
	e.POST("/deny").WithText(`{}`).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusOK).Body().IsEqual("ok")

	// A filter that does not carry a sort request ignores the policy.
	e.POST("/plain").WithText(`{"name":"makis"}`).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusOK).Body().IsEqual("makis")
}

// Paginated takes the same options.
func TestPaginatedSortPolicy(t *testing.T) {
	collect := func(_ stdContext.Context, _ iris.PageOptions, filter userFilter) ([]string, int64, error) {
		return []string{filter.Sort}, 1, nil
	}

	app := iris.New()
	app.Post("/list", func(ctx iris.Context) {
		ctx.Paginated(collect, iris.AllowSort("name"))
	})

	e := httptest.New(t, app)
	e.POST("/list").WithText(`{"sort":"name"}`).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusOK).JSON().Object().Value("items").Array().ContainsAll("name")
	e.POST("/list").WithText(`{"sort":"password"}`).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusBadRequest).Body().Contains("password")
}

// ptrEmbedFilter embeds the shared filter through a pointer, so a body that
// carries none of its members leaves it nil. The client decides that.
type ptrEmbedFilter struct {
	*iris.ListFilter
	GroupID string `json:"group_id,omitzero" url:"group_id"`
}

// ptrReceiverFilter answers the sort request through a pointer receiver, so the
// method is not in the value's method set.
type ptrReceiverFilter struct {
	SortBy string `json:"sort,omitzero" url:"sort"`
}

func (f *ptrReceiverFilter) SortRequest() (column string, descending bool, requested bool) {
	return f.SortBy, false, f.SortBy != ""
}

// DeepInner embeds the shared filter through a pointer, one level down.
type DeepInner struct {
	*iris.ListFilter
	Tag string `json:"tag,omitzero" url:"tag"`
}

// deepFilter reaches the sort request through two pointer hops, so a body can
// allocate the outer one and leave the inner one nil.
type deepFilter struct {
	*DeepInner
	GroupID string `json:"group_id,omitzero" url:"group_id"`
}

// A nil embedded *ListFilter means "no sort requested", never a panic, and a
// SortRequest with a pointer receiver is still policed.
func TestBindPaginationSortPolicyFilterShapes(t *testing.T) {
	app := iris.New()
	app.Post("/embedded-pointer", func(ctx iris.Context) {
		_, filter, ok := ctx.BindPagination[ptrEmbedFilter](iris.AllowSort("name"))
		if !ok {
			return
		}
		if filter.ListFilter == nil {
			ctx.WriteString("none")
			return
		}
		ctx.WriteString(filter.Sort)
	})
	app.Post("/deep-pointer", func(ctx iris.Context) {
		_, filter, ok := ctx.BindPagination[deepFilter](iris.AllowSort("name"))
		if !ok {
			return
		}
		switch {
		case filter.DeepInner == nil:
			ctx.WriteString("none")
		case filter.ListFilter == nil:
			ctx.WriteString("inner")
		default:
			ctx.WriteString(filter.Sort)
		}
	})
	app.Post("/pointer-receiver", func(ctx iris.Context) {
		_, filter, ok := ctx.BindPagination[ptrReceiverFilter](iris.AllowSort("name"))
		if !ok {
			return
		}
		ctx.WriteString(filter.SortBy)
	})

	e := httptest.New(t, app)
	post := func(path, body string) *httptest.Request {
		return e.POST(path).WithText(body).WithHeader("Content-Type", "application/json")
	}

	// The embedded pointer is absent from the body: no sort, no panic.
	post("/embedded-pointer", `{"group_id":"g1"}`).Expect().
		Status(httptest.StatusOK).Body().IsEqual("none")
	post("/embedded-pointer", `{}`).Expect().Status(httptest.StatusOK).Body().IsEqual("none")
	post("/embedded-pointer", `{"sort":"name"}`).Expect().
		Status(httptest.StatusOK).Body().IsEqual("name")
	post("/embedded-pointer", `{"sort":"password"}`).Expect().
		Status(httptest.StatusBadRequest).Body().Contains("password")

	// Two pointer hops: the body allocates the middle struct and leaves the
	// filter itself nil.
	post("/deep-pointer", `{"tag":"t"}`).Expect().
		Status(httptest.StatusOK).Body().IsEqual("inner")
	post("/deep-pointer", `{"group_id":"g1"}`).Expect().
		Status(httptest.StatusOK).Body().IsEqual("none")
	post("/deep-pointer", `{"sort":"name"}`).Expect().
		Status(httptest.StatusOK).Body().IsEqual("name")
	post("/deep-pointer", `{"sort":"password"}`).Expect().
		Status(httptest.StatusBadRequest).Body().Contains("password")

	// A pointer-receiver SortRequest must not escape the policy.
	post("/pointer-receiver", `{"sort":"name"}`).Expect().
		Status(httptest.StatusOK).Body().IsEqual("name")
	post("/pointer-receiver", `{"sort":"password"}`).Expect().
		Status(httptest.StatusBadRequest).Body().Contains("password")
	post("/pointer-receiver", `{}`).Expect().Status(httptest.StatusOK).Body().IsEmpty()
}
