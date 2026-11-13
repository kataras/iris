package sessions

import (
	stdContext "context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/uid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type (
	// postgresDatabase is a PostgreSQL database implementation for session management.
	postgresDatabase[T any] struct {
		client          PostgresConn
		Options         Options
		tableName       string // default is "sessions".
		shiftExpiration bool   // Set to true to shift the expiration of the session to the current time plus the MaxAge on Commit.
	}

	// PostgresConn is an interface that defines the methods required to interact with a PostgreSQL database.
	PostgresConn interface {
		// Exec executes a query that doesn't return rows.
		Exec(ctx stdContext.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
		// QueryRow executes a query that is expected to return at most one row.
		QueryRow(ctx stdContext.Context, sql string, arguments ...any) pgx.Row
	}

	// SQLDatabase is an interface that extends the Database interface with methods specific to SQL databases.
	SQLDatabase[T any] interface {
		Database[T]
		// ShiftExpiration sets whether the session expiration should be shifted to the current time plus the MaxAge on Commit.
		// Handler commits a session after a request that changed it, so the expiration moves with a change
		// (or with a Commit called by hand), not with every request.
		ShiftExpiration(b bool)
		// StartCleanupScheduler starts a scheduler that periodically cleans up expired sessions from the database.
		StartCleanupScheduler(ctx stdContext.Context, interval time.Duration) <-chan error
	}
)

var _ Database[any] = (*postgresDatabase[any])(nil)

// DefaultTableName is the default name of the sessions table in the (Postgre)SQL database.
const DefaultTableName = "iris_sessions"

// NewPostgresDatabase creates and returns a new PostgreSQL database instance for session management with the provided options.
// It registers the generic type T with the gob package for encoding and decoding session data.
//
// It panics if
// - the table name is empty or
// - the PostgreSQL client is nil or
// - it fails to create the sessions table in the PostgreSQL database.
//
// Usage:
//
//	db := sessions.NewPostgresDatabase(sessions.DefaultTableName, pgDB, sessions.NewOptions(keyPair...))
func NewPostgresDatabase[T any](tableName string, conn PostgresConn, opts Options) SQLDatabase[T] {
	if tableName == "" {
		panic("table name is required")
	}

	if conn == nil {
		panic("postgres client is required")
	}

	var t T
	gob.Register(t)
	gob.Register(SessionData[T]{})

	db := &postgresDatabase[T]{
		client:    conn,
		Options:   opts.Apply(),
		tableName: tableName,
	}

	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 30*time.Second) // give a lot of time to wait for the table creation, just in case.
	defer cancel()
	if err := db.tryCreateTable(ctx); err != nil {
		panic(fmt.Errorf("failed to create sessions table: %w", err))
	}

	return db
}

// tryCreateTable creates the sessions table in the PostgreSQL database if it does not already exist.
// The table includes columns for session ID, creation time, expiration time, name, and value.
func (db *postgresDatabase[T]) tryCreateTable(ctx stdContext.Context) error {
	createTableQuery := `
    CREATE TABLE IF NOT EXISTS "` + db.tableName + `" (
        id UUID PRIMARY KEY,
        created_at TIMESTAMP NOT NULL DEFAULT clock_timestamp(),
        expires_at TIMESTAMP NOT NULL,
        name TEXT NOT NULL,
        value BYTEA NOT NULL,
        UNIQUE (id, name)
    );`
	// CREATE INDEX IF NOT EXISTS idx_name ON "` + db.tableName + `" (name);`
	// However, since we already have a unique constraint on the combination of id and name,
	// adding a separate index on name is NOT be necessary and it will slow down the insertions.
	_, err := db.client.Exec(ctx, createTableQuery)
	return err
}

func (db *postgresDatabase[T]) ShiftExpiration(b bool) {
	db.shiftExpiration = b
}

// upsertSession inserts a new session or updates an existing session in the PostgreSQL database.
// The session expiration time is calculated based on the Options.MaxAge value.
func (db *postgresDatabase[T]) upsertSession(ctx stdContext.Context, id string, name string, value string) error {
	insertQuery := `
    INSERT INTO "` + db.tableName + `" (id, expires_at, name, value)
    VALUES ($1, NOW() + $4, $2, $3)
    ON CONFLICT (id, name) DO UPDATE
    SET value = EXCLUDED.value`
	if db.shiftExpiration && db.Options.MaxAge > 0 {
		insertQuery += `, expires_at = NOW() + $4`
	}
	insertQuery += `;` // end of the query.

	_, err := db.client.Exec(ctx, insertQuery, id, name, []byte(value), db.Options.MaxAge)
	return err
}

// retrieveSessionData retrieves the session's value from the PostgreSQL database based on the session ID and name.
func (db *postgresDatabase[T]) retrieveSessionData(ctx stdContext.Context, id string, name string) (encodedData []byte, err error) {
	selectQuery := ` SELECT value FROM "` + db.tableName + `" 
    WHERE id = $1 AND name = $2 AND expires_at > NOW() LIMIT 1;`

	err = db.client.QueryRow(ctx, selectQuery, id, name).Scan(&encodedData)
	return
}

// deleteSession deletes a session from the PostgreSQL database based on the session ID and name.
func (db *postgresDatabase[T]) deleteSession(ctx stdContext.Context, id string, name string) error {
	deleteQuery := `DELETE FROM "` + db.tableName + `" WHERE id = $1 AND name = $2;`

	_, err := db.client.Exec(ctx, deleteQuery, id, name)
	return err
}

// cleanupExpiredSessions deletes all expired sessions from the PostgreSQL database.
func (db *postgresDatabase[T]) cleanupExpiredSessions(ctx stdContext.Context) error {
	cleanupQuery := `DELETE FROM "` + db.tableName + `" WHERE expires_at < NOW();`
	_, err := db.client.Exec(ctx, cleanupQuery)
	return err
}

// StartCleanupScheduler starts a scheduler that periodically cleans up expired sessions from the PostgreSQL database.
// It runs the cleanup job at the specified interval and returns a read-only error channel to report any errors.
// The scheduler stops when the context is canceled; the error channel is
// closed then, so ranging over it (as below) terminates. Errors are reported
// through a blocking send, but a pending error is dropped on cancel: cancel
// never waits for a slow (or absent) receiver.
// Usage:
//
//	errCh := db.StartCleanupScheduler(ctx, 4*time.Hour)
//	go func() {
//		for err := range errCh {
//			// handle the error
//		}
//	}()
func (db *postgresDatabase[T]) StartCleanupScheduler(ctx stdContext.Context, interval time.Duration) <-chan error {
	errCh := make(chan error)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(errCh)

		for {
			select {
			case <-ticker.C:
				err := db.cleanupExpiredSessions(ctx)
				if err != nil {
					select {
					case errCh <- fmt.Errorf("failed to clean up expired sessions: %w", err):
					case <-ctx.Done():
						// The caller stopped draining: drop the error
						// instead of blocking on the send forever.
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return errCh
}

// Notes:
// id (session id) unique(id,name)
// create_at
// updated_at (keep track ttl too) OR expires_at (timestamp(z))
// name (cookie name) unique(id,name)
// value (the encoded session value and flash messages)
//
// select query:
// select * from sessions where id = $1 and name = $2 and expires_at > now() limit 1; -- // and updated_at > now() - interval '1 day';
// cron delete query:
// delete from sessions where expires_at < now();

// Get returns a session from the request context or the postgres database.
func (db *postgresDatabase[T]) Get(ctx *context.Context, cookieName string) (*Session[T], error) {
	return Load(ctx, db, cookieName)
}

// Load loads the session from the postgres database.
func (db *postgresDatabase[T]) Load(ctx *context.Context, cookieName string, cookieValue string) (*Session[T], error) {
	session, err := db.load(ctx, cookieName, cookieValue)
	if err != nil {
		err = fmt.Errorf("load session from postgres failed: %s: %w", cookieName, err)
	}

	return session, err
}

func (db *postgresDatabase[T]) load(ctx *context.Context, cookieName string, cookieValue string) (*Session[T], error) {
	session := NewSession[T](db.Options)
	// bind session id.
	//
	// A cookie that does not decode (signed with other keys, older than MaxAge,
	// not a session id) is treated like no cookie: the visitor cannot fix it,
	// so this is a new session and the next Commit replaces the cookie.
	// So is an id that is not a UUID, which the id column would fail the query
	// for: a v12 cookie signed with the same keys decodes to such an id.
	if cookieValue == "" || db.Options.Decode(cookieName, cookieValue, &session.ID) != nil || !isValidSessionID(session.ID) {
		session.ID = uid.NewV7String() // time-ordered ids for storage-key locality.
		session.IsNew = true
		return session, nil
	}

	// bind data and any flash messages.
	result, err := db.retrieveSessionData(ctx, session.ID, cookieName) // assign the error.
	if err == nil {
		err = db.Options.Decode(cookieName, string(result), &session.SessionData) // assign the error.
		session.decoded = err == nil
	} else if err == pgx.ErrNoRows { // Cookie exists in the browser but not in the database for some reason.
		// we don't care about this error, it's just a new session and Commit will create it if it's missing,
		// however for security reasons we assign a new session id here.
		err = nil
		session.ID = uid.NewV7String()
	}

	return session, err
}

// Commit saves the session to the postgres database.
func (db *postgresDatabase[T]) Commit(ctx *context.Context, session *Session[T]) error {
	if err := db.commit(ctx, session.ID, session); err != nil {
		return fmt.Errorf("commit session to postgres failed: %s: %w", session.ID, err)
	}

	return nil
}

func (db *postgresDatabase[T]) commit(ctx *context.Context, id string, session *Session[T]) error {
	if session.Options.MaxAge < 0 {
		// delete the session record entirely.
		return db.deleteSession(ctx, id, session.Name())
	}

	encodedData, err := db.Options.Encode(session.Name(), session.SessionData)
	if err != nil {
		return err
	}

	return db.upsertSession(ctx, session.ID, session.Name(), encodedData)
}
