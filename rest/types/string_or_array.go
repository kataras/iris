package types

import (
	"encoding/json"
	"strings"
)

// StringOrArray type can be a single string or an array of strings.
// It marshals and unmarshals as a single string joined by commas.
type StringOrArray string

// UnmarshalJSON sets the value of the type from JSON.
func (soa *StringOrArray) UnmarshalJSON(data []byte) error {
	// Check for null JSON value or empty data
	if len(data) == 0 || isNull(data) {
		return nil // no need to set the StringOrArray to any value.
	}

	// Ensure data is long enough to be a valid JSON array or string.
	if len(data) >= 2 {
		// Check if the data represents an array of strings.
		if data[0] == '[' && data[len(data)-1] == ']' {
			// Trim the square brackets and split the string by commas.
			trimmed := strings.Trim(string(data), "[]")
			parts := strings.Split(trimmed, ",")
			for i, part := range parts {
				// Trim the quotes from each part.
				parts[i] = strings.Trim(part, "\"")
			}
			*soa = StringOrArray(strings.Join(parts, ","))
			return nil
		}

		// Check if the data is a single quoted string.
		if data[0] == '"' && data[len(data)-1] == '"' {
			*soa = StringOrArray(strings.Trim(string(data), "\""))
			return nil
		}
	}

	// return fmt.Errorf("string or array: must be a quoted string or an array of quoted strings but got: %q", string(data))
	return nil // just return nil if the data is not a valid JSON string or array, string will be empty.
}

// MarshalJSON returns the JSON encoding of the type.
func (soa StringOrArray) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(soa))
}

// String returns the string representation of the type.
func (soa StringOrArray) String() string {
	return string(soa)
}
