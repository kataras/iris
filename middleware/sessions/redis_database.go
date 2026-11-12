package sessions

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/uid"

	"github.com/redis/go-redis/v9"
)

type (
	redisDatabase[T any] struct {
		client  RedisClient
		Options Options
	}

	// RedisClient is the interface which both
	// go-redis's Client and Cluster Client implements.
	RedisClient interface {
		redis.Cmdable // Commands.
		io.Closer     // CloseConnection.
	}
)

var _ Database[any] = (*redisDatabase[any])(nil)

// NewRedisDatabase returns a new redis-backed session Database for the given client.
// It panics if client is nil.
//
// The client is built by the caller, so either a single node or a cluster works:
//
//	client := redis.NewClient(redis.Options{...})
//	// OR
//	client := redis.NewClusterClient(redis.ClusterOptions{...})
func NewRedisDatabase[T any](client RedisClient, opts Options) Database[T] {
	if client == nil {
		panic("redis client is required")
	}

	var t T
	gob.Register(t)
	gob.Register(SessionData[T]{})

	db := &redisDatabase[T]{
		client:  client,
		Options: opts.Apply(),
	}

	return db
}

// Get returns a session from the request context or the redis database.
func (db *redisDatabase[T]) Get(ctx *context.Context, cookieName string) (*Session[T], error) {
	return Load(ctx, db, cookieName)
}

// Load loads the session from the redis database.
func (db *redisDatabase[T]) Load(ctx *context.Context, cookieName string, cookieValue string) (*Session[T], error) {
	session, err := db.load(ctx, cookieName, cookieValue)
	if err != nil {
		key := db.makeKey(cookieName, cookieValue)
		if session != nil {
			key = db.makeKey(cookieName, session.ID)
		}
		err = fmt.Errorf("load session from redis failed: %s: %w", key, err)
	}

	return session, err
}

func (db *redisDatabase[T]) load(ctx *context.Context, cookieName string, cookieValue string) (*Session[T], error) {
	session := NewSession[T](db.Options)
	if cookieValue == "" {
		session.ID = uid.NewV7String() // time-ordered ids for storage-key locality.
		session.IsNew = true
		return session, nil
	}

	// bind session id.
	err := db.Options.Decode(cookieName, cookieValue, &session.ID) // assign the error.
	if err != nil {
		return nil, err // we can't decode the session id, which is important, in that case let's return nil and error.
	}
	// bind data and any flash messages.
	result, err := db.client.Get(ctx, db.makeKey(cookieName, session.ID)).Result() // assign the error.
	if err == nil {
		err = db.Options.Decode(cookieName, result, &session.SessionData) // assign the error.
	} else if errors.Is(err, redis.Nil) { // Cookie exists in the browser but not in the database for some reason.
		// we don't care about this error, it's just a new session and Commit will create it if it's missing,
		// however for security reasons we assign a new session id here.
		err = nil
		session.ID = uid.NewV7String()
	}

	return session, err
}

// Commit saves the session to the redis database.
func (db *redisDatabase[T]) Commit(ctx *context.Context, session *Session[T]) error {
	key := db.makeKey(session.Name(), session.ID)
	if err := db.commit(ctx, key, session); err != nil {
		return fmt.Errorf("commit session to redis failed: %s: %w", key, err)
	}

	return nil
}

func (db *redisDatabase[T]) commit(ctx *context.Context, key string, session *Session[T]) error {
	if session.Options.MaxAge < 0 {
		// delete the session record entirely.
		return db.client.Del(ctx, key).Err()
	}

	encodedData, err := db.Options.Encode(session.Name(), session.SessionData)
	if err != nil {
		return err
	}

	// Note: we could use HSET and HEXPIRE that are more memory-performant for large objects like sessions
	// but we already hashing them, so there is no such a need here, keep it simple.
	//
	// That we do here is just to "group" the sessions with the cookie name, e.g. cookieName:sessionID => sessionData.
	return db.client.Set(ctx, key, []byte(encodedData), session.Options.MaxAge).Err()
}

func (db *redisDatabase[T]) makeKey(cookieName, sessionID string) string {
	return cookieName + ":" + sessionID
}
