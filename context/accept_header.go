package context

import "strings"

func negotiationMatch(in []string, priorities []string) string {
	// e.g.
	// match json:
	// 	in: text/html, application/json
	// 	priorities: application/json
	// not match:
	// 	in: text/html, application/json
	// 	priorities: text/xml
	// match html:
	// 	in: text/html, application/json
	// 	priorities: */*
	// not match:
	// 	in: application/json
	// 	priorities: text/xml
	// match json:
	// 	in: text/html, application/*
	// 	priorities: application/json

	if len(priorities) == 0 {
		return ""
	}

	if len(in) == 0 {
		return priorities[0]
	}

	for _, accepted := range in {
		for _, p := range priorities {
			// wildcard is */* or text/* and etc.
			// so loop through each char.
			for i, n := 0, len(accepted); i < n; i++ {
				if accepted[i] != p[i] {
					break
				}

				if accepted[i] == '*' || p[i] == '*' {
					return p
				}

				if i == n-1 {
					return p
				}
			}
		}
	}

	return ""
}

func negotiateAcceptHeader(in []string, offers []string, bestOffer string) string {
	if bestOffer == "" {
		bestOffer = IDENTITY
	}

	bestQ := -1.0
	specs := parseAccept(in)
	for _, offer := range offers {
		for _, spec := range specs {
			if spec.Q > bestQ &&
				(spec.Value == "*" || spec.Value == offer) {
				bestQ = spec.Q
				bestOffer = offer
			}
		}
	}
	if bestQ == 0 {
		bestOffer = ""
	}
	return bestOffer
}

// acceptSpec describes an Accept* header.
type acceptSpec struct {
	Value string
	Q     float64
}

// parseAccept parses Accept* headers.
func parseAccept(in []string) (specs []acceptSpec) {
loop:
	for _, s := range in {
		for {
			var spec acceptSpec
			spec.Value, s = expectTokenSlash(s)
			if spec.Value == "" {
				continue loop
			}
			spec.Q = 1.0
			s = skipSpace(s)
			if strings.HasPrefix(s, ";") {
				s = skipSpace(s[1:])
				if !strings.HasPrefix(s, "q=") {
					continue loop
				}
				spec.Q, s = expectQuality(s[2:])
				if spec.Q < 0.0 {
					continue loop
				}
			}
			specs = append(specs, spec)
			s = skipSpace(s)
			if !strings.HasPrefix(s, ",") {
				continue loop
			}
			s = skipSpace(s[1:])
		}
	}
	return
}

func skipSpace(s string) (rest string) {
	i := 0
	for ; i < len(s); i++ {
		if octetTypes[s[i]]&isSpace == 0 {
			break
		}
	}
	return s[i:]
}

func expectTokenSlash(s string) (token, rest string) {
	i := 0
	for ; i < len(s); i++ {
		b := s[i]
		if (octetTypes[b]&isToken == 0) && b != '/' {
			break
		}
	}
	return s[:i], s[i:]
}

func expectQuality(s string) (q float64, rest string) {
	switch {
	case s == "":
		return -1, ""
	case s[0] == '0':
		q = 0
	case s[0] == '1':
		q = 1
	default:
		return -1, ""
	}
	s = s[1:]
	if !strings.HasPrefix(s, ".") {
		return q, s
	}
	s = s[1:]
	i := 0
	n := 0
	d := 1
	for ; i < len(s); i++ {
		b := s[i]
		if b < '0' || b > '9' {
			break
		}
		n = n*10 + int(b) - '0'
		d *= 10
	}
	return q + float64(n)/float64(d), s[i:]
}

// Octet types from RFC 2616.
var octetTypes [256]octetType

type octetType byte

const (
	isToken octetType = 1 << iota
	isSpace
)

func init() {
	// OCTET      = <any 8-bit sequence of data>
	// CHAR       = <any US-ASCII character (octets 0 - 127)>
	// CTL        = <any US-ASCII control character (octets 0 - 31) and DEL (127)>
	// CR         = <US-ASCII CR, carriage return (13)>
	// LF         = <US-ASCII LF, linefeed (10)>
	// SP         = <US-ASCII SP, space (32)>
	// HT         = <US-ASCII HT, horizontal-tab (9)>
	// <">        = <US-ASCII double-quote mark (34)>
	// CRLF       = CR LF
	// LWS        = [CRLF] 1*( SP | HT )
	// TEXT       = <any OCTET except CTLs, but including LWS>
	// separators = "(" | ")" | "<" | ">" | "@" | "," | ";" | ":" | "\" | <">
	//              | "/" | "[" | "]" | "?" | "=" | "{" | "}" | SP | HT
	// token      = 1*<any CHAR except CTLs or separators>
	// qdtext     = <any TEXT except <">>

	for c := 0; c < 256; c++ {
		var t octetType
		isCtl := c <= 31 || c == 127
		isChar := 0 <= c && c <= 127
		isSeparator := strings.ContainsRune(" \t\"(),/:;<=>?@[]\\{}", rune(c))
		if strings.ContainsRune(" \t\r\n", rune(c)) {
			t |= isSpace
		}
		if isChar && !isCtl && !isSeparator {
			t |= isToken
		}
		octetTypes[c] = t
	}
}
