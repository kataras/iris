package service

import (
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/errors"
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
	Config *config.Redis
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
	return ErrRedisClosed.Return()
}

// Set sets to the redis
// key string, value string, you can use utils.Serialize(&myobject{}) to convert an object to []byte
func (r *Service) Set(key string, value []byte, maxageseconds ...float64) (err error) { // map[interface{}]interface{}) (err error) {
	maxage := config.DefaultRedisMaxAgeSeconds //1 year
	c := r.pool.Get()
	defer c.Close()
	if err = c.Err(); err != nil {
		return
	}
	if len(maxageseconds) > 0 {
		if max := maxageseconds[0]; max >= 0 {
			maxage = max
		}
	}
	_, err = c.Do("SETEX", r.Config.Prefix+key, maxage, value)
	return
}

// Get returns value, err by its key
// you can use utils.Deserialize((.Get("yourkey"),&theobject{})
//returns nil and a filled error if something wrong happens
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

// GetString returns value, err by its key
// you can use utils.Deserialize((.GetString("yourkey"),&theobject{})
//returns empty string and a filled error if something wrong happens
func (r *Service) GetString(key string) (string, error) {
	redisVal, err := r.Get(key)
	if redisVal == nil {
		return "", ErrKeyNotFound.Format(key)
	}

	sVal, err := redis.String(redisVal, err)
	if err != nil {
		return "", err
	}
	return sVal, nil
}

// GetInt returns value, err by its key
// you can use utils.Deserialize((.GetInt("yourkey"),&theobject{})
//returns -1 int and a filled error if something wrong happens
func (r *Service) GetInt(key string) (int, error) {
	redisVal, err := r.Get(key)
	if redisVal == nil {
		return -1, ErrKeyNotFound.Format(key)
	}

	intVal, err := redis.Int(redisVal, err)
	if err != nil {
		return -1, err
	}
	return intVal, nil
}

// GetStringMap returns map[string]string, err by its key
//returns nil  and a filled error if something wrong happens
func (r *Service) GetStringMap(key string) (map[string]string, error) {
	redisVal, err := r.Get(key)
	if redisVal == nil {
		return nil, ErrKeyNotFound.Format(key)
	}

	_map, err := redis.StringMap(redisVal, err)
	if err != nil {
		return nil, err
	}
	return _map, nil
}

// GetAll returns all keys and their values from a specific key (map[string]string)
// returns a filled error if something bad happened
func (r *Service) GetAll(key string) (map[string]string, error) {
	c := r.pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		return nil, err
	}

	reply, err := c.Do("HGETALL", r.Config.Prefix+key)

	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, ErrKeyNotFound.Format(key)
	}

	return redis.StringMap(reply, err)

}

// GetAllKeysByPrefix returns all []string keys by a key prefix from the redis
func (r *Service) GetAllKeysByPrefix(prefix string) ([]string, error) {
	c := r.pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		return nil, err
	}

	reply, err := c.Do("KEYS", r.Config.Prefix+prefix)

	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, ErrKeyNotFound.Format(prefix)
	}
	return redis.Strings(reply, err)

}

// Delete removes redis entry by specific key
func (r *Service) Delete(key string) error {
	c := r.pool.Get()
	defer c.Close()
	if _, err := c.Do("DEL", r.Config.Prefix+key); err != nil {
		return err
	}
	return nil
}

func dial(network string, addr string, pass string) (redis.Conn, error) {
	if network == "" {
		network = config.DefaultRedisNetwork
	}
	if addr == "" {
		addr = config.DefaultRedisAddr
	}
	c, err := redis.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	if pass != "" {
		if _, err := c.Do("AUTH", pass); err != nil {
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
		c.IdleTimeout = config.DefaultRedisIdleTimeout
	}

	if c.Network == "" {
		c.Network = config.DefaultRedisNetwork
	}

	if c.Addr == "" {
		c.Addr = config.DefaultRedisAddr
	}

	if c.MaxAgeSeconds <= 0 {
		c.MaxAgeSeconds = config.DefaultRedisMaxAgeSeconds
	}

	pool := &redis.Pool{IdleTimeout: config.DefaultRedisIdleTimeout, MaxIdle: c.MaxIdle, MaxActive: c.MaxActive}
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
			if _, err := red.Do("SELECT", c.Database); err != nil {
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
// to connect call the .Connect()
func New(cfg ...config.Redis) *Service {
	c := config.DefaultRedis().Merge(cfg)
	r := &Service{pool: &redis.Pool{}, Config: &c}
	return r
}
