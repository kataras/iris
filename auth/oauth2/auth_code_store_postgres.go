package oauth2

import (
	stdContext "context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// postgresAuthCodeStore is a PostgreSQL-based implementation of AuthCodeStore,
// storing and retrieving OAuth2 authorization codes in a PostgreSQL database.
type postgresAuthCodeStore struct {
	// conn is the PostgreSQL database connection, used for executing queries.
	conn PostgresConn

	// tableName is the name of the table to store authorization codes.
	tableName string
	// clientsTableName is the name of the related clients table for foreign key references.
	clientsTableName string
}

// NewPostgresAuthCodeStore creates a new instance of postgresAuthCodeStore using the specified
// clientsTableName, auth code tableName, and a PostgresConn connection.
// It ensures the table structure exists on initialization and returns
// an AuthCodeStore implementation for managing OAuth2 authorization codes.
func NewPostgresAuthCodeStore(clientsTableName, tableName string, conn PostgresConn) AuthCodeStore {
	if tableName == "" {
		panic("table name is required")
	}

	if conn == nil {
		panic("postgres client connection is required")
	}

	store := &postgresAuthCodeStore{
		conn:             conn,
		tableName:        tableName,
		clientsTableName: clientsTableName,
	}

	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 30*time.Second)
	defer cancel()
	if err := store.tryCreateTable(ctx); err != nil {
		panic(fmt.Errorf("failed to create clients table: %w", err))
	}

	return store
}

// tryCreateTable attempts to create the auth code table if it does not already exist,
// including its columns and relationships. It enforces a one-to-many relationship
// between the clients table and the auth codes via ON DELETE CASCADE.
func (s *postgresAuthCodeStore) tryCreateTable(ctx stdContext.Context) error {
	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS "%s" (
		code VARCHAR(255) PRIMARY KEY,
		created_at TIMESTAMP NOT NULL DEFAULT clock_timestamp(),
		updated_at TIMESTAMP NOT NULL DEFAULT clock_timestamp(),
		expires_at TIMESTAMP NOT NULL,
		client_id UUID NOT NULL REFERENCES "%s"(id) ON DELETE CASCADE,
		user_data JSONB NOT NULL,
		state VARCHAR(255) NOT NULL,
		scopes VARCHAR[] NOT NULL,
		code_challenge TEXT NOT NULL DEFAULT '',
		code_challenge_method TEXT NOT NULL DEFAULT ''
	);`, s.tableName, s.clientsTableName)
	_, err := s.conn.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create authorization codes table: %w", err)
	}

	// Tables created before PKCE columns exist need the columns added in place.
	for _, column := range []string{"code_challenge", "code_challenge_method"} {
		alter := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN IF NOT EXISTS %s TEXT NOT NULL DEFAULT ''`, s.tableName, column)
		if _, err = s.conn.Exec(ctx, alter); err != nil {
			return fmt.Errorf("failed to add %s to authorization codes table: %w", column, err)
		}
	}

	if err := createUpdatedAtTriggerFunction(ctx, s.conn); err != nil {
		return err
	}

	return createUpdatedAtTrigger(ctx, s.conn, s.tableName)
}

// InsertAuthCode inserts a new record into the auth code table. The code, timestamps,
// expiration, client ID, user data, state, and scopes are stored. This method
// enables retrieval of the code for the authorization flow.
func (s *postgresAuthCodeStore) InsertAuthCode(ctx stdContext.Context, authCode *AuthCode) error {
	query := fmt.Sprintf(`
    INSERT INTO "%s" (code, expires_at, client_id, user_data, state, scopes, code_challenge, code_challenge_method)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, s.tableName)

	_, err := s.conn.Exec(ctx, query,
		authCode.Code,
		authCode.ExpiresAt,
		authCode.ClientID,
		authCode.UserData,
		authCode.State,
		authCode.Scopes,
		authCode.CodeChallenge,
		authCode.CodeChallengeMethod,
	)
	return err
}

// GetAuthCode fetches a single auth code record from the table by its code value.
// If found, the record is returned along with a bool indicating existence. Otherwise, ok is false.
func (s *postgresAuthCodeStore) GetAuthCode(ctx stdContext.Context, code string) (*AuthCode, bool, error) {
	query := fmt.Sprintf(`SELECT code, created_at, updated_at, expires_at, client_id, user_data, state, scopes, code_challenge, code_challenge_method FROM "%s" WHERE code = $1`, s.tableName)

	row := s.conn.QueryRow(ctx, query, code)
	authCode := &AuthCode{}
	err := row.Scan(
		&authCode.Code,
		&authCode.CreatedAt,
		&authCode.UpdatedAt,
		&authCode.ExpiresAt,
		&authCode.ClientID,
		&authCode.UserData,
		&authCode.State,
		&authCode.Scopes,
		&authCode.CodeChallenge,
		&authCode.CodeChallengeMethod,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	return authCode, true, nil
}

// GetAndDeleteAuthCode retrieves an authorization code by its code value
// and deletes it from the table in a single operation. It returns the retrieved
// auth code, a bool indicating if it was found, and any error encountered.
func (s *postgresAuthCodeStore) GetAndDeleteAuthCode(ctx stdContext.Context, code string) (*AuthCode, bool, error) {
	authCode, ok, err := s.GetAuthCode(ctx, code)
	if !ok || err != nil {
		return nil, ok, err
	}

	query := fmt.Sprintf(`DELETE FROM "%s" WHERE code = $1`, s.tableName)
	_, err = s.conn.Exec(ctx, query, code)
	return authCode, true, err
}
