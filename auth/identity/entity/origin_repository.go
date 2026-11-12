package entity

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/kataras/iris/v14/middleware/cors"

	"github.com/kataras/pg"
)

// OriginRepository handles allowed CORS origin pattern database operations.
type OriginRepository struct {
	*pg.Repository[Origin]
}

// NewOriginRepository creates a new OriginRepository instance.
func NewOriginRepository(db *pg.DB) *OriginRepository {
	return &OriginRepository{
		Repository: pg.NewRepository[Origin](db),
	}
}

// InTransaction executes a function within a database transaction.
func (r *OriginRepository) InTransaction(ctx context.Context, fn func(*OriginRepository) error) error {
	conn := r.Repository.DB()
	if conn.IsTransaction() {
		return fn(r)
	}

	return conn.InTransaction(ctx, func(db *pg.DB) error {
		return fn(NewOriginRepository(db))
	})
}

func (r *OriginRepository) ListOriginPatterns(ctx context.Context) ([]string, error) {
	query := `SELECT pattern FROM origins;`
	return r.Repository.DB().QuerySlice[string](ctx, query)
}

// UpsertOriginPatterns inserts or updates origin patterns in the database.
// It merges the new patterns with existing ones, ensuring no duplicates.
// If the patterns are already present, it returns false without making changes.
// If any patterns are inserted, it returns true.
// It returns an error if the operation fails.
// Note that the stored patterns are re-normalized on every call, even when no patterns are
// given, so a stored set which is not already merged is deleted and re-inserted.
func (r *OriginRepository) UpsertOriginPatterns(ctx context.Context, patterns ...string) (bool, error) {
	storedPatterns, err := r.ListOriginPatterns(ctx)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return false, err
	}

	patternsToStore := cors.MergeOriginPatterns(append(storedPatterns, patterns...))
	// They are not modified at all.
	if slices.Equal(storedPatterns, patternsToStore) {
		return false, nil
	}

	values := make([]string, len(patternsToStore))
	args := make([]any, len(patternsToStore))
	for i, p := range patternsToStore {
		values[i] = fmt.Sprintf("($%d)", i+1)
		args[i] = p
	}

	var affected bool
	err = r.InTransaction(ctx, func(r *OriginRepository) error {
		affected, err = r.DeleteOriginPatterns(ctx, storedPatterns...)
		if err != nil {
			return fmt.Errorf("failed to delete existing patterns: %w", err)
		}

		query := fmt.Sprintf("INSERT INTO origins (pattern) VALUES %s ON CONFLICT (pattern) DO NOTHING;", strings.Join(values, ","))
		cmd, err := r.Repository.Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to insert patterns: %w", err)
		}
		if !affected {
			affected = cmd.RowsAffected() > 0
		}
		return nil
	})

	return affected, err
}

// DeleteOriginsExceptWildcardPattern deletes all origin patterns except the wildcard one.
func (r *OriginRepository) DeleteOriginsExceptWildcardPattern(ctx context.Context) (bool, error) {
	query := `DELETE FROM origins WHERE pattern != '*';`
	cmd, err := r.Repository.Exec(ctx, query)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *OriginRepository) DeleteOriginPatterns(ctx context.Context, patterns ...string) (bool, error) {
	if len(patterns) == 0 {
		return false, nil
	}

	query := `DELETE FROM origins WHERE pattern = ANY($1);`
	cmd, err := r.Repository.Exec(ctx, query, patterns)
	if err != nil {
		return false, err
	}
	return cmd.RowsAffected() > 0, nil
}
