package fileserver_test

import (
	"compress/gzip"
	"io"
	"net/http"
	nethttptest "net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
	"github.com/kataras/iris/v14/middleware/compress"
)

// docsSite lays out a documentation site whose pages use relative links
// (css/app.css, page2.html), the case of the issue #1803:
//
//	<root>/<indexName>, page2.html, css/app.css
//	<root>/doc/<indexName>, page2.html, css/app.css
//	<root>/doc/guide/<indexName>, page2.html, css/app.css
//	<root>/my docs/<indexName>, page2.html, css/app.css
//	<root>/assets/logo.txt                                (a directory with no index)
//
// Every file names its own directory in its body, so a test can tell
// which css/app.css a relative link ended at.
func docsSite(t *testing.T, indexName string) string {
	t.Helper()

	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		name := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	for _, dir := range []string{"", "/doc", "/doc/guide", "/my docs"} {
		rel := strings.TrimPrefix(dir, "/")
		write(rel+"/"+indexName, indexBody(dir))
		write(rel+"/page2.html", "page2 of ["+dir+"]")
		write(rel+"/css/app.css", "/* css of ["+dir+"] */")
	}
	write("assets/logo.txt", "logo")

	return root
}

func indexBody(dir string) string {
	return `<html><head><link href="css/app.css" rel="stylesheet"></head>` +
		`<body>index of [` + dir + `] <a href="page2.html">next</a></body></html>`
}

type routerMode struct {
	name string
	edit func(*iris.Options)
}

// The path correction of the router decides what happens to a trailing slash,
// so every redirect of the file server is checked under its three modes.
var routerModes = []routerMode{
	{"PathCorrection", func(*iris.Options) {}},
	{"DisablePathCorrectionRedirection", func(o *iris.Options) { o.Router.DisablePathCorrectionRedirection = true }},
	{"DisablePathCorrection", func(o *iris.Options) { o.Router.DisablePathCorrection = true }},
}

func newDocsApp(t *testing.T, mode routerMode, register func(app *iris.Application)) http.Handler {
	t.Helper()

	app := iris.New(iris.Defaults(mode.edit))
	register(app)
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}
	return app
}

// do sends a request the way a server receives it from a browser:
// target is a path, with or without a query, and becomes the RequestURI.
func do(h http.Handler, method, target string, header http.Header) *nethttptest.ResponseRecorder {
	req := nethttptest.NewRequest(method, target, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	rec := nethttptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type hop struct {
	URL      string // what the browser asked for, path and query.
	Code     int
	Location string // as the server sent it.
}

// browse asks for start and follows the redirects the way a browser does:
// a Location is resolved against the URL that answered with it.
// It fails the test on a redirect loop.
func browse(t *testing.T, h http.Handler, start string, header http.Header) ([]hop, *nethttptest.ResponseRecorder) {
	t.Helper()

	cur, err := url.Parse("http://example.com" + start)
	if err != nil {
		t.Fatal(err)
	}

	var hops []hop
	for range 10 {
		rec := do(h, http.MethodGet, cur.RequestURI(), header)
		loc := rec.Header().Get("Location")
		hops = append(hops, hop{URL: cur.RequestURI(), Code: rec.Code, Location: loc})
		if rec.Code < 300 || rec.Code >= 400 || loc == "" {
			return hops, rec
		}

		if cur, err = cur.Parse(loc); err != nil {
			t.Fatal(err)
		}
	}

	t.Fatalf("%s: redirect loop: %+v", start, hops)
	return nil, nil
}

// resolve returns the URL a browser asks for when the page at pageURL links to ref.
func resolve(t *testing.T, pageURL, ref string) string {
	t.Helper()

	base, err := url.Parse("http://example.com" + pageURL)
	if err != nil {
		t.Fatal(err)
	}
	u, err := base.Parse(ref)
	if err != nil {
		t.Fatal(err)
	}
	return u.RequestURI()
}

// expectIndexPage browses to start and expects to end at wantURL, on the index
// page of dir, with both relative links of that page loading the files of dir.
func expectIndexPage(t *testing.T, h http.Handler, start, wantURL, dir string) {
	t.Helper()

	hops, rec := browse(t, h, start, nil)
	last := hops[len(hops)-1]
	if last.Code != http.StatusOK || last.URL != wantURL {
		t.Errorf("%s: ended at %s (%d), want %s (200): %+v", start, last.URL, last.Code, wantURL, hops)
		return
	}
	if got := rec.Body.String(); got != indexBody(dir) {
		t.Errorf("%s: ended at %s with body %q, want the index of [%s]", start, last.URL, got, dir)
		return
	}

	for ref, want := range map[string]string{
		"css/app.css": "/* css of [" + dir + "] */",
		"page2.html":  "page2 of [" + dir + "]",
	} {
		target := resolve(t, last.URL, ref)
		if got := do(h, http.MethodGet, target, nil); got.Code != http.StatusOK || got.Body.String() != want {
			t.Errorf("%s: ended at %s, where %s resolves to %s: got %d %q, want 200 %q",
				start, last.URL, ref, target, got.Code, got.Body.String(), want)
		}
	}
}

// expectIndexPages checks the three URLs of each directory index under prefix:
// the directory, the directory with a trailing slash and the index file itself.
//
// With RedirectToIndex the visitor ends on the URL of the index file. The one
// exception is a trailing slash that reaches the file server (path correction
// off, or on without its redirection): the page is served there, as the
// relative links resolve inside the directory already.
func expectIndexPages(t *testing.T, h http.Handler, mode routerMode, prefix, indexName string, dirs ...string) {
	t.Helper()

	for _, escapedDir := range dirs {
		dir, err := url.PathUnescape(escapedDir)
		if err != nil {
			t.Fatal(err)
		}

		indexURL := prefix + escapedDir + "/" + indexName
		slashURL := prefix + escapedDir + "/"

		wantOfSlash := indexURL
		if mode.name != "PathCorrection" || slashURL == "/" {
			wantOfSlash = slashURL
		}

		expectIndexPage(t, h, indexURL, indexURL, dir)
		expectIndexPage(t, h, slashURL, wantOfSlash, dir)
		if slashless := prefix + escapedDir; slashless != "" {
			expectIndexPage(t, h, slashless, indexURL, dir)
		}
	}
}

type response struct {
	method, target string
	code           int
	location       string
	body           string // not checked when empty.
}

func expectResponses(t *testing.T, h http.Handler, tests []response) {
	t.Helper()

	for _, tt := range tests {
		rec := do(h, tt.method, tt.target, nil)
		if got := rec.Header().Get("Location"); rec.Code != tt.code || got != tt.location {
			t.Errorf("%s %s: got %d Location=%q, want %d Location=%q", tt.method, tt.target, rec.Code, got, tt.code, tt.location)
			continue
		}
		if tt.body != "" && rec.Body.String() != tt.body {
			t.Errorf("%s %s: body %q, want %q", tt.method, tt.target, rec.Body.String(), tt.body)
		}
	}
}

// Without RedirectToIndex nothing changes: the index file is sent back to its
// directory and the directory serves the index at its own URL, so a relative
// link of a nested index page resolves against the parent path.
func TestFileServerIndexRedirectsByDefault(t *testing.T) {
	root := docsSite(t, "index.html")

	// What the file server itself answers, whatever the router does with a trailing slash.
	same := []response{
		{http.MethodGet, "/static/doc/index.html", 301, "./", ""},
		{http.MethodHead, "/static/doc/index.html", 301, "./", ""},
		{http.MethodGet, "/static/doc/index.html?x=1", 301, "./?x=1", ""},
		{http.MethodGet, "/static/doc/guide/index.html", 301, "./", ""},
		{http.MethodGet, "/static/index.html", 301, "./", ""},
		{http.MethodGet, "/static/doc", 200, "", indexBody("/doc")},
		{http.MethodGet, "/static/doc?x=1", 200, "", indexBody("/doc")},
		{http.MethodHead, "/static/doc", 200, "", ""},
		{http.MethodGet, "/static/doc/guide", 200, "", indexBody("/doc/guide")},
		{http.MethodGet, "/static/my%20docs", 200, "", indexBody("/my docs")},
		{http.MethodGet, "/static", 200, "", indexBody("")},
		{http.MethodGet, "/static/doc/page2.html", 200, "", "page2 of [/doc]"},
		{http.MethodGet, "/static/assets", 404, "", ""},
		{http.MethodGet, "/static/missing.html", 404, "", ""},
	}

	for _, mode := range routerModes {
		t.Run(mode.name, func(t *testing.T) {
			h := newDocsApp(t, mode, func(app *iris.Application) {
				app.Router("/static", fileserver.New(root, fileserver.DefaultOptions))
			})
			expectResponses(t, h, same)

			// The trailing slash of a directory.
			rec := do(h, http.MethodGet, "/static/doc/", nil)
			if mode.name == "PathCorrection" {
				if loc := rec.Header().Get("Location"); rec.Code != 301 || resolve(t, "/static/doc/", loc) != "/static/doc" {
					t.Errorf("GET /static/doc/: got %d Location=%q, want the 301 of the router to /static/doc", rec.Code, loc)
				}
			} else if rec.Code != 200 || rec.Body.String() != indexBody("/doc") {
				t.Errorf("GET /static/doc/: got %d %q, want the index of [/doc]", rec.Code, rec.Body.String())
			}
		})
	}

	// The case of the issue: the visitor of doc/index.html ends at /static/doc,
	// where css/app.css is /static/css/app.css, the stylesheet of the parent.
	h := newDocsApp(t, routerModes[0], func(app *iris.Application) {
		app.Router("/static", fileserver.New(root, fileserver.DefaultOptions))
	})
	hops, _ := browse(t, h, "/static/doc/index.html", nil)
	if last := hops[len(hops)-1]; last.URL != "/static/doc" || last.Code != 200 {
		t.Fatalf("ended at %s (%d), want /static/doc (200): %+v", last.URL, last.Code, hops)
	}
	if got := do(h, http.MethodGet, resolve(t, "/static/doc", "css/app.css"), nil).Body.String(); got != "/* css of [] */" {
		t.Fatalf("css/app.css from /static/doc: got %q, want the stylesheet of the parent directory", got)
	}
}

// With RedirectToIndex every URL of a directory index ends, with no redirect
// loop, on a URL from which the relative links of the page load.
func TestFileServerRedirectToIndex(t *testing.T) {
	root := docsSite(t, "index.html")
	opts := fileserver.DefaultOptions
	opts.RedirectToIndex = true

	for _, mode := range routerModes {
		t.Run(mode.name, func(t *testing.T) {
			h := newDocsApp(t, mode, func(app *iris.Application) {
				app.Router("/static", fileserver.New(root, opts))
			})

			expectIndexPages(t, h, mode, "/static", "index.html", "", "/doc", "/doc/guide", "/my%20docs")

			expectResponses(t, h, []response{
				// The index file is served at its own URL.
				{http.MethodGet, "/static/doc/index.html", 200, "", indexBody("/doc")},
				{http.MethodHead, "/static/doc/index.html", 200, "", ""},
				{http.MethodGet, "/static/doc/index.html?x=1", 200, "", indexBody("/doc")},
				{http.MethodGet, "/static/index.html", 200, "", indexBody("")},
				// A directory with no trailing slash is sent to its index file,
				// the mount path included. The query stays.
				{http.MethodGet, "/static/doc", 302, "./doc/index.html", ""},
				{http.MethodHead, "/static/doc", 302, "./doc/index.html", ""},
				{http.MethodGet, "/static/doc?x=1", 302, "./doc/index.html?x=1", ""},
				{http.MethodGet, "/static/doc/guide", 302, "./guide/index.html", ""},
				{http.MethodGet, "/static/my%20docs", 302, "./my%20docs/index.html", ""},
				{http.MethodGet, "/static", 302, "./static/index.html", ""},
				// Everything else is what it was.
				{http.MethodGet, "/static/doc/page2.html", 200, "", "page2 of [/doc]"},
				{http.MethodGet, "/static/assets", 404, "", ""},
				{http.MethodGet, "/static/assets/logo.txt", 200, "", "logo"},
				{http.MethodGet, "/static/missing.html", 404, "", ""},
			})

			// The index file asked with a trailing slash is not sent in circles.
			hops, rec := browse(t, h, "/static/doc/index.html/", nil)
			if rec.Code != 200 || rec.Body.String() != indexBody("/doc") {
				t.Errorf("/static/doc/index.html/: got %d %q, want the index of [/doc]: %+v", rec.Code, rec.Body.String(), hops)
			}
		})
	}
}

func TestFileServerRedirectToIndexCustomIndexName(t *testing.T) {
	root := docsSite(t, "home.html")

	for _, mode := range routerModes {
		t.Run(mode.name, func(t *testing.T) {
			h := newDocsApp(t, mode, func(app *iris.Application) {
				app.Router("/static", fileserver.New(root, fileserver.Options{IndexName: "home.html", RedirectToIndex: true}))
			})

			expectIndexPages(t, h, mode, "/static", "home.html", "", "/doc", "/doc/guide")
			expectResponses(t, h, []response{
				{http.MethodGet, "/static/doc", 302, "./doc/home.html", ""},
				{http.MethodGet, "/static/doc/home.html", 200, "", indexBody("/doc")},
				{http.MethodGet, "/static/doc/index.html", 404, "", ""},
			})
		})
	}
}

// The index name is written into the Location of the directory redirect, so
// a name with a character that a URL gives a meaning to (a space, '#', '%')
// is escaped there. Unescaped, "index#1.html" sends the client to "index"
// and "100%.html" is not a URL at all.
func TestFileServerRedirectToIndexEscapedIndexName(t *testing.T) {
	for indexName, escaped := range map[string]string{
		"my index.html": "my%20index.html",
		"index#1.html":  "index%231.html",
		"100%.html":     "100%25.html",
	} {
		root := docsSite(t, indexName)

		for _, mode := range routerModes {
			t.Run(indexName+"/"+mode.name, func(t *testing.T) {
				h := newDocsApp(t, mode, func(app *iris.Application) {
					app.Router("/static", fileserver.New(root, fileserver.Options{
						IndexName:       indexName,
						RedirectToIndex: true,
						PushTargets:     map[string][]string{"/doc": {"css/app.css"}},
					}))
				})

				expectResponses(t, h, []response{
					{http.MethodGet, "/static/doc", 302, "./doc/" + escaped, ""},
					{http.MethodGet, "/static/doc/" + escaped, 200, "", indexBody("/doc")},
				})
				expectIndexPages(t, h, mode, "/static", escaped, "", "/doc", "/doc/guide")

				// The push targets of the page are still those of its directory.
				rec := &pushRecorder{ResponseRecorder: nethttptest.NewRecorder()}
				h.ServeHTTP(rec, nethttptest.NewRequest(http.MethodGet, "/static/doc/"+escaped, nil))
				if want := []string{"/static/doc/css/app.css"}; !slices.Equal(rec.pushed, want) {
					t.Errorf("GET /static/doc/%s: pushed %v, want %v", escaped, rec.pushed, want)
				}
			})
		}
	}
}

// A directory that has no index file is not redirected: it is listed at the
// URL it was asked for, or it is a 404 without ShowList, as before.
func TestFileServerRedirectToIndexShowList(t *testing.T) {
	root := docsSite(t, "index.html")

	for _, mode := range routerModes {
		t.Run(mode.name, func(t *testing.T) {
			h := newDocsApp(t, mode, func(app *iris.Application) {
				app.Router("/static", fileserver.New(root, fileserver.Options{
					IndexName:       "index.html",
					RedirectToIndex: true,
					ShowList:        true,
					DirList:         fileserver.DirList,
				}))
			})

			for _, start := range []string{"/static/assets", "/static/assets/", "/static/doc/css"} {
				hops, rec := browse(t, h, start, nil)
				last := hops[len(hops)-1]
				if last.Code != 200 || strings.Contains(last.URL, "index.html") {
					t.Errorf("%s: ended at %s (%d), want the listing of the directory: %+v", start, last.URL, last.Code, hops)
				}
				if body := rec.Body.String(); !strings.Contains(body, "Current Directory") {
					t.Errorf("%s: not a listing: %q", start, body)
				}
			}

			if body := do(h, http.MethodGet, "/static/assets", nil).Body.String(); !strings.Contains(body, `href="/static/assets/logo.txt"`) {
				t.Errorf("the listing of /static/assets does not link to its file: %q", body)
			}

			expectIndexPages(t, h, mode, "/static", "index.html", "", "/doc", "/doc/guide")
		})
	}
}

// A file server mounted on "/" serves its root index at "/": that URL ends
// with a slash, the relative links resolve from it.
func TestFileServerRedirectToIndexMountedOnRoot(t *testing.T) {
	root := docsSite(t, "index.html")

	for _, mode := range routerModes {
		t.Run(mode.name, func(t *testing.T) {
			h := newDocsApp(t, mode, func(app *iris.Application) {
				app.Router("/", fileserver.New(root, fileserver.Options{IndexName: "index.html", RedirectToIndex: true}))
			})

			expectIndexPages(t, h, mode, "", "index.html", "", "/doc", "/doc/guide")
			expectResponses(t, h, []response{
				{http.MethodGet, "/", 200, "", indexBody("")},
				{http.MethodGet, "/index.html", 200, "", indexBody("")},
				{http.MethodGet, "/doc", 302, "./doc/index.html", ""},
			})
		})
	}
}

// The redirect is relative, so it follows any prefix the file server is under.
func TestFileServerRedirectToIndexUnderParty(t *testing.T) {
	root := docsSite(t, "index.html")

	for _, mode := range routerModes {
		t.Run(mode.name, func(t *testing.T) {
			h := newDocsApp(t, mode, func(app *iris.Application) {
				app.Router("/v1", func(r iris.Router) {
					r.Router("/docs", func(r iris.Router) {
						r.Router("/site", fileserver.New(root, fileserver.Options{IndexName: "index.html", RedirectToIndex: true}))
					})
				})
			})

			expectIndexPages(t, h, mode, "/v1/docs/site", "index.html", "", "/doc", "/doc/guide")
			expectResponses(t, h, []response{
				{http.MethodGet, "/v1/docs/site", 302, "./site/index.html", ""},
				{http.MethodGet, "/v1/docs/site/doc", 302, "./doc/index.html", ""},
			})
		})
	}
}

// RedirectToIndex does nothing in SPA mode: the client-side router owns the
// URLs, and the index is served on paths of any depth.
func TestFileServerRedirectToIndexSPA(t *testing.T) {
	root := docsSite(t, "index.html")

	for _, mode := range routerModes {
		t.Run(mode.name, func(t *testing.T) {
			var handlers [2]http.Handler
			for i, redirectToIndex := range []bool{false, true} {
				handlers[i] = newDocsApp(t, mode, func(app *iris.Application) {
					app.Router("/app", fileserver.New(root, fileserver.Options{
						IndexName:       "index.html",
						SPA:             true,
						RedirectToIndex: redirectToIndex,
					}))
				})
			}

			for _, method := range []string{http.MethodGet, http.MethodHead} {
				for _, target := range []string{
					"/app", "/app/", "/app/index.html", "/app/index.html?x=1",
					"/app/doc", "/app/doc/", "/app/doc/index.html",
					"/app/some/client/route", "/app/doc/page2.html", "/app/assets",
				} {
					off, on := do(handlers[0], method, target, nil), do(handlers[1], method, target, nil)
					if off.Code != on.Code || off.Header().Get("Location") != on.Header().Get("Location") || off.Body.String() != on.Body.String() {
						t.Errorf("%s %s: %d Location=%q without RedirectToIndex, %d Location=%q with it",
							method, target, off.Code, off.Header().Get("Location"), on.Code, on.Header().Get("Location"))
					}
				}
			}

			expectResponses(t, handlers[1], []response{
				{http.MethodGet, "/app", 200, "", indexBody("")},
				{http.MethodGet, "/app/index.html", 301, "./", ""},
				{http.MethodGet, "/app/doc", 200, "", indexBody("/doc")},
				{http.MethodGet, "/app/some/client/route", 200, "", indexBody("")},
			})
		})
	}
}

type pushRecorder struct {
	*nethttptest.ResponseRecorder
	pushed []string
}

func (p *pushRecorder) Push(target string, _ *http.PushOptions) error {
	p.pushed = append(p.pushed, target)
	return nil
}

// PushTargets and PushTargetsRegexp are keyed by the directory of an index
// page. That stays true when the page is served at the URL of its index file.
func TestFileServerRedirectToIndexPushTargets(t *testing.T) {
	root := docsSite(t, "index.html")

	newHandler := func(redirectToIndex bool) http.Handler {
		return newDocsApp(t, routerModes[0], func(app *iris.Application) {
			app.Router("/static", fileserver.New(root, fileserver.Options{
				IndexName:       "index.html",
				RedirectToIndex: redirectToIndex,
				PushTargets: map[string][]string{
					"/":    {"css/app.css"},
					"/doc": {"css/app.css", "/static/doc/page2.html"},
				},
				PushTargetsRegexp: map[string]*regexp.Regexp{
					"/doc/guide": regexp.MustCompile(`\.css$`),
				},
			}))
		})
	}

	pushedBy := func(h http.Handler, target string) (int, []string) {
		rec := &pushRecorder{ResponseRecorder: nethttptest.NewRecorder()}
		h.ServeHTTP(rec, nethttptest.NewRequest(http.MethodGet, target, nil))
		return rec.Code, rec.pushed
	}

	want := map[string][]string{
		"":           {"/static/css/app.css"},
		"/doc":       {"/static/doc/css/app.css", "/static/doc/page2.html"},
		"/doc/guide": {"/static/doc/guide/css/app.css"},
	}

	off, on := newHandler(false), newHandler(true)
	for dir, targets := range want {
		// As before: the page of the directory URL pushes.
		if code, pushed := pushedBy(off, "/static"+dir); code != 200 || !slices.Equal(pushed, targets) {
			t.Errorf("without RedirectToIndex, GET /static%s: %d pushed %v, want 200 %v", dir, code, pushed, targets)
		}

		// With it: the redirect pushes nothing, the page of the index URL pushes the same.
		if code, pushed := pushedBy(on, "/static"+dir); code != 302 || len(pushed) != 0 {
			t.Errorf("with RedirectToIndex, GET /static%s: %d pushed %v, want 302 and no push", dir, code, pushed)
		}
		if code, pushed := pushedBy(on, "/static"+dir+"/index.html"); code != 200 || !slices.Equal(pushed, targets) {
			t.Errorf("with RedirectToIndex, GET /static%s/index.html: %d pushed %v, want 200 %v", dir, code, pushed, targets)
		}
	}
}

func gunzip(t *testing.T, rec *nethttptest.ResponseRecorder) string {
	t.Helper()

	if enc := rec.Header().Get("Content-Encoding"); enc != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", enc)
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The in-memory cache with its pre-compressed files, and the compress
// middleware in front of the file server: same redirect, same page.
func TestFileServerRedirectToIndexCompressed(t *testing.T) {
	root := docsSite(t, "index.html")
	acceptGzip := http.Header{"Accept-Encoding": {"gzip"}}

	registrations := map[string]func(app *iris.Application){
		"Cache": func(app *iris.Application) {
			app.Router("/static", fileserver.New(root, fileserver.Options{
				IndexName:       "index.html",
				RedirectToIndex: true,
				Cache:           fileserver.DirCacheOptions{Enable: true, CompressMinSize: 1, Encodings: []string{"gzip"}},
			}))
		},
		"CompressMiddleware": func(app *iris.Application) {
			app.Use(compress.Handler)
			app.Router("/static", fileserver.New(root, fileserver.Options{IndexName: "index.html", RedirectToIndex: true}))
		},
	}

	for name, register := range registrations {
		for _, mode := range routerModes {
			t.Run(name+"/"+mode.name, func(t *testing.T) {
				h := newDocsApp(t, mode, register)

				if rec := do(h, http.MethodGet, "/static/doc", acceptGzip); rec.Code != 302 || rec.Header().Get("Location") != "./doc/index.html" {
					t.Fatalf("GET /static/doc: got %d Location=%q, want 302 ./doc/index.html", rec.Code, rec.Header().Get("Location"))
				}

				for _, start := range []string{"/static/doc", "/static/doc/index.html", "/static", "/static/index.html"} {
					hops, rec := browse(t, h, start, acceptGzip)
					last := hops[len(hops)-1]
					dir := strings.TrimPrefix(strings.TrimSuffix(start, "/index.html"), "/static")
					if want := "/static" + dir + "/index.html"; last.Code != 200 || last.URL != want {
						t.Fatalf("%s: ended at %s (%d), want %s (200): %+v", start, last.URL, last.Code, want, hops)
					}
					if got := gunzip(t, rec); got != indexBody(dir) {
						t.Fatalf("%s: body %q, want the index of [%s]", start, got, dir)
					}
				}
			})
		}
	}
}

// The redirect is built from the path the client sent, RequestURI. A request
// that carries none (a handler called by hand, not by a server) or carries an
// absolute URL there (a request sent to a proxy) is served in place, as before.
func TestFileServerRedirectToIndexNoRequestURI(t *testing.T) {
	root := docsSite(t, "index.html")
	h := newDocsApp(t, routerModes[0], func(app *iris.Application) {
		app.Router("/static", fileserver.New(root, fileserver.Options{IndexName: "index.html", RedirectToIndex: true}))
	})

	byHand, err := http.NewRequest(http.MethodGet, "http://example.com/static/doc", nil)
	if err != nil {
		t.Fatal(err)
	}
	absolute := nethttptest.NewRequest(http.MethodGet, "http://example.com/static/doc", nil)

	for _, req := range []*http.Request{byHand, absolute} {
		rec := nethttptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 200 || rec.Body.String() != indexBody("/doc") {
			t.Fatalf("RequestURI %q: got %d %q, want the index of [/doc] in place", req.RequestURI, rec.Code, rec.Body.String())
		}
	}
}
