// Run with: go test . (or: go test -v -run TestListTasks .)
//
// The same application as ../iris-httptest, tested with nothing but the
// standard library. The entire test client is three lines:
//
//	app.Build()                    // wires the router; ServeHTTP needs it
//	rec := httptest.NewRecorder()
//	app.ServeHTTP(rec, httptest.NewRequest("GET", "/api/tasks", nil))
//
// No socket is involved: ServeHTTP runs the real router and the real
// middleware chain on the calling goroutine and writes into rec. Everything
// else here is ordinary Go: encoding/json for bodies, if statements for
// assertions, three small helpers so the tests read as request/assert pairs.
//
// Against ../iris-httptest: more lines per assertion, but no dependency beyond
// the framework, failure messages you wrote yourself, and a *http.Request and
// a *http.Response that any other Go HTTP tool understands. Pick one style per
// project and stay with it.
package main

import (
	"bytes"
	stdContext "context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"uuid"

	"github.com/kataras/iris/v14"
)

// testConfig holds the credentials the tests choose. Nothing here is a secret,
// and that is the point of passing configuration into newApp instead of
// reading the environment inside it.
var testConfig = Config{AdminUser: "admin", AdminPassword: "test-password"}

// apiError mirrors the envelope the rest/errors package writes. Decoding into
// a struct rather than a map[string]any keeps the assertions type-safe and
// states the error contract once, in one readable place.
type apiError struct {
	Code struct {
		CanonicalName string `json:"canonical_name"`
		Status        int    `json:"status"`
	} `json:"http_error_code"`
	Message    string `json:"message"`
	Validation []struct {
		Field  string `json:"field"`
		Value  any    `json:"value"`
		Reason string `json:"reason"`
	} `json:"validation"`
}

// TestTaskLifecycle walks one resource from creation to completion, so each
// step feeds the next with a value the server generated.
func TestTaskLifecycle(t *testing.T) {
	app := newTestApp(t, NewMemStore())

	rec := do(t, app, http.MethodPost, "/api/tasks", TaskInput{Title: "buy milk"})
	assertStatus(t, rec, http.StatusCreated)

	// ctx.Created answers {"id":"..."}. Decoding into a struct means a
	// renamed field fails the test instead of quietly yielding a zero value.
	created := decodeJSON[struct {
		ID uuid.UUID `json:"id"`
	}](t, rec)
	if created.ID == uuid.Nil() {
		t.Fatal("expected a generated id in the 201 body")
	}

	// Asserting the middleware's stamp proves it is really mounted; a test
	// that only reads bodies passes with the middleware deleted.
	if got := rec.Header().Get("X-Api-Version"); got != "v1" {
		t.Errorf("X-Api-Version: got %q, want %q", got, "v1")
	}

	path := "/api/tasks/" + created.ID.String()

	rec = do(t, app, http.MethodGet, path, nil)
	assertStatus(t, rec, http.StatusOK)
	if task := decodeJSON[Task](t, rec); task.Title != "buy milk" || task.Done {
		t.Errorf("task: got %+v, want title=%q done=false", task, "buy milk")
	}

	rec = do(t, app, http.MethodPatch, path+"/done", nil)
	assertStatus(t, rec, http.StatusNoContent)
	// A 204 carries no body. Assert it, or a handler that starts writing one
	// will break clients long before any test notices.
	if rec.Body.Len() != 0 {
		t.Errorf("204 body: got %q, want empty", rec.Body.String())
	}

	rec = do(t, app, http.MethodGet, path, nil)
	assertStatus(t, rec, http.StatusOK)
	if !decodeJSON[Task](t, rec).Done {
		t.Error("the task should be done after PATCH .../done")
	}
}

// TestListTasks is table-driven: one row per case, one t.Run per row. A
// failure names its row and `go test -run 'TestListTasks/open_only'` reruns
// exactly that case. Neither is possible with a loop of bare assertions.
func TestListTasks(t *testing.T) {
	store := NewMemStore()
	seed(t, store,
		Task{ID: uuid.New(), Title: "buy milk"},
		Task{ID: uuid.New(), Title: "write tests"},
		Task{ID: uuid.New(), Title: "ship it", Done: true},
	)

	// These subtests only read, so they can share one application. A subtest
	// that mutates state should build its own, as TestTaskLifecycle does.
	app := newTestApp(t, store)

	for _, tc := range []struct {
		name string
		path string
		want []string
	}{
		{"no filter", "/api/tasks", []string{"buy milk", "write tests", "ship it"}},
		{"open only", "/api/tasks?open=true", []string{"buy milk", "write tests"}},
		{"explicit false", "/api/tasks?open=false", []string{"buy milk", "write tests", "ship it"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, app, http.MethodGet, tc.path, nil)
			assertStatus(t, rec, http.StatusOK)

			titles := make([]string, 0, len(tc.want))
			for _, task := range decodeJSON[[]Task](t, rec) {
				titles = append(titles, task.Title)
			}

			// slices.Equal over decoded values pins the contents AND the
			// order the API promises, with no assertion library needed.
			if !slices.Equal(titles, tc.want) {
				t.Errorf("titles: got %v, want %v", titles, tc.want)
			}
		})
	}
}

// TestCreateTaskValidation pins the 400 wire format. The exact JSON a client
// receives is part of the API contract: if a refactor changes it, this test is
// the one that must be updated on purpose rather than by accident.
func TestCreateTaskValidation(t *testing.T) {
	app := newTestApp(t, NewMemStore())

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
			rec := do(t, app, http.MethodPost, "/api/tasks", TaskInput{Title: tc.title})
			assertStatus(t, rec, http.StatusBadRequest)

			body := decodeJSON[apiError](t, rec)

			// Clients should branch on the canonical name, not the number: it
			// stays stable even if the status code is ever reclassified.
			if body.Code.CanonicalName != "INVALID_ARGUMENT" || body.Code.Status != 400 {
				t.Fatalf("http_error_code: got %+v, want INVALID_ARGUMENT/400", body.Code)
			}

			// One entry per failing field, carrying the value the client sent
			// and every broken rule of that field joined into one reason.
			if len(body.Validation) != 1 {
				t.Fatalf("validation: got %d entries, want 1 (body: %s)",
					len(body.Validation), rec.Body.String())
			}
			if got := body.Validation[0]; got.Field != "title" || got.Reason != tc.reason {
				t.Errorf("validation[0]: got %+v, want field=title reason=%q", got, tc.reason)
			}
		})
	}
}

// TestNotFound separates the two 404s an API produces. A status-only assertion
// cannot tell them apart, and they come from completely different places.
func TestNotFound(t *testing.T) {
	app := newTestApp(t, NewMemStore())

	t.Run("mapped sentinel", func(t *testing.T) {
		// A well-formed id that does not exist: the repository returns
		// ErrTaskNotFound and the central error map renders it as 404.
		rec := do(t, app, http.MethodGet, "/api/tasks/"+uuid.New().String(), nil)
		assertStatus(t, rec, http.StatusNotFound)

		if got := decodeJSON[apiError](t, rec).Code.CanonicalName; got != "NOT_FOUND" {
			t.Errorf("canonical_name: got %q, want NOT_FOUND", got)
		}
	})

	t.Run("malformed id", func(t *testing.T) {
		// {id:uuid} does not match, so the router answers before any handler
		// runs. Same status code, entirely different origin.
		assertStatus(t, do(t, app, http.MethodGet, "/api/tasks/not-a-uuid", nil),
			http.StatusNotFound)
	})
}

// TestAdminRequiresCredentials asserts the pair that defines a guard: what it
// blocks and what it lets through. Testing only the happy path would pass just
// as well with the middleware removed.
func TestAdminRequiresCredentials(t *testing.T) {
	app := newTestApp(t, NewMemStore())

	for _, tc := range []struct {
		name     string
		user     string
		password string
		want     int
	}{
		{"no credentials", "", "", http.StatusUnauthorized},
		{"wrong password", "admin", "nope", http.StatusUnauthorized},
		{"unknown user", "root", testConfig.AdminPassword, http.StatusUnauthorized},
		{"valid", testConfig.AdminUser, testConfig.AdminPassword, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var opts []func(*http.Request)
			if tc.user != "" {
				opts = append(opts, withBasicAuth(tc.user, tc.password))
			}

			rec := do(t, app, http.MethodGet, "/api/admin/stats", nil, opts...)
			assertStatus(t, rec, tc.want)

			if tc.want == http.StatusUnauthorized {
				// Without the challenge header a browser never prompts, so a
				// 401 alone is not proof the guard behaves correctly.
				if got := rec.Header().Get("WWW-Authenticate"); !strings.HasPrefix(got, "Basic") {
					t.Errorf("WWW-Authenticate: got %q, want a Basic challenge", got)
				}
				return
			}

			stats := decodeJSON[struct {
				Open  int    `json:"open"`
				Admin string `json:"admin"`
			}](t, rec)
			// The identity in the response comes from ctx.User(), so this also
			// proves the middleware published who authenticated.
			if stats.Admin != testConfig.AdminUser {
				t.Errorf("admin: got %q, want %q", stats.Admin, testConfig.AdminUser)
			}
		})
	}
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

// TestInternalErrorsDoNotLeak proves a property of the whole system with five
// lines of stub: an error the central map does not know about (an outage, a
// bug) becomes a generic 500 while the real cause stays in the server log.
func TestInternalErrorsDoNotLeak(t *testing.T) {
	app := newTestApp(t, failingRepo{NewMemStore()})

	rec := do(t, app, http.MethodGet, "/api/tasks", nil)
	assertStatus(t, rec, http.StatusInternalServerError)

	// The valuable assertions here are the negative ones: no host, no port,
	// no driver name, no stack trace.
	if body := rec.Body.String(); strings.Contains(body, "10.0.0.3") ||
		strings.Contains(body, "pq:") || strings.Contains(body, "stack") {
		t.Fatalf("the internal error leaked to the client: %s", body)
	}

	if got := decodeJSON[apiError](t, rec); got.Message != "server error" {
		t.Errorf("message: got %q, want %q", got.Message, "server error")
	}
}

// TestAgainstARealServer swaps the recorder for a real socket.
// httptest.NewServer starts the application on a random loopback port, so the
// request crosses a real TCP connection driven by a real http.Client. Use it
// for what a recorder cannot reproduce: client timeouts, redirect policies,
// connection reuse, streaming, TLS (httptest.NewTLSServer) or a generated API
// client you want to test against the server it targets.
func TestAgainstARealServer(t *testing.T) {
	app := newTestApp(t, NewMemStore())

	srv := httptest.NewServer(app) // *iris.Application is an http.Handler.
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/tasks", "application/json",
		strings.NewReader(`{"title":"ship it"}`))
	if err != nil {
		t.Fatalf("POST /api/tasks: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var created struct {
		ID uuid.UUID `json:"id"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.ID == uuid.Nil() {
		t.Fatal("expected a generated id in the 201 body")
	}

	// The framework's httptest.NewServer(t, app) collapses the Build above and
	// this NewServer call into one line, if you prefer not to do it by hand.
}

// newTestApp builds the application exactly as main does, minus Listen. Build
// is what fills in the http.Handler that ServeHTTP dispatches to: calling
// ServeHTTP on an application that was never built panics on a nil handler.
func newTestApp(t *testing.T, repo Repository) *iris.Application {
	t.Helper()

	app := newApp(testConfig, repo)
	app.Logger().SetLevel("disable") // keep the test output about the tests

	if err := app.Build(); err != nil {
		t.Fatalf("build application: %v", err)
	}

	return app
}

// do sends one request through the application and returns the recorder. This
// is the only plumbing the file needs; wrapping it once keeps every test a
// sequence of request/assert pairs.
func do(t *testing.T, app *iris.Application, method, path string, body any,
	opts ...func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()

	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}

		payload = bytes.NewReader(encoded)
	}

	// httptest.NewRequest is the server-side constructor: it fills in Host and
	// RemoteAddr and panics on a malformed target, which in a test is exactly
	// the feedback you want. (http.NewRequest is the client-side one.)
	req := httptest.NewRequest(method, path, payload)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for _, opt := range opts {
		opt(req)
	}

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req) // the real router and the real middleware chain.

	return rec
}

// withBasicAuth is an option for do; more of them (a bearer token, a custom
// header) stay one-liners because the seam is just *http.Request.
func withBasicAuth(username, password string) func(*http.Request) {
	return func(r *http.Request) { r.SetBasicAuth(username, password) }
}

// decodeJSON unmarshals a recorded body into T. The generic form removes the
// var/err/if dance from every test and reports the status and raw body when
// the response was not the JSON the test expected.
func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %T: %v (status %d, body %q)", out, err, rec.Code, rec.Body.String())
	}

	return out
}

// assertStatus fails with the body attached: the difference between "expected
// 200, got 400" and knowing why on the first run.
func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()

	if rec.Code != want {
		t.Fatalf("status: got %d, want %d (body: %s)", rec.Code, want, rec.Body.String())
	}
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
