package cors

import (
	"slices"
	"strings"
)

// MatchPattern checks if the given origin matches the given pattern.
// It returns true if the pattern is "*" or if the origin matches the pattern.
// The pattern can contain '*' wildcards:
// - "https://example.com" matches exactly that origin
// - "https://*.example.com" matches any subdomain of example.com but no the domain itself
// - "https://www.example.*" matches any TLD for www.example
// - "https://www.example*" matches any domain starting with "https://www.example"
// - "*" matches any origin.
func MatchPattern(origin, pattern string) bool {
	return matchOriginPattern(origin, pattern)
}

// MergeOriginPatterns returns a new slice containing only the unique origins,
// removing any origin that is covered by a wider wildcard pattern.
// For example:
//   - ["https://www.example.com", "https://*.example.com"] => ["https://*.example.com"]
//   - ["https://foo.example.com", "https://*.example.com", "*"] => ["*"]
//
// but "https://*.example.com" does not match "https://example.com" as the
// wildcard accepts a subdomain, so they both kept.
func MergeOriginPatterns(origins []string) []string {
	// Remove duplicates first.
	seen := make(map[string]struct{}, len(origins))
	unique := make([]string, 0, len(origins))
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if _, ok := seen[o]; !ok {
			seen[o] = struct{}{}
			unique = append(unique, o)
		}
	}

	// If "*" is present, return only ["*"].
	if slices.Contains(unique, "*") {
		return []string{"*"}
	}

	// Remove any origin that is covered by a wider pattern.
	result := make([]string, 0, len(unique))
outer:
	for i, pattern := range unique {
		for j, other := range unique {
			if i == j {
				continue
			}
			if matchOriginPattern(pattern, other) && pattern != other {
				// pattern is covered by other.
				continue outer
			}
		}
		result = append(result, pattern)
	}
	return result
}

// matchOriginPattern reports whether "origin" matches "pattern".
//
// Two rules bound what a '*' may swallow:
//
//   - a '*' never matches '/', so a wildcard cannot cross out of the authority
//     into a path;
//   - a '*' standing in a label position, i.e. directly after a '.', matches
//     exactly ONE label, so "https://www.example.*" means "any TLD for
//     www.example" as documented.
//
// Note(@kataras): the second rule is the one that mattered. The previous
// implementation skipped its suffix check whenever the pattern ended with '*',
// so only the prefix had to match and "https://www.example.*" accepted
// "https://www.example.evil.com" - any attacker-registered domain, which with
// credentials enabled is a cross-origin read of authenticated data.
func matchOriginPattern(origin, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if !strings.Contains(pattern, "*") {
		return origin == pattern
	}

	return matchWildcardPattern(origin, pattern, false)
}

// matchWildcardPattern matches "origin" against "pattern", where "labelOnly"
// reports whether a leading wildcard of the pattern stands in a label position.
func matchWildcardPattern(origin, pattern string, labelOnly bool) bool {
	star := strings.IndexByte(pattern, '*')
	if star < 0 {
		return origin == pattern
	}

	prefix := pattern[:star]
	if !strings.HasPrefix(origin, prefix) {
		return false
	}
	origin = origin[len(prefix):]

	if prefix != "" {
		labelOnly = strings.HasSuffix(prefix, ".")
	}

	rest := pattern[star+1:]
	if rest == "" {
		// A trailing wildcard must consume at least one character.
		return origin != "" && wildcardCanConsume(origin, labelOnly)
	}

	// Try every split point: the wildcard consumes origin[:i], the rest of the
	// pattern has to match what follows.
	for i := 1; i <= len(origin); i++ {
		if !wildcardCanConsume(origin[:i], labelOnly) {
			break
		}
		if matchWildcardPattern(origin[i:], rest, labelOnly) {
			return true
		}
	}

	return false
}

// wildcardCanConsume reports whether a '*' is allowed to swallow "s".
func wildcardCanConsume(s string, labelOnly bool) bool {
	if strings.ContainsRune(s, '/') {
		return false
	}

	if labelOnly && strings.ContainsRune(s, '.') {
		return false
	}

	return true
}
