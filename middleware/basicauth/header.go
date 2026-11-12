package basicauth

import (
	"encoding/base64"
	"strings"
)

const (
	colonLiteral         = ":"
	basicSpaceLiteral    = "Basic "
	basicSpaceLiteralLen = len(basicSpaceLiteral)
)

// encodeHeader builds an Authorization header value for the given credentials.
//
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

// decodeHeader parses an Authorization header value of the "Basic" scheme.
// It returns the decoded "username:password" pair as a whole and split,
// like net/http's parseBasicAuth does. Only the first colon separates the two,
// so a password may contain colons while a username may not.
func decodeHeader(header string) (fullUser, username, password string, ok bool) {
	if len(header) < basicSpaceLiteralLen || !strings.EqualFold(header[:basicSpaceLiteralLen], basicSpaceLiteral) {
		return
	}

	c, err := base64.StdEncoding.DecodeString(header[basicSpaceLiteralLen:])
	if err != nil {
		return
	}

	fullUser = string(c)
	username, password, ok = strings.Cut(fullUser, colonLiteral)
	if !ok {
		return "", "", "", false
	}

	return fullUser, username, password, true
}
