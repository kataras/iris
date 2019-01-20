package service

import (
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/kataras/iris/core/errors"
)

var (
	// ErrRedisClosed an error with message 'Redis is already closed'
	ErrRedisClosed = errors.New("Redis is already closed")
	// ErrKeyNotFound an error with message 'Key $thekey doesn't found'
	ErrKeyNotFound = errors.New("Key '%s' doesn't found")
)

// Service the Redis service, contains the config and the redis pool
type Service struct {
	// Connected is true when the Service has already connected
	Connected bool
	// Config the redis config for this redis
	Config *Config
	pool   *redis.Pool
}

// PingPong sends a ping and receives a pong, if no pong received then returns false and filled error
func (r *Service) PingPong() (bool, error) {
	c := r.pool.Get()
	defer c.Close()
	msg, err := c.Do("PING")
	if err != nil || msg == nil {
		return false, err
	}
	return (msg == "PONG"), nil
}

// CloseConnection closes the redis connection
func (r *Service) CloseConnection() error {
	if r.pool != nil {
		return r.pool.Close()
	}
	return ErrRedisClosed
}

// Set sets a key-value to the redis store.
// The expiration is setted by the MaxAgeSeconds.
func (r *Service) Set(key string, value interface{}, secondsLifetime int64) (err error) {
	c := r.pool.Get()
	defer c.Close()
	if c.Err() != nil {
		return c.Err()
	}

	// if has expiration, then use the "EX" to delete the key automatically.
	if secondsLifetime > 0 {
		_, err = c.Do("SETEX", r.Config.Prefix+key, secondsLifetime, value)
	} else {
		_, err = c.Do("SET", r.Config.Prefix+key, value)
	}

	return
}

// Get returns value, err by its key
//returns nil and a filled error if something bad happened.
func (r *Service) Get(key string) (interface{}, error) {
	c := r.pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		return nil, err
	}

	redisVal, err := c.Do("GET", r.Config.Prefix+key)

	if err != nil {
		return nil, err
	}
	if redisVal == nil {
		return nil, ErrKeyNotFound.Format(key)
	}
	return redisVal, nil
}

// TTL returns the seconds to expire, if the key has expiration and error if action failed.
// Read more at: https://redis.io/commands/ttl
func (r *Service) TTL(key string) (seconds int64, hasExpiration bool, found bool) {
	c := r.pool.Get()
	defer c.Close()
	redisVal, err := c.Do("TTL", r.Config.Prefix+key)
	if err != nil {
		return -2, false, false
	}
	seconds = redisVal.(int64)
	// if -1 means the key has unlimited life time.
	hasExpiration = seconds > -1
	// if -2 means key does not exist.
	found = !(c.Err() != nil || seconds == -2)
	return
}

func (r *Service) updateTTLConn(c redis.Conn, key string, newSecondsLifeTime int64) error {
	reply, err := c.Do("EXPIRE", r.Config.Prefix+key, newSecondsLifeTime)

	if err != nil {
		return err
	}

	// https://redis.io/commands/expire#return-value
	//
	// 1 if the timeout was set.
	// 0 if key does not exist.
	if hadTTLOrExists, ok := reply.(int); ok {
		if hadTTLOrExists == 1 {
			return nil
		} else if hadTTLOrExists == 0 {
			return fmt.Errorf("unable to update expiration, the key '%s' was stored without ttl", key)
		} // do not check for -1.
	}

	return nil
}

// UpdateTTL will update the ttl of a key.
// Using the "EXPIRE" command.
// Read more at: https://redis.io/commands/expire#refreshing-expires
func (r *Service) UpdateTTL(key string, newSecondsLifeTime int64) error {
	c := r.pool.Get()
	defer c.Close()
	err := c.Err()
	if err != nil {
		return err
	}

	return r.updateTTLConn(c, key, newSecondsLifeTime)
}

// UpdateTTLMany like `UpdateTTL` but for all keys starting with that "prefix",
// it is a bit faster operation if you need to update all sessions keys (although it can be even faster if we used hash but this will limit other features),
// look the `sessions/Database#OnUpdateExpiration` for example.
func (r *Service) UpdateTTLMany(prefix string, newSecondsLifeTime int64) error {
	c := r.pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		return err
	}

	keys, err := r.getKeysConn(c, prefix)
	if err != nil {
		return err
	}

	for _, key := range keys {
		if err = r.updateTTLConn(c, key, newSecondsLifeTime); err != nil { // fail on first error.
			return err
		}
	}

	return err
}

// GetAll returns all redis entries using the "SCAN" command (2.8+).
func (r *Service) GetAll() (interface{}, error) {
	c := r.pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		return nil, err
	}

	redisVal, err := c.Do("SCAN", 0) // 0 -> cursor

	if err != nil {
		return nil, err
	}

	if redisVal == nil {
		return nil, err
	}

	return redisVal, nil
}

func (r *Service) getKeysConn(c redis.Conn, prefix string) ([]string, error) {
	if err := c.Send("SCAN", 0, "MATCH", r.Config.Prefix+prefix+"*", "COUNT", 9999999999); err != nil {
		return nil, err
	}

	if err := c.Flush(); err != nil {
		return nil, err
	}

	reply, err := c.Receive()
	if err != nil || reply == nil {
		return nil, err
	}

	// it returns []interface, with two entries, the first one is "0" and the second one is a slice of the keys as []interface{uint8....}.

	if keysInterface, ok := reply.([]interface{}); ok {
		if len(keysInterface) == 2 {
			// take the second, it must contain the slice of keys.
			if keysSliceAsBytes, ok := keysInterface[1].([]interface{}); ok {
				keys := make([]string, len(keysSliceAsBytes), len(keysSliceAsBytes))
				for i, k := range keysSliceAsBytes {
					keys[i] = fmt.Sprintf("%s", k)[len(r.Config.Prefix):]
				}

				return keys, nil
			}
		}
	}

	return nil, nil
}

// GetKeys returns all redis keys using the "SCAN" with MATCH command.
// Read more at:  https://redis.io/commands/scan#the-match-option.
func (r *Service) GetKeys(prefix string) ([]string, error) {
	c := r.pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		return nil, err
	}

	return r.getKeysConn(c, prefix)
}

// GetBytes returns value, err by its key
// you can use utils.Deserialize((.GetBytes("yourkey"),&theobject{})
//returns nil and a filled error if something wrong happens
func (r *Service) GetBytes(key string) ([]byte, error) {
	c := r.pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		return nil, err
	}

	redisVal, err := c.Do("GET", r.Config.Prefix+key)

	if err != nil {
		return nil, err
	}
	if redisVal == nil {
		return nil, ErrKeyNotFound.Format(key)
	}

	return redis.Bytes(redisVal, err)
}

// Delete removes redis entry by specific key
func (r *Service) Delete(key string) error {
	c := r.pool.Get()
	defer c.Close()

	_, err := c.Do("DEL", r.Config.Prefix+key)
	return err
}

func dial(network string, addr string, pass string) (redis.Conn, error) {
	if network == "" {
		network = DefaultRedisNetwork
	}
	if addr == "" {
		addr = DefaultRedisAddr
	}
	c, err := redis.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	if pass != "" {
		if _, err = c.Do("AUTH", pass); err != nil {
			c.Close()
			return nil, err
		}
	}
	return c, err
}

// Connect connects to the redis, called only once
func (r *Service) Connect() {
	c := r.Config

	if c.IdleTimeout <= 0 {
		c.IdleTimeout = DefaultRedisIdleTimeout
	}

	if c.Network == "" {
		c.Network = DefaultRedisNetwork
	}

	if c.Addr == "" {
		c.Addr = DefaultRedisAddr
	}

	pool := &redis.Pool{IdleTimeout: c.IdleTimeout, MaxIdle: c.MaxIdle, MaxActive: c.MaxActive}
	pool.TestOnBorrow = func(c redis.Conn, t time.Time) error {
		_, err := c.Do("PING")
		return err
	}

	if c.Database != "" {
		pool.Dial = func() (redis.Conn, error) {
			red, err := dial(c.Network, c.Addr, c.Password)
			if err != nil {
				return nil, err
			}
			if _, err = red.Do("SELECT", c.Database); err != nil {
				red.Close()
				return nil, err
			}
			return red, err
		}
	} else {
		pool.Dial = func() (redis.Conn, error) {
			return dial(c.Network, c.Addr, c.Password)
		}
	}
	r.Connected = true
	r.pool = pool
}

// New returns a Redis service filled by the passed config
// to connect call the .Connect().
func New(cfg ...Config) *Service {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	r := &Service{pool: &redis.Pool{}, Config: &c}
	return r
}
