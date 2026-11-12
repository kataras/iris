package fileserver_test

import (
	"net/http"
	nethttptest "net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
	"github.com/kataras/iris/v14/httptest"
)

// buildFileServerApp lays out:
//
//	<parent>/secret.txt        (OUTSIDE the served root)
//	<parent>/public/hello.txt
//	<parent>/public/app.js
//	<parent>/public/.env
//	<parent>/public/sub/nested.txt
//
// and serves <parent>/public at /static.
func buildFileServerApp(t *testing.T, opts fileserver.Options) *iris.Application {
	t.Helper()

	parent := t.TempDir()
	root := filepath.Join(parent, "public")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("hello.txt", "hello world")
	write("app.js", "export const x = 1;")
	write(".env", "SECRET=topsecret")
	write(filepath.Join("sub", "nested.txt"), "nested")
	write(filepath.Join("sub", ".secret"), "nested secret")
	write(filepath.Join(".git", "config"), "[core]")

	if err := os.WriteFile(filepath.Join(parent, "secret.txt"), []byte("TOP SECRET OUTSIDE ROOT"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := iris.New()
	app.Router("/static", fileserver.New(root, opts))
	return app
}

func TestFileServerServesFiles(t *testing.T) {
	app := buildFileServerApp(t, fileserver.Options{})
	e := httptest.New(t, app)

	e.GET("/static/hello.txt").Expect().Status(httptest.StatusOK).Body().IsEqual("hello world")
	e.GET("/static/sub/nested.txt").Expect().Status(httptest.StatusOK).Body().IsEqual("nested")
	e.GET("/static/missing.txt").Expect().Status(httptest.StatusNotFound)
}

func TestFileServerJavaScriptContentType(t *testing.T) {
	app := buildFileServerApp(t, fileserver.Options{})
	e := httptest.New(t, app)

	resp := e.GET("/static/app.js").Expect()
	resp.Status(httptest.StatusOK)
	if ct := resp.Raw().Header.Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Fatalf("app.js Content-Type = %q, want a javascript media type", ct)
	}
}

func TestFileServerDirectoryListingDisabledByDefault(t *testing.T) {
	app := buildFileServerApp(t, fileserver.Options{})
	e := httptest.New(t, app)
	// ShowList is false and there is no IndexName: a directory must not be listed.
	e.GET("/static/sub/").Expect().Status(httptest.StatusNotFound)
}

func TestFileServerDirectoryListingEnabled(t *testing.T) {
	app := buildFileServerApp(t, fileserver.Options{ShowList: true})
	e := httptest.New(t, app)
	body := e.GET("/static/sub/").Expect().Status(httptest.StatusOK).Body().Raw()
	if !strings.Contains(body, "nested.txt") {
		t.Fatalf("directory listing does not contain nested.txt:\n%s", body)
	}
}

// TestFileServerHiddenFilesServedByDefault documents the default behavior:
// without a validator, dotfiles such as .env ARE served. Use fileserver.BlockHidden
// (see TestFileServerBlockHidden) to deny them.
func TestFileServerHiddenFilesServedByDefault(t *testing.T) {
	app := buildFileServerApp(t, fileserver.Options{})
	e := httptest.New(t, app)
	e.GET("/static/.env").Expect().Status(httptest.StatusOK).Body().Contains("SECRET")
}

// TestFileServerBlockHidden verifies the BlockHidden AssetValidator denies hidden
// files while still serving regular ones.
func TestFileServerBlockHidden(t *testing.T) {
	app := buildFileServerApp(t, fileserver.Options{AssetValidator: fileserver.BlockHidden})
	e := httptest.New(t, app)

	e.GET("/static/hello.txt").Expect().Status(httptest.StatusOK).Body().IsEqual("hello world")
	e.GET("/static/.env").Expect().Status(httptest.StatusNotFound)
	// Nested hidden files and hidden directories are denied too.
	e.GET("/static/sub/.secret").Expect().Status(httptest.StatusNotFound)
	e.GET("/static/.git/config").Expect().Status(httptest.StatusNotFound)
}

// TestFileServerPathTraversalDoesNotEscapeRoot sends encoded traversal payloads
// (which survive client-side URL normalization and reach the server) and asserts
// that none of them can read a file located outside the served root.
func TestFileServerPathTraversalDoesNotEscapeRoot(t *testing.T) {
	app := buildFileServerApp(t, fileserver.Options{})
	e := httptest.New(t, app)

	for _, p := range []string{
		"/static/..%2fsecret.txt",
		"/static/%2e%2e%2fsecret.txt",
		"/static/%2e%2e/secret.txt",
		"/static/sub/..%2f..%2fsecret.txt",
	} {
		body := e.GET(p).Expect().Body().Raw()
		if strings.Contains(body, "TOP SECRET OUTSIDE ROOT") {
			t.Fatalf("path traversal %q leaked a file outside the served root", p)
		}
	}
}

// DirList renders into a text/html response. The heading used to write
// Request.RequestURI - the raw, client-supplied request line including the query
// string - unescaped, which is a reflected XSS in the application's own origin.
//
// The request is built by hand: an HTTP client would percent-encode the payload,
// but an attacker writes the request line directly and Go keeps RequestURI raw.
func TestFileServerDirListEscapesRequestURI(t *testing.T) {
	app := buildFileServerApp(t, fileserver.Options{
		ShowList: true,
		DirList:  fileserver.DirList,
	})
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	// No raw spaces: a request line cannot carry them, and slashes work just as
	// well as attribute separators in HTML.
	const payload = `<img/src=x/onerror=alert(document.cookie)>`

	req := nethttptest.NewRequest(http.MethodGet, "/static/sub?"+payload, nil)
	rec := nethttptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code == http.StatusMovedPermanently {
		// The listing lives on the trailing-slash form; follow the redirect once.
		req = nethttptest.NewRequest(http.MethodGet, rec.Header().Get("Location"), nil)
		rec = nethttptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, expected 200", rec.Code)
	}

	body := rec.Body.String()
	if strings.Contains(body, payload) {
		t.Fatalf("the request URI was written into the HTML response unescaped:\n%s", body)
	}
	if !strings.Contains(body, "&lt;img/src=x") {
		t.Fatalf("expected the request URI to be HTML-escaped:\n%s", body)
	}
}

// IsHidden must not panic on a FileInfo whose Sys() is not a Win32 attribute
// block: it is nil for embed.FS entries and this package's own type for a cached
// file system, and it runs at request time for every listed entry.
func TestFileServerDirListWithCachedFS(t *testing.T) {
	app := buildFileServerApp(t, fileserver.Options{
		ShowList: true,
		Cache:    fileserver.DirCacheOptions{Enable: true, Encodings: []string{"gzip"}},
	})
	e := httptest.New(t, app)

	e.GET("/static/sub/").Expect().Status(httptest.StatusOK).Body().Contains("nested.txt")
}
