package entity

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/kataras/pg"
)

// UserGroupRepository handles user group-related database operations.
type UserGroupRepository struct {
	*pg.Repository[UserGroup]
}

// NewUserGroupRepository creates a new UserGroupRepository instance.
func NewUserGroupRepository(db *pg.DB) *UserGroupRepository {
	return &UserGroupRepository{
		Repository: pg.NewRepository[UserGroup](db),
	}
}

// InTransaction executes a function within a database transaction.
func (r *UserGroupRepository) InTransaction(ctx context.Context, fn func(*UserGroupRepository) error) error {
	conn := r.Repository.DB()
	if conn.IsTransaction() {
		return fn(r)
	}

	return conn.InTransaction(ctx, func(db *pg.DB) error {
		return fn(NewUserGroupRepository(db))
	})
}

// GetUserGroupByID retrieves a user group by its ID.
func (r *UserGroupRepository) GetUserGroupByID(ctx context.Context, id string) (UserGroup, error) {
	return r.Repository.SelectByID(ctx, id)
}

// DeleteUserGroupByID deletes a user group by its ID.
func (r *UserGroupRepository) DeleteUserGroupByID(ctx context.Context, id string) (bool, error) {
	return r.Repository.DeleteByID(ctx, id)
}

// DeleteUserGroupByName deletes a user group by its name.
func (r *UserGroupRepository) DeleteUserGroupByName(ctx context.Context, name string) (bool, error) {
	query := `DELETE FROM user_groups WHERE name = $1;`
	return r.MutateSingle(ctx, query, name)
}

func (r *UserGroupRepository) UpsertUserGroup(ctx context.Context, group UserGroup) (id string, created bool, err error) {
	query := `INSERT INTO user_groups (name, filter) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET filter = EXCLUDED.filter RETURNING id;`
	err = r.Repository.QueryRow(ctx, query, group.Name, group.Filter).Scan(&id)
	if err == nil {
		created = group.ID != id
	}

	return
}

var (
	validUserGroupFieldRegex = regexp.MustCompile(`^(id|created_at|updated_at|name)$`)
)

// UserGroupFilterOptions defines the filter options for querying user groups.
type UserGroupFilterOptions struct {
	Sort           string `json:"sort,omitempty"`  // Optional. Sort expression. The columns that dietitians wish to order by.
	SortDescending bool   `json:"sort_descending"` // Optional. It sorts the result set in descending order by expression.
	Name           string `json:"name,omitempty"`  // Optional. Filter by user group name.
}

// GetSortOrder returns the sort order (ASC or DESC) based on the SortDescending flag.
func (f *UserGroupFilterOptions) GetSortOrder() string {
	if f.SortDescending {
		return DESC
	}

	return ASC
}

// validate checks if the UserGroupFilterOptions contains valid sort.
func (f *UserGroupFilterOptions) validate() error {
	if !validUserGroupFieldRegex.MatchString(f.Sort) { // sorting of the available fields.
		return fmt.Errorf("%w: %q", ErrInvalidSortOption, f.Sort)
	}

	return nil
}

// CountUserGroups returns the total number of user groups matching the filter options.
func (r *UserGroupRepository) CountUserGroups(ctx context.Context, f *UserGroupFilterOptions) (total int64, err error) {
	query := `SELECT COUNT(id) FROM user_groups WHERE ($1='' OR name ILIKE '%' || $1 || '%')`
	err = r.QueryRow(ctx, query, f.Name).Scan(&total)
	if err == pg.ErrNoRows {
		return 0, nil
	}

	return
}

// ListUserGroups queries the database for user groups based on pagination and filter options.
func (r *UserGroupRepository) ListUserGroups(ctx context.Context, l PageOptions, f *UserGroupFilterOptions) ([]UserGroup, error) {
	query := `SELECT * FROM user_groups WHERE ($1='' OR name ILIKE '%' || $1 || '%')`
	query += fmt.Sprintf(` ORDER BY "`+f.Sort+`" %s LIMIT %d OFFSET %d;`, f.GetSortOrder(), l.GetLimit(), l.GetOffset())

	groups, err := r.Select(ctx, query, f.Name)
	return groups, err
}

// ListUserGroupsPaginated retrieves a paginated list of user groups along with the total count.
func (r *UserGroupRepository) ListUserGroupsPaginated(ctx context.Context, l PageOptions, f *UserGroupFilterOptions) ([]UserGroup, int64, error) {
	if err := f.validate(); err != nil {
		return nil, 0, fmt.Errorf("validate filter options: %w", err)
	}

	var (
		totalCount int64
		list       []UserGroup
		err        error
	)

	totalCount, err = r.CountUserGroups(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("count user groups: %w", err)
	}

	if totalCount == 0 {
		return nil, 0, nil
	}

	list, err = r.ListUserGroups(ctx, l, f)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, 0, fmt.Errorf("list user groups: %w", err)
	}

	return list, totalCount, nil
}
