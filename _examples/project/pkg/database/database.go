package database

type DB struct {
	/* ... */
}

func Open(connString string) *DB {
	return &DB{}
}
