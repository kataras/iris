package httpcache

import (
	"bufio"
	"fmt"
	"io"
	"net"
	nethttptest "net/http/httptest"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/compress"
	"github.com/kataras/iris/v14/middleware/recorder"
)

// fakeStore stands in for DragonflyDB/Redis for one test. It listens on a
// loopback port the system picks and speaks the part of RESP2 this package
// sends: PING, GET, SET, SCAN and UNLINK/DEL.
type fakeStore struct {
	ln net.Listener

	mu   sync.Mutex
	data map[string][]byte
}

func newFakeStore(t *testing.T) *fakeStore {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	s := &fakeStore{ln: ln, data: make(map[string][]byte)}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.serve(conn)
		}
	}()

	return s
}

// readCommand reads one command, an array of bulk strings.
func readCommand(r *bufio.Reader) ([][]byte, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}

	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("unexpected line %q", line)
	}

	n, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, err
	}

	args := make([][]byte, 0, n)
	for range n {
		head, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}

		size, err := strconv.Atoi(strings.TrimRight(head, "\r\n")[1:])
		if err != nil {
			return nil, err
		}

		buf := make([]byte, size+2) // the argument and its CRLF.
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, buf[:size])
	}

	return args, nil
}

func (s *fakeStore) serve(conn net.Conn) {
	defer conn.Close()

	r := bufio.NewReader(conn)
	for {
		args, err := readCommand(r)
		if err != nil || len(args) == 0 {
			return
		}

		switch cmd := strings.ToUpper(string(args[0])); cmd {
		case "HELLO":
			// Answer as an old server does: the client falls back to RESP2.
			io.WriteString(conn, "-ERR unknown command 'HELLO'\r\n")
		case "PING":
			io.WriteString(conn, "+PONG\r\n")
		case "CLIENT":
			io.WriteString(conn, "+OK\r\n")
		case "SET":
			s.mu.Lock()
			s.data[string(args[1])] = slices.Clone(args[2])
			s.mu.Unlock()
			io.WriteString(conn, "+OK\r\n")
		case "GET":
			s.mu.Lock()
			v, ok := s.data[string(args[1])]
			s.mu.Unlock()
			if !ok {
				io.WriteString(conn, "$-1\r\n")
				continue
			}
			fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(v), v)
		case "SCAN":
			prefix := ""
			for i := 2; i+1 < len(args); i += 2 {
				if strings.EqualFold(string(args[i]), "MATCH") {
					prefix = strings.TrimSuffix(string(args[i+1]), "*")
				}
			}

			var keys []string
			for _, k := range s.keys() {
				if strings.HasPrefix(k, prefix) {
					keys = append(keys, k)
				}
			}

			fmt.Fprintf(conn, "*2\r\n$1\r\n0\r\n*%d\r\n", len(keys))
			for _, k := range keys {
				fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(k), k)
			}
		case "UNLINK", "DEL":
			s.mu.Lock()
			n := 0
			for _, k := range args[1:] {
				if _, ok := s.data[string(k)]; ok {
					delete(s.data, string(k))
					n++
				}
			}
			s.mu.Unlock()
			fmt.Fprintf(conn, ":%d\r\n", n)
		default:
			fmt.Fprintf(conn, "-ERR unknown command '%s'\r\n", cmd)
		}
	}
}

// keys returns the keys held by the store, sorted.
func (s *fakeStore) keys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	return keys
}

// newTestCache returns a Cache over a fakeStore of its own.
func newTestCache(t *testing.T) (*Cache, *fakeStore) {
	t.Helper()

	store := newFakeStore(t)

	c, err := New(Options{
		Addr:           store.ln.Addr().String(),
		TTL:            time.Minute,
		CommandTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { c.Close() })

	return c, store
}

// replayed is the Cache-Control of a response that came out of the store.
var replayed = regexp.MustCompile(`^public, max-age=\d+$`)

// cacheControlRoute registers a GET route that answers with the given
// Cache-Control field lines and with a body that names the run, so a test can
// tell a stored body from a new one. It returns the number of runs.
func cacheControlRoute(app *iris.Application, path string, cacheControl ...string) *int {
	runs := new(int)

	app.Get(path, func(ctx iris.Context) {
		*runs++

		for i, value := range cacheControl {
			if i == 0 {
				// Replaces what a handler before this one may have set.
				ctx.SetHeader("Cache-Control", value)
				continue
			}

			ctx.AddHeader("Cache-Control", value)
		}
		ctx.Text(fmt.Sprintf("run %d", *runs))
	})

	return runs
}

// A response that says private, no-store or no-cache is meant for one client,
// for no cache at all, or for no reuse without asking the handler again. It
// used to be stored on its 200 alone, and every later request got the stored
// body with "Cache-Control: public, max-age=N" in place of the handler's own
// header.
func TestHandlerLeavesARestrictedResponseOutOfTheStore(t *testing.T) {
	tests := []struct {
		name         string
		cacheControl []string // the field lines the route handler sets.
	}{
		{"private", []string{"private"}},
		{"no-store", []string{"no-store"}},
		{"no-cache", []string{"no-cache"}},
		{"upper case", []string{"Private"}},
		{"not the first directive", []string{"max-age=60, must-revalidate, no-store"}},
		{"spaces and tabs", []string{"max-age=60 ,\t NO-CACHE  "}},
		{"second field line", []string{"max-age=60", "no-store"}},
		{"private with field names", []string{`private="Set-Cookie, X-Account"`}},
		{"no-cache with a field name", []string{`max-age=60, no-cache="Set-Cookie"`}},
		{"after a quoted comma", []string{`ext="a, b", private`}},
		{"after an escaped quote", []string{`ext="a \" b", no-store`}},
		{"a quote left open", []string{`max-age=60, ext="a, no-store`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, store := newTestCache(t)

			app := iris.New()
			app.Use(c.Handler())
			runs := cacheControlRoute(app, "/", tt.cacheControl...)

			e := httptest.New(t, app)

			for run := 1; run <= 2; run++ {
				res := e.GET("/").Expect().Status(httptest.StatusOK)
				res.Body().IsEqual(fmt.Sprintf("run %d", run))

				if got := res.Raw().Header.Values("Cache-Control"); !slices.Equal(got, tt.cacheControl) {
					t.Fatalf("request %d: Cache-Control is %q, the handler set %q", run, got, tt.cacheControl)
				}
			}

			if *runs != 2 {
				t.Fatalf("the route handler ran %d times for 2 requests", *runs)
			}

			if keys := store.keys(); len(keys) != 0 {
				t.Fatalf("the response was stored: %q", keys)
			}
		})
	}
}

// Every other 200 is stored and replayed as it was before, also when one of
// the three words is part of another directive's name or of a quoted argument.
func TestHandlerStoresAResponseThatIsNotRestricted(t *testing.T) {
	tests := []struct {
		name         string
		cacheControl []string
	}{
		{"no header", nil},
		{"max-age", []string{"max-age=60"}},
		{"public", []string{"public, max-age=60"}},
		{"inside another name", []string{"x-private, no-store-here=1, my-no-cache"}},
		{"inside a quoted argument", []string{`max-age=60, ext="private, no-store, no-cache"`}},
		{"inside a quoted argument after an escaped quote", []string{`max-age=60, ext="a \" b, private"`}},
		{"as an argument", []string{"ext=private, max-age=60"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, store := newTestCache(t)

			app := iris.New()
			app.Use(c.Handler())
			runs := cacheControlRoute(app, "/", tt.cacheControl...)

			e := httptest.New(t, app)

			first := e.GET("/").Expect().Status(httptest.StatusOK)
			first.Body().IsEqual("run 1")
			if got := first.Raw().Header.Values("Cache-Control"); !slices.Equal(got, tt.cacheControl) {
				t.Fatalf("first request: Cache-Control is %q, the handler set %q", got, tt.cacheControl)
			}

			second := e.GET("/").Expect().Status(httptest.StatusOK)
			second.Body().IsEqual("run 1")
			if got := second.Raw().Header.Values("Cache-Control"); len(got) != 1 || !replayed.MatchString(got[0]) {
				t.Fatalf("second request: Cache-Control is %q, expected public with a max-age", got)
			}

			if *runs != 1 {
				t.Fatalf("the route handler ran %d times for 2 requests", *runs)
			}

			if keys := store.keys(); len(keys) != 1 {
				t.Fatalf("expected one stored response, got %q", keys)
			}
		})
	}
}

// The Cache-Control of a response is not always set by the route handler. A
// handler registered before the cache handler sets it on the writer under the
// recorder, where the headers recorded after it do not show it.
func TestHandlerSeesACacheControlSetBeforeIt(t *testing.T) {
	private := func(ctx iris.Context) {
		ctx.SetHeader("Cache-Control", "private")
		ctx.Next()
	}

	tests := []struct {
		name  string
		chain func(c *Cache) []iris.Handler
	}{
		{"before the cache handler", func(c *Cache) []iris.Handler {
			return []iris.Handler{private, c.Handler()}
		}},
		{"before a recorder the cache handler shares", func(c *Cache) []iris.Handler {
			return []iris.Handler{private, recorder.Response, c.Handler()}
		}},
		{"between that recorder and the cache handler", func(c *Cache) []iris.Handler {
			return []iris.Handler{recorder.Response, private, c.Handler()}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, store := newTestCache(t)

			app := iris.New()
			app.Use(tt.chain(c)...)
			account := cacheControlRoute(app, "/account")
			// The route handler has the last word: this one is public again.
			catalog := cacheControlRoute(app, "/catalog", "public, max-age=60")

			e := httptest.New(t, app)

			for run := 1; run <= 2; run++ {
				res := e.GET("/account").Expect().Status(httptest.StatusOK)
				res.Body().IsEqual(fmt.Sprintf("run %d", run))
				res.Header("Cache-Control").IsEqual("private")
			}

			if *account != 2 {
				t.Fatalf("the private route handler ran %d times for 2 requests", *account)
			}

			if keys := store.keys(); len(keys) != 0 {
				t.Fatalf("the private response was stored: %q", keys)
			}

			e.GET("/catalog").Expect().Status(httptest.StatusOK).Header("Cache-Control").IsEqual("public, max-age=60")
			e.GET("/catalog").Expect().Status(httptest.StatusOK).Body().IsEqual("run 1")

			if *catalog != 1 {
				t.Fatalf("the public route handler ran %d times for 2 requests", *catalog)
			}
		})
	}
}

// The writer under the cache handler's recorder is not always the one an
// earlier handler wrote to. With security headers, a recorder (the access
// log's) and compression registered in that order, the cache handler records
// over the compression writer, whose headers are the first recorder's: the
// Cache-Control is one writer further down. The compression writer is only
// there for a client that accepts an encoding, which every browser does.
func TestHandlerSeesACacheControlSetUnderAnotherRecorder(t *testing.T) {
	c, store := newTestCache(t)

	noStore := func(ctx iris.Context) {
		ctx.SetHeader("Cache-Control", "no-store")
		ctx.Next()
	}

	app := iris.New()
	app.Use(noStore, recorder.Response, compress.Safe, c.Handler())
	account := cacheControlRoute(app, "/account")
	// The route handler has the last word here too.
	catalog := cacheControlRoute(app, "/catalog", "public, max-age=60")

	e := httptest.New(t, app)

	for run := 1; run <= 2; run++ {
		res := e.GET("/account").WithHeader("Accept-Encoding", "gzip").Expect().Status(httptest.StatusOK)
		res.Header("Content-Encoding").IsEqual("gzip") // the compression writer is in the chain.

		if got := res.Raw().Header.Values("Cache-Control"); !slices.Equal(got, []string{"no-store"}) {
			t.Fatalf("request %d: Cache-Control is %q, the first handler set no-store", run, got)
		}
	}

	if *account != 2 {
		t.Fatalf("the no-store route handler ran %d times for 2 requests", *account)
	}

	if keys := store.keys(); len(keys) != 0 {
		t.Fatalf("the no-store response was stored: %q", keys)
	}

	for run := 1; run <= 2; run++ {
		res := e.GET("/catalog").WithHeader("Accept-Encoding", "gzip").Expect().Status(httptest.StatusOK)
		res.Header("Content-Encoding").IsEqual("gzip")

		if got := res.Raw().Header.Get("Cache-Control"); !replayed.MatchString(got) {
			t.Fatalf("public route, request %d: Cache-Control is %q, expected public with a max-age", run, got)
		}
	}

	if *catalog != 1 {
		t.Fatalf("the public route handler ran %d times for 2 requests", *catalog)
	}

	if keys := store.keys(); len(keys) != 1 {
		t.Fatalf("expected the public response alone in the store, got %q", keys)
	}
}

// WriteTo replaced any Cache-Control with "public, max-age=N". A stored
// response that carries one of the three directives (written by an earlier
// version, or by a caller of Set) keeps it, and so does a writer that already
// has one when the stored response has none.
func TestResponseWriteToKeepsARestrictedCacheControl(t *testing.T) {
	tests := []struct {
		name   string
		stored []string // the Cache-Control of the stored response.
		writer string   // the Cache-Control already on the writer.
		want   *regexp.Regexp
	}{
		{"none", nil, "", replayed},
		{"stored max-age", []string{"max-age=60"}, "", replayed},
		{"stored private", []string{"private"}, "", regexp.MustCompile(`^private$`)},
		{"stored no-store on a second line", []string{"max-age=60", "No-Store"}, "", regexp.MustCompile(`^max-age=60\|No-Store$`)},
		{"writer no-cache", nil, "no-cache", regexp.MustCompile(`^no-cache$`)},
		{"writer max-age", nil, "max-age=60", replayed},
		{"stored public over writer private", []string{"public"}, "private", replayed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := Response{
				ExpiresAt: time.Now().Add(time.Minute),
				Body:      []byte("stored"),
				Headers:   map[string][]string{"Content-Type": {"text/plain"}},
			}
			if tt.stored != nil {
				resp.Headers["Cache-Control"] = tt.stored
			}

			w := nethttptest.NewRecorder()
			if tt.writer != "" {
				w.Header().Set("Cache-Control", tt.writer)
			}

			if err := resp.WriteTo(w); err != nil {
				t.Fatal(err)
			}

			if got := strings.Join(w.Header().Values("Cache-Control"), "|"); !tt.want.MatchString(got) {
				t.Fatalf("Cache-Control is %q, expected %s", got, tt.want)
			}

			if got := w.Body.String(); got != "stored" {
				t.Fatalf("body is %q", got)
			}
		})
	}
}
