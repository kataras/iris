package record

import "time"

// Record The structure written to the database
type Record struct {
	Data      []byte
	DeathTime time.Time
}
