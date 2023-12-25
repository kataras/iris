package memstore

import (
	"sync"
	"time"
)

var (
	// Clock is the default clock to get the current time,
	// it can be used for testing purposes too.
	//
	// Defaults to time.Now.
	Clock func() time.Time = time.Now

	// ExpireDelete may be set on Cookie.Expire for expiring the given cookie.
	ExpireDelete = time.Unix(0, 0) // time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
)

// LifeTime controls the session expiration datetime.
type LifeTime struct {
	// Remember, tip for the future:
	// No need of gob.Register, because we embed the time.Time.
	// And serious bug which has a result of me spending my whole evening:
	// Because of gob encoding it doesn't encodes/decodes the other fields if time.Time is embedded
	// (this should be a bug(go1.9-rc1) or not. We don't care atm)
	time.Time
	timer *time.Timer

	// StartTime holds the Now of the Begin.
	Begun time.Time

	mu sync.RWMutex
}

// NewLifeTime returns a pointer to an empty LifeTime instance.
func NewLifeTime() *LifeTime {
	return &LifeTime{}
}

// Begin will begin the life based on the Clock (time.Now()).Add(d).
// Use `Continue` to continue from a stored time(database-based session does that).
func (lt *LifeTime) Begin(d time.Duration, onExpire func()) {
	if d <= 0 {
		return
	}

	now := Clock()

	lt.mu.Lock()
	lt.Begun = now
	lt.Time = now.Add(d)
	lt.timer = time.AfterFunc(d, onExpire)
	lt.mu.Unlock()
}

// Revive will continue the life based on the stored Time.
// Other words that could be used for this func are: Continue, Restore, Resc.
func (lt *LifeTime) Revive(onExpire func()) {
	lt.mu.RLock()
	t := lt.Time
	lt.mu.RUnlock()

	if t.IsZero() {
		return
	}

	now := Clock()
	if t.After(now) {
		d := t.Sub(now)
		lt.mu.Lock()
		if lt.timer != nil {
			lt.timer.Stop() // Stop the existing timer, if any.
		}
		lt.Begun = now
		lt.timer = time.AfterFunc(d, onExpire) // and execute on-time the new onExpire function.
		lt.mu.Unlock()
	}
}

// Shift resets the lifetime based on "d".
func (lt *LifeTime) Shift(d time.Duration) {
	lt.mu.Lock()
	if d > 0 && lt.timer != nil {
		now := Clock()
		lt.Begun = now
		lt.Time = now.Add(d)
		lt.timer.Reset(d)
	}
	lt.mu.Unlock()
}

// ExpireNow reduce the lifetime completely.
func (lt *LifeTime) ExpireNow() {
	lt.mu.Lock()
	lt.Time = ExpireDelete
	if lt.timer != nil {
		lt.timer.Stop()
	}
	lt.mu.Unlock()
}

// HasExpired reports whether "lt" represents is expired.
func (lt *LifeTime) HasExpired() bool {
	lt.mu.RLock()
	t := lt.Time
	lt.mu.RUnlock()

	if t.IsZero() {
		return false
	}

	return t.Before(Clock())
}

// DurationUntilExpiration returns the duration until expires, it can return negative number if expired,
// a call to `HasExpired` may be useful before calling this `Dur` function.
func (lt *LifeTime) DurationUntilExpiration() time.Duration {
	lt.mu.RLock()
	t := lt.Time
	lt.mu.RUnlock()

	return t.Sub(Clock())
}
