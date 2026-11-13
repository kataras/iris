package sessions

import (
	"encoding/gob"
	"time"

	"github.com/kataras/iris/v14/context"
)

// Database is an interface that defines methods for session management.
// It provides methods to get, load, and commit sessions.
//
// Example usage:
//
//	sessionDB := sessions.NewCookieDatabase[UserSession](sessionOptions)
//	[...]
//	func handler(ctx iris.Context) {
//	    session, err := sessionDB.Get(ctx, "my_session")
//	    if err != nil {
//	        // handle error
//	    }
//	    // use the session
//	    session.Data = UserSession{ // or session.Set(UserSession{...})
//	        ID:       "1",
//	        Username: "kataras",
//	    }
//	    if err := session.Commit(ctx); err != nil {
//	        // handle error
//	    }
//	}
//
// If you want to use the same cookie name for all sessions (most common usage),
// you can use the `Single` function. This function simplifies session retrieval
// by not requiring you to provide a cookie name each time you want to retrieve the session:
//
// Example usage:
//
//	sessionDB := sessions.Single(
//	    sessions.NewCookieDatabase[UserSession](sessionOptions),
//	    "my_session",
//	)
//	[...]
//	func handler(ctx iris.Context) {
//	    session, err := sessionDB.Get(ctx)
//	    if err != nil {
//	        // handle error
//	    }
//	    // use the session
//	}
type Database[T any] interface {
	// Get should return a cached session based on the cookie name.
	// It retrieves the session associated with the given cookie name from the context.
	Get(ctx *context.Context, cookieName string) (*Session[T], error)

	// Load should return a cached or a new session based on the cookie name and value.
	// If the cookie value is empty, it means that this is a new session and the database should create a new record for it.
	// Otherwise, it should return the session based on the cookie value.
	// A cookie value that does not decode should be a new session too, not an error:
	// Handler answers 500 on any error returned here, and the visitor cannot fix the cookie.
	Load(ctx *context.Context, cookieName, cookieValue string) (*Session[T], error)

	// Commit should persist the session to the database.
	// It saves the session data to the database.
	// If the session's Option's MaxAge is set to < 0
	// then the database should remove its relative session record.
	Commit(ctx *context.Context, session *Session[T]) error
}

// SingleDatabase is a database extension that defines a method to get a single session.
// It provides a method to get a single session based on the provided cookie name,
// so the caller can get a session without having to provide the cookie name each time.
type SingleDatabase[T any] struct {
	Database[T]

	// Name is the name of the cookie.
	Name string
}

// Single creates and returns a new SingleDatabase instance with the provided database and cookie name.
func Single[T any](database Database[T], cookieName string) *SingleDatabase[T] {
	return &SingleDatabase[T]{
		Database: database,
		Name:     cookieName,
	}
}

// Get retrieves the session from the request context.
func (db *SingleDatabase[T]) Get(ctx *context.Context) (*Session[T], error) {
	return db.Database.Get(ctx, db.Name)
}

// cookieDatabase is a concrete implementation of the Database interface using cookies.
type cookieDatabase[T any] struct {
	Options Options
}

var _ Database[any] = (*cookieDatabase[any])(nil)

func init() {
	gob.Register(map[string]any{})
	gob.Register(time.Time{})
}

// NewCookieDatabase creates and returns a new cookieDatabase instance with the provided options.
// It registers the generic type T with the gob package for encoding and decoding session data.
// The cookie-based session database requires a key pair in the session options
// because it stores the actual session data inside the cookie. The key pair is
// used to securely encode and decode the session data to ensure data integrity
// and confidentiality.
//
// Usage:
//
//	sessionOptions := sessions.NewOptions(hashKey, blockKey)
//	sessionDB := sessions.NewCookieDatabase[UserSession](sessionOptions)
//
// NewOptions forwards the key pairs to securecookie.CodecsFromPairs, which reads
// each pair as a hash key followed by an optional block key.
func NewCookieDatabase[T any](opts Options) Database[T] {
	if len(opts.DataCodecs) == 0 {
		panic("cookie-based session database requires a key pair in the session options")
	}

	var t T
	gob.Register(t)
	gob.Register(SessionData[T]{})

	return &cookieDatabase[T]{
		Options: opts.Apply(),
	}
}

// Get retrieves the session associated with the given cookie name from the context.
// It uses the Get function to fetch the session from the database.
func (db *cookieDatabase[T]) Get(ctx *context.Context, cookieName string) (*Session[T], error) {
	return Load(ctx, db, cookieName)
}

// Load retrieves a cached or a new session based on the cookie name and value.
// If the cookie value is empty, it creates a new session. Otherwise, it decodes the session data from the cookie value.
// A value that does not decode (signed with other keys, older than MaxAge,
// written for another data type) is a new session too, not an error:
// the visitor cannot fix that cookie, so the next Commit replaces it.
func (db *cookieDatabase[T]) Load(ctx *context.Context, cookieName, cookieValue string) (*Session[T], error) {
	session := NewSession[T](db.Options)
	if cookieValue == "" {
		session.IsNew = true
		return session, nil
	}

	if err := db.Options.Decode(cookieName, cookieValue, &session.SessionData); err != nil {
		// A new value: a failed decode may have filled part of the data.
		session = NewSession[T](db.Options)
		session.IsNew = true
		return session, nil
	}

	session.decoded = true
	return session, nil
}

// Commit saves the session data to the database.
// It persists the session associated with the given context.
func (db *cookieDatabase[T]) Commit(ctx *context.Context, session *Session[T]) error {
	return nil
}
