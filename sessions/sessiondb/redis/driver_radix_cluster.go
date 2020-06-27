package redis

import (
	"errors"
	"fmt"
	"github.com/mediocregopher/radix/v3"
	"strconv"
	"time"
)

// RadixDriver the Redis service based on the radix go client,
// contains the config and the redis cluster.
type RadixClusterDriver struct {
	Connected bool   //Connected is true when the Service has already connected
	Config    Config //Config the read-only redis database config.
	cluster   *radix.Cluster
}

// Connect connects to the redis, called only once
func (r *RadixClusterDriver) Connect(c Config) error {
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
	if c.Delim == "" {
		c.Delim = DefaultDelim
	}
	if len(c.Clusters) < 1 {
		return errors.New("cluster empty")
	}

	var options []radix.DialOpt
	if c.Password != "" {
		options = append(options, radix.DialAuthPass(c.Password))
	}
	if c.Timeout > 0 {
		options = append(options, radix.DialTimeout(c.Timeout))
	}

	customConnFunc := func(network, addr string) (radix.Conn, error) {
		return radix.Dial(network, addr,
			radix.DialTimeout(1*time.Minute),
			radix.DialAuthPass(c.Password),
		)
	}
	poolFunc := func(network, addr string) (radix.Client, error) {
		return radix.NewPool(network, addr, c.MaxActive, radix.PoolConnFunc(customConnFunc))
	}
	cluster, err := radix.NewCluster(c.Clusters, radix.ClusterPoolFunc(poolFunc))
	if err != nil {
		return err
	}

	r.Connected = true
	r.cluster = cluster
	r.Config = c
	return nil
}

// PingPong sends a ping and receives a pong, if no pong received then returns false and filled error
func (r *RadixClusterDriver) PingPong() (bool, error) {
	var msg string
	err := r.cluster.Do(radix.Cmd(&msg, "PING"))
	if err != nil {
		return false, err
	}

	return (msg == "PONG"), nil
}

// CloseConnection closes the redis connection.
func (r *RadixClusterDriver) CloseConnection() error {
	if r.cluster != nil {
		return r.cluster.Close()
	}

	return ErrRedisClosed
}

// Set sets a key-value to the redis store.
// The expiration is setted by the secondsLifetime.
func (r *RadixClusterDriver) Set(key string, value interface{}, secondsLifetime int64) error {
	var cmd radix.CmdAction
	// if has expiration, then use the "EX" to delete the key automatically.
	if secondsLifetime > 0 {
		cmd = radix.FlatCmd(nil, "SETEX", r.Config.Prefix+key, secondsLifetime, value)
	} else {
		cmd = radix.FlatCmd(nil, "SET", r.Config.Prefix+key, value) // MSET same performance...
	}

	return r.cluster.Do(cmd)
}

// Get returns value, err by its key
// returns nil and a filled error if something bad happened.
func (r *RadixClusterDriver) Get(key string) (interface{}, error) {
	var redisVal interface{}
	mn := radix.MaybeNil{Rcv: &redisVal}

	err := r.cluster.Do(radix.Cmd(&mn, "GET", r.Config.Prefix+key))
	if err != nil {
		return nil, err
	}
	if mn.Nil {
		return nil, fmt.Errorf("%s: %w", key, ErrKeyNotFound)
	}

	return redisVal, nil
}

// TTL returns the seconds to expire, if the key has expiration and error if action failed.
// Read more at: https://redis.io/commands/ttl
func (r *RadixClusterDriver) TTL(key string) (seconds int64, hasExpiration bool, found bool) {
	var redisVal interface{}
	err := r.cluster.Do(radix.Cmd(&redisVal, "TTL", r.Config.Prefix+key))
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

func (r *RadixClusterDriver) updateTTLConn(key string, newSecondsLifeTime int64) error {
	var reply int
	err := r.cluster.Do(radix.FlatCmd(&reply, "EXPIRE", r.Config.Prefix+key, newSecondsLifeTime))
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

func (r RadixClusterDriver) getKeys(cursor, prefix string) ([]string, error) {
	var res scanResult
	err := r.cluster.Do(radix.Cmd(&res, "SCAN", cursor, "MATCH", r.Config.Prefix+prefix+"", "COUNT", "300000"))
	if err != nil {
		return nil, err
	}

	keys := res.keys[0:]
	if res.cur != "0" {
		moreKeys, err := r.getKeys(res.cur, prefix)
		if err != nil {
			return nil, err
		}

		keys = append(keys, moreKeys...)
	}

	return keys, nil
}

// UpdateTTL will update the ttl of a key.
// Using the "EXPIRE" command.
// Read more at: https://redis.io/commands/expire#refreshing-expires
func (r *RadixClusterDriver) UpdateTTL(key string, newSecondsLifeTime int64) error {
	return r.updateTTLConn(key, newSecondsLifeTime)
}

// UpdateTTLMany like UpdateTTL but for all keys starting with that "prefix",
// it is a bit faster operation if you need to update all sessions keys (although it can be even faster if we used hash but this will limit other features),
// look the sessions/Database#OnUpdateExpiration for example.
func (r *RadixClusterDriver) UpdateTTLMany(prefix string, newSecondsLifeTime int64) error {
	keys, err := r.getKeys("0", prefix)
	if err != nil {
		return err
	}

	for _, key := range keys {
		if err = r.updateTTLConn(key, newSecondsLifeTime); err != nil { // fail on first error.
			return err
		}
	}

	return err
}

// GetAll returns all redis entries using the "SCAN" command (2.8+).
func (r *RadixClusterDriver) GetAll() (interface{}, error) {
	var redisVal []interface{}
	mn := radix.MaybeNil{Rcv: &redisVal}
	err := r.cluster.Do(radix.Cmd(&mn, "SCAN", strconv.Itoa(0))) // 0 -> cursor
	if err != nil {
		return nil, err
	}

	if mn.Nil {
		return nil, err
	}

	return redisVal, nil
}

// GetKeys returns all redis keys using the "SCAN" with MATCH command.
// Read more at: https://redis.io/commands/scan#the-match-option.
func (r *RadixClusterDriver) GetKeys(prefix string) ([]string, error) {
	return r.getKeys("0", prefix)
}

// Delete removes redis entry by specific key
func (r *RadixClusterDriver) Delete(key string) error {
	return r.cluster.Do(radix.Cmd(nil, "DEL", r.Config.Prefix+key))
}
