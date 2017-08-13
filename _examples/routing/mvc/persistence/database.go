package persistence

// Database is our imaginary storage.
type Database struct {
	Connstring string
}

func OpenDatabase(connstring string) *Database {
	return &Database{Connstring: connstring}
}
