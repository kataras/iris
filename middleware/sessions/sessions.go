// Package sessions provides typed HTTP sessions backed by a cookie, Redis or
// a SQL database.
//
// It has no single Handler entry point: build the settings with NewOptions,
// pass that Options value to a database constructor (NewCookieDatabase,
// NewRedisDatabase, NewPostgresDatabase) and register Handler[T](db, cookieName)
// or Single(db, cookieName). A session is read back with Get[T] or Load[T],
// because the type of the session data is the caller's, not the package's.
package sessions

import (
	"github.com/kataras/iris/v14/context"
)

const sessionsContextKeyPrefix = "iris.sessions."

// makeContextKey generates a unique context key for storing session data
// based on the provided cookie name.
func makeContextKey(cookieName string) string {
	return sessionsContextKeyPrefix + cookieName
}

// sessionContextValue holds the session and any associated error.
type sessionContextValue[T any] struct {
	session *Session[T]
	err     error
}

// storeSessionContextValue stores the session and error in the context
// using a generated context key based on the cookie name.
func storeSessionContextValue[T any](ctx *context.Context, cookieName string, s *Session[T], err error) {
	contextKey := makeContextKey(cookieName) // no s.name cuz it is nil on begin.
	ctx.Values().Set(contextKey, &sessionContextValue[T]{
		session: s,
		err:     err,
	})
}

// getSessionContextValue retrieves the sessionContextValue from the context
// using the generated context key based on the cookie name.
func getSessionContextValue[T any](ctx *context.Context, cookieName string) *sessionContextValue[T] {
	contextKey := makeContextKey(cookieName)
	contextValue, _ := ctx.ValueAs[*sessionContextValue[T]](contextKey)
	return contextValue
}

// removeSessionContextValue removes the sessionContextValue from the context
// using the generated context key based on the cookie name.
func removeSessionContextValue(ctx *context.Context, cookieName string) {
	contextKey := makeContextKey(cookieName)
	ctx.Values().Remove(contextKey)
}

// Load retrieves the session associated with the given cookie name from the context or database.
// If the session is not found in the context, it attempts to load it from the database.
// It stores the session and any error in the context for future retrieval.
func Load[T any](ctx *context.Context, db Database[T], cookieName string) (*Session[T], error) {
	if contextValue := getSessionContextValue[T](ctx, cookieName); contextValue != nil { // try to load it from the context.
		return contextValue.session, contextValue.err
	}

	var cookieValue string

	if cookie, err := ctx.Cookie(cookieName); err == nil { // only Name and Value are available.
		cookieValue = cookie.Value
	} // else not found, cookie not exist, database should create a new session when "cookieValue" is empty.
	session, err := db.Load(ctx, cookieName, cookieValue)
	if session != nil {
		session.database = db
		session.name = cookieName
	}

	storeSessionContextValue(ctx, cookieName, session, err)
	return session, err
}
