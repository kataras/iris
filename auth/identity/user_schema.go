package identity

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"
	"uuid"

	"github.com/kataras/iris/v14/rest/errors/validation"
)

// UserAttribute represents a user attribute in the user schema.
type UserAttribute struct {
	// Name of the attribute (e.g., firstname, lastname, email)
	Name string `json:"name" yaml:"Name"`
	// Type of the attribute (e.g., text, number, email, date, phone)
	Type string `json:"type" yaml:"Type"`
	// Additional parameters for the attribute type, if any.
	Params []string `json:"params,omitempty" yaml:"Params"`
	// Indexed indicates whether the attribute should be indexed in the database.
	Indexed bool `json:"indexed,omitzero" yaml:"Indexed"`
	// Required indicates whether the attribute is required.
	Required bool `json:"required,omitzero" yaml:"Required"`
	// Access is who may write the attribute.
	// "self" (the default) is a profile field a client with users:update may change.
	// "privileged" is written by an admin client, or by a server client whose scopes
	// include users:attr:<name>. The account holder does not write it.
	Access string `json:"access,omitempty" yaml:"Access"`
}

type (
	// ValueValidatorFunc is a function type that validates a value against a specific condition.
	// It returns an error if the value is invalid, or nil if it is valid.
	ValueValidatorFunc = func(value any) error

	// UserSchema describes the schema for user attributes, including validation rules and database indexes.
	UserSchema struct {
		Attrs []*UserAttribute `json:"attrs,omitempty"` // List of user attributes with their types and validation rules.

		fieldValidators map[string]ValueValidatorFunc
		// For JSONB fields in PostgreSQL, the GIN index is the standard and
		// best choice for key/value lookups and
		// containment queries (e.g., attrs @> '{"foo":"bar"}').
		databaseIndexes []string
		// List of required fields that must be present in the user attributes.
		requiredFields map[string]struct{}
		// privilegedFields are attributes the subject cannot write.
		privilegedFields map[string]struct{}
	}
)

// NewUserSchema creates a new UserSchema instance.
// It initializes the fieldValidators map to store custom field validators.
// This validator is used to validate user attributes based on the provided JSON schema.
// Note that if a user validator exists for an attribute but it's missing, it's considered a required one.
func NewUserSchema() *UserSchema {
	return &UserSchema{
		fieldValidators:  make(map[string]ValueValidatorFunc),
		requiredFields:   make(map[string]struct{}),
		privilegedFields: make(map[string]struct{}),
	}
}

// Set adds a custom validator for a specific field.
// The field is the key in the JSON schema,
// and the validator is a function that takes a value and returns an error if the value is invalid.
func (s *UserSchema) Set(field string, validator ValueValidatorFunc) *UserSchema {
	s.fieldValidators[field] = validator
	return s
}

// SetIndex adds a field to the list of database indexes.
// It is a no-op if no validator has been registered for that field yet,
// so call Set/SetText/SetEmail (and so on) for the field first.
func (s *UserSchema) SetIndex(field string) *UserSchema {
	if _, ok := s.fieldValidators[field]; ok {
		if !slices.Contains(s.databaseIndexes, field) {
			s.databaseIndexes = append(s.databaseIndexes, field)
		}
	}

	return s
}

// SetRequired adds a field to the list of required fields.
// It is a no-op if no validator has been registered for that field yet,
// so call Set/SetText/SetEmail (and so on) for the field first.
func (s *UserSchema) SetRequired(field string) *UserSchema {
	if _, ok := s.fieldValidators[field]; ok {
		s.requiredFields[field] = struct{}{}
	}
	return s
}

// isRequired checks if a field is required in the schema.
func (s *UserSchema) isRequired(field string) bool {
	_, exists := s.requiredFields[field]
	return exists
}

// SetText adds a validator for a string field with specified minimum and maximum length.
func (s *UserSchema) SetText(field string, minLength, maxLength int) *UserSchema {
	return s.Set(field, func(value any) error {
		if value == nil {
			if s.isRequired(field) || minLength > 0 {
				return fmt.Errorf("missing required field %s", field)
			} else {
				// If the value is nil and not required, we consider it valid.
				return nil
			}
		}

		if minLength == 0 && maxLength == 0 {
			// No length to validate against, consider it valid.
			return nil
		}

		strValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("invalid %s type: expected string but got: %T", field, value)
		}

		if errText := validation.StringLength(minLength, maxLength)(strValue); errText != "" {
			return fmt.Errorf("%s", errText)
		}

		return nil
	})
}

// SetNumber adds a validator for a numeric field with specified minimum and maximum values.
func (s *UserSchema) SetNumber(field string, min, max float64) *UserSchema {
	return s.Set(field, func(value any) error {
		if value == nil {
			if s.isRequired(field) || min > 0 {
				return fmt.Errorf("missing required field %s", field)
			} else {
				// If the value is nil and not required, we consider it valid.
				return nil
			}
		}

		if min == 0 && max == 0 {
			// No range to validate against, consider it valid.
			return nil
		}

		numValue, ok := value.(float64)
		if !ok {
			return fmt.Errorf("invalid %s type: expected number but got: %T", field, value)
		}

		if max == 0 { // do not force max if not exist.
			if numValue < min {
				return fmt.Errorf("%s must be at least %f", field, min)
			}
		} else if numValue < min || numValue > max {
			return fmt.Errorf("%s must be between %f and %f", field, min, max)
		}

		return nil
	})
}

// SetEmail adds a validator for an email field.
func (s *UserSchema) SetEmail(field string) *UserSchema {
	return s.Set(field, func(value any) error {
		if value == nil {
			if s.isRequired(field) {
				return fmt.Errorf("missing required field %s", field)
			} else {
				// If the value is nil and not required, we consider it valid.
				return nil
			}
		}

		strValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("invalid %s type: expected string but got: %T", field, value)
		}

		if errText := validation.Email(strValue); errText != "" {
			return fmt.Errorf("%s", errText)
		}

		return nil
	})
}

// SetUUID adds a validator for a UUIDv4 field.
func (s *UserSchema) SetUUID(field string) *UserSchema {
	return s.Set(field, func(value any) error {
		if value == nil {
			if s.isRequired(field) {
				return fmt.Errorf("missing required field %s", field)
			} else {
				// If the value is nil and not required, we consider it valid.
				return nil
			}
		}

		strValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("invalid %s type: expected string but got: %T", field, value)
		}

		if _, err := uuid.Parse(strValue); err != nil {
			return fmt.Errorf("invalid %s form: expected UUID form but got: %s: %w", field, strValue, err)
		}

		return nil
	})
}

// SetDate adds a validator for a date field with specified minimum and maximum years.
// The date should be in the format YYYY-MM-DD.
func (s *UserSchema) SetDate(field string, minYear, maxYear int) *UserSchema {
	return s.Set(field, func(value any) error {
		if value == nil {
			if s.isRequired(field) || minYear > 0 {
				return fmt.Errorf("missing required field %s", field)
			} else {
				// If the value is nil and not required, we consider it valid.
				return nil
			}
		}

		// Check if the value is a string
		strValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("invalid %s type: expected date string (YYYY-MM-DD) but got: %T", field, value)
		}

		// Parse the date string
		date, err := time.Parse("2006-01-02", strValue)
		if err != nil {
			return fmt.Errorf("%s must be a valid date in YYYY-MM-DD format", field)
		}

		// Check if the year is in the valid range
		year := date.Year()
		if minYear > 0 && year < minYear {
			return fmt.Errorf("%s year must be at least %d", field, minYear)
		}

		if maxYear > 0 && year > maxYear {
			return fmt.Errorf("%s year must be at most %d", field, maxYear)
		}

		return nil
	})
}

// SetPhone adds a validator for a phone field with optional phone format type.
func (s *UserSchema) SetPhone(field string, format string) *UserSchema {
	return s.Set(field, func(value any) error {
		if value == nil {
			if s.isRequired(field) {
				return fmt.Errorf("missing required field %s", field)
			} else {
				// If the value is nil and not required, we consider it valid.
				return nil
			}
		}

		strValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("invalid %s type: expected string but got: %T", field, value)
		}

		// Default to any phone type if not specified.
		phoneType := validation.ParsePhoneType(format)

		if errText := validation.PhoneNumber(phoneType)(strValue); errText != "" {
			return fmt.Errorf("%s", errText)
		}

		return nil
	})
}

// SetBool adds a validator for a boolean field.
// It accepts a boolean value or a string representation of a boolean ("true" or "false").
// If the value is nil it is considered valid, unless the field was marked as required
// through SetRequired, in which case a "missing required field" error is returned.
func (s *UserSchema) SetBool(field string) *UserSchema {
	return s.Set(field, func(value any) error {
		if value == nil {
			if s.isRequired(field) {
				return fmt.Errorf("missing required field %s", field)
			} else {
				// If the value is nil and not required, we consider it valid.
				return nil
			}
		}

		switch v := value.(type) {
		case bool:
			// If the value is already a boolean, we check its value.
			return nil
		case string:
			_, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("invalid %s type: expected boolean string but got: %T (%s)", field, value, v)
			}
			return nil
		default:
			return fmt.Errorf("invalid %s type: expected boolean or string but got: %T (%v)", field, value, v)
		}
	})
}

// SetSlice adds a validator for a slice field.
// The field should be a slice of a specific type (e.g., []string, []int, etc.).
// The sliceType parameter specifies the expected type of the slice elements.
// If sliceType is empty, it defaults to "any", meaning the slice can contain any
// type of elements.
//
// Available slice types:
// - text: a slice of strings
// - number: a slice of floats or integers.
// - bool: a slice of booleans
// - any: a slice of any type (any)
// If sliceType is empty, it defaults to "any".
func (s *UserSchema) SetSlice(field string, sliceType string, minLength, maxLength int) *UserSchema {
	if !slices.Contains([]string{"text", "number", "bool", "any", ""}, sliceType) {
		panic(fmt.Errorf("invalid slice type %s for field %s: must be one of string, int, float, bool, any", sliceType, field))
	}

	if minLength > 0 && maxLength > 0 && minLength > maxLength {
		panic(fmt.Errorf("minLength (%d) cannot be greater than maxLength (%d) for field %s", minLength, maxLength, field))
	}

	if minLength < 0 {
		minLength = 0 // Ensure minLength is non-negative.
	}

	if maxLength < 0 {
		maxLength = 0 // Ensure maxLength is non-negative.
	}

	checkSliceValueType := func(sliceValue []any, sliceType string) bool {
		if minLength > 0 && len(sliceValue) < minLength {
			return false // If the slice length is less than minLength, it's invalid.
		}

		if maxLength > 0 && len(sliceValue) > maxLength {
			return false // If the slice length is greater than maxLength, it's invalid.
		}

		for _, item := range sliceValue {
			switch sliceType {
			case "text":
				if _, ok := item.(string); !ok {
					return false
				}
			case "number":
				// Check for both int and float64 types
				if _, ok := item.(int); ok {
					continue // Valid int, check next item
				}
				if _, ok := item.(float64); ok { // JSON numbers are float64
					continue // Valid float64, check next item
				}
				return false // Not a valid number type
			case "bool":
				if _, ok := item.(bool); !ok {
					return false
				}
			default:
				// For "any" or empty type, we assume it's valid.
				continue
			}
		}

		return true // All items passed validation
	}

	isValidSliceType := func(value any) bool {
		switch v := value.(type) {
		case nil:
			if minLength > 0 {
				return false // If the slice length is less than minLengh, it's invalid.
			}
			return true // If the value is nil, we consider it valid.
		case []string:
			if sliceType == "text" || sliceType == "any" {
				if minLength > 0 && len(v) < minLength {
					return false // If the slice length is less than minLengh, it's invalid.
				}

				if maxLength > 0 && len(v) > maxLength {
					return false // If the slice length is greater than maxLength, it's invalid.
				}

				return true // If the slice is a string slice and matches the expected type, it's valid
			}
		case []int:
			if sliceType == "number" || sliceType == "any" {
				if minLength > 0 && len(v) < minLength {
					return false // If the slice length is less than minLengh, it's invalid.
				}

				if maxLength > 0 && len(v) > maxLength {
					return false // If the slice length is greater than maxLength, it's invalid.
				}

				return true // If the slice is a number slice and matches the expected type, it's valid
			}
		case []float64:
			if sliceType == "number" || sliceType == "any" {
				if minLength > 0 && len(v) < minLength {
					return false // If the slice length is less than minLengh, it's invalid.
				}

				if maxLength > 0 && len(v) > maxLength {
					return false // If the slice length is greater than maxLength, it's invalid.
				}

				return true // If the slice is a number slice and matches the expected type, it's valid
			}
		case []bool:
			if sliceType == "bool" || sliceType == "any" {
				if minLength > 0 && len(v) < minLength {
					return false // If the slice length is less than minLengh, it's invalid.
				}

				if maxLength > 0 && len(v) > maxLength {
					return false // If the slice length is greater than maxLength, it's invalid.
				}

				return true // If the slice is a boolean slice and matches the expected type, it's valid
			}
		case []byte:
			// If the value is a byte slice, we need to unmarshal it into a slice.
			var sliceValue []any
			if err := json.Unmarshal(v, &sliceValue); err != nil {
				return false // If unmarshaling fails, it's not a valid slice.
			}

			// Check if the unmarshaled slice matches the expected type.
			return checkSliceValueType(sliceValue, sliceType)
		case json.RawMessage:
			// If the value is a json.RawMessage, we need to unmarshal it into a slice.
			var sliceValue []any
			if err := json.Unmarshal(v, &sliceValue); err != nil {
				return false // If unmarshaling fails, it's not a valid slice.
			}

			// Check if the unmarshaled slice matches the expected type.
			return checkSliceValueType(sliceValue, sliceType)
		case []any:
			// Handle []any for all slice types, not just "any"
			// This is the most common case when JSON is unmarshaled
			return checkSliceValueType(v, sliceType)
		}

		// If the value is not a slice or a compatible type, it's not valid.
		return false
	}

	return s.Set(field, func(value any) error {
		if value == nil {
			if s.isRequired(field) || minLength > 0 {
				return fmt.Errorf("missing required field %s", field)
			} else {
				// If the value is nil and not required, we consider it valid.
				return nil
			}
		}

		if !isValidSliceType(value) {
			if minLength > 0 && maxLength > 0 {
				return fmt.Errorf("invalid %s type: expected slice of %s with length between %d and %d but got: %T", field, sliceType, minLength, maxLength, value)
			} else if minLength > 0 {
				return fmt.Errorf("invalid %s type: expected slice of %s with minimum length %d but got: %T", field, sliceType, minLength, value)
			} else if maxLength > 0 {
				return fmt.Errorf("invalid %s type: expected slice of %s with maximum length %d but got: %T", field, sliceType, maxLength, value)
			}

			return fmt.Errorf("invalid %s type: expected slice of %s but got: %T", field, sliceType, value)
		}

		return nil
	})
}

// SetMap adds a validator for a map field.
// The field should be a map[string]any and the requiredKeys, if any,
// are the keys the map is checked to contain.
// If requiredKeys is empty then nothing is validated at all, not even the map type,
// and any value is accepted.
// Otherwise, if the value is a byte slice or json.RawMessage, it will be unmarshaled into a map,
// and if the value is not a map, cannot be unmarshaled into a map or misses one of the
// requiredKeys, an error is returned.
func (s *UserSchema) SetMap(field string, requiredKeys []string) *UserSchema {
	return s.Set(field, func(value any) error {
		if len(requiredKeys) == 0 {
			// No required keys to validate against, consider it valid.
			return nil
		}

		if value == nil {
			if s.isRequired(field) {
				return fmt.Errorf("missing required field %s", field)
			} else if len(requiredKeys) > 0 {
				return fmt.Errorf("missing required keys %v in %s: value is nil", requiredKeys, field)
			} else {
				// If the value is nil and not required, we consider it valid.
				return nil
			}
		}

		var mapValue map[string]any
		switch v := value.(type) {
		case map[string]any:
			mapValue = v
		case []byte:
			// If the value is a byte slice, we need to unmarshal it into a map.
			if err := json.Unmarshal(v, &mapValue); err != nil {
				return fmt.Errorf("invalid %s type: expected map but got %T: %w", field, v, err)
			}
		case json.RawMessage:
			// If the value is a json.RawMessage or any other type, we need to unmarshal it into a map.
			if err := json.Unmarshal(v, &mapValue); err != nil {
				return fmt.Errorf("invalid %s type: expected map but got %T: %w", field, v, err)
			}
		default:
			return fmt.Errorf("invalid %s type: expected map but got: %T", field, value)
		}

		for _, key := range requiredKeys {
			if _, exists := mapValue[key]; !exists {
				return fmt.Errorf("missing required key %s in %s", key, field)
			}
		}

		return nil
	})
}

// not needed at all because passwords are always coming encrypted by the client
// func (s *UserJSONSchema) SetPassword(field string, )

// HasRules checks if the UserSchema has any validation rules defined.
func (s *UserSchema) HasRules() bool {
	return s != nil && len(s.fieldValidators) > 0
}

// Validate validates the user attributes against the defined schema.
// It checks for the presence of required fields, validates their types, and applies custom validators.
// Only registered fields and the system's ReservedUserFields are accepted;
// any other unregistered field will cause validation to fail.
// If the schema has no rules at all then Validate returns nil without any check.
func (s *UserSchema) Validate(attrs json.RawMessage) error {
	if !s.HasRules() {
		return nil // No validation rules defined, nothing to validate.
	}

	var fields map[string]any
	if err := json.Unmarshal(attrs, &fields); err != nil {
		return fmt.Errorf("user json schema: invalid attrs: %v", err)
	}

	var errs []error

	// First, check that all input fields are either registered or reserved
	for field := range fields {
		if !slices.Contains(ReservedUserFields, field) {
			if _, isRegistered := s.fieldValidators[field]; !isRegistered {
				errs = append(errs, fmt.Errorf("unregistered field %s is not allowed", field))
			}
		}
	}

	// Then validate all registered fields
	for field, validator := range s.fieldValidators {
		// Checked on the parser step.
		// if slices.Contains(ReservedUserFields, field) {
		// 	errs = append(errs, fmt.Errorf("user json schema: field %s is reserved by the system and cannot be used", field))
		// 	continue
		// }

		if value, exists := fields[field]; exists {
			if err := validator(value); err != nil {
				errs = append(errs, fmt.Errorf("%s validation failed: %v", field, err))
				continue
			}
		} else {
			if s.isRequired(field) {
				errs = append(errs, fmt.Errorf("missing required field %s", field))
				continue
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("user json schema validation failed: %w", errors.Join(errs...))
	}

	return nil
}

// GetDatabaseIndexes returns the list of database indexes for the user attributes.
func (s *UserSchema) GetDatabaseIndexes() []string {
	if s == nil {
		return nil
	}
	return s.databaseIndexes
}
