package database

type sqlite struct{}

func (db *sqlite) Exec(q string) error { return nil }
