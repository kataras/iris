package sessions_test

import (
	stdContext "context"
	"io"
	"net/http"
	nethttptest "net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/sessions"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

// Handler commits a session after the request only when the request changed
// it. It used to commit every session of a request that ended below status
// 400: a request that only read the session, and overlapped one that wrote
// it, stored its old copy over the write, and brought back a session that the
// other request had just deleted.
//
// What the deferred commit did besides that still happens. The tests below
// take the cases one by one, on the three shipped databases.

// unchangedUser has the kinds of field that a copy of the loaded session has
// to compare equal on: a slice, a map and a time.
type unchangedUser struct {
	Visits int
	Name   string
	Roles  []string
	Meta   map[string]string
	Since  time.Time
}

// recordingRedisClient is fakeRedisClient behind a lock, because some of the
// requests below overlap, with a log of the writes it receives.
type recordingRedisClient struct {
	*fakeRedisClient

	mu     sync.Mutex
	writes []string
}

func (c *recordingRedisClient) Get(ctx stdContext.Context, key string) *redis.StringCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fakeRedisClient.Get(ctx, key)
}

func (c *recordingRedisClient) Set(ctx stdContext.Context, key string, value any, ttl time.Duration) *redis.StatusCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writes = append(c.writes, "SET ttl="+ttl.String())
	return c.fakeRedisClient.Set(ctx, key, value, ttl)
}

func (c *recordingRedisClient) Del(ctx stdContext.Context, keys ...string) *redis.IntCmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writes = append(c.writes, "DEL")
	return c.fakeRedisClient.Del(ctx, keys...)
}

func (c *recordingRedisClient) takeWrites() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	writes := c.writes
	c.writes = nil
	return writes
}

func (c *recordingRedisClient) records() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.data)
}

func (c *recordingRedisClient) forget() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.data)
}

// recordingPostgresConn is fakePostgresConn behind a lock, with a log of the
// writes it receives.
type recordingPostgresConn struct {
	*fakePostgresConn

	mu     sync.Mutex
	writes []string
}

func (c *recordingPostgresConn) Exec(ctx stdContext.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch {
	case len(args) == 0: // CREATE TABLE.
	case !strings.Contains(sql, "INSERT INTO"):
		c.writes = append(c.writes, "DELETE")
	case strings.Contains(sql, "expires_at = NOW()"): // ShiftExpiration(true).
		c.writes = append(c.writes, "UPSERT value, expires_at")
	default:
		c.writes = append(c.writes, "UPSERT value")
	}

	return c.fakePostgresConn.Exec(ctx, sql, args...)
}

func (c *recordingPostgresConn) QueryRow(ctx stdContext.Context, sql string, args ...any) pgx.Row {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fakePostgresConn.QueryRow(ctx, sql, args...)
}

func (c *recordingPostgresConn) takeWrites() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	writes := c.writes
	c.writes = nil
	return writes
}

func (c *recordingPostgresConn) records() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.rows)
}

func (c *recordingPostgresConn) forget() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.rows)
}

// unchangedStore is one of the shipped databases over a store that records
// what is written to it.
type unchangedStore struct {
	name string
	db   sessions.Database[unchangedUser]

	// hasStore is false for the cookie database: its cookie is the record.
	hasStore bool
	// takeWrites returns the writes the store received since the last call.
	takeWrites func() []string
	// records returns how many records the store holds and forget removes them.
	records func() int
	forget  func()
	// write is what the store logs when a session is stored.
	write string
}

func unchangedStores() []*unchangedStore {
	opts := staleOptions() // MaxAge is 24 hours.

	redisClient := &recordingRedisClient{fakeRedisClient: &fakeRedisClient{data: make(map[string]string)}}
	postgresConn := &recordingPostgresConn{fakePostgresConn: &fakePostgresConn{rows: make(map[string][]byte)}}
	shiftingConn := &recordingPostgresConn{fakePostgresConn: &fakePostgresConn{rows: make(map[string][]byte)}}

	shiftingDB := sessions.NewPostgresDatabase[unchangedUser](sessions.DefaultTableName, shiftingConn, opts)
	shiftingDB.ShiftExpiration(true)

	return []*unchangedStore{
		{
			name:       "cookie",
			db:         sessions.NewCookieDatabase[unchangedUser](opts),
			takeWrites: func() []string { return nil },
			records:    func() int { return 0 },
			forget:     func() {},
		},
		{
			name:       "redis",
			db:         sessions.NewRedisDatabase[unchangedUser](redisClient, opts),
			hasStore:   true,
			takeWrites: redisClient.takeWrites,
			records:    redisClient.records,
			forget:     redisClient.forget,
			write:      "SET ttl=24h0m0s",
		},
		{
			name:       "postgres",
			db:         sessions.NewPostgresDatabase[unchangedUser](sessions.DefaultTableName, postgresConn, opts),
			hasStore:   true,
			takeWrites: postgresConn.takeWrites,
			records:    postgresConn.records,
			forget:     postgresConn.forget,
			write:      "UPSERT value",
		},
		{
			name:       "postgres with ShiftExpiration",
			db:         shiftingDB,
			hasStore:   true,
			takeWrites: shiftingConn.takeWrites,
			records:    shiftingConn.records,
			forget:     shiftingConn.forget,
			write:      "UPSERT value, expires_at",
		},
	}
}

// newUnchangedApp registers Handler, a route that changes the session, a
// route that only reads it and the routes of the test.
func newUnchangedApp(t *testing.T, db sessions.Database[unchangedUser], routes func(app *iris.Application)) *iris.Application {
	t.Helper()

	app := iris.New()
	app.Use(sessions.Handler(db, staleCookieName))

	app.Get("/write", func(ctx iris.Context) {
		sessions.Get[unchangedUser](ctx).Data.Visits++
		ctx.WriteString("written")
	})

	app.Get("/read", func(ctx iris.Context) {
		session := sessions.Get[unchangedUser](ctx)
		ctx.Writef("visits=%d new=%t", session.Data.Visits, session.IsNew)
	})

	if routes != nil {
		routes(app)
	}

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}
	return app
}

// serveSession sends GET path through the application, with the session
// cookie when cookieValue is not empty. It does not use the test client of
// the httptest package, because two of the tests run two requests at once.
func serveSession(app *iris.Application, path, cookieValue string) *http.Response {
	req := nethttptest.NewRequest(http.MethodGet, path, nil)
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: staleCookieName, Value: cookieValue})
	}

	rec := nethttptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec.Result()
}

// expectBody fails the test unless the response is a 200 with that body.
func expectBody(t *testing.T, resp *http.Response, expected string) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || string(body) != expected {
		t.Fatalf("expected a 200 with %q, got %d with %q", expected, resp.StatusCode, body)
	}
}

// expectNotCommitted fails the test when the request sent a cookie or wrote
// to the store.
func expectNotCommitted(t *testing.T, store *unchangedStore, resp *http.Response) {
	t.Helper()

	if lines := resp.Header["Set-Cookie"]; len(lines) != 0 {
		t.Fatalf("a request that did not change the session sent a cookie: %v", lines)
	}
	if writes := store.takeWrites(); len(writes) != 0 {
		t.Fatalf("a request that did not change the session wrote to the store: %v", writes)
	}
}

// expectCommitted fails the test unless the request stored the session and
// sent a new, live cookie for it. It returns that cookie.
func expectCommitted(t *testing.T, store *unchangedStore, resp *http.Response, sentCookieValue string) *http.Cookie {
	t.Helper()

	cookie := sessionCookie(t, resp)
	if cookie.Value == "" || cookie.Value == sentCookieValue || cookie.MaxAge <= 0 {
		t.Fatalf("expected a new session cookie, got %v", resp.Header["Set-Cookie"])
	}
	if writes := store.takeWrites(); store.hasStore && !slices.Equal(writes, []string{store.write}) {
		t.Fatalf("expected the store to receive [%s], got %v", store.write, writes)
	}
	return cookie
}

// expectExpiredCookie fails the test unless the response removes the session
// cookie from the browser: no value and an expiry in the past.
func expectExpiredCookie(t *testing.T, resp *http.Response) {
	t.Helper()

	if cookie := sessionCookie(t, resp); cookie.Value != "" || !cookie.Expires.Before(time.Now()) {
		t.Fatalf("expected an expired session cookie, got %v", resp.Header["Set-Cookie"])
	}
}

// startSession returns the cookie of a session that one request wrote.
func startSession(t *testing.T, store *unchangedStore, app *iris.Application) *http.Cookie {
	t.Helper()

	resp := serveSession(app, "/write", "")
	expectBody(t, resp, "written")
	return expectCommitted(t, store, resp, "")
}

// The defect: a request that only reads is still running while another
// request writes the session. When the first one ends, the write must stand.
func TestHandlerReadDoesNotOverwriteConcurrentWrite(t *testing.T) {
	for _, store := range unchangedStores() {
		t.Run(store.name, func(t *testing.T) {
			loaded, release := make(chan struct{}), make(chan struct{})

			app := newUnchangedApp(t, store.db, func(app *iris.Application) {
				app.Get("/slow-read", func(ctx iris.Context) {
					session := sessions.Get[unchangedUser](ctx)
					close(loaded)
					<-release
					ctx.Writef("visits=%d", session.Data.Visits)
				})
			})

			browser := startSession(t, store, app) // visits=1.

			slowRead := make(chan *http.Response)
			go func() { slowRead <- serveSession(app, "/slow-read", browser.Value) }()
			<-loaded

			// The write arrives and is answered while the read is still running.
			resp := serveSession(app, "/write", browser.Value)
			expectBody(t, resp, "written")
			written := expectCommitted(t, store, resp, browser.Value) // visits=2.

			close(release)
			resp = <-slowRead
			expectBody(t, resp, "visits=1") // it read before the write.

			// The browser keeps the cookie of the response that arrived last.
			latest := written
			for _, cookie := range resp.Cookies() {
				if cookie.Name == staleCookieName {
					latest = cookie
				}
			}

			later := serveSession(app, "/read", latest.Value)
			expectBody(t, later, "visits=2 new=false")
			expectNotCommitted(t, store, resp)
		})
	}
}

// The same old copy against a sign-out: the request that only reads ends
// after another request deleted the session.
func TestHandlerReadDoesNotRestoreDeletedSession(t *testing.T) {
	for _, store := range unchangedStores() {
		t.Run(store.name, func(t *testing.T) {
			loaded, release := make(chan struct{}), make(chan struct{})

			app := newUnchangedApp(t, store.db, func(app *iris.Application) {
				app.Get("/slow-read", func(ctx iris.Context) {
					session := sessions.Get[unchangedUser](ctx)
					close(loaded)
					<-release
					ctx.Writef("visits=%d", session.Data.Visits)
				})

				app.Get("/logout", func(ctx iris.Context) {
					if err := sessions.Get[unchangedUser](ctx).Delete(ctx); err != nil {
						ctx.StopWithError(http.StatusInternalServerError, err)
						return
					}
					ctx.WriteString("bye")
				})
			})

			browser := startSession(t, store, app)

			slowRead := make(chan *http.Response)
			go func() { slowRead <- serveSession(app, "/slow-read", browser.Value) }()
			<-loaded

			resp := serveSession(app, "/logout", browser.Value)
			expectBody(t, resp, "bye")
			expectExpiredCookie(t, resp)
			if records := store.records(); records != 0 {
				t.Fatalf("expected the sign-out to delete the record, the store holds %d", records)
			}
			store.takeWrites()

			close(release)
			resp = <-slowRead
			expectBody(t, resp, "visits=1")

			// No cookie that signs the visitor in again, no record in the store.
			expectNotCommitted(t, store, resp)
			if records := store.records(); records != 0 {
				t.Fatalf("the deleted session is back in the store: %d record(s)", records)
			}
		})
	}
}

// The cost of the fix, pinned: a request that does not change the session
// sends no cookie and does not write to the store, so nothing moves the
// expiry of the session. It ends Options.MaxAge after its last change.
//
// The session holds a slice, a time and two maps with several entries. Two
// gob encodings of the same maps differ from one request to the next, so the
// loop also fails a comparison of encodings.
func TestHandlerUnchangedSessionIsNotCommitted(t *testing.T) {
	for _, store := range unchangedStores() {
		t.Run(store.name, func(t *testing.T) {
			app := newUnchangedApp(t, store.db, func(app *iris.Application) {
				app.Get("/fill", func(ctx iris.Context) {
					session := sessions.Get[unchangedUser](ctx)
					session.Set(unchangedUser{
						Visits: 1,
						Name:   "kataras",
						Roles:  []string{"admin", "editor"},
						Meta:   map[string]string{"a": "1", "b": "2", "c": "3", "d": "4", "e": "5", "f": "6"},
						Since:  time.Now(),
					})
					for _, key := range []string{"one", "two", "three", "four", "five", "six"} {
						session.SetFlash(key, "a message for "+key)
					}
					session.SetExtra("extra")
					ctx.WriteString("filled")
				})

				// Reads everything, flash messages included, and removes nothing.
				app.Get("/read-all", func(ctx iris.Context) {
					session := sessions.Get[unchangedUser](ctx)
					_, mistyped := session.GetFlash[int]("one") // not an int: it stays.
					extra, _ := session.GetExtra[string]()
					ctx.Writef("visits=%d roles=%d meta=%d flashes=%d mistyped=%t extra=%s",
						session.Data.Visits, len(session.Data.Roles), len(session.Data.Meta), len(session.Flashes(nil)), mistyped, extra)
				})
			})

			resp := serveSession(app, "/fill", "")
			expectBody(t, resp, "filled")
			browser := expectCommitted(t, store, resp, "")

			for range 20 {
				resp = serveSession(app, "/read-all", browser.Value)
				expectBody(t, resp, "visits=1 roles=2 meta=6 flashes=6 mistyped=false extra=extra")
				expectNotCommitted(t, store, resp)
			}
		})
	}
}

// A handler can change an element of a slice or an entry of a map of the
// session without assigning anything to the session. That is a change: it is
// stored, and the request after it reads it and commits nothing.
func TestHandlerCommitsChangeMadeInPlace(t *testing.T) {
	for _, store := range unchangedStores() {
		t.Run(store.name, func(t *testing.T) {
			app := newUnchangedApp(t, store.db, func(app *iris.Application) {
				app.Get("/fill", func(ctx iris.Context) {
					sessions.Get[unchangedUser](ctx).Set(unchangedUser{
						Roles: []string{"admin", "editor"},
						Meta:  map[string]string{"theme": "light"},
					})
					ctx.WriteString("filled")
				})

				app.Get("/promote", func(ctx iris.Context) {
					sessions.Get[unchangedUser](ctx).Data.Roles[0] = "owner"
					ctx.WriteString("promoted")
				})

				app.Get("/theme", func(ctx iris.Context) {
					sessions.Get[unchangedUser](ctx).Data.Meta["theme"] = "dark"
					ctx.WriteString("themed")
				})

				app.Get("/show", func(ctx iris.Context) {
					data := sessions.Get[unchangedUser](ctx).Data
					ctx.Writef("%s %s", data.Roles[0], data.Meta["theme"])
				})
			})

			resp := serveSession(app, "/fill", "")
			expectBody(t, resp, "filled")
			browser := expectCommitted(t, store, resp, "")

			resp = serveSession(app, "/promote", browser.Value)
			expectBody(t, resp, "promoted")
			browser = expectCommitted(t, store, resp, browser.Value)

			resp = serveSession(app, "/theme", browser.Value)
			expectBody(t, resp, "themed")
			browser = expectCommitted(t, store, resp, browser.Value)

			resp = serveSession(app, "/show", browser.Value)
			expectBody(t, resp, "owner dark")
			expectNotCommitted(t, store, resp)
		})
	}
}

// Session.Commit is the way to keep a session alive on a route that only
// reads: called by hand it still stores the session and sends the cookie,
// changed or not.
func TestHandlerCommitByHandRewritesUnchangedSession(t *testing.T) {
	for _, store := range unchangedStores() {
		t.Run(store.name, func(t *testing.T) {
			app := newUnchangedApp(t, store.db, func(app *iris.Application) {
				app.Get("/keep-alive", func(ctx iris.Context) {
					session := sessions.Get[unchangedUser](ctx)
					if err := session.Commit(ctx); err != nil {
						ctx.StopWithError(http.StatusInternalServerError, err)
						return
					}
					ctx.Writef("visits=%d", session.Data.Visits)
				})
			})

			browser := startSession(t, store, app)

			resp := serveSession(app, "/keep-alive", browser.Value)
			expectBody(t, resp, "visits=1")
			expectCommitted(t, store, resp, browser.Value)
		})
	}
}

// A request that ends with status 400 or above commits nothing, as before,
// also when it changed the session: the change is dropped.
func TestHandlerDoesNotCommitChangeOfFailedRequest(t *testing.T) {
	for _, store := range unchangedStores() {
		t.Run(store.name, func(t *testing.T) {
			app := newUnchangedApp(t, store.db, func(app *iris.Application) {
				app.Get("/refused", func(ctx iris.Context) {
					sessions.Get[unchangedUser](ctx).Data.Visits += 100
					ctx.StopWithStatus(http.StatusForbidden)
				})
			})

			browser := startSession(t, store, app)

			resp := serveSession(app, "/refused", browser.Value)
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("expected a 403, got %d", resp.StatusCode)
			}
			expectNotCommitted(t, store, resp)

			resp = serveSession(app, "/read", browser.Value)
			expectBody(t, resp, "visits=1 new=false")
			expectNotCommitted(t, store, resp)
		})
	}
}

// A new session is created and its cookie sent, also by a route that only
// reads. The request after it changes nothing and commits nothing.
func TestHandlerCommitsNewSession(t *testing.T) {
	for _, store := range unchangedStores() {
		t.Run(store.name, func(t *testing.T) {
			app := newUnchangedApp(t, store.db, nil)

			resp := serveSession(app, "/read", "")
			expectBody(t, resp, "visits=0 new=true")
			browser := expectCommitted(t, store, resp, "")
			if records := store.records(); store.hasStore && records != 1 {
				t.Fatalf("expected the new session in the store, it holds %d record(s)", records)
			}

			resp = serveSession(app, "/read", browser.Value)
			expectBody(t, resp, "visits=0 new=false")
			expectNotCommitted(t, store, resp)
		})
	}
}

// GetFlash removes the message it returns. Only the deferred commit stores
// that, so the message is not shown twice.
func TestHandlerCommitsRemovedFlashMessage(t *testing.T) {
	for _, store := range unchangedStores() {
		t.Run(store.name, func(t *testing.T) {
			app := newUnchangedApp(t, store.db, func(app *iris.Application) {
				app.Get("/flash-set", func(ctx iris.Context) {
					sessions.Get[unchangedUser](ctx).SetFlash("notice", "saved")
					ctx.WriteString("ok")
				})

				app.Get("/flash-get", func(ctx iris.Context) {
					message, ok := sessions.Get[unchangedUser](ctx).GetFlash[string]("notice")
					ctx.Writef("notice=%s ok=%t", message, ok)
				})
			})

			browser := startSession(t, store, app)

			resp := serveSession(app, "/flash-set", browser.Value)
			expectBody(t, resp, "ok")
			browser = expectCommitted(t, store, resp, browser.Value)

			resp = serveSession(app, "/flash-get", browser.Value)
			expectBody(t, resp, "notice=saved ok=true")
			browser = expectCommitted(t, store, resp, browser.Value)

			// The message is gone, and a read that finds none changes nothing.
			resp = serveSession(app, "/flash-get", browser.Value)
			expectBody(t, resp, "notice= ok=false")
			expectNotCommitted(t, store, resp)
		})
	}
}

// A change to the Options of the session reaches the cookie and the store,
// with the data left alone.
func TestHandlerCommitsOptionsChange(t *testing.T) {
	for _, store := range unchangedStores() {
		t.Run(store.name, func(t *testing.T) {
			app := newUnchangedApp(t, store.db, func(app *iris.Application) {
				app.Get("/remember-me", func(ctx iris.Context) {
					sessions.Get[unchangedUser](ctx).Options.MaxAge = 720 * time.Hour
					ctx.WriteString("remembered")
				})

				// No Commit and no Delete.
				app.Get("/expire", func(ctx iris.Context) {
					sessions.Get[unchangedUser](ctx).Options.MaxAge = -1
					ctx.WriteString("expired")
				})
			})

			browser := startSession(t, store, app)

			// MaxAge raised.
			if store.name == "redis" {
				store.write = "SET ttl=720h0m0s" // the record lives as long as the cookie.
			}
			resp := serveSession(app, "/remember-me", browser.Value)
			expectBody(t, resp, "remembered")
			browser = expectCommitted(t, store, resp, browser.Value)
			if expected := int((720 * time.Hour).Seconds()); browser.MaxAge != expected {
				t.Fatalf("expected the cookie to live %d seconds, got %d", expected, browser.MaxAge)
			}

			// MaxAge = -1: the record is deleted and the cookie expires.
			resp = serveSession(app, "/expire", browser.Value)
			expectBody(t, resp, "expired")
			expectExpiredCookie(t, resp)
			if !store.hasStore {
				return
			}
			if writes := store.takeWrites(); len(writes) != 1 || !strings.HasPrefix(writes[0], "DEL") {
				t.Fatalf("expected the store to receive one delete, got %v", writes)
			}
			if records := store.records(); records != 0 {
				t.Fatalf("expected no record in the store, it holds %d", records)
			}
		})
	}
}

// MaxAge is not the only field of Options that a handler can change. Another
// attribute of the cookie reaches the browser too, with the data left alone.
func TestHandlerCommitsCookieAttributeChange(t *testing.T) {
	for _, store := range unchangedStores() {
		t.Run(store.name, func(t *testing.T) {
			app := newUnchangedApp(t, store.db, func(app *iris.Application) {
				app.Get("/account-only", func(ctx iris.Context) {
					sessions.Get[unchangedUser](ctx).Options.Path = "/account"
					ctx.WriteString("moved")
				})
			})

			browser := startSession(t, store, app)

			resp := serveSession(app, "/account-only", browser.Value)
			expectBody(t, resp, "moved")
			browser = expectCommitted(t, store, resp, browser.Value)
			if browser.Path != "/account" {
				t.Fatalf("expected the cookie on the path /account, got %q", browser.Path)
			}
		})
	}
}

// SetExtra alone is a change: the extra data is stored, and the request
// after it reads it and commits nothing.
func TestHandlerCommitsExtraChange(t *testing.T) {
	for _, store := range unchangedStores() {
		t.Run(store.name, func(t *testing.T) {
			app := newUnchangedApp(t, store.db, func(app *iris.Application) {
				app.Get("/extra-set", func(ctx iris.Context) {
					sessions.Get[unchangedUser](ctx).SetExtra("dark")
					ctx.WriteString("ok")
				})

				app.Get("/extra-get", func(ctx iris.Context) {
					extra, _ := sessions.Get[unchangedUser](ctx).GetExtra[string]()
					ctx.Writef("extra=%s", extra)
				})
			})

			browser := startSession(t, store, app)

			resp := serveSession(app, "/extra-set", browser.Value)
			expectBody(t, resp, "ok")
			browser = expectCommitted(t, store, resp, browser.Value)

			resp = serveSession(app, "/extra-get", browser.Value)
			expectBody(t, resp, "extra=dark")
			expectNotCommitted(t, store, resp)
		})
	}
}

// A handler that gives the session another id by hand, without RegenerateID
// and without Commit, has changed it: the session is stored under the new id
// and the cookie is replaced.
func TestHandlerCommitsIDChange(t *testing.T) {
	const movedTo = "0195d2a0-1c2b-7d3e-8f4a-5b6c7d8e9f01"

	for _, store := range unchangedStores() {
		if !store.hasStore {
			continue
		}

		t.Run(store.name, func(t *testing.T) {
			app := newUnchangedApp(t, store.db, func(app *iris.Application) {
				app.Get("/move", func(ctx iris.Context) {
					sessions.Get[unchangedUser](ctx).ID = movedTo
					ctx.WriteString("moved")
				})
			})

			browser := startSession(t, store, app)

			resp := serveSession(app, "/move", browser.Value)
			expectBody(t, resp, "moved")
			browser = expectCommitted(t, store, resp, browser.Value)
			if records := store.records(); records != 2 {
				t.Fatalf("expected the old and the new record in the store, it holds %d", records)
			}

			resp = serveSession(app, "/read", browser.Value)
			expectBody(t, resp, "visits=1 new=false")
			expectNotCommitted(t, store, resp)
		})
	}
}

// A valid cookie whose record is gone from the store: the database answers
// with a session under a new id that does not report IsNew. Its record is
// created and the cookie is replaced, also by a route that only reads.
func TestHandlerCommitsSessionWhoseRecordIsGone(t *testing.T) {
	for _, store := range unchangedStores() {
		if !store.hasStore {
			continue
		}

		t.Run(store.name, func(t *testing.T) {
			app := newUnchangedApp(t, store.db, nil)

			browser := startSession(t, store, app)
			store.forget()

			resp := serveSession(app, "/read", browser.Value)
			expectBody(t, resp, "visits=0 new=false")
			browser = expectCommitted(t, store, resp, browser.Value)
			if records := store.records(); records != 1 {
				t.Fatalf("expected the session in the store again, it holds %d record(s)", records)
			}

			// The new cookie names the new record: nothing left to commit.
			resp = serveSession(app, "/read", browser.Value)
			expectBody(t, resp, "visits=0 new=false")
			expectNotCommitted(t, store, resp)
		})
	}
}

// A cookie that does not decode is replaced by a route that only reads too.
func TestHandlerReplacesUndecodableCookieOnRead(t *testing.T) {
	for _, store := range unchangedStores() {
		for _, cookie := range staleCookies(t) {
			t.Run(store.name+"/"+cookie.name, func(t *testing.T) {
				app := newUnchangedApp(t, store.db, nil)
				store.takeWrites()

				resp := serveSession(app, "/read", cookie.value)
				if cookie.decodes && store.name == "redis" {
					// The value decodes to an id with no record: a session
					// under a new id that does not report IsNew.
					expectBody(t, resp, "visits=0 new=false")
				} else {
					expectBody(t, resp, "visits=0 new=true")
				}
				expectCommitted(t, store, resp, cookie.value)
			})
		}
	}
}

// A handler that runs before Handler can load the session through the
// database and change it. Handler did not see what it was before that, so it
// commits such a session as it always did.
func TestHandlerCommitsSessionLoadedBeforeIt(t *testing.T) {
	for _, store := range unchangedStores() {
		t.Run(store.name, func(t *testing.T) {
			app := iris.New()
			app.Use(func(ctx iris.Context) {
				session, err := store.db.Get(ctx, staleCookieName)
				if err != nil {
					ctx.StopWithError(http.StatusInternalServerError, err)
					return
				}
				session.Data.Visits++
				ctx.Next()
			})
			app.Use(sessions.Handler(store.db, staleCookieName))
			app.Get("/read", func(ctx iris.Context) {
				ctx.Writef("visits=%d", sessions.Get[unchangedUser](ctx).Data.Visits)
			})
			if err := app.Build(); err != nil {
				t.Fatal(err)
			}

			resp := serveSession(app, "/read", "")
			expectBody(t, resp, "visits=1")
			browser := expectCommitted(t, store, resp, "")

			resp = serveSession(app, "/read", browser.Value)
			expectBody(t, resp, "visits=2")
			browser = expectCommitted(t, store, resp, browser.Value)

			resp = serveSession(app, "/read", browser.Value)
			expectBody(t, resp, "visits=3")
		})
	}
}

// The session of a custom Database is committed after every request, as it
// always was. Its data can be a value that other requests hold too (a store
// that keeps sessions in memory), and Handler would have to read that data to
// tell whether the request changed it.
func TestHandlerCommitsSessionOfCustomDatabase(t *testing.T) {
	db := newMemServerDatabase(staleOptions())

	app := iris.New()
	app.Use(sessions.Handler(db, staleCookieName))
	app.Get("/write", func(ctx iris.Context) {
		sessions.Get[regenUserSession](ctx).Set(regenUserSession{Username: "kataras"})
		ctx.WriteString("written")
	})
	app.Get("/read", func(ctx iris.Context) {
		ctx.WriteString(sessions.Get[regenUserSession](ctx).Data.Username)
	})
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	resp := serveSession(app, "/write", "")
	expectBody(t, resp, "written")
	browser := sessionCookie(t, resp)
	stored := db.records[db.key(staleCookieName, firstRecordID(t, db))]

	resp = serveSession(app, "/read", browser.Value)
	expectBody(t, resp, "kataras")
	if cookie := sessionCookie(t, resp); cookie.Value == "" || cookie.MaxAge <= 0 {
		t.Fatalf("expected the session cookie to be sent again, got %v", resp.Header["Set-Cookie"])
	}
	// Each commit signs the value again, so a different value was rewritten.
	if db.records[db.key(staleCookieName, firstRecordID(t, db))] == stored {
		t.Fatal("expected the session of a custom database to be stored again")
	}
}

// A custom Database can hand out data that the gob encoding cannot walk, a
// value that points to itself for one. Handler never reads the data of such a
// session, so the request is served.
func TestHandlerDoesNotReadDataOfCustomDatabase(t *testing.T) {
	db := &cyclicDatabase{}
	db.node = &cyclicNode{Name: "kataras"}
	db.node.Next = db.node

	app := iris.New()
	app.Use(sessions.Handler[*cyclicNode](db, staleCookieName))
	app.Get("/read", func(ctx iris.Context) {
		ctx.WriteString(sessions.Get[*cyclicNode](ctx).Data.Name)
	})
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	for range 3 {
		resp := serveSession(app, "/read", "0195d2a0-1c2b-7d3e-8f4a-5b6c7d8e9f01")
		expectBody(t, resp, "kataras")
	}
	if db.commits != 3 {
		t.Fatalf("expected a commit after each of the 3 requests, got %d", db.commits)
	}
}

// cyclicNode points to itself: a gob encoder that walks it never returns.
type cyclicNode struct {
	Name string
	Next *cyclicNode
}

// cyclicDatabase keeps one session in memory and hands the same data to every
// request. It has no key pair, so the cookie value is the session id.
type cyclicDatabase struct {
	node    *cyclicNode
	commits int
}

func (db *cyclicDatabase) Get(ctx iris.Context, name string) (*sessions.Session[*cyclicNode], error) {
	return sessions.Load(ctx, db, name)
}

func (db *cyclicDatabase) Load(_ iris.Context, _, value string) (*sessions.Session[*cyclicNode], error) {
	session := sessions.NewSession[*cyclicNode](sessions.Options{Path: "/", MaxAge: time.Hour})
	session.ID = value
	session.Data = db.node
	return session, nil
}

func (db *cyclicDatabase) Commit(iris.Context, *sessions.Session[*cyclicNode]) error {
	db.commits++
	return nil
}

// firstRecordID returns the id of the one record a memServerDatabase holds.
func firstRecordID(t *testing.T, db *memServerDatabase) string {
	t.Helper()

	if len(db.records) != 1 {
		t.Fatalf("expected one record, got %d", len(db.records))
	}
	for key := range db.records {
		return strings.TrimPrefix(key, staleCookieName+":")
	}
	return ""
}
