package context_test

import (
	stdContext "context"
	"io"
	"math"
	"net/http"
	nethttptest "net/http/httptest"
	"strings"
	"testing"

	"github.com/kataras/golog"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/pagination"
	"github.com/kataras/iris/v14/httptest"
)

// serve builds the app and serves it over a real net/http server, so
// net/http's write rules (a 304 refuses a body) apply like in production.
func serve(t *testing.T, app *iris.Application) *nethttptest.Server {
	t.Helper()

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := nethttptest.NewServer(app)
	t.Cleanup(srv.Close)
	return srv
}

// captureErrorLogs records every error-level (or worse) log line of the app.
func captureErrorLogs(app *iris.Application) *[]string {
	var lines []string
	app.Logger().Handle(func(l *golog.Log) bool {
		if l.Level <= golog.ErrorLevel {
			lines = append(lines, l.Message)
		}
		return true // captured, keep it out of the test output.
	})
	return &lines
}

// A zero count is a valid 200 answer. It used to set 304, which net/http
// answers by refusing the JSON body (http.ErrBodyNotAllowed), and that
// write failure was then logged as a server error.
func TestCountZeroRenders200(t *testing.T) {
	app := iris.New()
	logged := captureErrorLogs(app)

	var handled bool
	app.Get("/count", func(ctx iris.Context) {
		handled = ctx.Count[int64](0, nil)
	})

	srv := serve(t, app)
	resp, err := http.Get(srv.URL + "/count")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := strings.TrimSpace(string(body)); got != `{"count":0}` {
		t.Fatalf("body = %q, want {\"count\":0}", got)
	}
	if !handled {
		t.Fatal("Count must report success for a zero count")
	}
	if len(*logged) > 0 {
		t.Fatalf("nothing should be logged at error level, got: %q", *logged)
	}
}

func withPageSizes(t *testing.T, maxSize, defaultSize int) {
	t.Helper()
	prevMax, prevDefault := pagination.MaxPageSize, pagination.DefaultPageSize
	pagination.MaxPageSize, pagination.DefaultPageSize = maxSize, defaultSize
	t.Cleanup(func() { pagination.MaxPageSize, pagination.DefaultPageSize = prevMax, prevDefault })
}

func TestPageOptionsGetLimit(t *testing.T) {
	withPageSizes(t, 10, 3)

	cases := []struct {
		name string
		size int
		want int
	}{
		{"zero uses the default", 0, 3},
		{"negative uses the default", -5, 3},
		{"in range is kept", 7, 7},
		{"equal to the maximum is kept", 10, 10},
		{"above the maximum clamps to the maximum", 11, 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := context.PageOptions{Size: tc.size}
			if got := opts.GetLimit(); got != tc.want {
				t.Fatalf("GetLimit(Size=%d) = %d, want %d", tc.size, got, tc.want)
			}
		})
	}
}

func TestPageOptionsGetOffset(t *testing.T) {
	withPageSizes(t, 10, 3)

	cases := []struct {
		name       string
		page, size int
		want       int
	}{
		{"page zero", 0, 5, 0},
		{"first page", 1, 5, 0},
		{"third page", 3, 5, 10},
		{"size above the maximum offsets by the clamped size", 2, 50, 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := context.PageOptions{Page: tc.page, Size: tc.size}
			if got := opts.GetOffset(); got != tc.want {
				t.Fatalf("GetOffset(Page=%d, Size=%d) = %d, want %d", tc.page, tc.size, got, tc.want)
			}
		})
	}
}

func TestNewPageBoundaries(t *testing.T) {
	withPageSizes(t, 10, 3)

	items := func(n int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	}

	cases := []struct {
		name        string
		opts        context.PageOptions
		items       []int
		total       int64
		currentPage int
		totalPages  int
		totalItems  int64
		hasNext     bool
	}{
		{"first of four", context.PageOptions{Page: 1, Size: 3}, items(3), 10, 1, 4, 10, true},
		{"last page short", context.PageOptions{Page: 4, Size: 3}, items(1), 10, 4, 4, 10, false},
		{"exact multiple", context.PageOptions{Page: 3, Size: 3}, items(3), 9, 3, 3, 9, false},
		{"page past the end with items", context.PageOptions{Page: 9, Size: 3}, items(1), 10, 9, 4, 10, false},
		{"total zero", context.PageOptions{Page: 1, Size: 3}, nil, 0, 1, 0, 0, false},
		// A page past the end is an empty page of a non-empty result set, not an
		// empty result set: the client asked for page 9 of 10 rows and must be
		// told so, or a pager renders "0 of 0" and loses its way back.
		{"page past the end with no items", context.PageOptions{Page: 9, Size: 3}, nil, 10, 9, 4, 10, false},
		{"first page with no items", context.PageOptions{Page: 1, Size: 3}, nil, 10, 1, 4, 10, false},
		{"size above the maximum", context.PageOptions{Page: 1, Size: 50}, items(10), 25, 1, 3, 25, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page := context.NewPage(tc.opts, nil, tc.items, tc.total)
			if page.CurrentPage != tc.currentPage || page.TotalPages != tc.totalPages ||
				page.TotalItems != tc.totalItems || page.HasNextPage != tc.hasNext {
				t.Fatalf("got current=%d total_pages=%d total_items=%d has_next=%v, want %d %d %d %v",
					page.CurrentPage, page.TotalPages, page.TotalItems, page.HasNextPage,
					tc.currentPage, tc.totalPages, tc.totalItems, tc.hasNext)
			}
			if page.PageSize != len(tc.items) {
				t.Fatalf("page_size = %d, want %d", page.PageSize, len(tc.items))
			}
			if page.Items == nil {
				t.Fatal("items must never be nil")
			}
		})
	}
}

// The page count is integer arithmetic: a float64 ceiling loses the low bits
// of a total above 2^53 and reports one page too few.
func TestNewPageCountIsExact(t *testing.T) {
	if math.MaxInt < 1<<53 {
		t.Skip("needs a 64-bit int")
	}
	withPageSizes(t, 10, 3)

	total := int64(1<<53 + 1)
	page := context.NewPage(context.PageOptions{Page: 1, Size: 1}, nil, []int{1}, total)
	if want := int(total); page.TotalPages != want {
		t.Fatalf("total_pages = %d, want %d", page.TotalPages, want)
	}
}

func TestPageTransformKeepsHasNextPage(t *testing.T) {
	withPageSizes(t, 10, 3)

	page := context.NewPage(context.PageOptions{Page: 1, Size: 3}, nil, []int{1, 2, 3}, 10)
	if !page.HasNextPage {
		t.Fatal("setup: expected a next page")
	}

	out, err := page.Transform[string](func(i int) (string, error) { return "x", nil })
	if err != nil {
		t.Fatal(err)
	}
	if !out.HasNextPage {
		t.Fatal("Transform dropped HasNextPage")
	}
}

// A float total count rounds half up instead of truncating.
func TestPaginatedRoundsFloatTotal(t *testing.T) {
	app := iris.New()
	app.Get("/list", func(ctx iris.Context) {
		ctx.Paginated(func(_ stdContext.Context, _ context.PageOptions, _ struct{}) ([]int, float64, error) {
			return []int{1}, 2.5, nil
		})
	})

	e := httptest.New(t, app)
	e.GET("/list").Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("total_items", 3)
}

// The exported size variables can be set to zero; the limit must stay
// positive so NewPage never divides by zero inside a request handler.
func TestPageSizeVariablesZero(t *testing.T) {
	cases := []struct {
		name         string
		maxSize      int
		defaultSize  int
		size         int
		wantLimit    int
		wantTotalPgs int
	}{
		{"max zero disables the clamp", 0, 3, 50, 50, 1},
		{"default zero falls back to one", 10, 0, 0, 1, 10},
		{"both zero", 0, 0, 0, 1, 10},
		{"negative default", 10, -4, 0, 1, 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withPageSizes(t, tc.maxSize, tc.defaultSize)
			opts := context.PageOptions{Page: 1, Size: tc.size}
			if got := opts.GetLimit(); got != tc.wantLimit {
				t.Fatalf("GetLimit = %d, want %d", got, tc.wantLimit)
			}
			page := context.NewPage(opts, nil, []int{1}, 10) // must not panic.
			if page.TotalPages != tc.wantTotalPgs {
				t.Fatalf("total_pages = %d, want %d", page.TotalPages, tc.wantTotalPgs)
			}
		})
	}
}

// The page types are aliases of core/pkg/pagination, so values cross freely.
var (
	_ context.Page[int]         = pagination.Page[int]{}
	_ iris.Page[int]            = pagination.Page[int]{}
	_ context.PageOptions       = pagination.PageOptions{}
	_ context.ListFilter        = pagination.ListFilter{}
	_ iris.CountResponse[int64] = pagination.CountResponse[int64]{}
)

// A QUERY list reads the page from the URL and the filter from the body,
// exactly like POST.
func TestPaginatedOverQuery(t *testing.T) {
	type filter struct {
		context.ListFilter
		Author string `json:"author"`
	}

	app := iris.New()
	app.Query("/books", func(ctx iris.Context) {
		ctx.Paginated(func(_ stdContext.Context, opts context.PageOptions, f filter) ([]string, int64, error) {
			return []string{f.Author + " " + f.Term}, 7, nil
		})
	})

	e := httptest.New(t, app)
	page := e.Request(iris.MethodQuery, "/books").WithQuery("page", 2).WithQuery("size", 3).
		WithJSON(map[string]any{"author": "Pike", "term": "go"}).
		Expect().Status(httptest.StatusOK).JSON().Object()
	page.HasValue("current_page", 2)
	page.HasValue("total_pages", 3)
	page.HasValue("items", []string{"Pike go"})
}

// Patches reach the moved size bounds.
func TestPatchesPageSize(t *testing.T) {
	prevMax, prevDefault := pagination.MaxPageSize, pagination.DefaultPageSize
	t.Cleanup(func() { pagination.MaxPageSize, pagination.DefaultPageSize = prevMax, prevDefault })

	iris.Patches().Context().SetMaxPageSize(50)
	iris.Patches().Context().SetDefaultPageSize(10)

	if got := (&context.PageOptions{}).GetLimit(); got != 10 {
		t.Fatalf("expected the patched default 10, got %d", got)
	}
	if got := (&context.PageOptions{Size: 500}).GetLimit(); got != 50 {
		t.Fatalf("expected the patched max 50, got %d", got)
	}
}
