package sqlx

import "strings"

// snakeCase converts a given string to a friendly snake case, e.g.
// - userId to user_id
// - ID     to id
// - ProviderAPIKey to provider_api_key
// - Option to option
func snakeCase(camel string) string {
	var (
		b            strings.Builder
		prevWasUpper bool
	)

	for i, c := range camel {
		if isUppercase(c) { // it's upper.
			if b.Len() > 0 && !prevWasUpper { // it's not the first and the previous was not uppercased too (e.g  "ID").
				b.WriteRune('_')
			} else { // check for XxxAPIKey, it should be written as xxx_api_key.
				next := i + 1
				if next > 1 && len(camel)-1 > next {
					if !isUppercase(rune(camel[next])) {
						b.WriteRune('_')
					}
				}
			}

			b.WriteRune(c - 'A' + 'a') // write its lowercase version.
			prevWasUpper = true
		} else {
			b.WriteRune(c) // write it as it is, it's already lowercased.
			prevWasUpper = false
		}
	}

	return b.String()
}

func isUppercase(c rune) bool {
	return 'A' <= c && c <= 'Z'
}
