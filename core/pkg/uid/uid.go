// Package uid provides tiny helpers over the standard library's uuid package
// for the two id shapes the framework generates: unpredictable version 4 ids
// and time-ordered version 7 ids.
package uid

import "uuid"

// NewString returns a new random (version 4) UUID in its canonical string form.
// Use it for identifiers that must be unpredictable and must not leak their
// creation time, such as token ids.
func NewString() string {
	return uuid.New().String()
}

// NewV7String returns a new time-ordered (version 7) UUID in its canonical
// string form. Use it for identifiers that are stored and looked up as keys:
// the timestamp prefix keeps neighboring inserts adjacent (storage-key
// locality), which databases and caches reward. Note that a v7 id embeds its
// creation time, so it is not suitable where that would be a leak. Session
// ids qualify: their 74 random bits keep them unguessable, and the moment a
// session was created is not a secret.
func NewV7String() string {
	return uuid.NewV7().String()
}
