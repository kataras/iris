package sessions

import (
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/middleware/recorder"
)

const handlerSessionContextKey = sessionsContextKeyPrefix + "handler"

// Handler returns a new session handler.
// It retrieves the session from the database based on the cookie name.
// It commits the session after the request is completed if the Commit function is not already called.
// The session is stored in the context and can be retrieved using the Get function.
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
//		if err := sess.RegenerateID(ctx); err != nil { ... }
//		sess.Data.UserID = user.ID
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
				if !session.commitCalled && rr.StatusCode() < 400 { // if any other error or commited already, then don't try to persist the session.
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
