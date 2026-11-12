package identity

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseUserSchema(t *testing.T) {
	tests := []struct {
		name        string
		attrs       []*UserAttribute // Now directly pass []*UserAttribute
		wantErr     bool
		errContains string
		check       func(*testing.T, *UserSchema)
	}{
		{
			name: "basic text field",
			attrs: []*UserAttribute{
				{Name: "firstname", Type: "text", Params: []string{"3", "15"}, Indexed: false},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if len(schema.Attrs) != 1 || schema.Attrs[0].Name != "firstname" {
					t.Errorf("expected firstname attribute, got %+v", schema.Attrs)
				}
				// Test the validator works
				err := schema.Validate(json.RawMessage(`{"firstname": "John"}`))
				if err != nil {
					t.Errorf("valid firstname should pass validation: %v", err)
				}
				// Test validation fails for too short
				err = schema.Validate(json.RawMessage(`{"firstname": "Jo"}`))
				if err == nil {
					t.Error("firstname with 2 chars should fail validation")
				}
			},
		},
		{
			name: "number field",
			attrs: []*UserAttribute{
				{Name: "age", Type: "number", Params: []string{"18", "105"}},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if schema.Attrs[0].Type != "number" {
					t.Errorf("expected number type, got %s", schema.Attrs[0].Type)
				}
				// Test valid age
				err := schema.Validate(json.RawMessage(`{"age": 25}`))
				if err != nil {
					t.Errorf("valid age should pass validation: %v", err)
				}
				// Test age too low
				err = schema.Validate(json.RawMessage(`{"age": 15}`))
				if err == nil {
					t.Error("age below 18 should fail validation")
				}
			},
		},
		{
			name: "email field with auto-detection",
			attrs: []*UserAttribute{
				{Name: "email", Indexed: true}, // Type should be auto-detected as "email"
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if schema.Attrs[0].Type != "email" {
					t.Errorf("expected email type to be auto-detected, got %s", schema.Attrs[0].Type)
				}
				// Test valid email
				err := schema.Validate(json.RawMessage(`{"email": "user@example.com"}`))
				if err != nil {
					t.Errorf("valid email should pass validation: %v", err)
				}
				// Test invalid email
				err = schema.Validate(json.RawMessage(`{"email": "invalid-email"}`))
				if err == nil {
					t.Error("invalid email should fail validation")
				}
			},
		},
		{
			name: "date field with 'now' parameter",
			attrs: []*UserAttribute{
				{Name: "birthdate", Type: "date", Params: []string{"1900", "now"}},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if schema.Attrs[0].Type != "date" {
					t.Errorf("expected date type, got %s", schema.Attrs[0].Type)
				}
				// Test valid date
				err := schema.Validate(json.RawMessage(`{"birthdate": "1990-01-01"}`))
				if err != nil {
					t.Errorf("valid date should pass validation: %v", err)
				}
				// Test invalid date format
				err = schema.Validate(json.RawMessage(`{"birthdate": "1990/01/01"}`))
				if err == nil {
					t.Error("invalid date format should fail validation")
				}
			},
		},
		{
			name: "phone field with auto-detection",
			attrs: []*UserAttribute{
				{Name: "phone"}, // Type should be auto-detected as "phone"
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if schema.Attrs[0].Type != "phone" {
					t.Errorf("expected phone type to be auto-detected, got %s", schema.Attrs[0].Type)
				}
			},
		},
		{
			name: "bool field",
			attrs: []*UserAttribute{
				{Name: "is_active", Type: "bool"},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if schema.Attrs[0].Type != "bool" {
					t.Errorf("expected bool type, got %s", schema.Attrs[0].Type)
				}
				// Test valid boolean
				err := schema.Validate(json.RawMessage(`{"is_active": true}`))
				if err != nil {
					t.Errorf("valid boolean should pass validation: %v", err)
				}
				// Test string boolean
				err = schema.Validate(json.RawMessage(`{"is_active": "false"}`))
				if err != nil {
					t.Errorf("string boolean should pass validation: %v", err)
				}
				// Test nil value
				err = schema.Validate(json.RawMessage(`{"is_active": null}`))
				if err != nil {
					t.Errorf("null boolean should pass validation: %v", err)
				}
				// Test invalid boolean
				err = schema.Validate(json.RawMessage(`{"is_active": "maybe"}`))
				if err == nil {
					t.Error("invalid boolean string should fail validation")
				}
			},
		},
		{
			name: "map field with required keys",
			attrs: []*UserAttribute{
				{Name: "preferences", Type: "map", Params: []string{"theme", "language"}},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if schema.Attrs[0].Type != "map" {
					t.Errorf("expected map type, got %s", schema.Attrs[0].Type)
				}
				// Test valid map with required keys
				err := schema.Validate(json.RawMessage(`{"preferences": {"theme": "dark", "language": "en", "extra": "value"}}`))
				if err != nil {
					t.Errorf("valid map with required keys should pass validation: %v", err)
				}
				// Test map missing required key
				err = schema.Validate(json.RawMessage(`{"preferences": {"theme": "dark"}}`))
				if err == nil {
					t.Error("map missing required key should fail validation")
				}
				// Test null map
				err = schema.Validate(json.RawMessage(`{"preferences": null}`))
				if err == nil {
					t.Error("null map should fail validation as it requires keys")
				}
			},
		},
		{
			name: "map field without required keys",
			attrs: []*UserAttribute{
				{Name: "metadata", Type: "map"},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				// Test any map structure
				err := schema.Validate(json.RawMessage(`{"metadata": {"any": "value", "nested": {"key": "value"}}}`))
				if err != nil {
					t.Errorf("any map structure should pass validation: %v", err)
				}
				// Test empty map
				err = schema.Validate(json.RawMessage(`{"metadata": {}}`))
				if err != nil {
					t.Errorf("empty map should pass validation: %v", err)
				}
			},
		},
		{
			name: "multiple fields with indexes",
			attrs: []*UserAttribute{
				{Name: "firstname", Type: "text", Params: []string{"3", "15"}, Indexed: true},
				{Name: "email", Type: "email", Indexed: true},
				{Name: "age", Type: "number", Params: []string{"18", "105"}},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if len(schema.Attrs) != 3 {
					t.Errorf("expected 3 attributes, got %d", len(schema.Attrs))
				}
				indexes := schema.GetDatabaseIndexes()
				if len(indexes) != 2 {
					t.Errorf("expected 2 database indexes, got %d: %v", len(indexes), indexes)
				}
			},
		},
		{
			name: "default text type when none specified",
			attrs: []*UserAttribute{
				{Name: "nickname"}, // No type specified, should default to "text"
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if schema.Attrs[0].Type != "text" {
					t.Errorf("expected default type to be text, got %s", schema.Attrs[0].Type)
				}
			},
		},
		{
			name:    "empty attributes list",
			attrs:   []*UserAttribute{},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if len(schema.Attrs) != 0 {
					t.Errorf("expected 0 attributes, got %d", len(schema.Attrs))
				}
				if schema.HasRules() {
					t.Error("empty schema should not have rules")
				}
			},
		},
		{
			name:    "nil attributes list",
			attrs:   nil,
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if len(schema.Attrs) != 0 {
					t.Errorf("expected 0 attributes, got %d", len(schema.Attrs))
				}
			},
		},
		{
			name: "missing field name",
			attrs: []*UserAttribute{
				{Type: "text"}, // No name specified
			},
			wantErr:     true,
			errContains: "field validation requires a name",
		},
		{
			name: "reserved field name",
			attrs: []*UserAttribute{
				{Name: "id", Type: "text"}, // "id" should be a reserved field
			},
			wantErr:     true,
			errContains: "reserved by the system",
		},
		{
			name: "unsupported field type",
			attrs: []*UserAttribute{
				{Name: "test", Type: "unknown"},
			},
			wantErr:     true,
			errContains: "unsupported field type",
		},
		{
			name: "invalid number parameter",
			attrs: []*UserAttribute{
				{Name: "age", Type: "number", Params: []string{"invalid", "100"}},
			},
			wantErr:     true,
			errContains: "invalid parameter",
		},
		{
			name: "birthdate auto-detection",
			attrs: []*UserAttribute{
				{Name: "birthdate"}, // Should auto-detect as date type
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if schema.Attrs[0].Type != "date" {
					t.Errorf("expected birthdate to auto-detect as date type, got %s", schema.Attrs[0].Type)
				}
			},
		},
		{
			name: "slice field - text type with length constraints",
			attrs: []*UserAttribute{
				{Name: "tags", Type: "slice", Params: []string{"text", "2", "5"}},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if schema.Attrs[0].Type != "slice" {
					t.Errorf("expected slice type, got %s", schema.Attrs[0].Type)
				}
				// Test valid string slice
				err := schema.Validate(json.RawMessage(`{"tags": ["tag1", "tag2", "tag3"]}`))
				if err != nil {
					t.Errorf("valid string slice should pass validation: %v", err)
				}
				// Test slice too short
				err = schema.Validate(json.RawMessage(`{"tags": ["tag1"]}`))
				if err == nil {
					t.Error("slice below minimum length should fail validation")
				}
				// Test slice too long
				err = schema.Validate(json.RawMessage(`{"tags": ["tag1", "tag2", "tag3", "tag4", "tag5", "tag6"]}`))
				if err == nil {
					t.Error("slice above maximum length should fail validation")
				}
				// Test mixed types in slice (should fail for text type)
				err = schema.Validate(json.RawMessage(`{"tags": ["tag1", 123, "tag3"]}`))
				if err == nil {
					t.Error("mixed types in text slice should fail validation")
				}
			},
		},
		{
			name: "slice field - number type",
			attrs: []*UserAttribute{
				{Name: "scores", Type: "slice", Params: []string{"number", "1", "10"}},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				// Test valid number slice
				err := schema.Validate(json.RawMessage(`{"scores": [85.5, 92, 78.3]}`))
				if err != nil {
					t.Errorf("valid number slice should pass validation: %v", err)
				}
				// Test integer slice
				err = schema.Validate(json.RawMessage(`{"scores": [85, 92, 78]}`))
				if err != nil {
					t.Errorf("valid integer slice should pass validation: %v", err)
				}
				// Test invalid type in number slice
				err = schema.Validate(json.RawMessage(`{"scores": [85, "invalid", 78]}`))
				if err == nil {
					t.Error("non-number in number slice should fail validation")
				}
			},
		},
		{
			name: "slice field - bool type",
			attrs: []*UserAttribute{
				{Name: "flags", Type: "slice", Params: []string{"bool", "0", "0"}}, // No length constraints
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				// Test valid boolean slice
				err := schema.Validate(json.RawMessage(`{"flags": [true, false, true]}`))
				if err != nil {
					t.Errorf("valid boolean slice should pass validation: %v", err)
				}
				// Test empty slice
				err = schema.Validate(json.RawMessage(`{"flags": []}`))
				if err != nil {
					t.Errorf("empty slice should pass validation: %v", err)
				}
				// Test invalid type in boolean slice
				err = schema.Validate(json.RawMessage(`{"flags": [true, "invalid", false]}`))
				if err == nil {
					t.Error("non-boolean in boolean slice should fail validation")
				}
			},
		},
		{
			name: "slice field - any type",
			attrs: []*UserAttribute{
				{Name: "mixed_data", Type: "slice", Params: []string{"any", "1", "5"}},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				// Test mixed types slice
				err := schema.Validate(json.RawMessage(`{"mixed_data": ["text", 123, true, {"key": "value"}]}`))
				if err != nil {
					t.Errorf("mixed types slice should pass validation: %v", err)
				}
				// Test slice too short
				err = schema.Validate(json.RawMessage(`{"mixed_data": []}`))
				if err == nil {
					t.Error("empty slice should fail when minimum length is required")
				}
			},
		},
		{
			name: "slice field - default to any type",
			attrs: []*UserAttribute{
				{Name: "any_data", Type: "slice", Params: []string{"", "0", "0"}}, // Empty type defaults to "any"
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				// Test any slice types
				err := schema.Validate(json.RawMessage(`{"any_data": [1, "text", true, null]}`))
				if err != nil {
					t.Errorf("any type slice should pass validation: %v", err)
				}
			},
		},
		{
			name: "slice field - no parameters (defaults)",
			attrs: []*UserAttribute{
				{Name: "default_slice", Type: "slice"},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				// Test with default parameters (should accept any type, any length)
				err := schema.Validate(json.RawMessage(`{"default_slice": []}`))
				if err != nil {
					t.Errorf("empty slice should pass with defaults: %v", err)
				}
				err = schema.Validate(json.RawMessage(`{"default_slice": [1, "text", true]}`))
				if err != nil {
					t.Errorf("mixed slice should pass with defaults: %v", err)
				}
			},
		},
		{
			name: "slice field - null value",
			attrs: []*UserAttribute{
				{Name: "nullable_slice", Type: "slice", Params: []string{"text", "1", "3"}},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				// Test null value (should fail if minimum length > 0)
				err := schema.Validate(json.RawMessage(`{"nullable_slice": null}`))
				if err == nil {
					t.Error("null slice should fail when minimum length is required")
				}
			},
		},
		{
			name: "slice field - null value with no minimum",
			attrs: []*UserAttribute{
				{Name: "optional_slice", Type: "slice", Params: []string{"text", "0", "3"}},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				// Test null value (should pass if minimum length is 0)
				err := schema.Validate(json.RawMessage(`{"optional_slice": null}`))
				if err != nil {
					t.Errorf("null slice should pass when minimum length is 0: %v", err)
				}
			},
		},
		{
			name: "slice field - JSON string input",
			attrs: []*UserAttribute{
				{Name: "json_slice", Type: "slice", Params: []string{"text", "1", "3"}},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				// Note: This tests the JSON unmarshaling capability of the SetSlice method
				// when the input comes as bytes or json.RawMessage
				err := schema.Validate(json.RawMessage(`{"json_slice": ["item1", "item2"]}`))
				if err != nil {
					t.Errorf("JSON string slice should pass validation: %v", err)
				}
			},
		},
		{
			name: "slice field - invalid non-slice input",
			attrs: []*UserAttribute{
				{Name: "invalid_slice", Type: "slice", Params: []string{"text", "0", "3"}},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				// Test non-slice input
				err := schema.Validate(json.RawMessage(`{"invalid_slice": "not_a_slice"}`))
				if err == nil {
					t.Error("non-slice input should fail validation")
				}
				err = schema.Validate(json.RawMessage(`{"invalid_slice": 123}`))
				if err == nil {
					t.Error("non-slice input should fail validation")
				}
			},
		},
		{
			name: "comprehensive schema with all types",
			attrs: []*UserAttribute{
				{Name: "firstname", Type: "text", Params: []string{"2", "50"}, Indexed: true},
				{Name: "age", Type: "number", Params: []string{"0", "150"}},
				{Name: "email", Type: "email", Indexed: true},
				{Name: "birthdate", Type: "date", Params: []string{"1900", "now"}},
				{Name: "phone", Type: "phone", Params: []string{"international"}},
				{Name: "is_active", Type: "bool"},
				{Name: "tags", Type: "slice", Params: []string{"text", "0", "10"}},
				{Name: "scores", Type: "slice", Params: []string{"number", "1", "5"}},
				{Name: "preferences", Type: "map", Params: []string{"theme", "language"}},
				{Name: "metadata", Type: "map"},
			},
			wantErr: false,
			check: func(t *testing.T, schema *UserSchema) {
				if len(schema.Attrs) != 10 {
					t.Errorf("expected 10 attributes, got %d", len(schema.Attrs))
				}

				// Test a comprehensive valid input
				validInput := `{
					"firstname": "John",
					"age": 30,
					"email": "john@example.com",
					"birthdate": "1993-01-01",
					"phone": "+1234567890",
					"is_active": true,
					"tags": ["developer", "golang"],
					"scores": [85.5, 92.0, 78.3],
					"preferences": {"theme": "dark", "language": "en"},
					"metadata": {"department": "engineering", "level": "senior"}
				}`

				err := schema.Validate(json.RawMessage(validInput))
				if err != nil {
					t.Errorf("comprehensive valid input should pass validation: %v", err)
				}

				// Check database indexes
				indexes := schema.GetDatabaseIndexes()
				if len(indexes) != 2 {
					t.Errorf("expected 2 database indexes, got %d: %v", len(indexes), indexes)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := parseUserSchema(tt.attrs)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseUserSchema() expected error, but got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("parseUserSchema() error = %v, want it to contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseUserSchema() unexpected error: %v", err)
				return
			}

			if validator == nil {
				t.Error("parseUserSchema() returned nil validator")
				return
			}

			if tt.check != nil {
				tt.check(t, validator)
			}
		})
	}
}

// helper function to check if a string contains a substring, case-insensitive
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
