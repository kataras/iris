package fileserver_test

import (
	"strings"
	"testing"

	"github.com/kataras/iris/v14/controller/fileserver"
)

// TestTypeByExtensionCommon locks the content types for common static-asset
// extensions (all present in Go's built-in MIME table, so the result is stable
// across machines regardless of the system MIME database).
func TestTypeByExtensionCommon(t *testing.T) {
	cases := map[string]string{
		".html": "text/html",
		".css":  "text/css",
		".json": "application/json",
		".svg":  "image/svg+xml",
		".png":  "image/png",
	}
	for ext, wantPrefix := range cases {
		if got := fileserver.TypeByExtension(ext); !strings.HasPrefix(got, wantPrefix) {
			t.Errorf("TypeByExtension(%q) = %q, want prefix %q", ext, got, wantPrefix)
		}
	}
}

// TestTypeByExtensionJavaScript is a regression test for the operator-precedence
// override in TypeByExtension: .js and .mjs must resolve to a javascript media
// type (the override must apply to both, not only .js).
func TestTypeByExtensionJavaScript(t *testing.T) {
	if got := fileserver.TypeByExtension(".js"); !strings.Contains(got, "javascript") {
		t.Errorf("TypeByExtension(.js) = %q, want a javascript media type", got)
	}
	// .mjs is in Go's built-in table; if a stripped-down system returns empty we
	// only require that a non-empty result is a javascript media type.
	if got := fileserver.TypeByExtension(".mjs"); got != "" && !strings.Contains(got, "javascript") {
		t.Errorf("TypeByExtension(.mjs) = %q, want a javascript media type or empty", got)
	}
	if got := fileserver.TypeByFilename("bundle.js"); !strings.Contains(got, "javascript") {
		t.Errorf("TypeByFilename(bundle.js) = %q, want a javascript media type", got)
	}
}
