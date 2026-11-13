package sessions_test

import (
	stdContext "context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/sessions"

	"github.com/gorilla/securecookie"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

// A session cookie the server cannot read is not the visitor's to fix: it was
// signed with keys the server no longer has, it is older than MaxAge, or a v12
// application wrote it under the same name. The shipped databases start a new
// session for it, as they do for a request without the cookie, and the Handler
// replaces it. Only a failure of the store itself is an error (a 500).

const staleCookieName = "sid"

var (
	staleHashKey  = []byte("0123456789abcdef0123456789abcdef")
	staleBlockKey = []byte("fedcba9876543210")
)

type staleUser struct{ Name string }

// errStoreDown is what a client returns while its store is unreachable.
var errStoreDown = errors.New("dial tcp 10.0.0.7:6379: connect: connection refused")

// fakeRedisClient answers the three commands the redis session database sends
// (GET, SET, DEL) from a map. Any other command panics on the nil Cmdable.
type fakeRedisClient struct {
	redis.Cmdable

	data map[string]string
	down error // when set, every command fails with it.
}

func (c *fakeRedisClient) Close() error { return nil }

func (c *fakeRedisClient) Get(ctx stdContext.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, "get", key)
	value, ok := c.data[key]
	switch {
	case c.down != nil:
		cmd.SetErr(c.down)
	case ok:
		cmd.SetVal(value)
	default:
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

func (c *fakeRedisClient) Set(ctx stdContext.Context, key string, value any, _ time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "set", key)
	if c.down != nil {
		cmd.SetErr(c.down)
		return cmd
	}
	c.data[key] = string(value.([]byte))
	return cmd
}

func (c *fakeRedisClient) Del(ctx stdContext.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "del")
	if c.down != nil {
		cmd.SetErr(c.down)
		return cmd
	}
	for _, key := range keys {
		delete(c.data, key)
	}
	return cmd
}

// fakePostgresConn keeps the rows of the sessions table in a map. Like the
// table's UUID column, it fails a statement whose id is not a UUID.
type fakePostgresConn struct {
	rows map[string][]byte // "name:id" -> value
	down error             // when set, every statement fails with it.
}

func (c *fakePostgresConn) key(args []any) (string, error) {
	id, name := args[0].(string), args[1].(string)
	if _, err := uuid.Parse(id); err != nil {
		return "", fmt.Errorf("invalid input syntax for type uuid: %q", id)
	}
	return name + ":" + id, nil
}

func (c *fakePostgresConn) Exec(_ stdContext.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if c.down != nil {
		return pgconn.CommandTag{}, c.down
	}
	if len(args) == 0 { // CREATE TABLE.
		return pgconn.CommandTag{}, nil
	}

	key, err := c.key(args)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	if strings.Contains(sql, "INSERT INTO") {
		c.rows[key] = args[2].([]byte)
	} else {
		delete(c.rows, key)
	}
	return pgconn.CommandTag{}, nil
}

func (c *fakePostgresConn) QueryRow(_ stdContext.Context, _ string, args ...any) pgx.Row {
	if c.down != nil {
		return fakePostgresRow{err: c.down}
	}

	key, err := c.key(args)
	if err != nil {
		return fakePostgresRow{err: err}
	}
	value, ok := c.rows[key]
	if !ok {
		return fakePostgresRow{err: pgx.ErrNoRows}
	}
	return fakePostgresRow{value: value}
}

type fakePostgresRow struct {
	value []byte
	err   error
}

func (r fakePostgresRow) Scan(dest ...any) error {
	if r.err == nil {
		*dest[0].(*[]byte) = r.value
	}
	return r.err
}

// staleDatabase is one of the three shipped databases over its fake store.
type staleDatabase struct {
	name string
	new  func(sessions.Options) sessions.Database[staleUser]
	// fail makes the store behind the last database built by new unreachable.
	// It is nil for the cookie database, which has no store.
	fail func()
}

func staleDatabases() []*staleDatabase {
	cookieDB := &staleDatabase{name: "cookie"}
	cookieDB.new = func(opts sessions.Options) sessions.Database[staleUser] {
		return sessions.NewCookieDatabase[staleUser](opts)
	}

	redisDB := &staleDatabase{name: "redis"}
	redisDB.new = func(opts sessions.Options) sessions.Database[staleUser] {
		client := &fakeRedisClient{data: make(map[string]string)}
		redisDB.fail = func() { client.down = errStoreDown }
		return sessions.NewRedisDatabase[staleUser](client, opts)
	}

	postgresDB := &staleDatabase{name: "postgres"}
	postgresDB.new = func(opts sessions.Options) sessions.Database[staleUser] {
		conn := &fakePostgresConn{rows: make(map[string][]byte)}
		postgresDB.fail = func() { conn.down = errStoreDown }
		return sessions.NewPostgresDatabase[staleUser](sessions.DefaultTableName, conn, opts)
	}

	return []*staleDatabase{cookieDB, redisDB, postgresDB}
}

func staleOptions() sessions.Options {
	return sessions.NewOptions(staleHashKey, staleBlockKey)
}

// staleCookie is a value of the session cookie that the server did not issue
// with its current keys and settings.
type staleCookie struct {
	name  string
	value string
	// decodes is true when the value passes the securecookie check and only
	// its contents are not what v14 writes.
	decodes bool
}

func staleCookies(t *testing.T) []staleCookie {
	t.Helper()

	otherKeys := sessions.NewOptions([]byte("another-hash-key-32-characters!!"), []byte("another-block-16"))
	signedWithOtherKeys, err := otherKeys.Encode(staleCookieName, uuid.NewV7().String())
	if err != nil {
		t.Fatal(err)
	}

	// What v12 wrote with Config.Encoding set to securecookie.New(hashKey, blockKey):
	// the session id through the gob serializer securecookie uses by default.
	fromV12, err := securecookie.New(staleHashKey, staleBlockKey).Encode(staleCookieName, uuid.New().String())
	if err != nil {
		t.Fatal(err)
	}

	return []staleCookie{
		{name: "a plain id", value: "a95828e7-a194-49e8-89cf-d76a987faa2e"},
		{name: "not a session", value: "not-a-session"},
		{name: "signed with other keys", value: signedWithOtherKeys},
		{name: "written by v12 with the same keys", value: fromV12, decodes: true},
	}
}

func newStaleCookieApp(handler iris.Handler) *iris.Application {
	app := iris.New()
	app.Use(handler)

	app.Get("/set", func(ctx iris.Context) {
		session := sessions.Get[staleUser](ctx)
		session.Set(staleUser{Name: "kataras"})
		ctx.WriteString("ok")
	})

	app.Get("/get", func(ctx iris.Context) {
		session := sessions.Get[staleUser](ctx)
		ctx.Writef("name=%s new=%t", session.Data.Name, session.IsNew)
	})

	return app
}

// sessionCookie returns the one session cookie the response sets.
func sessionCookie(t *testing.T, resp *http.Response) *http.Cookie {
	t.Helper()

	var found []*http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == staleCookieName {
			found = append(found, cookie)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected one %s cookie in the response, got %v", staleCookieName, resp.Header["Set-Cookie"])
	}
	return found[0]
}

func TestHandlerUndecodableCookieStartsNewSession(t *testing.T) {
	for _, db := range staleDatabases() {
		for _, cookie := range staleCookies(t) {
			t.Run(db.name+"/"+cookie.name, func(t *testing.T) {
				e := httptest.New(t, newStaleCookieApp(sessions.Handler(db.new(staleOptions()), staleCookieName)))

				// The route runs and the response replaces the cookie.
				resp := e.GET("/set").WithCookie(staleCookieName, cookie.value).Expect().Status(httptest.StatusOK)
				replaced := sessionCookie(t, resp.Raw())
				if replaced.Value == "" || replaced.Value == cookie.value || replaced.MaxAge <= 0 {
					t.Fatalf("expected a new session cookie, got %v", resp.Raw().Header["Set-Cookie"])
				}

				// The new cookie carries the session from here on.
				e.GET("/get").WithCookie(staleCookieName, replaced.Value).Expect().
					Status(httptest.StatusOK).Body().IsEqual("name=kataras new=false")
			})
		}
	}
}

// Without DataCodecs the cookie value is the id itself, and an id that is not a
// UUID does not validate (ErrInvalidSessionID).
func TestHandlerInvalidSessionIDStartsNewSession(t *testing.T) {
	for _, db := range staleDatabases() {
		if db.name == "cookie" { // the cookie database requires DataCodecs.
			continue
		}

		t.Run(db.name, func(t *testing.T) {
			e := httptest.New(t, newStaleCookieApp(sessions.Handler(db.new(sessions.NewOptions()), staleCookieName)))

			resp := e.GET("/set").WithCookie(staleCookieName, "not-a-session").Expect().Status(httptest.StatusOK)
			replaced := sessionCookie(t, resp.Raw())
			if !sessions.IsValidSessionID(replaced.Value) {
				t.Fatalf("expected a new session id, got %q", replaced.Value)
			}

			e.GET("/get").WithCookie(staleCookieName, replaced.Value).Expect().
				Status(httptest.StatusOK).Body().IsEqual("name=kataras new=false")
		})
	}
}

// The read-only handler shares the load: the route runs with a new, empty
// session. It never commits, so it does not write a cookie either.
func TestReadOnlyHandlerUndecodableCookie(t *testing.T) {
	for _, db := range staleDatabases() {
		for _, cookie := range staleCookies(t) {
			if cookie.decodes && db.name == "redis" {
				// The value decodes to an id with no record. That already was
				// a session under a new id, and it does not report IsNew.
				continue
			}

			t.Run(db.name+"/"+cookie.name, func(t *testing.T) {
				e := httptest.New(t, newStaleCookieApp(sessions.ReadOnlyHandler(db.new(staleOptions()), staleCookieName)))

				resp := e.GET("/get").WithCookie(staleCookieName, cookie.value).Expect().Status(httptest.StatusOK)
				resp.Body().IsEqual("name= new=true")
				if lines := resp.Raw().Header["Set-Cookie"]; len(lines) != 0 {
					t.Fatalf("the read-only handler wrote a cookie: %v", lines)
				}
			})
		}
	}
}

// A handler that asks the database itself gets the new session, not an error.
func TestDatabaseGetUndecodableCookie(t *testing.T) {
	for _, db := range staleDatabases() {
		t.Run(db.name, func(t *testing.T) {
			sessionDB := db.new(staleOptions())

			app := iris.New()
			app.Get("/", func(ctx iris.Context) {
				session, err := sessionDB.Get(ctx, staleCookieName)
				if err != nil {
					ctx.StopWithText(httptest.StatusInternalServerError, err.Error())
					return
				}
				// The cookie database keeps no id.
				ctx.Writef("new=%t id=%t", session.IsNew, session.ID == "" || sessions.IsValidSessionID(session.ID))
			})

			httptest.New(t, app).GET("/").WithCookie(staleCookieName, "not-a-session").Expect().
				Status(httptest.StatusOK).Body().IsEqual("new=true id=true")
		})
	}
}

// A cookie older than MaxAge fails the securecookie timestamp check.
func TestHandlerCookieOlderThanMaxAgeStartsNewSession(t *testing.T) {
	t.Parallel() // it sleeps.

	opts := staleOptions()
	opts.MaxAge = time.Second

	type aged struct {
		name   string
		e      *httptest.Expect
		cookie *http.Cookie
	}

	var cases []aged
	for _, db := range staleDatabases() {
		e := httptest.New(t, newStaleCookieApp(sessions.Handler(db.new(opts), staleCookieName)))
		cookie := sessionCookie(t, e.GET("/set").Expect().Status(httptest.StatusOK).Raw())
		cases = append(cases, aged{name: db.name, e: e, cookie: cookie})
	}

	time.Sleep(2100 * time.Millisecond)

	for _, c := range cases {
		resp := c.e.GET("/get").WithCookie(staleCookieName, c.cookie.Value).Expect().Status(httptest.StatusOK)
		resp.Body().IsEqual("name= new=true")
		if replaced := sessionCookie(t, resp.Raw()); replaced.Value == "" || replaced.Value == c.cookie.Value {
			t.Fatalf("%s: expected a new session cookie, got %v", c.name, resp.Raw().Header["Set-Cookie"])
		}
	}
}

// The store being unreachable is not the cookie's fault. It stays a 500, with
// the store's error on the Context as a private one: no client reads it.
func TestHandlerStoreFailureStaysInternalServerError(t *testing.T) {
	for _, db := range staleDatabases() {
		if db.name == "cookie" {
			continue
		}

		t.Run(db.name, func(t *testing.T) {
			var (
				stored   error
				isPublic bool
			)

			app := newStaleCookieApp(sessions.Handler(db.new(staleOptions()), staleCookieName))
			app.UseRouter(func(ctx iris.Context) {
				ctx.Next()
				isPublic, stored = ctx.ErrPublic()
			})

			e := httptest.New(t, app)
			valid := sessionCookie(t, e.GET("/set").Expect().Status(httptest.StatusOK).Raw())

			db.fail()

			// With the valid cookie the load fails. With no cookie, or one that
			// does not decode, the new session fails to commit.
			for _, value := range []string{valid.Value, "", "not-a-session"} {
				req := e.GET("/get")
				if value != "" {
					req = req.WithCookie(staleCookieName, value)
				}

				body := req.Expect().Status(httptest.StatusInternalServerError).Body().Raw()
				if !errors.Is(stored, errStoreDown) || isPublic {
					t.Fatalf("cookie %q: expected the private store failure on the Context, got: %v (public: %t)", value, stored, isPublic)
				}
				if strings.Contains(body, "10.0.0.7") || strings.Contains(body, "connection refused") {
					t.Fatalf("cookie %q: the store failure text reached the client: %s", value, body)
				}
			}
		})
	}
}
