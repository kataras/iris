package oauth2

import (
	stdContext "context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kataras/iris/v14/core/pkg/pagination"
)

// PostgresConn is an interface that defines the methods required to interact with a PostgreSQL database (pool connection).
type PostgresConn interface {
	// Exec executes a query that doesn't return rows.
	Exec(ctx stdContext.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	// QueryRow executes a query that is expected to return at most one row.
	QueryRow(ctx stdContext.Context, sql string, arguments ...any) pgx.Row
	// Query executes a query that returns rows.
	Query(ctx stdContext.Context, sql string, args ...any) (pgx.Rows, error)
}

// postgresClientStore provides an implementation of ClientStore for PostgreSQL databases.
type postgresClientStore struct {
	// conn is the PostgreSQL connection used by the client store.
	conn PostgresConn

	// tableName is the name of the clients table.
	tableName string
}

// NewPostgresClientStore initializes a new postgresClientStore with the given table name
// and ensures the table is created if it doesn’t exist.
func NewPostgresClientStore(tableName string, conn PostgresConn) ClientStore {
	if tableName == "" {
		panic("table name is required")
	}

	if conn == nil {
		panic("postgres client connection is required")
	}

	store := &postgresClientStore{
		conn:      conn,
		tableName: tableName,
	}

	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 30*time.Second) // give a lot of time to wait for the table creation, just in case.
	defer cancel()
	if err := store.tryCreateTable(ctx); err != nil {
		panic(fmt.Errorf("failed to create clients table: %w", err))
	}

	return store
}

var createUpdatedAtTriggerFunctionOnce sync.Once

func createUpdatedAtTriggerFunction(ctx stdContext.Context, conn PostgresConn) (err error) {
	createUpdatedAtTriggerFunctionOnce.Do(func() {
		// Create or replace a trigger function to update 'updated_at' on every UPDATE.
		triggerFuncQuery := `
		CREATE OR REPLACE FUNCTION update_modified_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = clock_timestamp();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;`
		if _, err = conn.Exec(ctx, triggerFuncQuery); err != nil {
			err = fmt.Errorf("failed to create the trigger function: update_modified_column: %w", err)
		}
	})

	return err
}

func createUpdatedAtTrigger(ctx stdContext.Context, conn PostgresConn, tableName string) error {
	// Create the trigger if it doesn't already exist.
	// Note: Postgres doesn't have IF NOT EXISTS for triggers, so this uses a simple query that tries creating unconditionally.
	// If you want a safer approach, you'd query pg_trigger first or wrap in a block.
	triggerQuery := fmt.Sprintf(`
	DO $$
	BEGIN
		CREATE TRIGGER %[1]s_update_modified_column_trigger
		BEFORE UPDATE ON "%[1]s"
		FOR EACH ROW
		EXECUTE PROCEDURE update_modified_column();
	EXCEPTION
		WHEN duplicate_object THEN
			-- Trigger already exists; do nothing
	END
	$$;`,
		tableName,
	)

	if _, err := conn.Exec(ctx, triggerQuery); err != nil {
		return fmt.Errorf("failed to create the update_modified_column trigger: %w", err)
	}

	return nil
}

// tryCreateTable creates the clients table if it does not already exist,
// setting up columns for storing client properties and defaults.
func (s *postgresClientStore) tryCreateTable(ctx stdContext.Context) error {
	query := fmt.Sprintf(`
    CREATE TABLE IF NOT EXISTS "%s" (
        id UUID PRIMARY KEY,
        created_at TIMESTAMP NOT NULL DEFAULT clock_timestamp(),
        updated_at TIMESTAMP NOT NULL DEFAULT clock_timestamp(),
        name VARCHAR(255) NOT NULL UNIQUE,
		website VARCHAR(255) NOT NULL,
		logo_url VARCHAR(255) NOT NULL,
		redirect_uri TEXT NOT NULL,
		verification_uri TEXT NOT NULL DEFAULT '',
		secret VARCHAR(255) NOT NULL,
		grant_types VARCHAR[] NOT NULL,
		scopes VARCHAR[] NOT NULL,
		code_max_age INT4 NOT NULL DEFAULT 3600,
		access_max_age INT4 NOT NULL DEFAULT 3600,
		refresh_max_age INT4 NOT NULL DEFAULT 604800,
		version INT2 NOT NULL DEFAULT 0
    )`, s.tableName)
	_, err := s.conn.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create clients table: %w", err)
	}

	if err := createUpdatedAtTriggerFunction(ctx, s.conn); err != nil {
		return err
	}

	return createUpdatedAtTrigger(ctx, s.conn, s.tableName)
}

// InsertClient stores a new client record in the PostgreSQL table.
func (s *postgresClientStore) InsertClient(ctx stdContext.Context, client *Client) error {
	query := fmt.Sprintf(`
    INSERT INTO "%s" (id, name, website, logo_url, redirect_uri, verification_uri, secret, grant_types, scopes, code_max_age, access_max_age, refresh_max_age, version)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`, s.tableName)

	_, err := s.conn.Exec(ctx, query,
		client.ID,
		client.Name,
		client.Website,
		client.LogoURL,
		client.RedirectURI,
		client.VerificationURI,
		client.Secret,
		client.GrantTypes,
		client.Scopes,
		client.CodeMaxAge,
		client.AccessMaxAge,
		client.RefreshMaxAge,
		client.Version,
	)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation, the name is taken.
		return fmt.Errorf("%w: %s", ErrClientAlreadyExists, client.Name)
	}
	return err
}

// GetClient retrieves a client record by its ID from the PostgreSQL table.
func (s *postgresClientStore) GetClient(ctx stdContext.Context, id string) (*Client, bool, error) {
	return s.getClient(ctx, "id", id)
}

// GetClientByName retrieves a client record by its name from the PostgreSQL table.
func (s *postgresClientStore) GetClientByName(ctx stdContext.Context, name string) (*Client, bool, error) {
	return s.getClient(ctx, "name", name)
}

func (s *postgresClientStore) getClient(ctx stdContext.Context, column, value string) (*Client, bool, error) {
	query := fmt.Sprintf(`SELECT * FROM "%s" WHERE "%s" = $1`, s.tableName, column)

	client, err := scanClient(s.conn.QueryRow(ctx, query, value))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	return client, true, nil
}

// scanClient reads one row of the clients table, selected with SELECT *, into a Client.
func scanClient(s interface{ Scan(dest ...any) error }) (*Client, error) {
	client := &Client{}
	err := s.Scan(
		&client.ID,
		&client.CreatedAt,
		&client.UpdatedAt,
		&client.Name,
		&client.Website,
		&client.LogoURL,
		&client.RedirectURI,
		&client.VerificationURI,
		&client.Secret,
		&client.GrantTypes,
		&client.Scopes,
		&client.CodeMaxAge,
		&client.AccessMaxAge,
		&client.RefreshMaxAge,
		&client.Version,
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// clientsWhere is the WHERE clause shared by ListClients and CountClients.
// $1 is the name text and $2 is the scope, an empty value disables the condition.
const clientsWhere = `WHERE ($1 = '' OR name ILIKE '%' || $1 || '%') AND ($2 = '' OR $2 = ANY(scopes))`

// filterArgs returns the $1 and $2 arguments of clientsWhere.
func filterArgs(filter *ClientFilter) (name, scope string) {
	if filter == nil {
		return "", ""
	}
	return filter.Name, filter.Scope
}

// ListClients returns one page of the matching clients and the total number of matches.
func (s *postgresClientStore) ListClients(ctx stdContext.Context, page pagination.PageOptions, filter *ClientFilter) ([]*Client, int64, error) {
	column, err := filter.sortColumn() // a whitelisted column name, never user input.
	if err != nil {
		return nil, 0, err
	}
	direction := "ASC"
	if filter != nil && filter.SortDescending {
		direction = "DESC"
	}

	total, err := s.CountClients(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`SELECT * FROM "%s" %s ORDER BY %s %s, id LIMIT $3 OFFSET $4`, s.tableName, clientsWhere, column, direction)
	name, scope := filterArgs(filter)
	rows, err := s.conn.Query(ctx, query, name, scope, page.GetLimit(), page.GetOffset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	clients := make([]*Client, 0, page.GetLimit())
	for rows.Next() {
		client, err := scanClient(rows)
		if err != nil {
			return nil, 0, err
		}
		clients = append(clients, client)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}

// CountClients returns the number of matching clients.
func (s *postgresClientStore) CountClients(ctx stdContext.Context, filter *ClientFilter) (int64, error) {
	if _, err := filter.sortColumn(); err != nil {
		return 0, err
	}

	query := fmt.Sprintf(`SELECT COUNT(*) FROM "%s" %s`, s.tableName, clientsWhere)
	name, scope := filterArgs(filter)
	var total int64
	if err := s.conn.QueryRow(ctx, query, name, scope).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// DeleteClient removes a client record by its ID.
func (s *postgresClientStore) DeleteClient(ctx stdContext.Context, id string) (bool, error) {
	return s.execOne(ctx, fmt.Sprintf(`DELETE FROM "%s" WHERE id = $1`, s.tableName), id)
}

// UpdateClientScopes replaces the scopes of a client record, the secret and the version are untouched.
func (s *postgresClientStore) UpdateClientScopes(ctx stdContext.Context, id string, scopes []string) (bool, error) {
	if scopes == nil {
		scopes = []string{} // the column is NOT NULL.
	}
	return s.execOne(ctx, fmt.Sprintf(`UPDATE "%s" SET scopes = $2 WHERE id = $1`, s.tableName), id, scopes)
}

// IncrementClientVersion adds one to the version of a client record.
func (s *postgresClientStore) IncrementClientVersion(ctx stdContext.Context, id string) (bool, error) {
	return s.execOne(ctx, fmt.Sprintf(`UPDATE "%s" SET version = version + 1 WHERE id = $1`, s.tableName), id)
}

// execOne runs a statement and reports whether it affected a row.
func (s *postgresClientStore) execOne(ctx stdContext.Context, query string, args ...any) (bool, error) {
	tag, err := s.conn.Exec(ctx, query, args...)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
