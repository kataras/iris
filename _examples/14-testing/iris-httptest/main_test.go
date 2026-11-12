// Run with: go test . (or: go test -v -run TestListTasks .)
//
// httptest.New(t, app) builds the application (a build failure fails the test
// through t.Fatalf instead of panicking), silences its logs and binds an
// httpexpect client straight to the router. There is no port, no goroutine and
// no "wait for the server to come up", so the suite cannot be flaky for
// infrastructure reasons. Every test constructs its own application and its own
// store, so nothing is shared and nothing has to be reset between tests.
//
// Every knob lives on the same call:
//
//	httptest.New(t, app, httptest.Debug(true))       // dump each request/response
//	httptest.New(t, app, httptest.LogLevel("debug")) // un-silence the app logs
//	httptest.New(t, app, httptest.URL("http://example.com")) // base URL, needed for cookies
//	httptest.New(t, app, httptest.Strict(true))      // stop at the first failed assertion
package main

import (
	stdContext "context"
	"fmt"
	"strings"
	"testing"

	"uuid"

	"github.com/kataras/iris/v14/httptest"
)

// testConfig holds the credentials the tests choose. Nothing here is a secret,
// and that is the point of passing configuration into newApp instead of
// reading the environment inside it.
var testConfig = Config{AdminUser: "admin", AdminPassword: "test-password"}

// TestTaskLifecycle walks one resource from creation to completion. Each step
// feeds the next, which is why the generated id is read back with Raw().
func TestTaskLifecycle(t *testing.T) {
	e := httptest.New(t, newApp(testConfig, NewMemStore()))

	// WithJSON marshals the value and sets the Content-Type header for you.
	// Raw() steps out of the assertion chain and returns the Go value: here
	// the id, needed to address the resource in the requests that follow.
	id := e.POST("/api/tasks").WithJSON(TaskInput{Title: "buy milk"}).
		Expect().Status(httptest.StatusCreated).
		JSON().Object().Value("id").String().NotEmpty().Raw()

	// WithPath fills the {id} placeholder, so the test reads like the route it
	// exercises instead of a string concatenation.
	e.GET("/api/tasks/{id}").WithPath("id", id).
		Expect().Status(httptest.StatusOK).
		JSON().Object().
		HasValue("title", "buy milk").
		HasValue("done", false).
		// Asserting on an ABSENT key is what stops an internal field from
		// silently appearing in the response the day someone adds one.
		NotContainsKey("owner_email")

	// The positional form does the same substitution with less typing.
	e.PATCH("/api/tasks/{id}/done", id).
		Expect().Status(httptest.StatusNoContent).
		// A 204 must have no body at all; assert that, or a handler that
		// starts writing one will never be noticed.
		NoContent()

	e.GET("/api/tasks/{id}", id).
		Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("done", true)
}

// TestListTasks covers the collection endpoint: the JSON array, the ?open
// filter and the header stamped by the middleware.
func TestListTasks(t *testing.T) {
	store := NewMemStore()
	seed(t, store,
		Task{ID: uuid.New(), Title: "buy milk"},
		Task{ID: uuid.New(), Title: "write tests"},
		Task{ID: uuid.New(), Title: "ship it", Done: true},
	)

	e := httptest.New(t, newApp(testConfig, store))
	resp := e.GET("/api/tasks").Expect().Status(httptest.StatusOK)

	// Asserting the stamp proves the middleware is really mounted. A test that
	// only looks at bodies passes happily with the middleware deleted.
	resp.Header("X-Api-Version").IsEqual("v1")
	resp.HasContentType("application/json")

	list := resp.JSON().Array()
	list.Length().IsEqual(3)
	list.Value(0).Object().HasValue("title", "buy milk")
	// Path runs a JSONPath expression: one assertion for a field across every
	// element, which also pins the order the API promises.
	list.Path("$[*].title").Array().
		IsEqual([]string{"buy milk", "write tests", "ship it"})

	// Table-driven subtests: one row per case, one t.Run per row. A failure
	// names its row and `go test -run 'TestListTasks/open_only'` reruns just
	// that one. Both are impossible with a loop of bare assertions.
	for _, tc := range []struct {
		name  string
		open  string
		count int
	}{
		{"no filter", "", 3},
		{"open only", "true", 2},
		{"explicit false", "false", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A subtest gets its own client, so a failure is reported against
			// the subtest's *testing.T and names the right row.
			e := httptest.New(t, newApp(testConfig, store))

			req := e.GET("/api/tasks")
			if tc.open != "" {
				req = req.WithQuery("open", tc.open) // ?open=true
			}

			req.Expect().Status(httptest.StatusOK).
				JSON().Array().Length().IsEqual(tc.count)
		})
	}
}

// TestCreateTaskValidation pins the 400 wire format. The exact JSON a client
// receives is part of the API contract: if a refactor changes it, this test is
// the one that must be updated on purpose rather than by accident.
func TestCreateTaskValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		title  string
		reason string
	}{
		{"empty", "", "must not be empty, must be between 3 and 120 characters"},
		{"too short", "ab", "must be between 3 and 120 characters"},
		{"too long", strings.Repeat("x", 121), "must be between 3 and 120 characters"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := httptest.New(t, newApp(testConfig, NewMemStore()))

			body := e.POST("/api/tasks").WithJSON(TaskInput{Title: tc.title}).
				Expect().Status(httptest.StatusBadRequest).JSON().Object()

			// Clients should branch on the canonical name, not on the number:
			// it is stable even if the status code is ever reclassified.
			body.Value("http_error_code").Object().
				HasValue("canonical_name", "INVALID_ARGUMENT").
				HasValue("status", 400)

			// One entry per failing field, carrying the value the client sent
			// and every broken rule of that field joined into one reason.
			body.Value("validation").Array().Value(0).Object().
				HasValue("field", "title").
				HasValue("value", tc.title).
				HasValue("reason", tc.reason)
		})
	}

	// A body that is not JSON fails before Validate is ever reached, and it
	// must still produce the API's error shape, not a stack trace.
	t.Run("malformed json", func(t *testing.T) {
		e := httptest.New(t, newApp(testConfig, NewMemStore()))

		e.POST("/api/tasks").
			WithHeader("Content-Type", "application/json").
			WithBytes([]byte(`{"title": `)).
			Expect().Status(httptest.StatusBadRequest).
			JSON().Object().ContainsKey("http_error_code")
	})
}

// TestNotFound separates the two 404s an API produces. They look identical in
// a status-only assertion, and they come from completely different places.
func TestNotFound(t *testing.T) {
	e := httptest.New(t, newApp(testConfig, NewMemStore()))

	// A well-formed id that does not exist: the repository returns the
	// sentinel and the CENTRAL ERROR MAP renders it as 404 NOT_FOUND.
	e.GET("/api/tasks/{id}", uuid.New().String()).
		Expect().Status(httptest.StatusNotFound).
		JSON().Object().Value("http_error_code").Object().
		HasValue("canonical_name", "NOT_FOUND").
		HasValue("status", 404)

	// A malformed id never reaches a handler: the {id:uuid} macro rejects the
	// segment and the ROUTER answers. Same status, different origin.
	e.GET("/api/tasks/not-a-uuid").Expect().Status(httptest.StatusNotFound)
}

// TestAdminRequiresCredentials asserts the pair that defines a guard: what it
// blocks and what it lets through. Testing only the happy path would pass just
// as well with the middleware removed.
func TestAdminRequiresCredentials(t *testing.T) {
	e := httptest.New(t, newApp(testConfig, NewMemStore()))

	// The challenge header is what makes a browser prompt for credentials;
	// without it a 401 is only an error page.
	e.GET("/api/admin/stats").Expect().
		Status(httptest.StatusUnauthorized).
		Header("WWW-Authenticate").Contains("Basic")

	e.GET("/api/admin/stats").WithBasicAuth("admin", "wrong").
		Expect().Status(httptest.StatusUnauthorized)

	e.GET("/api/admin/stats").WithBasicAuth(testConfig.AdminUser, testConfig.AdminPassword).
		Expect().Status(httptest.StatusOK).
		JSON().Object().
		HasValue("open", 0).
		HasValue("admin", "admin") // ctx.User() really carries the identity.
}

// failingRepo embeds the Repository INTERFACE and overrides a single method;
// every other call falls through to the value in the embedded field. That is
// what makes a targeted fake a five-line struct: no mocking framework, no code
// generation, and it stops compiling the day the interface changes.
type failingRepo struct {
	Repository
}

func (failingRepo) List(stdContext.Context, bool) ([]Task, error) {
	return nil, fmt.Errorf("pq: connection refused on 10.0.0.3:5432")
}

// TestInternalErrorsDoNotLeak is the test worth internalizing: a five-line
// stub proves a property of the whole system. An error the central map does
// not know about (an outage, a bug) renders as a generic 500 while the real
// cause stays in the server log.
func TestInternalErrorsDoNotLeak(t *testing.T) {
	e := httptest.New(t, newApp(testConfig, failingRepo{NewMemStore()}))

	body := e.GET("/api/tasks").
		Expect().Status(httptest.StatusInternalServerError).JSON().Object()

	body.HasValue("message", "server error")
	body.Value("http_error_code").Object().HasValue("canonical_name", "INTERNAL")
	// The negative assertions are the valuable ones here: no host, no port, no
	// driver name, no stack trace.
	body.NotContainsValue("pq: connection refused on 10.0.0.3:5432")
	body.NotContainsKey("stack")
}

// TestVisitorCookie shows cookie assertions and the client's cookie jar.
// Cookies need a base URL: an in-memory client with no host has no domain to
// file them under, which is what httptest.URL (and the httptest.Domain
// shortcut) is for.
func TestVisitorCookie(t *testing.T) {
	e := httptest.New(t, newApp(testConfig, NewMemStore()),
		httptest.URL("http://example.com"))

	cookie := e.GET("/api/tasks").Expect().
		Status(httptest.StatusOK).Cookie("visitor")
	cookie.Value().NotEmpty()
	cookie.Path().IsEqual("/")

	// Raw() is the escape hatch for anything the fluent API does not cover.
	if !cookie.Raw().HttpOnly {
		t.Error("the visitor cookie must be HttpOnly, or a script can read it")
	}

	// The client keeps a cookie jar, so the next request sends the cookie back
	// and the middleware does not hand out a second one. Session-style flows
	// (sign in, then call a protected route) work for exactly this reason.
	e.GET("/api/tasks").Expect().
		Status(httptest.StatusOK).
		Cookies().NotContains("visitor")
}

// TestStrictMode shows the reporter knob. By default every failed assertion in
// a test is collected and reported, which gives the fullest picture; Strict
// stops at the first one, which is what you want when a later assertion would
// itself blow up on the earlier one's result.
func TestStrictMode(t *testing.T) {
	e := httptest.New(t, newApp(testConfig, NewMemStore()), httptest.Strict(true))

	e.GET("/api/tasks").Expect().Status(httptest.StatusOK).
		JSON().Array().IsEmpty()
}

// seed fills the store directly. Driving the API is more end-to-end, but for
// fixtures the repository is faster and keeps the arrange step from drowning
// the assertions.
func seed(t *testing.T, store *MemStore, tasks ...Task) {
	t.Helper()

	for _, task := range tasks {
		if err := store.Add(stdContext.Background(), task); err != nil {
			t.Fatalf("seed %q: %v", task.Title, err)
		}
	}
}
