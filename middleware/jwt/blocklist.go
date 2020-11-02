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
	InvalidateToken(token []byte, c Claims) error
	// Del should remove a token from the storage.
	Del(key string) error
	// Has should report whether a specific token exists in the storage.
	Has(key string) (bool, error)
	// Count should return the total amount of tokens stored.
	Count() (int64, error)
}

type blocklistConnect interface {
	Connect() error
	IsConnected() bool
}
