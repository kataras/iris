package auth

import "testing"

// TestIsSafeRelativeRedirect locks the open-redirect guard used by SigninHandler.
func TestIsSafeRelativeRedirect(t *testing.T) {
	safe := []string{
		"/",
		"/dashboard",
		"/a/b/c?q=1",
		"/path#frag",
	}
	unsafe := []string{
		"",
		"//evil.com",            // protocol-relative
		"https://evil.com",      // absolute
		"http://evil.com",       // absolute
		"javascript:alert(1)",   // scheme
		"mailto:x@y.com",        // scheme
		"foo/bar",               // not rooted
		"/\\evil.com",           // backslash trick -> //evil.com
		"\\\\evil.com",          // backslashes
		"/foo\r\nSet-Cookie: x", // CRLF / header injection
		"/foo\x00bar",           // NUL byte
		"/foo\tbar",             // tab
		"/foo\nbar",             // LF
	}

	for _, s := range safe {
		if !isSafeRelativeRedirect(s) {
			t.Errorf("isSafeRelativeRedirect(%q) = false, want true", s)
		}
	}
	for _, s := range unsafe {
		if isSafeRelativeRedirect(s) {
			t.Errorf("isSafeRelativeRedirect(%q) = true, want false", s)
		}
	}
}
