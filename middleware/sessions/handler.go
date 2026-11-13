package sessions

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"sync"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/middleware/recorder"
)

const handlerSessionContextKey = sessionsContextKeyPrefix + "handler"

// Handler returns a new session handler.
// It retrieves the session from the database based on the cookie name.
// The session is stored in the context and can be retrieved using the Get function.
//
// After the request it commits the session, unless Commit was called already,
// the response status is 400 or above, or the request left the session as it
// was loaded: same data, flash messages, id and Options. A commit of an
// unchanged session would store an old copy over what another request wrote,
// or deleted, while this one was running. A new session is always committed,
// and so is one that a handler registered before this one has loaded.
//
// That holds for the databases of this package (NewCookieDatabase,
// NewRedisDatabase, NewPostgresDatabase), which decode the data of a session
// for each request. The session of any other Database is committed after
// every request, as before: its data may be shared with other requests, so
// this handler does not read it.
//
// So a request that only reads the session sends no cookie and writes nothing
// to the store, and nothing moves the expiry of the session: the cookie and
// the Redis record end Options.MaxAge after the last change, not after the
// last request. Call Session.Commit in the handler to keep a session alive on
// a route that only reads it. Two requests that both change the session at
// the same time still lose one of the two writes.
//
// A response without the session cookie is one that a shared cache may store.
// Send "Cache-Control: private" on a page that depends on the session when a
// reverse proxy caches in front of the application. The httpcache middleware
// does not cache a request with cookies unless its key names every one of
// them, as long as its handler is registered before this one: registered
// after it, it stores the page made for a new session before the session
// cookie is written.
//
// Note: Use this when you want to manage a single session per request, otherwise use the session database's methods instead
// and don't register this Handler middleware at all.
//
// SECURITY: this handler does NOT rotate the session id on its own.
//
// Call `Session.RegenerateID` immediately after a privilege change - a
// successful login above all - or an attacker who plants a session id in the
// victim's browser keeps access to the authenticated session afterwards
// (session fixation).
//
//	func login(ctx iris.Context) {
//		// ...verify the credentials first...
//		sess := sessions.Get[MyData](ctx)
//		sess.Data.UserID = user.ID // before RegenerateID, which stores the session.
//		if err := sess.RegenerateID(ctx); err != nil { ... }
//	}
//
// The database re-issues a fresh id when a cookie names a session that has no
// server-side record, but that only defeats a planted id that was never used:
// an attacker who first creates a real session and then plants THAT id is not
// covered.
func Handler[T any](database Database[T], cookieName string) context.Handler {
	return handler(database, cookieName, true)
}

// ReadOnlyHandler returns a new session handler with read-only access.
// Useful for performance when you create and update the session from a single spot (using the session database directly)
// and you just want to read the session data from multiple routes and handlers.
func ReadOnlyHandler[T any](database Database[T], cookieName string) context.Handler {
	return handler(database, cookieName, false) // read-only access.
}

func handler[T any](database Database[T], cookieName string, writeAccess bool) context.Handler {
	return func(ctx *context.Context) {
		// A handler before this one may have loaded the session and changed it.
		loadedBefore := getSessionContextValue[T](ctx, cookieName) != nil

		session, err := database.Get(ctx, cookieName)
		if err != nil {
			// Private: a store failure names hosts, ports and sometimes users.
			// An error handler still reads it through ctx.GetErr; no client does.
			ctx.StopWithPlainError(500, context.PrivateError(err))
			return
		}

		// We could use a map of reflect.Type and cookie names to make it better-API on supporting multiple session names on
		// Get function but...that way we lose in performance.
		// So we store a fixed context key for the sessions managed by this Handler function.
		ctx.Values().Set(handlerSessionContextKey, session)

		if writeAccess { // if we enable read & write then make sure that the response is written as expected.
			var loaded *loadedSession[T]
			if !loadedBefore {
				loaded = newLoadedSession(session)
			}

			previous := ctx.ResponseWriter()
			rr, ok := recorder.AcquireResponseRecorder(previous)
			if ok {
				// Release the captured response when the whole handler chain is done,
				// and put the previous writer back on the Context first: a handler
				// registered outside this one (an access log, an error handler) reads
				// the Context's writer after the chain returns, and a released
				// recorder may already belong to another request.
				//
				// Deferred before the commit below, so it runs after it: the commit
				// still writes its cookie and its failure through the recorder.
				defer func() {
					ctx.SetResponseWriter(previous)
					recorder.ReleaseResponseRecorder(rr)
				}()
				// Set the response writer to the response recorder.
				ctx.SetResponseWriter(rr)
			}

			defer func() {
				// if any other error or committed already, then don't try to persist the session,
				// and neither when the request left it as it was loaded.
				if !session.commitCalled && rr.StatusCode() < 400 && !loaded.is(session) {
					if err := session.Commit(ctx); err != nil {
						rr.Body.Reset() // reset the response body, so we are sure that the main handler doesn't write any contents on commit error.
						// Private, like the Get failure above.
						ctx.StopWithPlainError(500, context.PrivateError(err))
						return
					}
				}
			}()
		}

		ctx.Next()
	}
}

// loadedSession is what a stored session was when Handler got it from the
// database, kept to tell after the request whether the request changed it.
type loadedSession[T any] struct {
	id      string
	options Options
	data    SessionData[T]
}

// newLoadedSession returns nil for a session that is committed whatever the
// request does with it: one that a database of this package did not decode
// from its store for this request (a new one, one whose record is gone, the
// session of a custom Database), and one whose data cannot be copied.
func newLoadedSession[T any](session *Session[T]) *loadedSession[T] {
	if !session.decoded {
		return nil
	}

	// A deep copy, made with the encoding the data is stored in: a handler
	// can change a slice or a map of the session in place. The copy is
	// compared by value, never by its encoding: gob writes the entries of a
	// map in no fixed order.
	loaded := &loadedSession[T]{id: session.ID, options: session.Options}
	if !copyThroughGob(session.SessionData, &loaded.data) {
		return nil
	}

	return loaded
}

// gobCopier is a gob encoder and a decoder on one stream. A stream describes
// a type once, so a copier that is used again does not build the encoding of
// the session's type for every request, which is most of what a copy costs.
type gobCopier struct {
	stream  bytes.Buffer
	encoder *gob.Encoder
	decoder *gob.Decoder
}

var gobCopiers = sync.Pool{
	New: func() any {
		c := new(gobCopier)
		c.encoder, c.decoder = gob.NewEncoder(&c.stream), gob.NewDecoder(&c.stream)
		return c
	},
}

// copyThroughGob sets what dst points to to a deep copy of src. It reports
// false when src cannot be encoded or its encoding cannot be decoded into dst.
// gob writes a value to the stream in parts, so a copy that fails can leave
// one behind, which the next copy through that copier would read as its own.
func copyThroughGob(src, dst any) bool {
	c := gobCopiers.Get().(*gobCopier)
	if c.encoder.Encode(src) != nil || c.decoder.Decode(dst) != nil {
		return false // the stream is broken: the copier is not used again.
	}

	gobCopiers.Put(c)
	return true
}

// is reports whether the session still is what it was when it was loaded.
func (loaded *loadedSession[T]) is(session *Session[T]) bool {
	if loaded == nil || session.ID != loaded.id {
		return false
	}

	was, now := loaded.options, session.Options
	was.DataCodecs, now.DataCodecs = nil, nil // shared by every session, and not a part of this one.

	return reflect.DeepEqual(was, now) && reflect.DeepEqual(loaded.data, session.SessionData)
}

// Get returns the session from the context.
// It returns nil if the session is not found or if the stored session's type
// is not the same as the one provided on the Handler function.
func Get[T any](ctx *context.Context) *Session[T] {
	v, ok := ctx.ValueAs[*Session[T]](handlerSessionContextKey)
	if !ok {
		return nil
	}

	return v
}
