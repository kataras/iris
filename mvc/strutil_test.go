package mvc

import (
	"testing"
)

func TestFindCtrlWords(t *testing.T) {
	var tests = map[string][]string{
		"UserController":            {"user"},
		"UserPostController":        {"user", "post"},
		"ProfileController":         {"profile"},
		"UserProfileController":     {"user", "profile"},
		"UserProfilePostController": {"user", "profile", "post"},
		"UserProfile":               {"user", "profile"},
		"Profile":                   {"profile"},
		"User":                      {"user"},
	}

	for ctrlName, expected := range tests {
		words := findCtrlWords(ctrlName)
		if len(expected) != len(words) {
			t.Fatalf("expected words and return don't have the same length: [%d] != [%d] | '%s' != '%s'",
				len(expected), len(words), expected, words)
		}
		for i, w := range words {
			if expected[i] != w {
				t.Fatalf("expected word is not equal with the return one: '%s' != '%s'", expected[i], w)
			}
		}
	}
}
