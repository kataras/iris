package sessions

import (
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"
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

	mu sync.RWMutex
}

// Begin will begin the life based on the time.Now().Add(d).
// Use `Continue` to continue from a stored time(database-based session does that).
func (lt *LifeTime) Begin(d time.Duration, onExpire func()) {
	if d <= 0 {
		return
	}

	lt.mu.Lock()
	lt.Time = time.Now().Add(d)
	lt.timer = time.AfterFunc(d, onExpire)
	lt.mu.Unlock()
}

// Revive will continue the life based on the stored Time.
// Other words that could be used for this func are: Continue, Restore, Resc.
func (lt *LifeTime) Revive(onExpire func()) {
	if lt.Time.IsZero() {
		return
	}

	now := time.Now()
	if lt.Time.After(now) {
		d := lt.Time.Sub(now)
		lt.mu.Lock()
		lt.timer = time.AfterFunc(d, onExpire)
		lt.mu.Unlock()
	}
}

// Shift resets the lifetime based on "d".
func (lt *LifeTime) Shift(d time.Duration) {
	lt.mu.Lock()
	if d > 0 && lt.timer != nil {
		lt.Time = time.Now().Add(d)
		lt.timer.Reset(d)
	}
	lt.mu.Unlock()
}

// ExpireNow reduce the lifetime completely.
func (lt *LifeTime) ExpireNow() {
	lt.mu.Lock()
	lt.Time = context.CookieExpireDelete
	if lt.timer != nil {
		lt.timer.Stop()
	}
	lt.mu.Unlock()
}

// HasExpired reports whether "lt" represents is expired.
func (lt *LifeTime) HasExpired() bool {
	if lt.IsZero() {
		return false
	}

	return lt.Time.Before(time.Now())
}

// DurationUntilExpiration returns the duration until expires, it can return negative number if expired,
// a call to `HasExpired` may be useful before calling this `Dur` function.
func (lt *LifeTime) DurationUntilExpiration() time.Duration {
	return time.Until(lt.Time)
}
