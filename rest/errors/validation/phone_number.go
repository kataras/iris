package validation

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Basic phone number patterns
	basicPhoneRegex = regexp.MustCompile(`^[+]?[0-9\s\-()]+$`)

	// E.164 format: +CountryCodeSubscriberNumber, e.g. +14155552671
	// Note: This should be strict - no spaces or formatting characters allowed
	e164Regex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

	// International format with optional country code
	internationalPhoneRegex = regexp.MustCompile(`^(?:\+|00)[1-9][0-9\s\-()]{7,20}$`)
)

// PhoneType represents different phone number format standards
type PhoneType string

const (
	PhoneAny           PhoneType = "any"            // Any reasonable phone format
	PhoneE164          PhoneType = "e164"           // E.164 international format
	PhoneNorthAmerican PhoneType = "north_american" // North American numbering plan
	PhoneInternational PhoneType = "international"  // International format with country code
)

// ParsePhoneType converts a string representation of a phone type into a PhoneType constant.
// It supports "e164", "north_american" (or "us"), and "international".
func ParsePhoneType(pt string) PhoneType {
	switch strings.ToLower(pt) {
	case "e164":
		return PhoneE164
	case "north_american", "us":
		return PhoneNorthAmerican
	case "international":
		return PhoneInternational
	default:
		return PhoneAny // Default to any reasonable phone format
	}
}

// PhoneNumber validates that a string represents a valid phone number according to the specified format.
//
// The function accepts a PhoneType parameter that determines the validation rules:
//
// PhoneAny (default): Accepts any reasonable phone number format with 7-15 digits.
//
//	Examples:
//	  - "555-123-4567"
//	  - "(555) 123-4567"
//	  - "+1 555 123 4567"
//	  - "5551234567"
//
// PhoneE164: Validates according to the E.164 international standard format.
//
//	This format requires a + prefix followed by the country code and subscriber number,
//	with no spaces, dashes, or other formatting characters.
//	Examples:
//	  - "+15551234567" (US/Canada)
//	  - "+442079460958" (UK)
//	  - "+61291234567" (Australia)
//
// PhoneNorthAmerican: Validates according to the North American Numbering Plan.
//
//	Requires a 3-digit area code (first digit 2-9) and a 7-digit local number (first digit of prefix 2-9).
//	Examples:
//	  - "(555) 123-4567"
//	  - "555-123-4567"
//	  - "5551234567"
//	  - "+1 (555) 123-4567"
//
// PhoneInternational: Validates international phone numbers with country codes.
//
//	Must start with a + or 00, followed by the country code and subscriber number.
//	Examples:
//	  - "+44 20 7946 0958" (UK)
//	  - "+61 2 9123 4567" (Australia)
//	  - "+49 30 12345678" (Germany)
//
// Returns an empty string if valid, or an error message if invalid.
// PhoneNumber validates that a string represents a valid phone number according to the specified format.
func PhoneNumber(phoneType PhoneType) func(string) string {
	return func(phone string) string {
		if phone == "" {
			return "phone number cannot be empty"
		}

		switch phoneType {
		case PhoneE164:
			// E.164 should be strictly formatted - no cleaning
			if !e164Regex.MatchString(phone) {
				return "must be a valid phone number in E.164 format (e.g., +14155552671)"
			}
		case PhoneNorthAmerican:
			// Use a more direct approach for North American numbers
			isValid := validateNorthAmericanPhone(phone)
			if !isValid {
				return "must be a valid North American phone number (e.g., (415) 555-2671)"
			}
		case PhoneInternational:
			// Remove common formatting characters for international numbers
			cleanedPhone := cleanPhoneNumber(phone)
			if !internationalPhoneRegex.MatchString(cleanedPhone) {
				return "must be a valid international phone number with country code (e.g., +44 20 1234 5678)"
			}
		case PhoneAny:
			fallthrough
		default:
			if !isBasicPhoneFormat(phone) {
				return "must be a valid phone number"
			}
		}

		return ""
	}
}

// validateNorthAmericanPhone checks if a phone number follows North American numbering plan rules (NANP rules).
func validateNorthAmericanPhone(phone string) bool {
	// Extract only digits from the phone number.
	digits := extractDigits(phone)

	// Check if we have a valid length
	switch len(digits) {
	case 10:
		// 10 digits is the standard NANP format (no country code).
		// Check area code (first digit) must be 2-9.
		if digits[0] < '2' || digits[0] > '9' {
			return false
		}
		// Check prefix (4th digit) must be 2-9.
		if digits[3] < '2' || digits[3] > '9' {
			return false
		}
		return true

	case 11:
		// Check if it starts with country code 1.
		if digits[0] != '1' {
			return false
		}
		// Check area code (after country code) must be 2-9.
		if digits[1] < '2' || digits[1] > '9' {
			return false
		}
		// Check prefix (7th digit) must be 2-9.
		if digits[4] < '2' || digits[4] > '9' {
			return false
		}
		return true

	default:
		// Any other length is invalid.
		return false
	}
}

// cleanPhoneNumber removes spaces, dashes, and parentheses from a phone number
func cleanPhoneNumber(phone string) string {
	// Replace common separators with nothing
	cleaned := strings.ReplaceAll(phone, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	return cleaned
}

// isBasicPhoneFormat checks if the string is a reasonable phone number format
func isBasicPhoneFormat(phone string) bool {
	// Check if the phone number matches the basic pattern
	if !basicPhoneRegex.MatchString(phone) {
		return false
	}

	// Count the number of digits
	digitCount := 0
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			digitCount++
		}
	}

	// Most phone numbers should have at least 7 digits and not more than 15
	return digitCount >= 7 && digitCount <= 15
}

// FormatPhoneNumber attempts to format a phone number into a standardized format
// Returns the formatted phone number and a bool indicating success
func FormatPhoneNumber(phone string, format PhoneType) (string, bool) {
	// Clean the phone number first
	cleaned := cleanPhoneNumber(phone)

	switch format {
	case PhoneE164:
		// If already in E.164 format, return as is
		if e164Regex.MatchString(cleaned) {
			return cleaned, true
		}

		// Extract digits only for conversion
		digits := extractDigits(phone)

		// North American format conversion
		if len(digits) == 10 && validateNorthAmericanPhone(phone) {
			return "+1" + digits, true
		} else if len(digits) == 11 && digits[0] == '1' && validateNorthAmericanPhone(phone) {
			return "+" + digits, true
		}

		return "", false

	case PhoneNorthAmerican:
		// Extract digits only
		digits := extractDigits(phone)

		// Check if it's a valid North American number
		if validateNorthAmericanPhone(phone) {
			if len(digits) == 10 {
				return fmt.Sprintf("(%s) %s-%s", digits[0:3], digits[3:6], digits[6:10]), true
			} else if len(digits) == 11 && digits[0] == '1' {
				return fmt.Sprintf("(%s) %s-%s", digits[1:4], digits[4:7], digits[7:11]), true
			}
		}

		return "", false

	case PhoneInternational:
		// If already in international format, preserve its structure
		if strings.HasPrefix(cleaned, "+") {
			// Handle formatting for specific country codes
			if strings.HasPrefix(cleaned, "+44") {
				// UK format
				if len(cleaned) >= 12 {
					return fmt.Sprintf("+44 %s %s %s", cleaned[3:5], cleaned[5:9], cleaned[9:]), true
				}
			} else if len(cleaned) >= 5 {
				// Generic international format - extract country code (1-3 digits)
				countryCodeEnd := 1 // Start after the +
				for countryCodeEnd < len(cleaned) && countryCodeEnd < 4 && cleaned[countryCodeEnd] >= '0' && cleaned[countryCodeEnd] <= '9' {
					countryCodeEnd++
				}

				countryCode := cleaned[0:countryCodeEnd]
				subscriberNumber := cleaned[countryCodeEnd:]

				// Format based on expected country patterns
				if countryCode == "+1" {
					// North American pattern
					if len(subscriberNumber) == 10 {
						return fmt.Sprintf("%s (%s) %s-%s",
							countryCode,
							subscriberNumber[0:3],
							subscriberNumber[3:6],
							subscriberNumber[6:10]), true
					}
				} else if countryCode == "+44" {
					// UK pattern
					if len(subscriberNumber) >= 9 {
						return fmt.Sprintf("%s %s %s %s",
							countryCode,
							subscriberNumber[0:2],
							subscriberNumber[2:6],
							subscriberNumber[6:]), true
					}
				} else {
					// Generic spacing pattern for readability
					return fmt.Sprintf("%s %s", countryCode, formatWithSpaces(subscriberNumber, 4)), true
				}
			}

			// If we can't apply specific formatting, keep as is
			return cleaned, true
		}

		return "", false

	default: // PhoneAny
		// Try each format in order of preference
		if validateNorthAmericanPhone(phone) {
			formatted, success := FormatPhoneNumber(phone, PhoneNorthAmerican)
			if success {
				return formatted, true
			}
		}

		if strings.HasPrefix(cleaned, "+") {
			formatted, success := FormatPhoneNumber(phone, PhoneInternational)
			if success {
				return formatted, true
			}
		}

		// Just check if it's a valid phone number format
		if isBasicPhoneFormat(phone) {
			// Extract digits for basic formatting
			digits := extractDigits(phone)

			// Apply basic formatting based on length
			if len(digits) == 10 {
				return fmt.Sprintf("(%s) %s-%s", digits[0:3], digits[3:6], digits[6:10]), true
			} else if len(digits) == 11 && digits[0] == '1' {
				return fmt.Sprintf("(%s) %s-%s", digits[1:4], digits[4:7], digits[7:11]), true
			}

			// Fall back to minimal formatting
			return formatWithSpaces(digits, 4), true
		}

		return "", false
	}
}

// extractDigits returns the ASCII digits of s, in order, discarding everything else.
func extractDigits(s string) string {
	var digits strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' {
			digits.WriteByte(byte(c))
		}
	}

	return digits.String()
}

// Helper function to format a string with spaces every n characters
func formatWithSpaces(s string, n int) string {
	if n <= 0 {
		return s
	}

	var result strings.Builder
	for i := 0; i < len(s); i += n {
		end := min(i+n, len(s))

		result.WriteString(s[i:end])
		if end < len(s) {
			result.WriteString(" ")
		}
	}

	return result.String()
}

// Phone adds an error if the string is not a valid phone number of the given type.
func (e *StringError) Phone(phoneType PhoneType) *StringError {
	e.Func(PhoneNumber(phoneType))
	return e
}
