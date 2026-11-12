package mimetypes_test

import (
	"mime"
	"strings"
	"testing"

	"github.com/kataras/iris/v14/internal/mimetypes"
)

func TestTypeByExtension(t *testing.T) {
	if got := mimetypes.TypeByExtension(".js"); !strings.Contains(got, "javascript") {
		t.Errorf("TypeByExtension(.js) = %q, want a javascript media type", got)
	}
	if got := mimetypes.TypeByExtension(".css"); !strings.HasPrefix(got, "text/css") {
		t.Errorf("TypeByExtension(.css) = %q, want text/css", got)
	}
	if got := mimetypes.TypeByFilename("bundle.js"); !strings.Contains(got, "javascript") {
		t.Errorf("TypeByFilename(bundle.js) = %q, want a javascript media type", got)
	}
}

// TestInitRegistersExtendedTable verifies the package init registered the
// extended MIME table into the standard library's mime package — this is the
// side-effect core/router relies on via its blank import.
func TestInitRegistersExtendedTable(t *testing.T) {
	// ".7z" is in the extended table but not in Go's built-in mime table.
	if got := mime.TypeByExtension(".7z"); !strings.Contains(got, "x-7z-compressed") {
		t.Errorf("stdlib mime.TypeByExtension(.7z) = %q, want it registered as application/x-7z-compressed", got)
	}
}
