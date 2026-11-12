package context_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	iris "github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
)

func captureJSON(t *testing.T, v any, opts ...context.JSON) string {
	t.Helper()

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		if err := ctx.JSON(v, opts...); err != nil {
			t.Errorf("ctx.JSON: %v", err)
		}
	})
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	return rec.Body.String()
}

// TestWriteJSONDefaults pins the encoding/json/v2-based defaults: no HTML
// escaping, deterministic (sorted) map order, a trailing newline, empty
// collections instead of null.
func TestWriteJSONDefaults(t *testing.T) {
	body := captureJSON(t, iris.Map{"b": "<tag> & co", "a": 1})
	if want := "{\"a\":1,\"b\":\"<tag> & co\"}\n"; body != want {
		t.Fatalf("got %q, want %q", body, want)
	}

	type collections struct {
		S []string       `json:"s"`
		M map[string]int `json:"m"`
	}
	body = captureJSON(t, collections{})
	if want := "{\"s\":[],\"m\":{}}\n"; body != want {
		t.Fatalf("nil collections: got %q, want %q", body, want)
	}
}

// TestWriteJSONEscapeHTML covers the opt-in that restores the old
// encoding/json escaping, and the bug the old byte-rewriting implementation
// had: a string containing a LITERAL \u003c was corrupted into invalid JSON
// when un-escaping was combined with Secure/ASCII/Prefix.
func TestWriteJSONEscapeHTML(t *testing.T) {
	body := captureJSON(t, iris.Map{"h": "<b>&</b>"}, context.JSON{EscapeHTML: true})
	if want := "{\"h\":\"\\u003cb\\u003e\\u0026\\u003c/b\\u003e\"}\n"; body != want {
		t.Fatalf("got %q, want %q", body, want)
	}

	// The six characters \u003c as literal content, with options that used
	// to trigger the byte-rewrite pass. The output must stay parseable and
	// round-trip the exact input.
	const literal = `literal \u003c stays`
	body = captureJSON(t, iris.Map{"v": literal}, context.JSON{Prefix: ")]}',\n"})
	payload := strings.TrimPrefix(body, ")]}',\n")
	var decoded map[string]string
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%q", err, body)
	}
	if decoded["v"] != literal {
		t.Fatalf("literal backslash-u content corrupted: got %q, want %q", decoded["v"], literal)
	}
}

// TestWriteJSONEscapeNonASCII pins correct \uXXXX escaping including UTF-16 surrogate
// pairs: the old implementation emitted an invalid five-hex escape (\u1f600)
// for astral-plane runes.
func TestWriteJSONEscapeNonASCII(t *testing.T) {
	body := captureJSON(t, iris.Map{"emoji": "😀", "greek": "γ"}, context.JSON{EscapeNonASCII: true})
	if want := "{\"emoji\":\"\\ud83d\\ude00\",\"greek\":\"\\u03b3\"}\n"; body != want {
		t.Fatalf("got %q, want %q", body, want)
	}

	var decoded map[string]string
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("ASCII output is not valid JSON: %v\n%q", err, body)
	}
	if decoded["emoji"] != "😀" {
		t.Fatalf("surrogate pair did not round-trip: %q", decoded["emoji"])
	}
}

// TestWriteJSONArrayPrefix pins the anti-hijacking prefix for top-level arrays,
// with and without indentation, and its absence for objects.
func TestWriteJSONArrayPrefix(t *testing.T) {
	if body := captureJSON(t, []string{"a", "b"}, context.JSON{ArrayPrefix: true}); body != "while(1);[\"a\",\"b\"]\n" {
		t.Fatalf("array: got %q", body)
	}
	if body := captureJSON(t, []string{"a"}, context.JSON{ArrayPrefix: true, Indent: "  "}); !strings.HasPrefix(body, "while(1);[") {
		t.Fatalf("indented array: got %q", body)
	}
	if body := captureJSON(t, iris.Map{"a": 1}, context.JSON{ArrayPrefix: true}); strings.HasPrefix(body, "while(1);") {
		t.Fatalf("object must not get the array prefix: got %q", body)
	}
}

// TestWriteJSONDuplicateNamesOnOutput pins that duplicate object members are
// rejected on output like they are on input: a body the framework writes must
// be readable by its own client. (rest/types.Merged merges overlapping keys
// instead of emitting both.)
func TestWriteJSONDuplicateNamesOnOutput(t *testing.T) {
	app := iris.New()
	app.Logger().SetLevel("disable")
	var jsonErr error
	app.Get("/", func(ctx iris.Context) {
		jsonErr = ctx.JSON(json.RawMessage(`{"a":1,"a":2}`))
	})
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if jsonErr == nil {
		t.Fatal("ctx.JSON accepted duplicate object names on output")
	}
	if body := rec.Body.String(); strings.Contains(body, `"a":2`) {
		t.Fatalf("the duplicate document reached the wire: %q", body)
	}
}

// TestWriteJSONInvalidUTF8Output pins the lenient output policy: a stray
// byte in a string value is mangled to U+FFFD, never a 500.
func TestWriteJSONInvalidUTF8Output(t *testing.T) {
	body := captureJSON(t, iris.Map{"v": "a\xffb"})
	if !strings.Contains(body, "\ufffd") {
		t.Fatalf("expected the invalid byte to be replaced, got %q", body)
	}
}
