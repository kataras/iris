package redis

import "time"

// Driver is the interface which each supported redis client
// should support in order to be used in the redis session database.
type Driver interface {
	Connect(c Config) error
	PingPong() (bool, error)
	CloseConnection() error
	Set(sid, key string, value interface{}) error
	Get(sid, key string) (interface{}, error)
	Exists(sid string) bool
	TTL(sid string) time.Duration
	UpdateTTL(sid string, newLifetime time.Duration) error
	GetAll(sid string) (map[string]string, error)
	GetKeys(sid string) ([]string, error)
	Len(sid string) int
	Delete(sid, key string) error
}

var (
	_ Driver = (*GoRedisDriver)(nil)
)

// GoRedis returns the default Driver for the redis sessions database
// It's the go-redis client. Learn more at: https://github.com/go-redis/redis.
func GoRedis() *GoRedisDriver {
	return &GoRedisDriver{}
}
