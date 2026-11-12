package identity

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// parseUserSchema creates a UserSchema from a list of UserAttribute definitions.
//
// Supported field types:
//   - "text": String field with optional min/max length validation
//   - "number": Numeric field with optional min/max value validation
//   - "email": Email field with built-in email format validation
//   - "date": Date field with optional min/max year validation (YYYY-MM-DD format)
//   - "phone": Phone number field with optional format validation
//   - "bool": Boolean field accepting true/false values or string representations
//   - "map": Map field accepting map[string]any with optional required key validation
//
// Example usage:
//
//	attrs := []*UserAttribute{
//		{Name: "firstname", Type: "text", Params: []string{"3", "15"}},            // Field with type and parameters
//		{Name: "lastname", Type: "text", Params: []string{"3", "15"}},             // Default type is text
//		{Name: "age", Type: "number", Params: []string{"18", "105"}},              // Numeric field with range
//		{Name: "email", Indexed: true, Required: true},                            // Special case, type is inferred as required email and indexed
//		{Name: "company_id", Type: "uuid", Indexed: true},                         // UUID field with indexing
//		{Name: "nickname", Type: "text", Required: true},                          // Text required field with no length restrictions
//		{Name: "comments", Type: "text"},                                          // Default text field with no restrictions
//		{Name: "birthdate", Type: "date", Params: []string{"1900", "now"}},        // Birth year between 1900 and current year
//		{Name: "is_active", Type: "bool"},                                         // Boolean field accepting true/false values
//		{Name: "preferences", Type: "map", Params: []string{"theme", "language"}}, // Map field with required keys
//		{Name: "metadata", Type: "map"},                                           // Map field with no required keys
//		{Name: "tags", Type: "slice", Params: []string{"text", "1", "10"}},        // Slice of text values between 1 and 10 items
//		{Name: "tags", Type: "slice", Params: []string{"text"}},                   // Slice of text values without restrictions
//		{Name: "tags", Type: "slice", Params: []string{"text", "1"}},              // Slice of text values with minimum 1 item
//	}
//
// schema, err := parseUserSchema(attrs)
func parseUserSchema(attrs []*UserAttribute) (*UserSchema, error) {
	if len(attrs) == 0 {
		return NewUserSchema(), nil
	}

	validator := NewUserSchema()
	validatorValue := reflect.ValueOf(validator)
	validatorType := validatorValue.Type()

	// Build a case-insensitive map of method names for quick lookup.
	methodMap := make(map[string]reflect.Method)
	for method := range validatorType.Methods() {
		methodMap[strings.ToLower(method.Name)] = method
	}

	for _, attr := range attrs {
		field := attr.Name
		fieldType := attr.Type
		params := attr.Params
		indexed := attr.Indexed
		required := attr.Required

		if field == "" {
			return nil, fmt.Errorf("field validation requires a name")
		}

		if slices.Contains(ReservedUserFields, field) {
			return nil, fmt.Errorf("field '%s' is reserved by the system and cannot be used", field)
		}

		// Special case: if field is "email" and no type specified, assume email type.
		if fieldType == "" && field == "email" {
			fieldType = "email"
		}

		// Special case for "birthdate" field, default to date type.
		if fieldType == "" && field == "birthdate" {
			fieldType = "date"
		}

		if fieldType == "" && field == "phone" {
			fieldType = "phone"
		}

		// Default to "text" type if not specified
		if fieldType == "" {
			fieldType = "text"
		}

		attr.Type = fieldType // Update the attribute type in case we modified it.

		// Match method name case-insensitively (e.g., "text" -> "SetText").
		methodName := "set" + strings.ToLower(fieldType)
		method, exists := methodMap[methodName]
		if !exists {
			return nil, fmt.Errorf("unsupported field type '%s' for '%s', supported types: %s",
				fieldType, field, getSupportedTypes(methodMap))
		}

		// First parameter is always the field name.
		args := []reflect.Value{reflect.ValueOf(field)}

		// Convert remaining parameters based on the method's expected parameter types.
		methodType := method.Type
		paramCount := methodType.NumIn() - 1 // -1 for the receiver.

		// Check if this is a date type with "now" parameter.
		isDateType := strings.ToLower(fieldType) == "date"

		// Skip the first parameter (field name).
		for i := 1; i < paramCount; i++ {
			paramIndex := i - 1
			paramType := methodType.In(i + 1) // +1 for the receiver.

			var paramValue any
			var convErr error

			if paramIndex < len(params) {
				paramStr := params[paramIndex]

				// Handle special "now" keyword for date fields
				if isDateType && strings.ToLower(paramStr) == "now" && paramType.Kind() == reflect.Int {
					paramValue = time.Now().Year()
				} else {
					// Parameter was provided, convert it.
					switch paramType.Kind() {
					case reflect.Slice:
						paramValue = params
						// // For slice types, we assume a comma-separated list of values.
						// if paramType.Elem().Kind() == reflect.String {
						// 	paramValue = strings.Split(paramStr, ",")
						// } else if paramType.Elem().Kind() == reflect.Int {
						// 	// Convert comma-separated integers.
						// 	strValues := strings.Split(paramStr, ",")
						// 	intValues := make([]int, len(strValues))
						// 	for j, strValue := range strValues {
						// 		intValue, err := strconv.Atoi(strings.TrimSpace(strValue))
						// 		if err != nil {
						// 			return nil, fmt.Errorf("invalid integer value '%s' for field '%s': %w",
						// 				strValue, field, err)
						// 		}
						// 		intValues[j] = intValue
						// 	}
						// 	paramValue = intValues
						// } else {
						// 	return nil, fmt.Errorf("unsupported slice type %v for field '%s'", paramType, field)
						// }
					case reflect.Int:
						paramValue, convErr = strconv.Atoi(paramStr)
					case reflect.Float64:
						paramValue, convErr = strconv.ParseFloat(paramStr, 64)
					case reflect.String:
						paramValue = paramStr
					case reflect.Bool:
						// Convert string to boolean, accepting "true"/"false" or "1"/"0".
						v, err := strconv.ParseBool(paramStr) // Normalize boolean string
						if err != nil {
							return nil, fmt.Errorf("invalid boolean value '%s' for field '%s': %w",
								paramStr, field, err)
						}

						paramValue = v
					default:
						return nil, fmt.Errorf("unsupported parameter type %v for field '%s'", paramType, field)
					}

					if convErr != nil {
						return nil, fmt.Errorf("invalid parameter '%s' for field '%s': %w",
							paramStr, field, convErr)
					}
				}
			} else {
				// No parameter provided, use zero value.
				paramValue = reflect.Zero(paramType).Interface()
			}

			args = append(args, reflect.ValueOf(paramValue))
		}

		// Call the method with the parameters
		method.Func.Call(append([]reflect.Value{validatorValue}, args...))

		if indexed {
			validator.SetIndex(field)
		}

		if required {
			validator.SetRequired(field)
		}

		access, accessErr := normalizeAttrAccess(attr.Access)
		if accessErr != nil {
			return nil, fmt.Errorf("field %q: %w", field, accessErr)
		}
		attr.Access = access
		if access == AttrAccessPrivileged {
			validator.markPrivileged(field)
		}
	}

	validator.Attrs = attrs
	return validator, nil
}

// getSupportedTypes returns a string listing all supported field types.
func getSupportedTypes(methodMap map[string]reflect.Method) string {
	types := make([]string, 0)
	for name := range methodMap {
		if strings.HasPrefix(name, "set") && name != "set" {
			// Extract just the type name without the "set" prefix.
			typeName := name[3:] // Skip "set" prefix.
			types = append(types, typeName)
		}
	}
	return strings.Join(types, ", ")
}
