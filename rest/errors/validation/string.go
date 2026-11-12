package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// StringError describes a string field validation error.
type StringError struct{ *FieldError[string] }

// String returns a new string validation error.
func String(field string, value string) *StringError {
	return &StringError{Field(field, value)}
}

// NotEmpty adds an error if the string is empty.
func (e *StringError) NotEmpty() *StringError {
	e.Func(NotEmpty)
	return e
}

// Fullname adds an error if the string is not a full name.
func (e *StringError) Fullname() *StringError {
	e.Func(Fullname)
	return e
}

// Length adds an error if the string length is not in the given range.
func (e *StringError) Length(min, max int) *StringError {
	e.Func(StringLength(min, max))
	return e
}

// Email adds an error if the string is not a valid email address.
func (e *StringError) Email() *StringError {
	e.Func(Email)
	return e
}

// NotEmpty accepts any string and returns a message if the value is empty.
func NotEmpty(s string) string {
	if s == "" {
		return "must not be empty"
	}

	return ""
}

// Fullname accepts any string and returns a message if the value is not a full name.
func Fullname(s string) string {
	if len(strings.Split(s, " ")) < 2 {
		return "must contain first and last name"
	}

	return ""
}

// StringLength accepts any string and returns a message if the length is not in the given range.
func StringLength(min, max int) func(s string) string {
	return func(s string) string {
		n := len(s)

		// Special case: if both min and max are 0, we interpret as "no constraints".
		if min == 0 && max == 0 {
			return ""
		}

		if min == max { // equal.
			if n != min {
				return fmt.Sprintf("must be %d characters", min)
			}

			return ""
		}

		if max == 0 && min > 0 { // only min provided (no maximum).
			if n < min {
				return fmt.Sprintf("must be at least %d characters", min)
			}
			return ""
		}

		if min == 0 && max > 0 { // only max provided (no minimum).
			if n > max {
				return fmt.Sprintf("must be at most %d characters", max)
			}
			return ""
		}

		// Both min and max provided
		if n < min || n > max {
			return fmt.Sprintf("must be between %d and %d characters", min, max)
		}

		return ""
	}
}

// emailRegex is a regular expression to validate email addresses.
// Supports apostrophes (single quotes) in the local part as per RFC 5322.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+'\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Email accepts any string and returns a message if the value is not a valid email address.
func Email(email string) string {
	if !emailRegex.MatchString(email) {
		return "must be a valid email address"
	}

	return ""
}

// PasswordStrength holds information about the password strength criteria
type PasswordStrength struct {
	HasUpper   bool
	HasLower   bool
	HasNumber  bool
	HasSpecial bool
	Length     int
}

// PasswordStrengthLevel represents the required strength of a password
type PasswordStrengthLevel string

const (
	// Password strength levels
	PasswordVeryWeak   PasswordStrengthLevel = "very weak"   // Length >= 4
	PasswordWeak       PasswordStrengthLevel = "weak"        // Length >= 8, 1 category
	PasswordMedium     PasswordStrengthLevel = "medium"      // Length >= 8, 2 categories
	PasswordStrong     PasswordStrengthLevel = "strong"      // Length >= 10, 3 categories
	PasswordVeryStrong PasswordStrengthLevel = "very strong" // Length >= 12, all categories
)

var (
	upperRegex   = regexp.MustCompile(`[A-Z]`)
	lowerRegex   = regexp.MustCompile(`[a-z]`)
	numberRegex  = regexp.MustCompile(`[0-9]`)
	specialRegex = regexp.MustCompile(`[^A-Za-z0-9]`)
)

// analyzePassword examines a password and returns its strength characteristics
func analyzePassword(password string) PasswordStrength {
	return PasswordStrength{
		HasUpper:   upperRegex.MatchString(password),
		HasLower:   lowerRegex.MatchString(password),
		HasNumber:  numberRegex.MatchString(password),
		HasSpecial: specialRegex.MatchString(password),
		Length:     len(password),
	}
}

// getPasswordLevel determines the strength level based on password characteristics
func getPasswordLevel(ps PasswordStrength) PasswordStrengthLevel {
	// Count the number of character categories used
	categories := 0
	if ps.HasUpper {
		categories++
	}
	if ps.HasLower {
		categories++
	}
	if ps.HasNumber {
		categories++
	}
	if ps.HasSpecial {
		categories++
	}

	// Determine password strength
	if ps.Length >= 12 && categories == 4 {
		return PasswordVeryStrong
	} else if ps.Length >= 10 && categories >= 3 {
		return PasswordStrong
	} else if ps.Length >= 8 && categories >= 2 {
		return PasswordMedium
	} else if ps.Length >= 8 && categories >= 1 {
		return PasswordWeak
	} else if ps.Length >= 4 {
		return PasswordVeryWeak
	}

	// Password is too short to meet any criteria
	return "" // Empty string indicates password doesn't meet any level
}

// StrongPassword accepts a password and a required strength level and returns
// a message if the password doesn't meet the required strength.
func StrongPassword(password string, requiredLevel PasswordStrengthLevel) string {
	strength := analyzePassword(password)
	level := getPasswordLevel(strength)

	// Handle too short passwords (less than 4 chars)
	if level == "" {
		return "password is too short (minimum 4 characters required)"
	}

	// Convert levels to numeric values for comparison
	var reqValue, actualValue int

	switch requiredLevel {
	case PasswordVeryWeak:
		reqValue = 1
	case PasswordWeak:
		reqValue = 2
	case PasswordMedium:
		reqValue = 3
	case PasswordStrong:
		reqValue = 4
	case PasswordVeryStrong:
		reqValue = 5
	}

	switch level {
	case PasswordVeryWeak:
		actualValue = 1
	case PasswordWeak:
		actualValue = 2
	case PasswordMedium:
		actualValue = 3
	case PasswordStrong:
		actualValue = 4
	case PasswordVeryStrong:
		actualValue = 5
	}

	if actualValue < reqValue {
		missingCriteria := []string{}

		switch requiredLevel {
		case PasswordVeryStrong:
			if !strength.HasUpper {
				missingCriteria = append(missingCriteria, "uppercase letters")
			}
			if !strength.HasLower {
				missingCriteria = append(missingCriteria, "lowercase letters")
			}
			if !strength.HasNumber {
				missingCriteria = append(missingCriteria, "numbers")
			}
			if !strength.HasSpecial {
				missingCriteria = append(missingCriteria, "special characters")
			}
			if strength.Length < 12 {
				missingCriteria = append(missingCriteria, "at least 12 characters")
			}
		case PasswordStrong:
			categories := 0
			if strength.HasUpper {
				categories++
			}
			if strength.HasLower {
				categories++
			}
			if strength.HasNumber {
				categories++
			}
			if strength.HasSpecial {
				categories++
			}

			if categories < 3 {
				missingCriteria = append(missingCriteria, "at least 3 character types (uppercase, lowercase, numbers, special)")
			}
			if strength.Length < 10 {
				missingCriteria = append(missingCriteria, "at least 10 characters")
			}
		case PasswordMedium:
			categories := 0
			if strength.HasUpper {
				categories++
			}
			if strength.HasLower {
				categories++
			}
			if strength.HasNumber {
				categories++
			}
			if strength.HasSpecial {
				categories++
			}

			if categories < 2 {
				missingCriteria = append(missingCriteria, "at least 2 character types (uppercase, lowercase, numbers, special)")
			}
			if strength.Length < 8 {
				missingCriteria = append(missingCriteria, "at least 8 characters")
			}
		case PasswordWeak:
			if strength.Length < 8 {
				missingCriteria = append(missingCriteria, "at least 8 characters")
			}
		}

		if len(missingCriteria) > 0 {
			return fmt.Sprintf("password must be %s and include %s", requiredLevel, strings.Join(missingCriteria, ", "))
		}

		return fmt.Sprintf("password must be %s", requiredLevel)
	}

	return ""
}

// Password adds a password strength validation to a StringError
func (e *StringError) Password(requiredLevel PasswordStrengthLevel) *StringError {
	e.Func(func(s string) string {
		return StrongPassword(s, requiredLevel)
	})
	return e
}
