package redis

import (
	stdContext "context"
	"io"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type (
	// Options is just a type alias for the go-redis Client Options.
	Options = redis.Options
	// ClusterOptions is just a type alias for the go-redis Cluster Client Options.
	ClusterOptions = redis.ClusterOptions
)

// GoRedisClient is the interface which both
// go-redis's Client and Cluster Client implements.
type GoRedisClient interface {
	redis.Cmdable // Commands.
	io.Closer     // CloseConnection.
}

// GoRedisDriver implements the Sessions Database Driver
// for the go-redis redis driver. See driver.go file.
type GoRedisDriver struct {
	// Both Client and ClusterClient implements this interface.
	// Custom one can be directly passed but if so, the
	// Connect method does nothing (so all connection and client settings are ignored).
	Client GoRedisClient
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
		opts.Password = c.Password
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

// SetClient sets an existing go redis client to the sessions redis driver.
//
// Returns itself.
func (r *GoRedisDriver) SetClient(goRedisClient GoRedisClient) *GoRedisDriver {
	r.Client = goRedisClient
	return r
}

// Connect initializes the redis client.
func (r *GoRedisDriver) Connect(c Config) error {
	if r.Client != nil { // if a custom one was given through SetClient.
		return nil
	}

	if len(c.Clusters) > 0 {
		r.Client = redis.NewClusterClient(r.mergeClusterOptions(c))
	} else {
		r.Client = redis.NewClient(r.mergeClientOptions(c))
	}

	return nil
}

// PingPong sends a ping message and reports whether
// the PONG message received successfully.
func (r *GoRedisDriver) PingPong() (bool, error) {
	pong, err := r.Client.Ping(defaultContext).Result()
	return pong == "PONG", err
}

// CloseConnection terminates the underline redis connection.
func (r *GoRedisDriver) CloseConnection() error {
	return r.Client.Close()
}

// Set stores a "value" based on the session's "key".
// The value should be type of []byte, so unmarshal can happen.
func (r *GoRedisDriver) Set(sid, key string, value interface{}) error {
	return r.Client.HSet(defaultContext, sid, key, value).Err()
}

// Get returns the associated value of the session's given "key".
func (r *GoRedisDriver) Get(sid, key string) (interface{}, error) {
	return r.Client.HGet(defaultContext, sid, key).Bytes()
}

// Exists reports whether a session exists or not.
func (r *GoRedisDriver) Exists(sid string) bool {
	n, err := r.Client.Exists(defaultContext, sid).Result()
	if err != nil {
		return false
	}

	return n > 0
}

// TTL returns any TTL value of the session.
func (r *GoRedisDriver) TTL(sid string) time.Duration {
	dur, err := r.Client.TTL(defaultContext, sid).Result()
	if err != nil {
		return 0
	}

	return dur
}

// UpdateTTL sets expiration duration of the session.
func (r *GoRedisDriver) UpdateTTL(sid string, newLifetime time.Duration) error {
	_, err := r.Client.Expire(defaultContext, sid, newLifetime).Result()
	return err
}

// GetAll returns all the key values under the session.
func (r *GoRedisDriver) GetAll(sid string) (map[string]string, error) {
	return r.Client.HGetAll(defaultContext, sid).Result()
}

// GetKeys returns all keys under the session.
func (r *GoRedisDriver) GetKeys(sid string) ([]string, error) {
	return r.Client.HKeys(defaultContext, sid).Result()
}

// Len returns the total length of key-values of the session.
func (r *GoRedisDriver) Len(sid string) int {
	return int(r.Client.HLen(defaultContext, sid).Val())
}

// Delete removes a value from the redis store.
func (r *GoRedisDriver) Delete(sid, key string) error {
	if key == "" {
		return r.Client.Del(defaultContext, sid).Err()
	}
	return r.Client.HDel(defaultContext, sid, key).Err()
}
