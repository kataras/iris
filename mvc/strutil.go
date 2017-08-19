package mvc

import (
	"strings"
	"unicode"
)

func findCtrlWords(ctrlName string) (w []string) {
	end := len(ctrlName)
	start := -1
	for i, n := 0, end; i < n; i++ {
		c := rune(ctrlName[i])
		if unicode.IsUpper(c) {
			// it doesn't count the last uppercase
			if start != -1 {
				end = i
				w = append(w, strings.ToLower(ctrlName[start:end]))
			}
			start = i
			continue
		}
		end = i + 1

	}

	// We can't omit the last name,  we have to take it.
	// because of controller names like
	// "UserProfile", we need to return "user", "profile"
	// if "UserController", we need to return "user"
	// if "User", we need to return "user".
	last := ctrlName[start:end]
	if last == ctrlSuffix {
		return
	}

	w = append(w, strings.ToLower(last))
	return
}
