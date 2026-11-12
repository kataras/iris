package rate

import "time"

// PurgeInterval exposes the resolved purge tick for the tests.
func (l *Limiter) PurgeInterval() time.Duration {
	return l.purgeInterval
}

// ShouldPurgeAfter reports whether the resolved purge condition drops a client
// whose last request was "idle" ago. It lets the tests observe the resolved max
// lifetime without waiting for it.
func (l *Limiter) ShouldPurgeAfter(idle time.Duration) bool {
	return l.purgeCondition(&Client{lastSeen: time.Now().Add(-idle)})
}

// DefaultPurgeInterval is the package's default purge tick, for the tests.
const DefaultPurgeInterval = defaultPurgeInterval

// MinPurgeLifetime is the lower bound of the derived max lifetime, for the tests.
const MinPurgeLifetime = minPurgeLifetime
