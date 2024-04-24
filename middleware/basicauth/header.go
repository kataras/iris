package basicauth

import (
	"encoding/base64"
	"strings"
)

const (
	spaceChar            = ' '
	colonChar            = ':'
	colonLiteral         = string(colonChar)
	basicLiteral         = "Basic"
	basicSpaceLiteral    = "Basic "
	basicSpaceLiteralLen = len(basicSpaceLiteral)
)

// The username and password are combined with a single colon (:).
// This means that the username itself cannot contain a colon.
// URL encoding (e.g. https://Aladdin:OpenSesame@www.example.com/index.html)
// has been deprecated by rfc3986.
func encodeHeader(username, password string) (string, bool) {
	if strings.Contains(username, colonLiteral) || strings.Contains(password, colonLiteral) {
		return "", false
	}
	fullUser := []byte(username + colonLiteral + password)
	header := basicSpaceLiteral + base64.StdEncoding.EncodeToString(fullUser)

	return header, true
}

// Like net/http.parseBasicAuth
func decodeHeader(header string) (fullUser, username, password string, ok bool) {
	if len(header) < basicSpaceLiteralLen || !strings.EqualFold(header[:basicSpaceLiteralLen], basicSpaceLiteral) {
		return
	}

	c, err := base64.StdEncoding.DecodeString(header[basicSpaceLiteralLen:])
	if err != nil {
		return
	}

	cs := string(c)
	s := strings.IndexByte(cs, colonChar)
	if s < 0 {
		return
	}
	return cs, cs[:s], cs[s+1:], true

	/*
		for i := 0; i < n; i++ {
			if header[i] == spaceChar {
				prefix := header[:i]
				if prefix != basicLiteral {
					return
				}

				if n <= i+1 {
					return
				}

				decodedFullUser, err := base64.RawStdEncoding.DecodeString(header[i+1:])
				if err != nil {
					return
				}

				fullUser = string(decodedFullUser)
				break
			}
		}

		n = len(fullUser)
		for i := n - 1; i > -1; i-- {
			if fullUser[i] == colonChar {
				username = fullUser[:i]
				password = fullUser[i+1:]

				if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
					ok = false
				} else {
					ok = true
				}

				return
			}
		}

		return*/
}
