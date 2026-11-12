package entity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"uuid"

	"github.com/kataras/iris/v14/auth/identity/password"
	"github.com/kataras/iris/v14/core/pkg/uid"

	"github.com/kataras/pg"
)

// UserRepository handles user-related database operations.
type UserRepository struct {
	*pg.Repository[User]
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db *pg.DB) *UserRepository {
	return &UserRepository{
		Repository: pg.NewRepository[User](db),
	}
}

// InTransaction executes a function within a database transaction.
func (r *UserRepository) InTransaction(ctx context.Context, fn func(*UserRepository) error) error {
	conn := r.Repository.DB()
	if conn.IsTransaction() {
		return fn(r)
	}

	return conn.InTransaction(ctx, func(db *pg.DB) error {
		return fn(NewUserRepository(db))
	})
}

// GetUserByID retrieves an active user by id.
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (entry User, err error) {
	query := `SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL LIMIT 1;`
	entry, err = r.SelectSingle(ctx, query, id)
	return
}

// GetUserByUsername retrieves an active user by username.
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (entry User, err error) {
	query := `SELECT * FROM users WHERE username = $1 AND deleted_at IS NULL LIMIT 1;`
	entry, err = r.SelectSingle(ctx, query, username)
	return
}

// GetUserByUsernameIncludingDeleted retrieves a user by username, including soft-deleted users.
func (r *UserRepository) GetUserByUsernameIncludingDeleted(ctx context.Context, username string) (entry User, err error) {
	query := `SELECT * FROM users WHERE username = $1 LIMIT 1;`
	entry, err = r.SelectSingle(ctx, query, username)
	return
}

type FetchOptions struct {
	IncludePassword bool // If true, includes password and OTP fields.
	IncludeDeleted  bool // If true, includes soft-deleted users.

	Column string // The column to filter by (e.g., "id", "username").
}

// SetColumn sets the column to filter by in the FetchOptions.
func (opts *FetchOptions) SetColumn(column string) *FetchOptions {
	opts.Column = column
	return opts
}

var commonColumns = []string{"id", "created_at", "updated_at", "deleted_at", "username"}
var commonFieldsWithPassword = append(commonColumns, "password", "otp", "otp_recovery_codes")
var commonFieldsWithAttrs = append(commonColumns, "attrs") // Include attrs field for non-flattened queries.
var commonFieldsWithPasswordAndAttrs = append(commonFieldsWithAttrs, append(commonFieldsWithPassword, "attrs")...)

// listFields returns the fields to be selected based on the FetchOptions.
func (opts *FetchOptions) listFields(flatenned bool) []string {
	if opts.IncludePassword {
		if !flatenned {
			return commonFieldsWithPasswordAndAttrs // Flattened query includes all fields.
		}
		return commonFieldsWithPassword // Non-flattened query includes password and OTP fields.
	}

	if !flatenned {
		return commonFieldsWithAttrs // Flattened query includes attrs field.
	}
	// Non-flattened query without password and OTP fields.
	// This is the most common case, so we preallocate for performance.
	return commonColumns // Non-flattened query includes only common fields.
}

// buildQuery builds the SELECT and WHERE clause for a single user fetch.
// Example: opts.BuildQuery(false) => ... WHERE u.id = $1 AND u.deleted_at IS NULL LIMIT 1;
func (opts *FetchOptions) buildQuery(flattened bool) string {
	fields := opts.listFields(flattened)

	pairs := make([]string, 0, len(fields))
	for _, f := range fields {
		pairs = append(pairs, fmt.Sprintf("'%s', u.%s", f, f))
	}
	fieldsStr := strings.Join(pairs, ", ")

	where := fmt.Sprintf("u.%s = $%d", opts.Column, 1)
	if !opts.IncludeDeleted {
		where += " AND u.deleted_at IS NULL"
	}

	query := fmt.Sprintf(`SELECT %s FROM users u WHERE %s LIMIT 1;`, fieldsStr, where)

	if flattened {
		query = fmt.Sprintf(`
        SELECT COALESCE(
            jsonb_build_object(%s) || u.attrs, '{}'::jsonb
        ) AS user
        FROM users u
        WHERE %s
        LIMIT 1;
    `, fieldsStr, where)
	}
	return query
}

func (r *UserRepository) GetUser(ctx context.Context, opts FetchOptions, argValue any) (entry User, err error) {
	query := opts.buildQuery(false)
	return r.SelectSingle(ctx, query, argValue)
}

func (r *UserRepository) GetUserFlattened(ctx context.Context, opts FetchOptions, argValue any) (entry FlatUser, err error) {
	query := opts.buildQuery(true)
	err = r.QueryRow(ctx, query, argValue).Scan(&entry)
	return
}

// GetUserIDByUsername retrieves a user's ID by username (admin access).
func (r *UserRepository) GetUserIDByUsername(ctx context.Context, username string) (userID string, err error) {
	query := `SELECT id FROM users WHERE username = $1 LIMIT 1;`
	err = r.QueryRow(ctx, query, username).Scan(&userID)
	return
}

func (r *UserRepository) BulkDeleteUsers(ctx context.Context, ids []string, soft bool) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	query := `UPDATE users SET deleted_at = clock_timestamp() WHERE id = ANY($1) AND deleted_at IS NULL;`
	if !soft {
		query = `DELETE FROM users WHERE id = ANY($1);`
	}

	return r.Mutate(ctx, query, ids)
}

func (r *UserRepository) DeleteUserByID(ctx context.Context, id string, soft bool) (bool, error) {
	query := `UPDATE users SET deleted_at = clock_timestamp() WHERE id = $1 AND deleted_at IS NULL;` // add of soft detection so the cache is not invalidated if no change.
	if !soft {
		query = `DELETE FROM users WHERE id = $1;`
	}

	return r.MutateSingle(ctx, query, id)
}

// DeleteUserByUsername deletes a user by username.
func (r *UserRepository) DeleteUserByUsername(ctx context.Context, username string, soft bool) (bool, error) {
	query := `UPDATE users SET deleted_at = clock_timestamp() WHERE username = $1 AND deleted_at IS NULL;`
	if !soft {
		query = `DELETE FROM users WHERE username = $1;`
	}

	return r.MutateSingle(ctx, query, username)
}

// RestoreUserByID restores a soft-deleted user by its ID.
func (r *UserRepository) RestoreUserByID(ctx context.Context, id string) (bool, error) {
	query := `UPDATE users SET deleted_at = NULL WHERE id = $1;`
	return r.MutateSingle(ctx, query, id)
}

// RestoreUserByUsername restores a user by its username.
func (r *UserRepository) RestoreUserByUsername(ctx context.Context, username string) (bool, error) {
	query := `UPDATE users SET deleted_at = NULL WHERE username = $1 AND deleted_at IS NOT NULL;`
	return r.MutateSingle(ctx, query, username)
}

func (r *UserRepository) RestoreUserAndAttrsByUsername(ctx context.Context, username string, attrs json.RawMessage) (bool, error) {
	query := `UPDATE users SET deleted_at = NULL, attrs = $2 WHERE username = $1 AND deleted_at IS NOT NULL;`
	return r.MutateSingle(ctx, query, username, attrs)
}

// UpdatePasswordByID updates the password for the given user ID.
func (r *UserRepository) UpdatePasswordByID(ctx context.Context, id, hashedPassword string) (bool, error) {
	query := `UPDATE users SET password = $1 WHERE id = $2 AND deleted_at IS NULL;`
	cmd, err := r.Exec(ctx, query, hashedPassword, id)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}

// UpdatePasswordByUsername updates the password for the given username.
func (r *UserRepository) UpdatePasswordByUsername(ctx context.Context, username, hashedPassword string) (bool, error) {
	query := `UPDATE users SET password = $1 WHERE username = $2 AND deleted_at IS NULL;`
	cmd, err := r.Exec(ctx, query, hashedPassword, username)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *UserRepository) UpdateUsers(ctx context.Context, users []User, columns ...string) (int64, error) {
	return r.UpdateOnlyColumns(ctx, columns, users...)
}

// PartialUpdateSpec represents a partial update operation for JSONB attrs.
type PartialUpdateSpec struct {
	ID       string         `json:"id"`
	Username string         `json:"username"`         // for identification if ID is not provided.
	Set      map[string]any `json:"set,omitempty"`    // paths to set (e.g., {"profile.name": "John", "profile": {"age": 30}}).
	Remove   []string       `json:"remove,omitempty"` // JSON paths to remove (e.g., ["profile.avatar", "settings.notifications"]).
}

// UpdateUsersPartial performs partial updates on the core fields and attrs JSONB column
// This allows updating specific nested fields without replacing the entire attrs object.
func (r *UserRepository) UpdateUsersPartial(ctx context.Context, updates []PartialUpdateSpec) (int64, error) {
	if len(updates) == 0 {
		return 0, nil
	}

	var totalAffected int64

	err := r.InTransaction(ctx, func(r *UserRepository) error {
		// Process each update individually to allow for different update operations.
		for _, update := range updates {
			affected, err := r.updateSingleUserPartialAttrs(ctx, update)
			if err != nil {
				return fmt.Errorf("failed to update user %s: %w", update.getUserIdentifier(), err)
			}
			totalAffected += affected
		}

		return nil
	})

	return totalAffected, err
}

// getUserIdentifier returns the user identifier (ID or username)
func (spec *PartialUpdateSpec) getUserIdentifier() string {
	if spec.ID != "" {
		return spec.ID
	}
	return spec.Username
}

// updateSingleUserPartialAttrs updates a single user's core fields and attrs using partial update operations.
func (r *UserRepository) updateSingleUserPartialAttrs(ctx context.Context, update PartialUpdateSpec) (int64, error) {
	setParts := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []any{}
	argIndex := 1

	// Core table fields that should be updated directly (not in attrs JSONB)
	coreFields := map[string]bool{
		"id": true, "username": true, "password": true,
		"created_at": true, "updated_at": true, "deleted_at": true,
		"otp": true, "otp_recovery_codes": true,
	}

	// Separate core fields from attrs fields
	coreUpdates := make(map[string]any)
	attrsUpdates := make(map[string]any)

	for path, value := range update.Set {
		if coreFields[path] {
			coreUpdates[path] = value
		} else {
			attrsUpdates[path] = value
		}
	}

	// Handle core field updates
	for field, value := range coreUpdates {
		// Skip certain protected fields
		if field == "id" {
			return 0, fmt.Errorf("updating 'id' field is not allowed for data integrity reasons")
		}
		if field == "created_at" || field == "updated_at" {
			return 0, fmt.Errorf("updating '%s' field is not allowed - it's automatically managed", field)
		}

		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	// Handle attrs JSONB operations
	hasAttrsOperations := len(attrsUpdates) > 0 || len(update.Remove) > 0

	if hasAttrsOperations {
		// Build a single combined expression for all attrs operations
		attrsExpr := "attrs::jsonb"

		// Step 1: Apply path sets using jsonb_set (only for non-core fields)
		for path, value := range attrsUpdates {
			valueJSON, err := json.Marshal(value)
			if err != nil {
				return 0, fmt.Errorf("failed to marshal value for path %s: %w", path, err)
			}

			// Convert dot notation to PostgreSQL path array (e.g., "profile.name" -> '{"profile","name"}')
			pgPath := convertDotNotationToPostgreSQLPath(path)
			attrsExpr = fmt.Sprintf("jsonb_set(%s, '%s', $%d::jsonb, true)", attrsExpr, pgPath, argIndex)
			args = append(args, string(valueJSON))
			argIndex++
		}

		// Step 2: Apply path removals using #- operator
		for _, path := range update.Remove {
			// Don't allow removing core fields from attrs (they shouldn't be there anyway)
			if coreFields[path] {
				return 0, fmt.Errorf("cannot remove core field '%s' from attrs - use direct field update instead", path)
			}
			// Convert dot notation to PostgreSQL path array
			pgPath := convertDotNotationToPostgreSQLPath(path)
			attrsExpr = fmt.Sprintf("%s #- '%s'", attrsExpr, pgPath)
		}

		// Add the single attrs update expression
		setParts = append(setParts, fmt.Sprintf("attrs = %s", attrsExpr))
	}

	// Build WHERE clause
	var whereClause string
	var identifier any

	if update.ID != "" {
		whereClause = fmt.Sprintf("id = $%d AND deleted_at IS NULL", argIndex)
		identifier = update.ID
	} else if update.Username != "" {
		whereClause = fmt.Sprintf("username = $%d AND deleted_at IS NULL", argIndex)
		identifier = update.Username
	} else {
		return 0, fmt.Errorf("either ID or Username must be provided")
	}
	args = append(args, identifier)

	// Build the complete UPDATE query
	query := fmt.Sprintf("UPDATE users SET %s WHERE %s", strings.Join(setParts, ", "), whereClause)

	result, err := r.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to execute partial update: %w", err)
	}

	return result.RowsAffected(), nil
}

// convertDotNotationToPostgreSQLPath converts "profile.name" to "{profile,name}"
func convertDotNotationToPostgreSQLPath(dotPath string) string {
	parts := strings.Split(dotPath, ".")
	return "{" + strings.Join(parts, ",") + "}"
}

/*
func (r *UserRepository) UpdateUser(ctx context.Context, value User, terms FilterTerms, columns ...string) (int64,error) {
    where, args, err := terms.Build()
    if err != nil {
        return 0, err
    }

    query := fmt.Sprintf("UPDATE users SET %s "+where, strings.Join(columns, ", "), len(args)+1)
    args = append(args, value.ID)

    cmd, err := r.Exec(ctx, query, append(value.ToArgs()...)...)
    if err != nil {
        return 0, err
    }

    return cmd.RowsAffected()
}*/

var reservedFields = map[string]struct{}{
	"id":                 {},
	"created_at":         {},
	"updated_at":         {},
	"deleted_at":         {},
	"username":           {},
	"otp":                {},
	"otp_recovery_codes": {},
	"attrs":              {},
}

func isReserved(field string) (reserved bool) {
	// Check if the field is a reserved field.
	_, reserved = reservedFields[field]
	return
}

var (
	validFieldRegex = regexp.MustCompile(`^(id|created_at|updated_at|username|deleted_at|attrs|attrs::jsonb|attrs->>'[^']+'|attrs::jsonb#>>'\{[^}]+\}'|\(attrs::jsonb\)(->'[^']+')*->>'[^']+'|\(attrs::jsonb\)->'[^']+'|\(attrs::jsonb\)->>'[^']+')$`)

	validOperatorRegex = regexp.MustCompile(`^(=|IS|IS NOT|ILIKE|@>|#>>|<|>|<=|>=)$`)
	validLogicRegex    = regexp.MustCompile(`^(AND|OR)?$`)
)

// FilterTerm represents a filter for querying users.
type FilterTerm struct {
	Field    string `json:"field"`    // e.g. "username", "email", "attrs->>'phone'"
	Operator string `json:"operator"` // e.g. "=", "ILIKE", "@>", etc.
	Value    any    `json:"value"`    // e.g. "makis", "%gmail%", JSON...
	Logic    string `json:"logic"`    // e.g. "AND", "OR" (defaults to AND if empty)
}

// Validate checks if the FilterTerm contains valid field, operator, and logic.
func (f *FilterTerm) Validate() error {
	// Rewrite custom fields to JSONB query if not reserved and not already a JSONB query.
	if !isReserved(f.Field) {
		/*
		   {"field": "extra.other_sub_extra_field", "operator": "=", "value": "983fb1be-a520-4c99-b76d-cbdf4b24f0a5"}
		   // SAME with:
		   // {"field": "extra,other_sub_extra_field", "operator": "#>>", "value": "983fb1be-a520-4c99-b76d-cbdf4b24f0a5"}
		*/
		if f.Operator == "#>>" {
			f.Field = fmt.Sprintf("attrs::jsonb#>>'{%s}'", f.Field)
			f.Operator = "=" // Change operator to = since #>> is now part of the field expression
		} else if f.Operator == "@>" {
			f.Field = "attrs::jsonb" // Also needs casting for @> operator
		} else if !strings.HasPrefix(f.Field, "attrs->>'") && !strings.HasPrefix(f.Field, "attrs::jsonb") {
			// For nested fields, use chained extraction with casting
			if strings.Contains(f.Field, ".") {
				parts := strings.Split(f.Field, ".")
				if len(parts) == 2 {
					f.Field = fmt.Sprintf("(attrs::jsonb)->'%s'->>'%s'", parts[0], parts[1])
				} else if len(parts) > 2 {
					f.Field = fmt.Sprintf("attrs::jsonb#>>'{%s}'", strings.Join(parts, ","))
				}
			} else {
				f.Field = fmt.Sprintf("(attrs::jsonb)->>'%s'", f.Field)
			}
		}
	}

	if f.Operator == "" {
		f.Operator = "=" // Default operator to "=" if none provided.
	}

	// Default logic to "AND" if none provided or invalid.
	upperLogic := strings.ToUpper(f.Logic)
	if upperLogic == "" {
		upperLogic = "AND"
	}

	if !validFieldRegex.MatchString(f.Field) {
		return fmt.Errorf("invalid field: %q", f.Field)
	}

	if !validOperatorRegex.MatchString(f.Operator) {
		return fmt.Errorf("invalid operator: %q", f.Operator)
	}

	if !validLogicRegex.MatchString(upperLogic) {
		return fmt.Errorf("invalid logic: %q", f.Logic)
	}

	f.Logic = upperLogic
	return nil
}

type FilterTerms []*FilterTerm

func (f FilterTerms) validate() error {
	// Validate the filter's field/operator/logic.
	for _, q := range f {
		if err := q.Validate(); err != nil {
			return fmt.Errorf("invalid query filter term: %s: %w", q.Field, err)
		}
	}

	return nil
}

// UserFilterOptions represents options for filtering and sorting users.
type UserFilterOptions struct {
	Sort           string      `json:"sort,omitempty"`        // Optional. Sort expression. The user column to order by, defaults to "created_at".
	SortDescending bool        `json:"sort_descending"`       // Optional. It sorts the result set in descending order by expression.
	Terms          FilterTerms `json:"terms,omitempty"`       // important to be pointer as its validate alters the Logic.
	IncludeDeleted bool        `json:"include_deleted"`       // Optional. Include soft-deleted users in the result set.
	IncludeIDs     []string    `json:"include_ids,omitempty"` // Optional. List of user IDs to include in the result set. If provided, only users with these IDs will be returned.
	resultQuery    string      // build once.
	resultArgs     []any
}

const (
	// ASC is the ascending order.
	ASC = "ASC"
	// DESC is the descending order.
	DESC = "DESC"
)

// GetSortOrder returns the sort order (ASC or DESC) based on the SortDescending flag.
func (f *UserFilterOptions) GetSortOrder() string {
	if f.SortDescending {
		return DESC
	}

	return ASC
}

// ErrInvalidSortOption is returned when the sort option is invalid.
var ErrInvalidSortOption = errors.New("invalid sort option")

// validate checks if the UserFilterOptions contains valid sort and query filters.
func (f *UserFilterOptions) validate() error {
	// Set default sort if empty
	if f.Sort == "" {
		f.Sort = "created_at" // Default sort by created_at
	}

	if !validFieldRegex.MatchString(f.Sort) { // sorting of the available fields.
		return fmt.Errorf("%w: %q", ErrInvalidSortOption, f.Sort)
	}

	// Validate the filter's field/operator/logic.
	return f.Terms.validate()
}

// ContainsTermField checks if the filter options contain a specific term field.
func (f *UserFilterOptions) ContainsTermField(fieldName string) bool {
	for _, term := range f.Terms {
		if strings.ToLower(term.Field) == fieldName {
			return true
		}
	}

	return false
}

// Build constructs the WHERE clause and arguments for the query based on the filters.
func (f *UserFilterOptions) Build() (string, []any, error) {
	var err error
	if f.resultQuery == "" {
		var allArgs []any
		var whereParts []string

		// Handle IncludeIDs first if provided
		if len(f.IncludeIDs) > 0 {
			placeholders := make([]string, len(f.IncludeIDs))
			for i, id := range f.IncludeIDs {
				placeholders[i] = fmt.Sprintf("$%d", len(allArgs)+1)
				allArgs = append(allArgs, id)
			}
			inClause := fmt.Sprintf("id IN (%s)", strings.Join(placeholders, ", "))
			whereParts = append(whereParts, inClause)
		}

		if !f.IncludeDeleted && !f.ContainsTermField("deleted_at") {
			// If IncludeDeleted is true or false (default) and the filter doesn't contain the delted_at field,
			// we need to include the deleted_at field in the filter.
			f.Terms = append(f.Terms, &FilterTerm{
				Field:    "deleted_at",
				Operator: "IS",
				Value:    "NULL",
				Logic:    "AND",
			})
		}

		if err = f.validate(); err != nil {
			return "", nil, err
		}

		// Build terms with adjusted placeholders to account for IncludeIDs args
		if len(f.Terms) > 0 {
			termsQuery, termsArgs, err := f.buildTermsWithOffset(len(allArgs))
			if err != nil {
				return "", nil, err
			}

			if termsQuery != "" {
				// Remove " WHERE " prefix from terms query since we'll add it ourselves
				termsQuery = strings.TrimPrefix(termsQuery, " WHERE ")
				whereParts = append(whereParts, termsQuery)
			}
			allArgs = append(allArgs, termsArgs...)
		}

		// Combine all WHERE conditions
		if len(whereParts) > 0 {
			f.resultQuery = " WHERE " + strings.Join(whereParts, " AND ")
		}
		f.resultArgs = allArgs
	}

	return f.resultQuery, f.resultArgs, err
}

// buildTermsWithOffset builds the filter terms with placeholder numbering starting from the given offset
func (f *UserFilterOptions) buildTermsWithOffset(offset int) (string, []any, error) {
	if len(f.Terms) == 0 {
		return "", nil, nil
	}

	var (
		parts []string
		args  []any
	)

	for i, q := range f.Terms {
		var condition string
		if q.Value == "NULL" {
			// Special case for NULL values.
			// Should be treated as a raw string and not an argument.
			// E.g. SELECT COUNT(u.id) FROM users u WHERE deleted_at IS NULL;
			// instead of SELECT COUNT(u.id) FROM users u WHERE deleted_at IS $1 [args = NULL]; which doesn't work.
			condition = fmt.Sprintf("%s %s %s", q.Field, q.Operator, q.Value)
		} else {
			placeholder := fmt.Sprintf("$%d", offset+len(args)+1)
			// For @> operator (JSONB contains), cast the placeholder to jsonb
			if q.Operator == "@>" {
				placeholder = placeholder + "::jsonb"
			}
			condition = fmt.Sprintf("%s %s %s", q.Field, q.Operator, placeholder)
			args = append(args, q.Value)
		}

		if i == 0 {
			// First condition has no prefix.
			parts = append(parts, condition)
		} else {
			// Subsequent conditions: prepend logic (defaults to AND, see `validate`).
			parts = append(parts, q.Logic+" "+condition)
		}
	}

	if len(parts) == 0 {
		return "", nil, nil
	}

	query := strings.Join(parts, " ")
	return query, args, nil
}

// CountUsers returns the total number of users matching the filter options.
func (r *UserRepository) CountUsers(ctx context.Context, f *UserFilterOptions) (total int64, err error) {
	where, args, err := f.Build()
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}

	query := `SELECT COUNT(u.id) FROM users u` + where + `;`

	// fmt.Printf("CountUsers query: %s, args: %v\n", query, args)

	err = r.QueryRow(ctx, query, args...).Scan(&total)
	if err == pg.ErrNoRows {
		return 0, nil
	}

	return
}

// ListUsers queries the database with multiple filters, supporting AND/OR logic.
func (r *UserRepository) ListUsers(ctx context.Context, l PageOptions, f *UserFilterOptions) ([]User, error) {
	where, args, err := f.Build()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}

	query := "SELECT id, created_at, updated_at, username, attrs, deleted_at FROM users u" + where
	query += fmt.Sprintf(` ORDER BY u."`+f.Sort+`" %s LIMIT %d OFFSET %d;`, f.GetSortOrder(), l.GetLimit(), l.GetOffset())

	users, err := r.Select(ctx, query, args...)
	return users, err
}

// ListUsersFlattened returns a list of users in a flattened JSON format.
// This is useful for APIs that require a flat structure without nested objects.
func (r *UserRepository) ListUsersFlattened(ctx context.Context, l PageOptions, f *UserFilterOptions) ([]FlatUser, error) {
	where, args, err := f.Build()
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
SELECT COALESCE(
    jsonb_agg(
        jsonb_build_object(
            'id', u.id,
            'created_at', u.created_at,
            'updated_at', u.updated_at,
            'deleted_at', u.deleted_at,
            'username', u.username
        ) || u.attrs
        ORDER BY u."created_at" DESC
    ), '[]'::jsonb
) AS users
FROM (
    SELECT *
    FROM users`+where+`
    ORDER BY created_at DESC
    LIMIT %d OFFSET %d
) u;
`, l.GetLimit(), l.GetOffset())

	var array json.RawMessage
	err = r.QueryRow(ctx, query, args...).Scan(&array)
	if err != nil {
		return nil, err
	}

	// Returning a single JSON array from SQL and unmarshalling to []json.RawMessage in Go is the
	// fastest and most efficient approach for our use case.
	// This is the recommended pattern for high-performance APIs with flexible/flat JSON output.
	users, err := FlatUserArrayToSlice(array)
	if err != nil {
		return nil, fmt.Errorf("unmarshal users: %w", err)
	}
	return users, nil
}

// ListUsersPaginated returns a list of users and its filtered total count.
// Example:
//
//	usersPage, totalCount, err := ...ListUsersPaginated(context.Background(), PageOptions{}, &UserFilterOptions{
//		Sort:           "created_at",
//		SortDescending: true,
//		Terms: []*FilterTerm{
//			{Field: "username", Operator: "ILIKE", Value: "%Su%"},
//			{ /*Logic defaults to AND*/ Field: "attrs->>'key91'", Operator: "=", Value: "148"},
//			// {Logic: "OR", Field: "attrs->>'key91'", Operator: "=", Value: "148"},
//		},
//	})
func (r *UserRepository) ListUsersPaginated(ctx context.Context, l PageOptions, f *UserFilterOptions) ([]User, int64, error) {
	var (
		totalCount int64
		list       []User
		err        error
	)

	totalCount, err = r.CountUsers(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	if totalCount == 0 {
		return nil, 0, nil
	}

	list, err = r.ListUsers(ctx, l, f)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	return list, totalCount, nil
}

// ListUsersPaginatedFlattened returns a list of users in a flattened JSON format and its filtered total count.
func (r *UserRepository) ListUsersPaginatedFlattened(ctx context.Context, l PageOptions, f *UserFilterOptions) ([]FlatUser, int64, error) {
	var (
		totalCount int64
		list       []FlatUser
		err        error
	)

	totalCount, err = r.CountUsers(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	if totalCount == 0 {
		return nil, 0, nil
	}

	list, err = r.ListUsersFlattened(ctx, l, f)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, 0, fmt.Errorf("list users (flattened): %w", err)
	}

	return list, totalCount, nil
}

// CountTotalUsers returns the total number of active and inactive users.
func (r *UserRepository) CountTotalUsers(ctx context.Context) (active, inactive int64, err error) {
	query := `SELECT
  COUNT(*) FILTER (WHERE deleted_at IS NULL) AS active_users,
  COUNT(*) FILTER (WHERE deleted_at IS NOT NULL) AS inactive_users
FROM users;`

	err = r.QueryRow(ctx, query).Scan(&active, &inactive)
	if err == pg.ErrNoRows {
		return 0, 0, nil
	}

	return
}

// FlatUserArrayToSlice converts a JSON raw message representing an array into a slice of json.RawMessage.
func FlatUserArrayToSlice(raw FlatUser) ([]FlatUser, error) {
	// raw = bytes.TrimSpace(raw)
	// if len(raw) == 0 {
	// 	return nil, nil
	// }
	// if raw[0] == '{' {
	// 	// Single object, wrap in array
	// 	raw = append([]byte{'['}, append(raw, ']')...)
	// }

	var arr []FlatUser
	err := json.Unmarshal(raw, &arr)
	return arr, err
}

// tryDecryptPassword attempts to decrypt an AES-encrypted password using the provided shared key.
// Returns the decrypted password and true if successful, or the original password and false if failed.
func tryDecryptPassword(encryptedPassword, sharedKeyHex string) (string, bool) {
	if sharedKeyHex == "" || !password.IsHexEncoded(encryptedPassword) {
		return encryptedPassword, false
	}

	decrypted, err := password.Decrypt(encryptedPassword, sharedKeyHex)
	if err != nil {
		return encryptedPassword, false
	}

	return decrypted, true
}

// BulkInsertUsers inserts multiple users in batches with automatic password generation.
// It inserts multiple users in batches with AES decryption support.
// If a user's password is empty, generates a strong random password and hashes it.
// Users are inserted in batches of 500 within separate transactions for optimal performance.
// If sharedKeyHex is provided, attempts to decrypt AES-encrypted passwords before hashing.
// Returns the total number of users successfully inserted and any error encountered.
func (r *UserRepository) BulkInsertUsers(ctx context.Context, sharedKeyHex string, users []User) (int64, error) {
	if len(users) == 0 {
		return 0, nil
	}

	const batchSize = 500
	var totalInserted int64

	// Process users in batches
	for i := 0; i < len(users); i += batchSize {
		end := min(i+batchSize, len(users))

		batch := users[i:end]

		// Process each batch in a transaction
		err := r.InTransaction(ctx, func(txRepo *UserRepository) error {
			inserted, err := txRepo.processBatch(ctx, batch, sharedKeyHex)
			if err != nil {
				return fmt.Errorf("failed to process batch %d-%d: %w", i, end-1, err)
			}
			totalInserted += inserted
			return nil
		})

		if err != nil {
			return totalInserted, err
		}
	}

	return totalInserted, nil
}

// processBatch handles the insertion of a single batch of users within a transaction,
// with support for AES password decryption.
func (r *UserRepository) processBatch(ctx context.Context, users []User, sharedKeyHex string) (int64, error) {
	// Prepare users for insertion: generate passwords and hash them
	processedUsers := make([]User, len(users))
	copy(processedUsers, users)

	for i := range processedUsers {
		user := &processedUsers[i]

		// Generate password if empty.
		if user.Password == "" {
			randomPassword, err := password.GenerateRandomPassword(12) // 12 character password.
			if err != nil {
				return 0, fmt.Errorf("failed to generate password for user %s: %w", user.Username, err)
			}
			user.Password = randomPassword
		} else {
			// Try to decrypt AES-encrypted password if shared key is provided.
			// This is useful here as the migration tool might have not encrypted the passwords
			// Or the uploaded JSON file which contains a list of Users contain plain passwords
			// instead of encrypted so the automatic decryption would fail (see normal signup).
			if decrypted, wasDecrypted := tryDecryptPassword(user.Password, sharedKeyHex); wasDecrypted {
				user.Password = decrypted
			}
			// If decryption failed or password was not encrypted, use original password.
		}

		// Hash the password (it should be plain text at this point).
		hashedPassword, err := password.HashPassword(user.Password)
		if err != nil {
			return 0, fmt.Errorf("failed to hash password for user %s: %w", user.Username, err)
		}
		user.Password = hashedPassword
	}

	// Build bulk insert query using raw SQL for better performance.
	return r.insertUsersSQL(ctx, processedUsers)
}

// insertUsersSQL performs bulk insert using raw SQL for optimal performance.
func (r *UserRepository) insertUsersSQL(ctx context.Context, users []User) (int64, error) {
	if len(users) == 0 {
		return 0, nil
	}

	// Build the VALUES clause for bulk insert
	valueStrings := make([]string, 0, len(users))
	valueArgs := make([]any, 0, len(users)*4) // id, username, password, attrs for each user.

	for i, user := range users {
		// Each user needs 4 placeholders: id, username, password, attrs.
		placeholder := fmt.Sprintf("($%d, $%d, $%d, $%d)", i*4+1, i*4+2, i*4+3, i*4+4)
		valueStrings = append(valueStrings, placeholder)

		// Ensure attrs is not nil
		attrs := user.Attrs
		if attrs == nil {
			attrs = json.RawMessage("{}")
		}

		id := user.ID
		if id != "" { // If ID is provided, validate it, could be provided for migration purposes.
			if _, err := uuid.Parse(id); err != nil {
				return 0, fmt.Errorf("%w: invalid UUID for user %s at index: %d: %w", ErrInvalidArgument, user.Username, i, err)
			}
		}

		if id == "" {
			// 	id = "DEFAULT" // Use a default value if ID is empty.
			//  id = "gen_random_uuid()" // Use a function to generate a new UUID if ID is empty.
			id = uid.NewString()
		}

		valueArgs = append(valueArgs, id, user.Username, user.Password, attrs)
	}

	// On a username conflict we update the existing row but deliberately keep its
	// primary key: never set id = EXCLUDED.id. Reassigning the id from caller
	// supplied input would let a bulk insert hijack an existing user's primary
	// key (and any rows referencing it). password/attrs are refreshed and a
	// soft-deleted user is restored, which is the intended admin upsert behavior.
	query := fmt.Sprintf(`
		INSERT INTO users (id, username, password, attrs)
		VALUES %s
		ON CONFLICT (username) DO UPDATE SET
		    password = EXCLUDED.password,
		    attrs = EXCLUDED.attrs,
		    deleted_at = NULL,  -- Restore user if it was soft-deleted.
		    updated_at = CURRENT_TIMESTAMP;
	`,
		strings.Join(valueStrings, ","))

	result, err := r.Exec(ctx, query, valueArgs...)
	if err != nil {
		return 0, fmt.Errorf("bulk insert failed: %w", err)
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected, nil
}
