package context

import (
	"testing"
)

// TestContextOptions_MimeToContentType checks the MimeToContentType method.
func TestContextOptions_MimeToContentType(t *testing.T) {
	opts := &ContextOptions{Charset: "iso-8859-1"}
	contentType := opts.MimeToContentType("text/html")
	expectedContentType := "text/html; charset=iso-8859-1"

	if contentType != expectedContentType {
		t.Errorf("MimeToContentType returned %s, expected %s", contentType, expectedContentType)
	}
}
