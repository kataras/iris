package jwt

import (
	"github.com/kataras/jwt"
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
	jwt.TokenValidator

	// InvalidateToken should invalidate a verified JWT token.
	InvalidateToken(token []byte, expiry int64)
	// Del should remove a token from the storage.
	Del(token []byte)
	// Count should return the total amount of tokens stored.
	Count() int
	// Has should report whether a specific token exists in the storage.
	Has(token []byte) bool
}
