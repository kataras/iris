package redis_hash

// Driver is the interface which each supported redis client
// should support in order to be used in the redis session database.
type Driver interface {
	Connect(c Config) error
	PingPong() (bool, error)
	CloseConnection() error
	Set(key, field string, value interface{}, secondsLifetime int64) error
	Get(key, field string) (interface{}, error)
	TTL(key string) (seconds int64, hasExpiration bool, found bool)
	UpdateTTL(key string, newSecondsLifeTime int64) error
	UpdateTTLMany(key string, newSecondsLifeTime int64) error
	GetKeys(key string) ([]string, error)
	Delete(key, field string) error
	Clear(key string) error
	Len(key string) (int, error)
}

var (
	_ Driver = (*RadixDriver)(nil)
)

func Radix() *RadixDriver {
	return &RadixDriver{}
}
