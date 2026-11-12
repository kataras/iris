package mvc_test

import (
	stdContext "context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"uuid"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

// A repository is what a service already has: five methods over the standard
// context. UseRepository is the shortcut from that to the six HTTP routes,
// with no controller type in between.

type Memo struct {
	ID     string `json:"id"`
	Tenant string `json:"tenant"`
	Title  string `json:"title"`
}

type MemoFilter struct {
	// Embedded so that the list route can take a sort policy: AllowSort and
	// DenySort read the filter through ListFilter's SortRequest method.
	iris.ListFilter

	Title string `json:"title"`
}

type memoRepository struct {
	mu    sync.Mutex
	items map[string]Memo
	seq   int
}

func newMemoRepository() *memoRepository {
	return &memoRepository{items: make(map[string]Memo)}
}

func (r *memoRepository) Create(_ stdContext.Context, v Memo) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	id := strconv.Itoa(r.seq)
	v.ID = id
	r.items[id] = v
	return id, nil
}

func (r *memoRepository) Get(_ stdContext.Context, id string) (Memo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	v, ok := r.items[id]
	if !ok {
		return Memo{}, errors.NotFound
	}
	return v, nil
}

func (r *memoRepository) Update(_ stdContext.Context, id string, v Memo, onlyColumns ...string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.items[id]
	if !ok {
		return false, errors.NotFound
	}

	if len(onlyColumns) > 0 {
		for _, column := range onlyColumns {
			if column == "title" {
				existing.Title = v.Title
			}
		}
	} else {
		existing.Title = v.Title
		if v.Tenant != "" {
			existing.Tenant = v.Tenant
		}
	}

	if existing == r.items[id] { // nothing changed.
		return false, nil
	}

	r.items[id] = existing
	return true, nil
}

func (r *memoRepository) Delete(_ stdContext.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[id]; !ok {
		return false, nil
	}
	delete(r.items, id)
	return true, nil
}

func (r *memoRepository) List(_ stdContext.Context, filter MemoFilter, limit, offset int) ([]Memo, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var items []Memo
	for _, v := range r.items {
		if filter.Title != "" && v.Title != filter.Title {
			continue
		}
		items = append(items, v)
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	total := int64(len(items))
	if offset >= len(items) {
		return nil, total, nil
	}
	items = items[offset:]
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items, total, nil
}

func TestUseRepositoryRoundTrip(t *testing.T) {
	app := iris.New()
	app.Router("/memos", mvc.UseRepository(newMemoRepository()))

	e := httptest.New(t, app)

	id := e.POST("/memos").WithJSON(Memo{Title: "first"}).Expect().
		Status(httptest.StatusCreated).JSON().Object().Value("id").String().Raw()

	e.GET("/memos/" + id).Expect().Status(httptest.StatusOK).
		JSON().Object().Value("title").IsEqual("first")

	// An update that changes something is a 204.
	e.PUT("/memos/" + id).WithJSON(Memo{Title: "second"}).Expect().
		Status(httptest.StatusNoContent)

	// The same update again changes nothing: 304, no body.
	e.PUT("/memos/" + id).WithJSON(Memo{Title: "second"}).Expect().
		Status(httptest.StatusNotModified)

	// PATCH with ?columns= updates the named columns only.
	e.PATCH("/memos/"+id).WithQuery("columns", "title").
		WithJSON(Memo{Title: "third"}).Expect().Status(httptest.StatusNoContent)
	e.GET("/memos/" + id).Expect().Status(httptest.StatusOK).
		JSON().Object().Value("title").IsEqual("third")

	// The comma spelling of the same list, which the handler package and the
	// repository routes used to parse differently.
	e.PATCH("/memos/"+id).WithQuery("columns", "title,tenant").
		WithJSON(Memo{Title: "fourth"}).Expect().Status(httptest.StatusNoContent)
	e.GET("/memos/" + id).Expect().Status(httptest.StatusOK).
		JSON().Object().Value("title").IsEqual("fourth")

	// A column that is not a member of the model is a 400 naming it, not a
	// silent no-op reported as success.
	e.PATCH("/memos/"+id).WithQuery("columns", "titel").
		WithJSON(Memo{Title: "fifth"}).Expect().Status(httptest.StatusBadRequest)

	page := e.POST("/memos/list").WithJSON(MemoFilter{}).Expect().
		Status(httptest.StatusOK).JSON().Object()
	page.Value("total_items").IsEqual(1)
	page.Value("items").Array().Length().IsEqual(1)

	// The list answers QUERY (RFC 10008) too.
	e.Request("QUERY", "/memos/list").WithJSON(MemoFilter{}).Expect().
		Status(httptest.StatusOK).JSON().Object().Value("total_items").IsEqual(1)

	e.DELETE("/memos/" + id).Expect().Status(httptest.StatusNoContent)
	// Nothing to delete now: 304.
	e.DELETE("/memos/" + id).Expect().Status(httptest.StatusNotModified)
}

func TestUseRepositoryGuardWritesOnly(t *testing.T) {
	guard := func(ctx iris.Context) {
		if ctx.Header("X-Token") != "letmein" {
			ctx.StopWithStatus(iris.StatusUnauthorized)
			return
		}
		ctx.Next()
	}

	repo := newMemoRepository()
	if _, err := repo.Create(stdContext.Background(), Memo{Title: "first"}); err != nil {
		t.Fatal(err)
	}

	app := iris.New()
	app.Router("/memos", mvc.UseRepository(repo, mvc.Guard(guard)))

	e := httptest.New(t, app)

	// Reads are open.
	e.GET("/memos/1").Expect().Status(httptest.StatusOK)
	e.POST("/memos/list").WithJSON(MemoFilter{}).Expect().Status(httptest.StatusOK)

	// Writes are guarded.
	e.POST("/memos").WithJSON(Memo{Title: "x"}).Expect().Status(httptest.StatusUnauthorized)
	e.PUT("/memos/1").WithJSON(Memo{Title: "x"}).Expect().Status(httptest.StatusUnauthorized)
	e.PATCH("/memos/1").WithJSON(Memo{Title: "x"}).Expect().Status(httptest.StatusUnauthorized)
	e.DELETE("/memos/1").Expect().Status(httptest.StatusUnauthorized)

	// ...and reachable with the token.
	e.POST("/memos").WithHeader("X-Token", "letmein").WithJSON(Memo{Title: "x"}).
		Expect().Status(httptest.StatusCreated)
}

func TestUseRepositoryWithout(t *testing.T) {
	app := iris.New()
	app.Router("/memos", mvc.UseRepository(newMemoRepository(),
		mvc.Without(mvc.Delete, mvc.List)))

	e := httptest.New(t, app)

	e.POST("/memos").WithJSON(Memo{Title: "first"}).Expect().Status(httptest.StatusCreated)
	e.GET("/memos/1").Expect().Status(httptest.StatusOK)

	e.DELETE("/memos/1").Expect().Status(httptest.StatusNotFound)
	e.POST("/memos/list").WithJSON(MemoFilter{}).Expect().Status(httptest.StatusNotFound)
}

func TestUseRepositoryBeforeWrite(t *testing.T) {
	repo := newMemoRepository()

	app := iris.New()
	app.Router("/memos", mvc.UseRepository(repo,
		mvc.BeforeWrite(func(ctx iris.Context, v *Memo) error {
			v.Tenant = ctx.Header("X-Tenant")
			return nil
		})))

	e := httptest.New(t, app)

	id := e.POST("/memos").WithHeader("X-Tenant", "acme").WithJSON(Memo{Title: "first"}).
		Expect().Status(httptest.StatusCreated).JSON().Object().Value("id").String().Raw()

	e.GET("/memos/" + id).Expect().Status(httptest.StatusOK).
		JSON().Object().Value("tenant").IsEqual("acme")

	// The hook runs on the update route too, which is a different handler.
	e.PUT("/memos/"+id).WithHeader("X-Tenant", "other").WithJSON(Memo{Title: "second"}).
		Expect().Status(httptest.StatusNoContent)
	e.GET("/memos/" + id).Expect().Status(httptest.StatusOK).
		JSON().Object().Value("tenant").IsEqual("other")

	// A hook error goes through the error pipeline.
	app2 := iris.New()
	app2.Router("/memos", mvc.UseRepository(newMemoRepository(),
		mvc.BeforeWrite(func(ctx iris.Context, v *Memo) error {
			return errors.InvalidArgument
		})))

	httptest.New(t, app2).POST("/memos").WithJSON(Memo{Title: "x"}).
		Expect().Status(httptest.StatusBadRequest)
}

func TestUseRepositoryIDParam(t *testing.T) {
	app := iris.New()
	app.Router("/memos", mvc.UseRepository(newMemoRepository(), mvc.IDParam("id")))

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	if route := app.Routes().GetByPath("/memos/{id:string}"); route == nil {
		t.Fatalf("IDParam did not override the derived parameter name, routes: %s", app.Routes().List())
	}
}

// The id macro derives from the repository's ID type parameter, exactly as it
// does for UseController.
func TestUseRepositoryIDMacro(t *testing.T) {
	app := iris.New()
	app.Router("/memos", mvc.UseRepository(newMemoRepository()))
	app.Router("/tickets", mvc.UseRepository(newTicketRepository()))

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/memos/{memo_id:string}", "/tickets/{ticket_id:uuid}"} {
		if route := app.Routes().GetByPath(path); route == nil {
			t.Fatalf("route %s was not registered", path)
		}
	}
}

// A hook declared for another model can only be a mistake, and the mistake is
// "the tenant id was never stamped", so it is caught at registration.
func TestUseRepositoryBeforeWriteWrongModel(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("a BeforeWrite hook of another model did not panic")
		}

		if msg := fmt.Sprint(r); !strings.Contains(msg, "mvc_test.Ticket") || !strings.Contains(msg, "mvc_test.Memo") {
			t.Fatalf("the panic message names neither type: %s", msg)
		}
	}()

	app := iris.New()
	app.Router("/memos", mvc.UseRepository(newMemoRepository(),
		mvc.BeforeWrite(func(ctx iris.Context, v *Ticket) error { return nil })))

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}
}

// A sort column reaches an ORDER BY clause, so the list route states which
// ones a client may name.
func TestUseRepositoryPagination(t *testing.T) {
	allowed := iris.New()
	allowed.Router("/memos", mvc.UseRepository(newMemoRepository(),
		mvc.Pagination(iris.AllowSort("title"))))

	e := httptest.New(t, allowed)
	e.POST("/memos/list").WithJSON(MemoFilter{}).Expect().Status(httptest.StatusOK)
	e.POST("/memos/list").WithJSON(MemoFilter{ListFilter: iris.ListFilter{Sort: "title"}}).
		Expect().Status(httptest.StatusOK)
	e.POST("/memos/list").WithJSON(MemoFilter{ListFilter: iris.ListFilter{Sort: "secret"}}).
		Expect().Status(httptest.StatusBadRequest)

	denied := iris.New()
	denied.Router("/memos", mvc.UseRepository(newMemoRepository(),
		mvc.Pagination(iris.DenySort("this list is ordered by rank"))))

	e = httptest.New(t, denied)
	e.POST("/memos/list").WithJSON(MemoFilter{}).Expect().Status(httptest.StatusOK)
	e.POST("/memos/list").WithJSON(MemoFilter{ListFilter: iris.ListFilter{Sort: "title"}}).
		Expect().Status(httptest.StatusBadRequest)
}

type Ticket struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
}

type TicketFilter struct{}

type ticketRepository struct{}

func newTicketRepository() *ticketRepository { return &ticketRepository{} }

func (r *ticketRepository) Create(_ stdContext.Context, v Ticket) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (r *ticketRepository) Get(_ stdContext.Context, id uuid.UUID) (Ticket, error) {
	return Ticket{ID: id}, nil
}

func (r *ticketRepository) Update(_ stdContext.Context, id uuid.UUID, v Ticket, onlyColumns ...string) (bool, error) {
	return true, nil
}

func (r *ticketRepository) Delete(_ stdContext.Context, id uuid.UUID) (bool, error) {
	return true, nil
}

func (r *ticketRepository) List(_ stdContext.Context, filter TicketFilter, limit, offset int) ([]Ticket, int64, error) {
	return nil, 0, nil
}
