package redis

import (
	"fmt"
	"github.com/mediocregopher/radix/v3"
)

// RadixDriver the Redis service based on the radix go client,
// contains the config and the redis cluster.
type RadixDriverHashed struct {
	Connected bool   //Connected is true when the Service has already connected
	Config    Config //Config the read-only redis database config.
	pool      radixPool
}

// Connect connects to the redis, called only once
func (r *RadixDriverHashed) Connect(c Config) error {
	if c.Timeout < 0 {
		c.Timeout = DefaultRedisTimeout
	}
	if c.Network == "" {
		c.Network = DefaultRedisNetwork
	}
	if c.Addr == "" {
		c.Addr = DefaultRedisAddr
	}
	if c.MaxActive == 0 {
		c.MaxActive = 10
	}

	var options []radix.DialOpt
	if c.Password != "" {
		options = append(options, radix.DialAuthPass(c.Password))
	}
	if c.Timeout > 0 {
		options = append(options, radix.DialTimeout(c.Timeout))
	}
	if c.TLSConfig != nil {
		options = append(options, radix.DialUseTLS(c.TLSConfig))
	}

	var pool radixPool
	var connFunc radix.ConnFunc
	connFunc = func(network, addr string) (radix.Conn, error) {
		return radix.Dial(network, addr, options...)
	}
	if len(c.Clusters) > 0 {
		poolFunc := func(network, addr string) (radix.Client, error) {
			return radix.NewPool(network, addr, c.MaxActive, radix.PoolConnFunc(connFunc))
		}
		cluster, err := radix.NewCluster(c.Clusters, radix.ClusterPoolFunc(poolFunc))
		if err != nil {
			return err
		}
		pool = cluster
	} else {
		p, err := radix.NewPool(c.Network, c.Addr, c.MaxActive, radix.PoolConnFunc(connFunc))
		if err != nil {
			return err
		}
		pool = p
	}

	r.Connected = true
	r.pool = pool
	r.Config = c
	return nil
}

// PingPong sends a ping and receives a pong, if no pong received then returns false and filled error
func (r *RadixDriverHashed) PingPong() (bool, error) {
	var msg string
	err := r.pool.Do(radix.Cmd(&msg, "PING"))
	if err != nil {
		return false, err
	}

	return (msg == "PONG"), nil
}

// CloseConnection closes the redis connection.
func (r *RadixDriverHashed) CloseConnection() error {
	if r.pool != nil {
		return r.pool.Close()
	}

	return ErrRedisClosed
}

// Using the "HSET key field value" command.
// The expiration is setted by the secondsLifetime.
func (r *RadixDriverHashed) Set(key, field string, value interface{}, secondsLifetime int64) error {
	var cmd radix.CmdAction

	cmd = radix.FlatCmd(nil, "HMSET", r.Config.Prefix+key, field, value)
	err := r.pool.Do(cmd)
	if err != nil {
		return err
	}
	if secondsLifetime > 0 {
		cmd = radix.FlatCmd(nil, "EXPIRE", r.Config.Prefix+key, secondsLifetime)
		return r.pool.Do(cmd)
	}

	return nil
}

// Using the "HGET key field" command.
// returns nil and a filled error if something bad happened.
func (r *RadixDriverHashed) Get(key, field string) (interface{}, error) {
	var redisVal interface{}
	mn := radix.MaybeNil{Rcv: &redisVal}

	err := r.pool.Do(radix.Cmd(&mn, "HGET", r.Config.Prefix+key, field))
	if err != nil {
		return nil, err
	}
	if mn.Nil {
		return nil, fmt.Errorf("%s %s: %w", r.Config.Prefix+key, field, ErrKeyNotFound)
	}

	return redisVal, nil
}

// TTL returns the seconds to expire, if the key has expiration and error if action failed.
// Read more at: https://redis.io/commands/ttl
func (r *RadixDriverHashed) TTL(key string) (seconds int64, hasExpiration bool, found bool) {
	var redisVal interface{}
	err := r.pool.Do(radix.Cmd(&redisVal, "TTL", r.Config.Prefix+key))
	if err != nil {
		return -2, false, false
	}
	seconds = redisVal.(int64)
	// if -1 means the key has unlimited life time.
	hasExpiration = seconds > -1
	// if -2 means key does not exist.
	found = seconds != -2
	return
}

func (r *RadixDriverHashed) updateTTLConn(key string, newSecondsLifeTime int64) error {
	var reply int
	err := r.pool.Do(radix.FlatCmd(&reply, "EXPIRE", r.Config.Prefix+key, newSecondsLifeTime))
	if err != nil {
		return err
	}

	// 1 if the timeout was set.
	// 0 if key does not exist.
	if reply == 1 {
		return nil
	} else if reply == 0 {
		return fmt.Errorf("unable to update expiration, the key '%s' was stored without ttl", key)
	} // do not check for -1.

	return nil
}

// Using the "HKEYS key" command.
func (r *RadixDriverHashed) getKeys(key string) ([]string, error) {
	var res []string
	err := r.pool.Do(radix.FlatCmd(&res, "HKEYS", r.Config.Prefix+key))
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Using the "EXPIRE" command.
func (r *RadixDriverHashed) UpdateTTL(key string, newSecondsLifeTime int64) error {
	return r.updateTTLConn(key, newSecondsLifeTime)
}

// UpdateTTLMany like `UpdateTTL` all keys.
// look the `sessions/Database#OnUpdateExpiration` for example.
func (r *RadixDriverHashed) UpdateTTLMany(key string, newSecondsLifeTime int64) error {
	return r.updateTTLConn(key, newSecondsLifeTime)
}

// GetKeys returns all redis hash keys using the "HKEYS key" with MATCH command.
func (r *RadixDriverHashed) GetKeys(key string) ([]string, error) {
	return r.getKeys(key)
}

// Using the "HDEL key field1" command.
// Delete removes redis entry by specific key
func (r *RadixDriverHashed) Delete(key, field string) error {
	return r.pool.Do(radix.Cmd(nil, "HDEL", r.Config.Prefix+key, field))
}

// Using the "DEL key" command.
func (r *RadixDriverHashed) Clear(key string) error {
	return r.pool.Do(radix.Cmd(nil, "DEL", r.Config.Prefix+key))
}

// Using the "HLEN key" command.
func (r *RadixDriverHashed) Len(key string) (int, error) {
	var length int
	err := r.pool.Do(radix.FlatCmd(&length, "HLEN", r.Config.Prefix+key))
	if err != nil {
		return 0, err
	}
	return length, nil
}

func (r *RadixDriverHashed) DeleteByKey(key string) error {
	return nil
}

func (r *RadixDriverHashed) GetAll() (interface{}, error) {
	return nil, nil
}

func (r *RadixDriverHashed) GetByKey(key string) (interface{}, error) {
	return nil, nil
}
