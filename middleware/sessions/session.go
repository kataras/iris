package sessions

import (
	"errors"
	"net/http"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/uid"
)

// SessionData is a wrapper struct that includes both the generic data and flash messages.
type SessionData[T any] struct {
	Data          T              // Data holds the session-specific data.
	FlashMessages map[string]any // FlashMessages holds temporary messages that are intended to be displayed only once.
	Extra         any            // Extra holds any extra data that the user wants to store.
}

// Session represents a user session with a generic data type T.
// It holds the session data, configuration options, and state information.
// If the ID is set by the database, no data, including flash messages, are stored in the session cookie.
// In such cases, the session data and flash messages should be handled by the database itself.
//
// Note: You should call Commit before any other writes to the response.
// Ensure that headers are written before the body to maintain proper session handling.
type Session[T any] struct {
	SessionData[T] // SessionData holds both the actual session-specific data and the flash messages.

	// ID is set by the database, optionally.
	ID string

	Options  Options // Options contains the session configuration settings.
	IsNew    bool    // IsNew indicates whether the session is newly created.
	name     string
	database Database[T]

	// state of commit
	commitCalled bool
}

// NewSession creates and returns a new Session instance with the provided options.
//
// Options is a value type, so the session gets its own copy: Expire, Commit and
// Destroy write to Options.MaxAge and must not reach the database's own copy.
// The DataCodecs slice is the one part still shared, by design, because it holds the keys.
func NewSession[T any](opts Options) *Session[T] {
	return &Session[T]{
		Options: opts,
	}
}

// Name returns the name of the session.
func (s *Session[T]) Name() string {
	return s.name
}

// Set sets the session's specific data.
func (s *Session[T]) Set(data T) {
	s.SessionData.Data = data
}

// SetExtra sets the session's extra data.
func (s *Session[T]) SetExtra(extra any) {
	s.SessionData.Extra = extra
}

// GetExtra returns the session's extra data as V.
// The second return value reports whether extra data was set AND is of type V.
//
// It is named GetExtra, not Extra: the promoted SessionData field of that
// name remains directly readable (session.Extra) where the raw any is wanted.
func (s *Session[T]) GetExtra[V any]() (V, bool) {
	v, ok := s.SessionData.Extra.(V)
	return v, ok
}

// SetFlash sets a flash message associated with the given key.
// Flash messages are temporary messages that are intended to be displayed only once.
// They are stored in the session and are removed after GetFlash is called.
func (s *Session[T]) SetFlash(key string, message any) {
	if s.FlashMessages == nil {
		s.FlashMessages = make(map[string]any)
	}

	s.FlashMessages[key] = message
}

// GetFlash returns the flash message associated with the given key as V.
// A message of the requested type is removed from the flash messages map;
// a message of a different type is left in place and (zero, false) is
// returned, so a mistyped read is never destructive.
//
// Use V = any for the old untyped behavior.
func (s *Session[T]) GetFlash[V any](key string) (V, bool) {
	v, ok := s.FlashMessages[key].(V)
	if ok {
		delete(s.FlashMessages, key)
	}

	return v, ok
}

// FlashDelete is a function that always returns true, indicating that the flash message should be deleted.
var FlashDelete = func(string, any) bool { return true }

// Flashes returns all the flash messages and optionally deletes them at the same time.
// The resulting map contains all the flash messages with their corresponding keys.
// If the deleteFn is not nil, it is called for each flash message
// to determine whether it should be deleted from the original map.
// Note that you should call Commit before any other writes to the response, headers should be written before body.
//
// To get and delete all flash messages, use the following:
//
//	flashes := session.Flashes(sessions.FlashDelete)
func (s *Session[T]) Flashes(deleteFn func(key string, message any) bool) map[string]any {
	flashes := make(map[string]any, len(s.SessionData.FlashMessages))
	for key, message := range s.SessionData.FlashMessages {
		flashes[key] = message
		if deleteFn != nil && deleteFn(key, message) {
			delete(s.SessionData.FlashMessages, key)
		}
	}
	return flashes
}

// Delete sets the session's MaxAge to -1, indicating that the session should be expired immediately.
// And calls Commit to delete the associated session cookie and any database's record.
func (s *Session[T]) Delete(ctx *context.Context) error {
	s.Options.MaxAge = -1
	return s.Commit(ctx)
}

// Commit saves the session data to the database and sets the session cookie in the context.
// If there is an error during the commit, it expires the session cookie.
//
// Note that you should call Commit before any other writes to the response.
func (s *Session[T]) Commit(ctx *context.Context) error {
	s.commitCalled = true // this is used only on Handler.

	// fire the database commit first, and if it fails, we will expire the cookie.
	commitErr := s.database.Commit(ctx, s)
	if commitErr != nil {
		s.Options.MaxAge = -1 // expire cookie if commit failed.
	}

	cookie, encodeErr := s.Cookie()
	if cookie != nil {
		if s.Options.MaxAge < 0 {
			removeSessionContextValue(ctx, s.name)
		}

		ctx.SetCookie(cookie)
	}

	return errors.Join(commitErr, encodeErr)
}

// RegenerateID rotates the session's server-side identifier to defend against
// session fixation attacks. Call it whenever the session's privilege level
// changes, most importantly right after a successful sign-in, so that any
// session identifier an attacker may have planted in the victim's browser can
// no longer be used to access the now-authenticated session.
//
// It deletes the record stored under the previous ID, assigns a freshly
// generated ID and persists the session (re-issuing the session cookie under
// the new ID). Set any authenticated data on the session before calling it;
// a subsequent Commit is not required, but calling it again is safe.
//
// RegenerateID is a no-op for cookie-based sessions: their data lives in the
// signed and encrypted cookie itself, so there is no server-side ID to rotate.
// It is likewise a no-op for sessions that were not loaded from a database.
//
// As with Commit, call RegenerateID before writing to the response body so that
// the updated Set-Cookie header is sent first.
func (s *Session[T]) RegenerateID(ctx *context.Context) error {
	if s.database == nil || s.ID == "" {
		// No server-side identifier to rotate (cookie-based session or a
		// session that was never loaded from a database).
		return nil
	}

	// Delete the record stored under the previous ID without disturbing the
	// live session's options or data. A MaxAge below zero instructs the
	// database to remove the record.
	expired := &Session[T]{
		SessionData: s.SessionData,
		ID:          s.ID,
		Options:     s.Options,
		name:        s.name,
		database:    s.database,
	}
	expired.Options.MaxAge = -1
	if err := s.database.Commit(ctx, expired); err != nil {
		return err
	}

	// Assign a fresh ID and persist the session (and its cookie) under it.
	// Version 7: time-ordered, so neighboring sessions land on neighboring
	// storage keys. Validation stays version-agnostic (Parse), so v4 and v7
	// ids coexist.
	s.ID = uid.NewV7String()
	s.IsNew = true
	return s.Commit(ctx)
}

// cookieValue encodes the session data or ID into a string for the cookie value.
// If the session is set to expire, it returns an empty string.
func (s *Session[T]) cookieValue() (cookieValue string, err error) {
	if s.Options.MaxAge <= 0 { // expire now.
		s.ID = ""
	} else { // error here is nil
		if s.ID != "" {
			cookieValue, err = s.Options.Encode(s.name, s.ID)
		} else {
			cookieValue, err = s.Options.Encode(s.name, s.SessionData)
		}
		if err != nil {
			s.Options.MaxAge = -1 // we remove the cookie on failure, the error will fallthrough to the caller.
		}
	}

	return cookieValue, err
}

// Cookie creates an http.Cookie based on the session's options and encoded value.
// It returns the cookie and any error encountered during encoding.
func (s *Session[T]) Cookie() (*http.Cookie, error) {
	cookieValue, err := s.cookieValue()
	if err != nil {
		return nil, err
	}

	return s.Options.Cookie(s.name, cookieValue), nil
}
