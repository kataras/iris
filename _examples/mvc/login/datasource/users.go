// file: datasource/users.go

package datasource

import (
	"errors"

	"github.com/kataras/iris/_examples/mvc/login/datamodels"
)

// Engine is from where to fetch the data, in this case the users.
type Engine uint32

const (
	// Memory stands for simple memory location;
	// map[int64] datamodels.User ready to use, it's our source in this example.
	Memory Engine = iota
	// Bolt for boltdb source location.
	Bolt
	// MySQL for mysql-compatible source location.
	MySQL
)

// LoadUsers returns all users(empty map) from the memory, for the shake of simplicty.
func LoadUsers(engine Engine) (map[int64]datamodels.User, error) {
	if engine != Memory {
		return nil, errors.New("for the shake of simplicity we're using a simple map as the data source")
	}

	return make(map[int64]datamodels.User), nil
}
