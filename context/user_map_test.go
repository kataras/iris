package context

import (
	"testing"
)

// UserMap normalizes a capitalized key to its lowercase form. The check used
// unicode.IsTitle, which matches only the ~31 runes of the Unicode Lt category
// and is false for every ASCII letter, so the normalization never ran.
func TestUserMapReadsCapitalizedKeys(t *testing.T) {
	u := UserMap{
		"id":       "42",
		"username": "admin",
		"email":    "admin@example.com",
	}

	username, err := u.GetUsername()
	if err != nil {
		t.Fatalf("GetUsername() error = %v", err)
	}

	if username != "admin" {
		t.Errorf("GetUsername() = %q, want %q", username, "admin")
	}

	capitalized := UserMap{"Username": "admin", "Id": "42", "Email": "admin@example.com"}

	if got, err := capitalized.GetUsername(); err != nil || got != "admin" {
		t.Errorf("GetUsername() on a capitalized key = (%q, %v), want (%q, nil)", got, err, "admin")
	}

	if got, err := capitalized.GetID(); err != nil || got != "42" {
		t.Errorf("GetID() on a capitalized key = (%q, %v), want (%q, nil)", got, err, "42")
	}

	if got, err := capitalized.GetEmail(); err != nil || got != "admin@example.com" {
		t.Errorf("GetEmail() on a capitalized key = (%q, %v), want it resolved", got, err)
	}
}

func TestUserMapMissingKey(t *testing.T) {
	u := UserMap{"username": "admin"}

	if _, err := u.GetEmail(); err != ErrNotSupported {
		t.Errorf("GetEmail() on a missing key returned %v, want ErrNotSupported", err)
	}
}

// An empty key used to index key[0] and panic.
func TestUserMapEmptyKey(t *testing.T) {
	empty := UserMap{}
	if got := empty.val(""); got != nil {
		t.Errorf("val(\"\") on an empty map = %v, want nil", got)
	}

	u := UserMap{"": "unnamed"}
	if got := u.val(""); got != "unnamed" {
		t.Errorf("val(\"\") = %v, want %q", got, "unnamed")
	}
}

// The lookup must not be so loose that it resolves an unrelated field.
func TestUserMapDoesNotMatchAnUnrelatedKey(t *testing.T) {
	u := UserMap{"user_name": "admin"}

	if _, err := u.GetUsername(); err != ErrNotSupported {
		t.Errorf("GetUsername() matched \"user_name\"; error = %v, want ErrNotSupported", err)
	}
}
