package jwt

import (
	"strings"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/middleware/recorder"
)

// TokenExtractor is a function that takes a context as input and returns
// a token. An empty string should be returned if no token found
// without additional information.
type TokenExtractor func(*context.Context) string

// FromHeader is a token extractor.
// It reads the token from the Authorization request header of form:
// Authorization: "Bearer {token}".
func FromHeader(ctx *context.Context) string {
	authHeader := ctx.Header("Authorization")
	if authHeader == "" {
		return ""
	}

	// pure check: authorization header format must be Bearer {token}
	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return ""
	}

	return authHeaderParts[1]
}

// FromQuery is a token extractor.
// It reads the token from the "token" url query parameter.
//
// Security note: tokens placed in the URL query string tend to leak into access
// logs, proxy logs, browser history and Referer headers. As of v14 FromQuery is
// NOT one of the default extractors; opt in explicitly with Verifier.WithQueryToken
// only when header-based extraction is not possible.
func FromQuery(ctx *context.Context) string {
	return ctx.Query().Get("token")
}

// FromJSON is a token extractor.
// Reads a json request body and extracts the json based on the given field.
// The request content-type should contain the: application/json header value, otherwise
// this method will not try to read and consume the body.
func FromJSON(jsonKey string) TokenExtractor {
	return func(ctx *context.Context) string {
		if !ctx.IsMediaType(context.ContentJSONHeaderValue) {
			return ""
		}

		var m context.Map
		_ = recorder.RecordRequestBody(ctx)
		if err := ctx.ReadJSON(&m); err != nil {
			return ""
		}

		if m == nil {
			return ""
		}

		v, ok := m[jsonKey]
		if !ok {
			return ""
		}

		tok, ok := v.(string)
		if !ok {
			return ""
		}

		return tok
	}
}
