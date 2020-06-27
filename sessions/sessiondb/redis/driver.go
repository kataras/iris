package redis

// Driver is the interface which each supported redis client
// should support in order to be used in the redis session database.
type Driver interface {
	Connect(c Config) error
	PingPong() (bool, error)
	CloseConnection() error
	Set(key string, value interface{}, secondsLifetime int64) error
	Get(key string) (interface{}, error)
	TTL(key string) (seconds int64, hasExpiration bool, found bool)
	UpdateTTL(key string, newSecondsLifeTime int64) error
	UpdateTTLMany(prefix string, newSecondsLifeTime int64) error
	GetAll() (interface{}, error)
	GetKeys(prefix string) ([]string, error)
	Delete(key string) error
}

var (
	_ Driver = (*RedigoDriver)(nil)
	_ Driver = (*RadixDriver)(nil)
	_ Driver = (*RadixClusterDriver)(nil)
)

// Redigo returns the driver for the redigo go redis client.
// Which is the default one.
// You can customize further any specific driver's properties.
func Redigo() *RedigoDriver {
	return &RedigoDriver{}
}

// Radix returns the driver for the radix go redis client.
func Radix() *RadixDriver {
	return &RadixDriver{}
}

// RadixCluster returns the driver for the radix go redis client(only support redis cluster).
func RadixCluster() *RadixClusterDriver {
	return &RadixClusterDriver{}
}
