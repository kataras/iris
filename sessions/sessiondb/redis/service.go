package redis

import (
	"fmt"
	"strconv"
	"time"

	"github.com/kataras/iris/core/errors"

	"github.com/mediocregopher/radix/v3"
)

const (
	// DefaultRedisNetwork the redis network option, "tcp".
	DefaultRedisNetwork = "tcp"
	// DefaultRedisAddr the redis address option, "127.0.0.1:6379".
	DefaultRedisAddr = "127.0.0.1:6379"
	// DefaultRedisTimeout the redis idle timeout option, time.Duration(30) * time.Second
	DefaultRedisTimeout = time.Duration(30) * time.Second
	// DefaultDelim ths redis delim option, "-".
	DefaultDelim = "-"
)

// Config the redis configuration used inside sessions
type Config struct {
	// Network protocol. Defaults to "tcp".
	Network string
	// Addr of the redis server. Defaults to "127.0.0.1:6379".
	Addr string
	// Password string .If no password then no 'AUTH'. Defaults to "".
	Password string
	// If Database is empty "" then no 'SELECT'. Defaults to "".
	Database string
	// MaxActive. Defaults to 10.
	MaxActive int
	// Timeout for connect, write and read, defautls to 30 seconds, 0 means no timeout.
	Timeout time.Duration
	// Prefix "myprefix-for-this-website". Defaults to "".
	Prefix string
	// Delim the delimeter for the keys on the sessiondb. Defaults to "-".
	Delim string
}

// DefaultConfig returns the default configuration for Redis service.
func DefaultConfig() Config {
	return Config{
		Network:   DefaultRedisNetwork,
		Addr:      DefaultRedisAddr,
		Password:  "",
		Database:  "",
		MaxActive: 10,
		Timeout:   DefaultRedisTimeout,
		Prefix:    "",
		Delim:     DefaultDelim,
	}
}

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
	pool   *radix.Pool
}

// newService returns a Redis service filled by the passed config
// to connect call the .Connect().
func newService(cfg ...Config) *Service {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}

	r := &Service{Config: &c}
	return r
}

// Connect connects to the redis, called only once
func (r *Service) Connect() error {
	c := r.Config

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

	customConnFunc := func(network, addr string) (radix.Conn, error) {
		var options []radix.DialOpt

		if c.Password != "" {
			options = append(options, radix.DialAuthPass(c.Password))
		}

		if c.Timeout > 0 {
			options = append(options, radix.DialTimeout(c.Timeout))
		}

		if c.Database != "" { //  *dialOpts.selectDb is not exported on the 3rd-party library,
			// but on its `DialSelectDB` option it does this:
			// do.selectDB = strconv.Itoa(db) -> (string to int)
			// so we can pass that string as int and it should work.
			dbIndex, err := strconv.Atoi(c.Database)
			if err == nil {
				options = append(options, radix.DialSelectDB(dbIndex))
			}

		}

		return radix.Dial(network, addr, options...)
	}

	pool, err := radix.NewPool(c.Network, c.Addr, c.MaxActive, radix.PoolConnFunc(customConnFunc))
	if err != nil {
		return err
	}

	r.Connected = true
	r.pool = pool
	return nil
}

// PingPong sends a ping and receives a pong, if no pong received then returns false and filled error
func (r *Service) PingPong() (bool, error) {
	var msg string
	err := r.pool.Do(radix.Cmd(&msg, "PING"))
	if err != nil {
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
func (r *Service) Set(key string, value interface{}, secondsLifetime int64) error {
	// fmt.Printf("%#+v. %T. %s\n", value, value, value)

	// if vB, ok := value.([]byte); ok && secondsLifetime <= 0 {
	// 	return r.pool.Do(radix.Cmd(nil, "MSET", r.Config.Prefix+key, string(vB)))
	// }

	var cmd radix.CmdAction
	// if has expiration, then use the "EX" to delete the key automatically.
	if secondsLifetime > 0 {
		cmd = radix.FlatCmd(nil, "SETEX", r.Config.Prefix+key, secondsLifetime, value)
	} else {
		cmd = radix.FlatCmd(nil, "SET", r.Config.Prefix+key, value) // MSET same performance...
	}

	return r.pool.Do(cmd)
}

// Get returns value, err by its key
//returns nil and a filled error if something bad happened.
func (r *Service) Get(key string) (interface{}, error) {
	var redisVal interface{}
	mn := radix.MaybeNil{Rcv: &redisVal}

	err := r.pool.Do(radix.Cmd(&mn, "GET", r.Config.Prefix+key))

	if err != nil {
		return nil, err
	}
	if mn.Nil {
		return nil, ErrKeyNotFound.Format(key)
	}
	return redisVal, nil
}

// TTL returns the seconds to expire, if the key has expiration and error if action failed.
// Read more at: https://redis.io/commands/ttl
func (r *Service) TTL(key string) (seconds int64, hasExpiration bool, found bool) {
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

func (r *Service) updateTTLConn(key string, newSecondsLifeTime int64) error {
	var reply int
	err := r.pool.Do(radix.FlatCmd(&reply, "EXPIRE", r.Config.Prefix+key, newSecondsLifeTime))
	if err != nil {
		return err
	}

	// https://redis.io/commands/expire#return-value
	//
	// 1 if the timeout was set.
	// 0 if key does not exist.

	if reply == 1 {
		return nil
	} else if reply == 0 {
		return fmt.Errorf("unable to update expiration, the key '%s' was stored without ttl", key)
	} // do not check for -1.

	return nil
}

// UpdateTTL will update the ttl of a key.
// Using the "EXPIRE" command.
// Read more at: https://redis.io/commands/expire#refreshing-expires
func (r *Service) UpdateTTL(key string, newSecondsLifeTime int64) error {
	return r.updateTTLConn(key, newSecondsLifeTime)
}

// UpdateTTLMany like `UpdateTTL` but for all keys starting with that "prefix",
// it is a bit faster operation if you need to update all sessions keys (although it can be even faster if we used hash but this will limit other features),
// look the `sessions/Database#OnUpdateExpiration` for example.
func (r *Service) UpdateTTLMany(prefix string, newSecondsLifeTime int64) error {
	keys, err := r.getKeys(prefix)
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
func (r *Service) GetAll() (interface{}, error) {
	var redisVal []interface{}
	mn := radix.MaybeNil{Rcv: &redisVal}
	err := r.pool.Do(radix.Cmd(&mn, "SCAN", strconv.Itoa(0))) // 0 -> cursor

	if err != nil {
		return nil, err
	}

	if mn.Nil {
		return nil, err
	}

	return redisVal, nil
}

func (r *Service) getKeys(prefix string) ([]string, error) {
	var keys []string
	// err := r.pool.Do(radix.Cmd(&keys, "MATCH", r.Config.Prefix+prefix+"*"))
	// if err != nil {
	// 	return nil, err
	// }

	scanner := radix.NewScanner(r.pool, radix.ScanOpts{
		Command: "SCAN",
		Pattern: r.Config.Prefix + prefix + r.Config.Delim + "*", // get all of this session except its root sid.
		//	Count: 9999999999,
	})

	var key string
	for scanner.Next(&key) {
		keys = append(keys, key)
	}
	if err := scanner.Close(); err != nil {
		return nil, err
	}

	// if err := c.Send("SCAN", 0, "MATCH", r.Config.Prefix+prefix+"*", "COUNT", 9999999999); err != nil {
	// 	return nil, err
	// }

	// if err := c.Flush(); err != nil {
	// 	return nil, err
	// }

	// reply, err := c.Receive()
	// if err != nil || reply == nil {
	// 	return nil, err
	// }

	// it returns []interface, with two entries, the first one is "0" and the second one is a slice of the keys as []interface{uint8....}.

	// if keysInterface, ok := reply.([]interface{}); ok {
	// 	if len(keysInterface) == 2 {
	// 		// take the second, it must contain the slice of keys.
	// 		if keysSliceAsBytes, ok := keysInterface[1].([]interface{}); ok {
	// 			keys := make([]string, len(keysSliceAsBytes), len(keysSliceAsBytes))
	// 			for i, k := range keysSliceAsBytes {
	// 				keys[i] = fmt.Sprintf("%s", k)[len(r.Config.Prefix):]
	// 			}

	// 			return keys, nil
	// 		}
	// 	}
	// }

	return keys, nil
}

// GetKeys returns all redis keys using the "SCAN" with MATCH command.
// Read more at:  https://redis.io/commands/scan#the-match-option.
func (r *Service) GetKeys(prefix string) ([]string, error) {
	return r.getKeys(prefix)
}

// GetBytes returns value, err by its key
// you can use utils.Deserialize((.GetBytes("yourkey"),&theobject{})
//returns nil and a filled error if something wrong happens
func (r *Service) GetBytes(key string) ([]byte, error) {
	var redisVal []byte
	mn := radix.MaybeNil{Rcv: &redisVal}
	err := r.pool.Do(radix.Cmd(&mn, "GET", r.Config.Prefix+key))
	if err != nil {
		return nil, err
	}
	if mn.Nil {
		return nil, ErrKeyNotFound.Format(key)
	}

	return redisVal, nil
}

// Delete removes redis entry by specific key
func (r *Service) Delete(key string) error {
	err := r.pool.Do(radix.Cmd(nil, "DEL", r.Config.Prefix+key))
	return err
}
