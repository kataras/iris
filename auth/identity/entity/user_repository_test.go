package entity

import (
	"testing"
)

func TestUserFilterOptions_Build_WithIncludeIDs(t *testing.T) {
	tests := []struct {
		name          string
		filterOptions *UserFilterOptions
		expectedQuery string
		expectedArgs  []any
		expectError   bool
	}{
		{
			name: "only IncludeIDs provided",
			filterOptions: &UserFilterOptions{
				IncludeIDs: []string{"id1", "id2", "id3"},
			},
			expectedQuery: " WHERE id IN ($1, $2, $3) AND deleted_at IS NULL",
			expectedArgs:  []any{"id1", "id2", "id3"},
		},
		{
			name: "IncludeIDs with additional filter terms",
			filterOptions: &UserFilterOptions{
				IncludeIDs: []string{"id1", "id2"},
				Terms: FilterTerms{
					{Field: "username", Operator: "ILIKE", Value: "%test%"},
				},
			},
			expectedQuery: " WHERE id IN ($1, $2) AND username ILIKE $3 AND deleted_at IS NULL",
			expectedArgs:  []any{"id1", "id2", "%test%"},
		},
		{
			name: "IncludeIDs with multiple filter terms",
			filterOptions: &UserFilterOptions{
				IncludeIDs: []string{"id1"},
				Terms: FilterTerms{
					{Field: "username", Operator: "=", Value: "testuser"},
					{Field: "attrs->>'role'", Operator: "=", Value: "admin", Logic: "AND"},
				},
			},
			expectedQuery: " WHERE id IN ($1) AND username = $2 AND attrs->>'role' = $3 AND deleted_at IS NULL",
			expectedArgs:  []any{"id1", "testuser", "admin"},
		},
		{
			name: "only filter terms without IncludeIDs",
			filterOptions: &UserFilterOptions{
				Terms: FilterTerms{
					{Field: "username", Operator: "ILIKE", Value: "%test%"},
				},
			},
			expectedQuery: " WHERE username ILIKE $1 AND deleted_at IS NULL",
			expectedArgs:  []any{"%test%"},
		},
		{
			name: "empty IncludeIDs array",
			filterOptions: &UserFilterOptions{
				IncludeIDs: []string{},
				Terms: FilterTerms{
					{Field: "username", Operator: "=", Value: "testuser"},
				},
			},
			expectedQuery: " WHERE username = $1 AND deleted_at IS NULL",
			expectedArgs:  []any{"testuser"},
		},
		{
			name: "IncludeDeleted with IncludeIDs",
			filterOptions: &UserFilterOptions{
				IncludeIDs:     []string{"id1", "id2"},
				IncludeDeleted: true,
			},
			expectedQuery: " WHERE id IN ($1, $2)",
			expectedArgs:  []any{"id1", "id2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args, err := tt.filterOptions.Build()

			if tt.expectError && err == nil {
				t.Errorf("expected an error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if query != tt.expectedQuery {
				t.Errorf("expected query %q, got %q", tt.expectedQuery, query)
			}

			if len(args) != len(tt.expectedArgs) {
				t.Errorf("expected %d args, got %d", len(tt.expectedArgs), len(args))
				return
			}

			for i, expectedArg := range tt.expectedArgs {
				if args[i] != expectedArg {
					t.Errorf("expected arg[%d] to be %v, got %v", i, expectedArg, args[i])
				}
			}
		})
	}
}

func TestUserFilterOptions_Build_PlaceholderNumbering(t *testing.T) {
	// Test that placeholder numbering is correct when combining IncludeIDs with filter terms
	filterOptions := &UserFilterOptions{
		IncludeIDs: []string{"id1", "id2", "id3"},
		Terms: FilterTerms{
			{Field: "username", Operator: "ILIKE", Value: "%user%"},
			{Field: "attrs->>'role'", Operator: "=", Value: "admin", Logic: "AND"},
			{Field: "attrs->>'status'", Operator: "=", Value: "active", Logic: "OR"},
		},
	}

	query, args, err := filterOptions.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedQuery := " WHERE id IN ($1, $2, $3) AND username ILIKE $4 AND attrs->>'role' = $5 OR attrs->>'status' = $6 AND deleted_at IS NULL"
	expectedArgs := []any{"id1", "id2", "id3", "%user%", "admin", "active"}

	if query != expectedQuery {
		t.Errorf("expected query %q, got %q", expectedQuery, query)
	}

	if len(args) != len(expectedArgs) {
		t.Errorf("expected %d args, got %d", len(expectedArgs), len(args))
		return
	}

	for i, expectedArg := range expectedArgs {
		if args[i] != expectedArg {
			t.Errorf("expected arg[%d] to be %v, got %v", i, expectedArg, args[i])
		}
	}
}
