package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/kataras/iris/v14/core/pkg/pagination"
)

type listBook struct {
	ID int `json:"id"`
}

type listFilter struct {
	ListFilter
	Author string `json:"author"`
}

// listServer serves "total" books over QUERY /books, paginated the way
// Context.Paginated does, and records every request it answered.
type listServer struct {
	*httptest.Server

	mu       sync.Mutex
	requests []listRequest
}

type listRequest struct {
	method string
	page   string
	size   string
	author string
}

func newListServer(t *testing.T, total int, failPage string) *listServer {
	t.Helper()

	s := &listServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var filter listFilter
		if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		q := r.URL.Query()
		s.mu.Lock()
		s.requests = append(s.requests, listRequest{r.Method, q.Get("page"), q.Get("size"), filter.Author})
		s.mu.Unlock()

		if r.Method != MethodQuery {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if q.Get("page") == failPage {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}

		page, _ := strconv.Atoi(q.Get("page"))
		size, _ := strconv.Atoi(q.Get("size"))
		opts := PageOptions{Page: page, Size: size}

		var items []listBook
		for id := opts.GetOffset() + 1; id <= min(total, opts.GetOffset()+opts.GetLimit()); id++ {
			items = append(items, listBook{ID: id})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pagination.NewPage(opts, filter, items, int64(total)))
	}))
	t.Cleanup(s.Close)

	return s
}

func (s *listServer) recorded() []listRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

func TestListOnePage(t *testing.T) {
	srv := newListServer(t, 5, "")
	c := New(BaseURL(srv.URL))

	page, err := List[listBook](defaultCtx, c, "/books", PageOptions{Page: 2, Size: 2}, listFilter{Author: "Pike"})
	if err != nil {
		t.Fatal(err)
	}

	if got := page.Items; !slices.Equal(got, []listBook{{3}, {4}}) {
		t.Fatalf("expected books 3 and 4, got %v", got)
	}
	if page.CurrentPage != 2 || page.TotalPages != 3 || page.TotalItems != 5 || !page.HasNextPage {
		t.Fatalf("unexpected page envelope: %+v", page)
	}

	want := []listRequest{{MethodQuery, "2", "2", "Pike"}}
	if got := srv.recorded(); !slices.Equal(got, want) {
		t.Fatalf("expected requests %v, got %v", want, got)
	}
}

func TestListMethodOverride(t *testing.T) {
	srv := newListServer(t, 1, "")
	c := New(BaseURL(srv.URL))

	_, err := List[listBook](defaultCtx, c, "/books", PageOptions{Page: 1}, listFilter{}, ListMethod(http.MethodPost))
	if apiErr, ok := errors.AsType[APIError](err); !ok || apiErr.Response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected the POST to reach the server and get 405, got %v", err)
	}

	if got := srv.recorded(); len(got) != 1 || got[0].method != http.MethodPost {
		t.Fatalf("expected one POST request, got %v", got)
	}
}

func TestItemsWalksEveryPage(t *testing.T) {
	srv := newListServer(t, 5, "")
	c := New(BaseURL(srv.URL))

	var ids []int
	for book, err := range Items[listBook](defaultCtx, c, "/books", listFilter{Author: "Pike"}, PageSize(2)) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, book.ID)
	}

	if !slices.Equal(ids, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("expected books 1 to 5, got %v", ids)
	}

	want := []listRequest{
		{MethodQuery, "1", "2", "Pike"},
		{MethodQuery, "2", "2", "Pike"},
		{MethodQuery, "3", "2", "Pike"},
	}
	if got := srv.recorded(); !slices.Equal(got, want) {
		t.Fatalf("expected requests %v, got %v", want, got)
	}
}

func TestItemsEmptyList(t *testing.T) {
	srv := newListServer(t, 0, "")
	c := New(BaseURL(srv.URL))

	for book, err := range Items[listBook](defaultCtx, c, "/books", listFilter{}) {
		t.Fatalf("expected no items, got %v, %v", book, err)
	}

	if got := srv.recorded(); len(got) != 1 {
		t.Fatalf("expected one request, got %v", got)
	}
}

func TestItemsBreakStopsRequesting(t *testing.T) {
	srv := newListServer(t, 5, "")
	c := New(BaseURL(srv.URL))

	for book, err := range Items[listBook](defaultCtx, c, "/books", listFilter{}, PageSize(2)) {
		if err != nil {
			t.Fatal(err)
		}
		if book.ID == 1 {
			break
		}
	}

	if got := srv.recorded(); len(got) != 1 {
		t.Fatalf("expected the break to stop after one request, got %v", got)
	}
}

func TestItemsYieldsTheErrorOnce(t *testing.T) {
	srv := newListServer(t, 5, "2")
	c := New(BaseURL(srv.URL))

	var (
		ids  []int
		errs []error
	)
	for book, err := range Items[listBook](defaultCtx, c, "/books", listFilter{}, PageSize(2)) {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ids = append(ids, book.ID)
	}

	if !slices.Equal(ids, []int{1, 2}) {
		t.Fatalf("expected the first page only, got %v", ids)
	}
	if len(errs) != 1 {
		t.Fatalf("expected exactly one error, got %v", errs)
	}
	if apiErr, ok := errors.AsType[APIError](errs[0]); !ok || apiErr.Response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected a 500 APIError, got %v", errs[0])
	}
}
