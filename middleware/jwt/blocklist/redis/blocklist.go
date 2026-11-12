package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync/atomic"

	"github.com/kataras/iris/v14/internal/signalhandler"
	"github.com/kataras/iris/v14/middleware/jwt"

	"github.com/redis/go-redis/v9"
)

var defaultContext = context.Background()

type (
	// Options is just a type alias for the go-redis Client Options.
	Options = redis.Options
	// ClusterOptions is just a type alias for the go-redis Cluster Client Options.
	ClusterOptions = redis.ClusterOptions
)

// Client is the interface which both
// go-redis Client and Cluster Client implements.
type Client interface {
	redis.Cmdable // Commands.
	io.Closer     // CloseConnection.
}

// Blocklist is a jwt.Blocklist backed by Redis.
type Blocklist struct {
	// GetKey is a function which can be used how to extract
	// the unique identifier for a token.
	// Required. By default the token key is extracted through the claims.ID ("jti").
	GetKey func(token []byte, claims jwt.Claims) string
	// Prefix the token key into the redis database.
	// Note that if you can also select a different database
	// through ClientOptions (or ClusterOptions).
	// Defaults to empty string (no prefix).
	Prefix string
	// Both Client and ClusterClient implements this interface.
	client    Client
	connected atomic.Bool
	// Customize any go-redis fields manually
	// before Connect.
	ClientOptions  Options
	ClusterOptions ClusterOptions
}

var _ jwt.Blocklist = (*Blocklist)(nil)

// NewBlocklist returns a new redis-based Blocklist.
// Modify its ClientOptions or ClusterOptions depending the application needs
// and call its Connect.
//
// Usage:
//
//	blocklist := NewBlocklist()
//	blocklist.ClientOptions.Addr = ...
//	err := blocklist.Connect()
//
// And register it:
//
//	verifier := jwt.NewVerifier(...)
//	verifier.Blocklist = blocklist
func NewBlocklist() *Blocklist {
	return &Blocklist{
		GetKey: defaultGetKey,
		Prefix: "",
		ClientOptions: Options{
			Addr: "127.0.0.1:6379",
			// The rest are defaulted to good values already.
		},
		// If its Addrs > 0 before connect then cluster client is used instead.
		ClusterOptions: ClusterOptions{},
	}
}

// defaultGetKey derives the storage key of a token.
//
// It prefers the "jti" claim, and falls back to a digest of the token itself
// when the signer did not set one.
//
// Note(@kataras): the fallback is not cosmetic. Returning claims.ID unchecked
// meant every token without a "jti" mapped to the EMPTY key, so invalidating one
// user's token wrote to "" and the existence check for every other token read
// from "" - one logout blocked every session in the system.
func defaultGetKey(token []byte, claims jwt.Claims) string {
	if claims.ID != "" {
		return claims.ID
	}

	sum := sha256.Sum256(token)
	return hex.EncodeToString(sum[:])
}

// Connect prepares the redis client and fires a ping response to it.
func (b *Blocklist) Connect() error {
	if b.Prefix != "" {
		getKey := b.GetKey
		b.GetKey = func(token []byte, claims jwt.Claims) string {
			return b.Prefix + getKey(token, claims)
		}
	}

	if len(b.ClusterOptions.Addrs) > 0 {
		// Use cluster client.
		b.client = redis.NewClusterClient(&b.ClusterOptions)
	} else {
		b.client = redis.NewClient(&b.ClientOptions)
	}

	_, err := b.client.Ping(defaultContext).Result()
	if err != nil {
		return err
	}

	signalhandler.Register(func() error {
		b.connected.Store(false)
		return b.client.Close()
	})
	b.connected.Store(true)

	return nil
}

// IsConnected reports whether the Connect function was called.
func (b *Blocklist) IsConnected() bool {
	return b.connected.Load()
}

// ValidateToken checks if the token exists and
func (b *Blocklist) ValidateToken(token []byte, c jwt.Claims, err error) error {
	if err != nil {
		if err == jwt.ErrExpired {
			b.Del(b.GetKey(token, c))
		}

		return err // respect the previous error.
	}

	has, err := b.Has(b.GetKey(token, c))
	if err != nil {
		return err
	} else if has {
		return jwt.ErrBlocked
	}

	return nil
}

// InvalidateToken invalidates a verified JWT token.
func (b *Blocklist) InvalidateToken(token []byte, c jwt.Claims) error {
	key := b.GetKey(token, c)
	return b.client.SetEx(defaultContext, key, token, c.Timeleft()).Err()
}

// Del removes a token from the storage.
func (b *Blocklist) Del(key string) error {
	return b.client.Del(defaultContext, key).Err()
}

// Has reports whether a specific token exists in the storage.
func (b *Blocklist) Has(key string) (bool, error) {
	n, err := b.client.Exists(defaultContext, key).Result()
	return n > 0, err
}

// Count returns the total amount of tokens stored.
func (b *Blocklist) Count() (int64, error) {
	if b.Prefix == "" {
		return b.client.DBSize(defaultContext).Result()
	}

	keys, err := b.getKeys(0)
	if err != nil {
		return 0, err
	}

	return int64(len(keys)), nil
}

func (b *Blocklist) getKeys(cursor uint64) ([]string, error) {
	keys, cursor, err := b.client.Scan(defaultContext, cursor, b.Prefix+"*", 300000).Result()
	if err != nil {
		return nil, err
	}

	if cursor != 0 {
		moreKeys, err := b.getKeys(cursor)
		if err != nil {
			return nil, err
		}

		keys = append(keys, moreKeys...)
	}

	return keys, nil
}
