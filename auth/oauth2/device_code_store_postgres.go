package oauth2

import (
	stdContext "context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// postgresDeviceCodeStore implements DeviceCodeStore using a PostgreSQL database.
type postgresDeviceCodeStore struct {
	// conn holds the connection interface to a PostgreSQL database.
	conn PostgresConn

	// tableName is the name of the table holding device codes.
	tableName string
	// clientsTableName references the clients table for foreign keys.
	clientsTableName string
}

// NewPostgresDeviceCodeStore creates a new postgresDeviceCodeStore instance and
// ensures the associated table is created if it doesn't exist.
func NewPostgresDeviceCodeStore(clientsTableName, tableName string, conn PostgresConn) DeviceCodeStore {
	if tableName == "" {
		panic("table name is required")
	}

	if conn == nil {
		panic("postgres client connection is required")
	}

	store := &postgresDeviceCodeStore{
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

// tryCreateTable attempts to create the device code table if it doesn't already exist,
// ensuring columns and constraints (such as ON DELETE CASCADE for client references).
func (s *postgresDeviceCodeStore) tryCreateTable(ctx stdContext.Context) error {
	createTableQuery := `
    CREATE TABLE IF NOT EXISTS "` + s.tableName + `" (
        code VARCHAR(255) PRIMARY KEY,
        created_at TIMESTAMP NOT NULL DEFAULT clock_timestamp(),
        updated_at TIMESTAMP NOT NULL DEFAULT clock_timestamp(),
		expires_at TIMESTAMP NOT NULL,
		client_id UUID NOT NULL REFERENCES "` + s.clientsTableName + `"(id) ON DELETE CASCADE,
		user_data JSONB NOT NULL,
		user_code VARCHAR(255) NOT NULL,
		scopes VARCHAR[] NOT NULL,
		interval INT NOT NULL DEFAULT 0
    );`
	_, err := s.conn.Exec(ctx, createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create device codes table: %w", err)
	}

	if err := createUpdatedAtTriggerFunction(ctx, s.conn); err != nil {
		return err
	}

	return createUpdatedAtTrigger(ctx, s.conn, s.tableName)
}

// InsertDeviceCode inserts a new device code record into the PostgreSQL table.
func (s *postgresDeviceCodeStore) InsertDeviceCode(ctx stdContext.Context, device *DeviceCode) error {
	insertQuery := `
		INSERT INTO "` + s.tableName + `" (code, created_at, updated_at, expires_at, client_id, user_data, user_code, scopes, interval)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`
	_, err := s.conn.Exec(ctx, insertQuery, device.Code, device.CreatedAt, device.UpdatedAt, device.ExpiresAt, device.ClientID, device.UserData, device.UserCode, device.Scopes, device.Interval)
	return err
}

// GetDeviceCode fetches a device code from the table by its code value.
func (s *postgresDeviceCodeStore) GetDeviceCode(ctx stdContext.Context, code string) (*DeviceCode, bool, error) {
	selectQuery := `SELECT code, created_at, updated_at, expires_at, client_id, user_data, user_code, scopes, interval FROM "` + s.tableName + `" WHERE code = $1 LIMIT 1;`
	row := s.conn.QueryRow(ctx, selectQuery, code)

	deviceCode := &DeviceCode{}
	err := row.Scan(&deviceCode.Code, &deviceCode.CreatedAt, &deviceCode.UpdatedAt, &deviceCode.ExpiresAt, &deviceCode.ClientID, &deviceCode.UserData, &deviceCode.UserCode, &deviceCode.Scopes, &deviceCode.Interval)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	return deviceCode, true, nil
}

// GetAndDeleteDeviceCode retrieves a device code record by its code and removes it from the table.
func (s *postgresDeviceCodeStore) GetAndDeleteDeviceCode(ctx stdContext.Context, code string) (*DeviceCode, bool, error) {
	deviceCode, ok, err := s.GetDeviceCode(ctx, code)
	if ok {
		deleteQuery := `DELETE FROM "` + s.tableName + `" WHERE code = $1;`
		_, err = s.conn.Exec(ctx, deleteQuery, code)
	}
	return deviceCode, ok, err
}

// GetDeviceCodeByUserCode finds a device code matching the provided userCode in the cache.
func (s *postgresDeviceCodeStore) GetDeviceCodeByUserCode(ctx stdContext.Context, userCode string) (*DeviceCode, bool, error) {
	selectQuery := `SELECT code, created_at, updated_at, expires_at, client_id, user_data, user_code, scopes, interval FROM "` + s.tableName + `" WHERE user_code = $1 LIMIT 1;`
	row := s.conn.QueryRow(ctx, selectQuery, userCode)

	deviceCode := &DeviceCode{}
	err := row.Scan(&deviceCode.Code, &deviceCode.CreatedAt, &deviceCode.UpdatedAt, &deviceCode.ExpiresAt, &deviceCode.ClientID, &deviceCode.UserData, &deviceCode.UserCode, &deviceCode.Scopes, &deviceCode.Interval)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	return deviceCode, true, nil
}
