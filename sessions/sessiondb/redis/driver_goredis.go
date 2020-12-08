package redis

import (
	stdContext "context"
	"io"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type (
	// Options is just a type alias for the go-redis Client Options.
	Options = redis.Options
	// ClusterOptions is just a type alias for the go-redis Cluster Client Options.
	ClusterOptions = redis.ClusterOptions
)

// GoRedisClient is the interface which both
// go-redis' Client and Cluster Client implements.
type GoRedisClient interface {
	redis.Cmdable // Commands.
	io.Closer     // CloseConnection.
}

// GoRedisDriver implements the Sessions Database Driver
// for the go-redis redis driver. See driver.go file.
type GoRedisDriver struct {
	// Both Client and ClusterClient implements this interface.
	client GoRedisClient
	// Customize any go-redis fields manually
	// before Connect.
	ClientOptions  Options
	ClusterOptions ClusterOptions
}

var defaultContext = stdContext.Background()

func (r *GoRedisDriver) mergeClientOptions(c Config) *Options {
	opts := r.ClientOptions
	if opts.Addr == "" {
		opts.Addr = c.Addr
	}

	if opts.Username == "" {
		opts.Username = c.Username
	}

	if opts.Password == "" {
		opts.Password = c.Password
	}

	if opts.DB == 0 {
		opts.DB, _ = strconv.Atoi(c.Database)
	}

	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = c.Timeout
	}

	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = c.Timeout
	}

	if opts.Network == "" {
		opts.Network = c.Network
	}

	if opts.TLSConfig == nil {
		opts.TLSConfig = c.TLSConfig
	}

	if opts.PoolSize == 0 {
		opts.PoolSize = c.MaxActive
	}

	return &opts
}

func (r *GoRedisDriver) mergeClusterOptions(c Config) *ClusterOptions {
	opts := r.ClusterOptions

	if opts.Username == "" {
		opts.Username = c.Username
	}

	if opts.Password == "" {
		opts.Username = c.Password
	}

	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = c.Timeout
	}

	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = c.Timeout
	}

	if opts.TLSConfig == nil {
		opts.TLSConfig = c.TLSConfig
	}

	if opts.PoolSize == 0 {
		opts.PoolSize = c.MaxActive
	}

	if len(opts.Addrs) == 0 {
		opts.Addrs = c.Clusters
	}

	return &opts
}

// Connect initializes the redis client.
func (r *GoRedisDriver) Connect(c Config) error {
	if len(c.Clusters) > 0 {
		r.client = redis.NewClusterClient(r.mergeClusterOptions(c))
	} else {
		r.client = redis.NewClient(r.mergeClientOptions(c))
	}

	return nil
}

// PingPong sends a ping message and reports whether
// the PONG message received successfully.
func (r *GoRedisDriver) PingPong() (bool, error) {
	pong, err := r.client.Ping(defaultContext).Result()
	return pong == "PONG", err
}

// CloseConnection terminates the underline redis connection.
func (r *GoRedisDriver) CloseConnection() error {
	return r.client.Close()
}

// Set stores a "value" based on the session's "key".
// The value should be type of []byte, so unmarshal can happen.
func (r *GoRedisDriver) Set(sid, key string, value interface{}) error {
	return r.client.HSet(defaultContext, sid, key, value).Err()
}

// Get returns the associated value of the session's given "key".
func (r *GoRedisDriver) Get(sid, key string) (interface{}, error) {
	return r.client.HGet(defaultContext, sid, key).Bytes()
}

// Exists reports whether a session exists or not.
func (r *GoRedisDriver) Exists(sid string) bool {
	n, err := r.client.Exists(defaultContext, sid).Result()
	if err != nil {
		return false
	}

	return n > 0
}

// TTL returns any TTL value of the session.
func (r *GoRedisDriver) TTL(sid string) time.Duration {
	dur, err := r.client.TTL(defaultContext, sid).Result()
	if err != nil {
		return 0
	}

	return dur
}

// UpdateTTL sets expiration duration of the session.
func (r *GoRedisDriver) UpdateTTL(sid string, newLifetime time.Duration) error {
	_, err := r.client.Expire(defaultContext, sid, newLifetime).Result()
	return err
}

// GetAll returns all the key values under the session.
func (r *GoRedisDriver) GetAll(sid string) (map[string]string, error) {
	return r.client.HGetAll(defaultContext, sid).Result()
}

// GetKeys returns all keys under the session.
func (r *GoRedisDriver) GetKeys(sid string) ([]string, error) {
	return r.client.HKeys(defaultContext, sid).Result()
}

// Len returns the total length of key-values of the session.
func (r *GoRedisDriver) Len(sid string) int {
	return int(r.client.HLen(defaultContext, sid).Val())
}

// Delete removes a value from the redis store.
func (r *GoRedisDriver) Delete(sid, key string) error {
	if key == "" {
		return r.client.Del(defaultContext, sid).Err()
	}
	return r.client.HDel(defaultContext, sid, key).Err()
}
