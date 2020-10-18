package jwt

import (
	stdContext "context"
	"sync"
	"time"
)

// Blocklist should hold and manage invalidated-by-server tokens.
// The `NewBlocklist` and `NewBlocklistContext` functions
// returns a memory storage of tokens,
// it is the internal "blocklist" struct.
//
// The end-developer can implement her/his own blocklist,
// e.g. a redis one to keep persistence of invalidated tokens on server restarts.
// and bind to the JWT middleware's Blocklist field.
type Blocklist interface {
	// Set should upsert a token to the storage.
	Set(token string, expiresAt time.Time)
	// Del should remove a token from the storage.
	Del(token string)
	// Count should return the total amount of tokens stored.
	Count() int
	// Has should report whether a specific token exists in the storage.
	Has(token string) bool
}

// blocklist is an in-memory storage of tokens that should be
// immediately invalidated by the server-side.
// The most common way to invalidate a token, e.g. on user logout,
// is to make the client-side remove the token itself.
// However, if someone else has access to that token,
// it could be still valid for new requests until its expiration.
type blocklist struct {
	entries map[string]time.Time // key = token | value = expiration time (to remove expired).
	mu      sync.RWMutex
}

// NewBlocklist returns a new up and running in-memory Token Blocklist.
// The returned value can be set to the JWT instance's Blocklist field.
func NewBlocklist(gcEvery time.Duration) Blocklist {
	return NewBlocklistContext(stdContext.Background(), gcEvery)
}

// NewBlocklistContext same as `NewBlocklist`
// but it also accepts a standard Go Context for GC cancelation.
func NewBlocklistContext(ctx stdContext.Context, gcEvery time.Duration) Blocklist {
	b := &blocklist{
		entries: make(map[string]time.Time),
	}

	if gcEvery > 0 {
		go b.runGC(ctx, gcEvery)
	}

	return b
}

// Set upserts a given token, with its expiration time,
// to the block list, so it's immediately invalidated by the server-side.
func (b *blocklist) Set(token string, expiresAt time.Time) {
	b.mu.Lock()
	b.entries[token] = expiresAt
	b.mu.Unlock()
}

// Del removes a "token" from the block list.
func (b *blocklist) Del(token string) {
	b.mu.Lock()
	delete(b.entries, token)
	b.mu.Unlock()
}

// Count returns the total amount of blocked tokens.
func (b *blocklist) Count() int {
	b.mu.RLock()
	n := len(b.entries)
	b.mu.RUnlock()

	return n
}

// Has reports whether the given "token" is blocked by the server.
// This method is called before the token verification,
// so even if was expired it is removed from the block list.
func (b *blocklist) Has(token string) bool {
	if token == "" {
		return false
	}

	b.mu.RLock()
	_, ok := b.entries[token]
	b.mu.RUnlock()

	/* No, the Blocklist will be used after the token is parsed,
	there we can call the Del method if err was ErrExpired.
	if ok {
		// As an extra step, to keep the list size as small as possible,
		// we delete it from list if it's going to be expired
		// ~in the next `blockedExpireLeeway` seconds.~
		// - Let's keep it easier for testing by not setting a leeway.
		// if time.Now().Add(blockedExpireLeeway).After(expiresAt) {
		if time.Now().After(expiresAt) {
			b.Del(token)
		}
	}*/

	return ok
}

// GC iterates over all entries and removes expired tokens.
// This method is helpful to keep the list size small.
// Depending on the application, the GC method can be scheduled
// to called every half or a whole hour.
// A good value for a GC cron task is the JWT's max age (default).
func (b *blocklist) GC() int {
	now := time.Now()
	var markedForDeletion []string

	b.mu.RLock()
	for token, expiresAt := range b.entries {
		if now.After(expiresAt) {
			markedForDeletion = append(markedForDeletion, token)
		}
	}
	b.mu.RUnlock()

	n := len(markedForDeletion)
	if n > 0 {
		for _, token := range markedForDeletion {
			b.Del(token)
		}
	}

	return n
}

func (b *blocklist) runGC(ctx stdContext.Context, every time.Duration) {
	t := time.NewTicker(every)

	for {
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
			b.GC()
		}
	}
}
