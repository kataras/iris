package httpcache

import (
	"fmt"
	"net/http"
	nethttptest "net/http/httptest"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/compress"
	"github.com/kataras/iris/v14/middleware/recorder"
	"github.com/kataras/iris/v14/middleware/sessions"
)

// newRequest returns a GET request with the given header lines, each one a
// "Name: value". Two lines of one name are two field lines.
func newRequest(path string, lines ...string) *http.Request {
	req := nethttptest.NewRequest(http.MethodGet, path, nil)
	for _, line := range lines {
		name, value, _ := strings.Cut(line, ":")
		req.Header.Add(name, strings.TrimSpace(value))
	}

	return req
}

// serve serves one request and expects a 200. It goes around the httptest
// client, whose cookie jar sends back every cookie a response has set: here a
// test names each cookie that a request sends.
func serve(t *testing.T, app *iris.Application, req *http.Request) *nethttptest.ResponseRecorder {
	t.Helper()

	rec := nethttptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("%s with %q: status %d", req.URL.Path, req.Header, rec.Code)
	}

	return rec
}

// identity is what a page behind a login answers with: the credentials that
// the request carries, read the way a handler reads them.
func identity(r *http.Request) string {
	who := "authorization=" + r.Header.Get("Authorization") + " proxy-authorization=" + r.Header.Get("Proxy-Authorization")
	for _, cookie := range r.Cookies() {
		who += " " + cookie.Name + "=" + cookie.Value
	}

	return who
}

// newIdentityApp returns an application with the cache handler, its key on the
// given headers and cookies, in front of a route that answers with the
// identity of the request. It returns the number of runs of that route and
// the store too.
func newIdentityApp(t *testing.T, headers, cookies []string) (*iris.Application, *int, *fakeStore) {
	t.Helper()

	c, store := newTestCache(t)
	c.Options.Headers, c.Options.Cookies = headers, cookies

	app := iris.New()
	app.Use(c.Handler())

	runs := new(int)
	app.Get("/", func(ctx iris.Context) {
		*runs++
		ctx.Text(identity(ctx.Request()))
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	return app, runs, store
}

// visit serves one request with the given header lines. A body that is not
// the identity of the request is a page made for another visitor. The test
// goes on after one, to report every visitor that got one.
func visit(t *testing.T, app *iris.Application, lines ...string) {
	t.Helper()

	req := newRequest("/", lines...)
	if got, want := serve(t, app, req).Body.String(), identity(req); got != want {
		t.Errorf("a request with %q got the page of another visitor: %q, expected %q", lines, got, want)
	}
}

// The cache key holds the headers that Options.Headers names and the cookies
// that Options.Cookies names. A request with a credential outside of them was
// looked up and stored as soon as either option named anything, so the page of
// one visitor went to the next one that sent the same named headers and
// cookies. The check that a request carries a cookie read the first Cookie
// line alone, where a handler reads them all.
func TestHandlerSkipsARequestWithACredentialOutsideTheKey(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		cookies []string
		first   []string   // the header lines of the first visitor.
		others  [][]string // those of the visitors after it.
	}{
		{
			name:    "an Authorization header, the key on Accept-Language",
			headers: []string{"Accept-Language"},
			first:   []string{"Authorization: Bearer alice"},
			others:  [][]string{{"Authorization: Bearer bob"}, {}, {"Cookie: sid=carol"}},
		},
		{
			name:    "an Authorization header, the key on the sid cookie",
			cookies: []string{"sid"},
			first:   []string{"Authorization: Bearer alice"},
			others:  [][]string{{"Authorization: Bearer bob"}, {}},
		},
		{
			name:    "another cookie, the key on the sid cookie",
			cookies: []string{"sid"},
			first:   []string{"Cookie: token=alice"},
			others:  [][]string{{"Cookie: token=bob"}, {}},
		},
		{
			name:    "a session cookie, the key on the theme cookie",
			cookies: []string{"theme"},
			first:   []string{"Cookie: session=alice; theme=dark"},
			others:  [][]string{{"Cookie: session=bob; theme=dark"}, {"Cookie: theme=dark"}},
		},
		{
			name:    "a cookie, the key on Authorization",
			headers: []string{"Authorization"},
			first:   []string{"Authorization: Bearer shared", "Cookie: sid=alice"},
			others:  [][]string{{"Authorization: Bearer shared", "Cookie: sid=bob"}, {"Authorization: Bearer shared"}},
		},
		{
			name:    "a Proxy-Authorization header, the key on Authorization",
			headers: []string{"Authorization"},
			first:   []string{"Proxy-Authorization: Basic alice"},
			others:  [][]string{{"Proxy-Authorization: Basic bob"}, {}},
		},
		{
			name:    "a cookie on a second Cookie line, the key on the sid cookie",
			cookies: []string{"sid"},
			first:   []string{"Cookie: sid=shared", "Cookie: token=alice"},
			others:  [][]string{{"Cookie: sid=shared", "Cookie: token=bob"}, {"Cookie: sid=shared"}},
		},
		{
			name:    "a cookie on a second Cookie line, the key on the Cookie header",
			headers: []string{"Cookie"},
			first:   []string{"Cookie: theme=dark", "Cookie: sid=alice"},
			others:  [][]string{{"Cookie: theme=dark", "Cookie: sid=bob"}, {"Cookie: theme=dark"}},
		},
		{
			name:   "a cookie under an empty Cookie line, the default key",
			first:  []string{"Cookie:", "Cookie: sid=alice"},
			others: [][]string{{}, {"Cookie:", "Cookie: sid=bob"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, runs, store := newIdentityApp(t, tt.headers, tt.cookies)

			// Twice: the second one is the one that would come out of the store.
			visit(t, app, tt.first...)
			visit(t, app, tt.first...)
			stored := store.keys()

			for _, other := range tt.others {
				visit(t, app, other...)
			}

			// And the other way round: what the others left in the store.
			visit(t, app, tt.first...)

			if len(stored) != 0 {
				t.Fatalf("the response of the first visitor was stored: %q", stored)
			}

			if *runs < 3 {
				t.Fatalf("the route handler ran %d times for the 3 requests of the first visitor", *runs)
			}
		})
	}
}

// The same with the session middleware behind the cache handler. A request
// that only reads its session sends no Set-Cookie, and a Set-Cookie was what
// kept the response of such a request out of the store.
func TestHandlerSkipsASessionCookieOutsideTheKey(t *testing.T) {
	type user struct{ Name string }

	c, store := newTestCache(t)
	c.Options.Cookies = []string{"theme"}

	db := sessions.NewCookieDatabase[user](sessions.NewOptions(
		[]byte("D*G-KaPdSgUkXp2s5v8y/B?E(H+MbQeThWmYq3t6w9z$C&F)J@NcRfUjXn2r4u7x"),
		[]byte("VkYp3s6v9y$B&E)H@McQfTjWmZq4t7w!")))

	app := iris.New()
	app.Use(c.Handler(), sessions.Handler(db, "session"))

	for _, name := range []string{"alice", "bob"} {
		app.Get("/login/"+name, func(ctx iris.Context) {
			sessions.Get[user](ctx).Set(user{Name: name})
			ctx.Text("signed in")
		})
	}

	app.Get("/me", func(ctx iris.Context) {
		ctx.Text("hello " + sessions.Get[user](ctx).Data.Name)
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	// login returns the Cookie line of a browser that holds the session
	// cookie of the user and a theme cookie.
	login := func(name string) string {
		for _, cookie := range serve(t, app, newRequest("/login/"+name)).Result().Cookies() {
			if cookie.Name == "session" {
				return "Cookie: session=" + cookie.Value + "; theme=dark"
			}
		}

		t.Fatalf("%s got no session cookie", name)
		return ""
	}

	alice, bob := login("alice"), login("bob")

	for _, visitor := range []struct{ cookie, page string }{
		{alice, "hello alice"},
		{alice, "hello alice"},
		{bob, "hello bob"},
		{alice, "hello alice"},
	} {
		res := serve(t, app, newRequest("/me", visitor.cookie))

		if got := res.Body.String(); got != visitor.page {
			t.Fatalf("the visitor of %q received %q", visitor.page, got)
		}

		if got := res.Header().Values("Set-Cookie"); len(got) != 0 {
			t.Fatalf("a request that only reads its session sent a cookie: %q", got)
		}
	}

	if keys := store.keys(); len(keys) != 0 {
		t.Fatalf("a page of a session was stored: %q", keys)
	}
}

// What was safe to cache is cached as before: a request without credentials,
// and a request whose credentials are all in the key, one entry for each
// visitor.
func TestHandlerCachesARequestWithItsCredentialsInTheKey(t *testing.T) {
	tests := []struct {
		name     string
		headers  []string
		cookies  []string
		visitors [][]string // the header lines of each visitor.
	}{
		{
			name:     "no credentials, the default key",
			visitors: [][]string{{}},
		},
		{
			name:     "an empty Cookie line, the default key",
			visitors: [][]string{{"Cookie:"}},
		},
		{
			name:     "no credentials, the key on Authorization and the sid cookie",
			headers:  []string{"Authorization"},
			cookies:  []string{"sid"},
			visitors: [][]string{{"Accept-Language: el"}},
		},
		{
			name:     "an Authorization header, the key on it",
			headers:  []string{"Authorization"},
			visitors: [][]string{{"Authorization: Bearer alice"}, {"Authorization: Bearer bob"}, {}},
		},
		{
			name:    "header names without regard to case",
			headers: []string{"authorization", "PROXY-AUTHORIZATION"},
			visitors: [][]string{
				{"Authorization: Bearer alice", "Proxy-Authorization: Basic alice"},
				{"Authorization: Bearer alice", "Proxy-Authorization: Basic bob"},
				{"Authorization: Bearer bob"},
			},
		},
		{
			name:     "the sid cookie alone, the key on it",
			cookies:  []string{"sid"},
			visitors: [][]string{{"Cookie: sid=alice"}, {"Cookie: sid=bob"}, {}},
		},
		{
			name:     "a header and two cookies, the key on the three",
			headers:  []string{"Authorization"},
			cookies:  []string{"sid", "theme"},
			visitors: [][]string{{"Authorization: Bearer alice", "Cookie: sid=alice; theme=dark"}, {"Cookie: sid=alice; theme=dark"}, {"Cookie: sid=alice"}},
		},
		{
			name:     "two Cookie lines, the key on the cookies of both",
			cookies:  []string{"sid", "theme"},
			visitors: [][]string{{"Cookie: theme=dark", "Cookie: sid=alice"}, {"Cookie: theme=dark", "Cookie: sid=bob"}},
		},
		{
			name:     "a cookie that is sent twice counts by its name",
			cookies:  []string{"sid"},
			visitors: [][]string{{"Cookie: sid=alice; sid=old"}, {"Cookie: sid=bob; sid=old"}},
		},
		{
			name:     "any cookies, the key on the Cookie header",
			headers:  []string{"Cookie"},
			visitors: [][]string{{"Cookie: sid=alice; theme=dark"}, {"Cookie: sid=bob; theme=dark"}, {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, runs, store := newIdentityApp(t, tt.headers, tt.cookies)

			for range 3 {
				for _, visitor := range tt.visitors {
					visit(t, app, visitor...)
				}
			}

			if *runs != len(tt.visitors) {
				t.Fatalf("the route handler ran %d times for %d visitors", *runs, len(tt.visitors))
			}

			if keys := store.keys(); len(keys) != len(tt.visitors) {
				t.Fatalf("expected one stored response for each of the %d visitors, got %q", len(tt.visitors), keys)
			}
		})
	}
}

// The cost of the rule. A request that sends a cookie the key does not name is
// not cached, whatever the cookie is for: under Cookies: ["sid"] that is every
// request of a browser that also holds an analytics or a theme cookie. Naming
// those cookies too caches it again, one entry for each set of values.
func TestHandlerSkipsARequestWithACookieTheKeyDoesNotName(t *testing.T) {
	const browser = "Cookie: sid=alice; _ga=GA1.2.3.4"

	app, runs, store := newIdentityApp(t, nil, []string{"sid"})

	visit(t, app, browser)
	visit(t, app, browser)

	if *runs != 2 {
		t.Fatalf("the route handler ran %d times for 2 requests", *runs)
	}

	if keys := store.keys(); len(keys) != 0 {
		t.Fatalf("the response was stored: %q", keys)
	}

	app, runs, store = newIdentityApp(t, nil, []string{"sid", "_ga"})

	visit(t, app, browser)
	visit(t, app, browser)

	if *runs != 1 {
		t.Fatalf("with both cookies named the route handler ran %d times for 2 requests", *runs)
	}

	if keys := store.keys(); len(keys) != 1 {
		t.Fatalf("with both cookies named, expected one stored response, got %q", keys)
	}
}

// The key holds what http.Request.Cookie returns for a name, and that method
// skips the parts of a Cookie line it cannot read. A request with a Cookie
// line that is not a list of name=value pairs, as http.ParseCookie reads one,
// is not cached. That takes in two lines http.Request.Cookie does read.
func TestHandlerSkipsACookieLineThatIsNotAListOfPairs(t *testing.T) {
	for _, line := range []string{
		`Cookie: sid=al"ice`,     // a value that is skipped.
		"Cookie: sid=alice; =x",  // a part with no name, skipped.
		"Cookie: sid=alice; x y", // a part whose name is not one, skipped.
		"Cookie: sid",            // no "=": read as a cookie with an empty value.
		"Cookie: sid=alice;",     // an empty part: read as sid alone.
	} {
		t.Run(line, func(t *testing.T) {
			app, runs, store := newIdentityApp(t, nil, []string{"sid"})

			visit(t, app, line)
			visit(t, app, line)

			if *runs != 2 {
				t.Fatalf("the route handler ran %d times for 2 requests", *runs)
			}

			if keys := store.keys(); len(keys) != 0 {
				t.Fatalf("the response was stored: %q", keys)
			}
		})
	}
}

// net/http reads no cookie at all from a request that sends more of them than
// its limit (GODEBUG httpcookiemaxnum, 3000 by default), and it counts them
// over all the Cookie lines. Each line of such a request can still be a list
// of pairs that the key names. The key holds none of them then, so the request
// would be looked up and stored as one that sends no cookie.
func TestHandlerSkipsMoreCookiesThanTheKeyReads(t *testing.T) {
	// Two lines of 2000 pairs: each line is under the limit, the request is not.
	line := "Cookie: sid=alice" + strings.Repeat("; sid=alice", 1999)

	if cookies := newRequest("/", line, line).Cookies(); len(cookies) != 0 {
		t.Skipf("net/http reads the %d cookies of the request", len(cookies))
	}

	app, runs, store := newIdentityApp(t, nil, []string{"sid"})

	visit(t, app, line, line)
	visit(t, app, line, line)

	if *runs != 2 {
		t.Fatalf("the route handler ran %d times for 2 requests", *runs)
	}

	if keys := store.keys(); len(keys) != 0 {
		t.Fatalf("the response was stored: %q", keys)
	}
}

// A request that sends no cookie carries no credential, so it is cached, also
// when Options.Cookies names a cookie. Its entry is not the one of a request
// that sends that cookie, in either order, and neither when the cookie has an
// empty value: the key used to leave an empty cookie out, as if it was not
// sent.
func TestHandlerKeepsARequestWithoutTheNamedCookieApart(t *testing.T) {
	for _, cookie := range []string{"Cookie: sid=alice", "Cookie: sid="} {
		for _, order := range [][2][]string{{nil, {cookie}}, {{cookie}, nil}} {
			t.Run(fmt.Sprintf("%q then %q", order[0], order[1]), func(t *testing.T) {
				app, runs, store := newIdentityApp(t, nil, []string{"sid"})

				for range 2 {
					visit(t, app, order[0]...)
					visit(t, app, order[0]...)
					visit(t, app, order[1]...)
					visit(t, app, order[1]...)
				}

				if *runs != 2 {
					t.Fatalf("the route handler ran %d times for 2 visitors", *runs)
				}

				if keys := store.keys(); len(keys) != 2 {
					t.Fatalf("expected two stored responses, got %q", keys)
				}
			})
		}
	}
}

// A response that sets a cookie is for one client and is not stored. The
// cookie is not always set by the route handler: a handler registered before
// the cache handler sets it on a writer under the recorder, where the headers
// recorded after it do not show it.
func TestHandlerSeesASetCookieSetBeforeIt(t *testing.T) {
	visitors := 0
	setCookie := func(ctx iris.Context) {
		visitors++
		ctx.SetCookie(&http.Cookie{Name: "visitor", Value: fmt.Sprint(visitors)})
		ctx.Next()
	}

	tests := []struct {
		name       string
		compressed bool
		chain      func(c *Cache) []iris.Handler
	}{
		{"before the cache handler", false, func(c *Cache) []iris.Handler {
			return []iris.Handler{setCookie, c.Handler()}
		}},
		{"before a recorder the cache handler shares", false, func(c *Cache) []iris.Handler {
			return []iris.Handler{setCookie, recorder.Response, c.Handler()}
		}},
		{"between that recorder and the cache handler", false, func(c *Cache) []iris.Handler {
			return []iris.Handler{recorder.Response, setCookie, c.Handler()}
		}},
		{"under another recorder and a compression writer", true, func(c *Cache) []iris.Handler {
			return []iris.Handler{setCookie, recorder.Response, compress.Safe, c.Handler()}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, store := newTestCache(t)

			welcome, catalog := new(int), new(int)

			app := iris.New()
			app.Get("/welcome", append(tt.chain(c), func(ctx iris.Context) {
				*welcome++
				ctx.Text("welcome")
			})...)
			// A route without that handler in front of it is cached.
			app.Get("/catalog", c.Handler(), func(ctx iris.Context) {
				*catalog++
				ctx.Text("catalog")
			})

			if err := app.Build(); err != nil {
				t.Fatal(err)
			}

			for range 2 {
				res := serve(t, app, newRequest("/welcome", "Accept-Encoding: gzip"))

				if got := res.Header().Values("Set-Cookie"); len(got) != 1 {
					t.Fatalf("expected the cookie of the first handler on the response, got %q", got)
				}

				// The compression writer is in the chain that has one.
				if got := res.Header().Get("Content-Encoding"); (got == "gzip") != tt.compressed {
					t.Fatalf("Content-Encoding is %q", got)
				}
			}

			if *welcome != 2 {
				t.Fatalf("the route handler ran %d times for 2 requests", *welcome)
			}

			if keys := store.keys(); len(keys) != 0 {
				t.Fatalf("a response that sets a cookie was stored: %q", keys)
			}

			serve(t, app, newRequest("/catalog"))
			serve(t, app, newRequest("/catalog"))

			if *catalog != 1 {
				t.Fatalf("the route handler without the cookie ran %d times for 2 requests", *catalog)
			}
		})
	}
}
