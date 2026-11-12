package validation

import (
	"testing"
)

func TestPhoneNumber(t *testing.T) {
	tests := []struct {
		name      string
		phoneType PhoneType
		value     string
		wantError bool
	}{
		// Any format tests
		{"Valid US number with parentheses", PhoneAny, "(555) 123-4567", false},
		{"Valid US number with dashes", PhoneAny, "555-123-4567", false},
		{"Valid US number with spaces", PhoneAny, "555 123 4567", false},
		{"Valid US number plain", PhoneAny, "5551234567", false},
		{"Valid international number", PhoneAny, "+44 20 7946 0958", false},
		{"Valid number with country code", PhoneAny, "+1 555 123 4567", false},
		{"Too few digits", PhoneAny, "123456", true},
		{"Too many digits", PhoneAny, "12345678901234567890", true},
		{"Invalid characters", PhoneAny, "555-ABC-4567", true},
		{"Empty string", PhoneAny, "", true},

		// E.164 format tests
		{"Valid E.164 format", PhoneE164, "+15551234567", false},
		{"Valid E.164 international", PhoneE164, "+442079460958", false},
		{"E.164 without plus", PhoneE164, "15551234567", true},
		{"E.164 with spaces", PhoneE164, "+1 555 123 4567", true},
		{"E.164 with dashes", PhoneE164, "+1-555-123-4567", true},
		{"E.164 too long", PhoneE164, "+1555123456789012345", true},

		// North American format tests
		{"Valid NA with parentheses", PhoneNorthAmerican, "(555) 234-5678", false},
		{"Valid NA with dashes", PhoneNorthAmerican, "555-234-5678", false},
		{"Valid NA with country code", PhoneNorthAmerican, "+1 (555) 234-5678", false},
		{"Valid NA plain", PhoneNorthAmerican, "5552345678", false},
		{"Invalid NA area code", PhoneNorthAmerican, "(055) 234-5678", true},
		{"Invalid NA prefix", PhoneNorthAmerican, "(555) 023-5678", true},
		{"Invalid NA length", PhoneNorthAmerican, "555-234-567", true},
		{"NA with letters", PhoneNorthAmerican, "(555) 234-ABCD", true},

		// International format tests
		{"Valid international with plus", PhoneInternational, "+44 20 7946 0958", false},
		{"Valid international compact", PhoneInternational, "+442079460958", false},
		{"Valid international with dashes", PhoneInternational, "+44-20-7946-0958", false},
		{"Valid international with parentheses", PhoneInternational, "+44 (0) 20 7946 0958", false},
		{"International without country code", PhoneInternational, "20 7946 0958", true},
		{"International with invalid prefix", PhoneInternational, "++44 20 7946 0958", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := PhoneNumber(tt.phoneType)
			result := validator(tt.value)

			hasError := result != ""
			if hasError != tt.wantError {
				t.Errorf("PhoneNumber(%v)(%q) = %q, wantError %v",
					tt.phoneType, tt.value, result, tt.wantError)
			}
		})
	}
}

func TestFormatPhoneNumber(t *testing.T) {
	tests := []struct {
		name        string
		phoneType   PhoneType
		input       string
		wantOutput  string
		wantSuccess bool
	}{
		// E.164 format
		{"E.164 already formatted", PhoneE164, "+15552345678", "+15552345678", true},
		{"E.164 from NA format", PhoneE164, "(555) 234-5678", "+15552345678", true},
		{"E.164 from NA with spaces", PhoneE164, "555 234 5678", "+15552345678", true},
		{"E.164 invalid input", PhoneE164, "not-a-phone", "", false},

		// North American format
		{"NA already formatted", PhoneNorthAmerican, "(555) 234-5678", "(555) 234-5678", true},
		{"NA from E.164", PhoneNorthAmerican, "+15552345678", "(555) 234-5678", true},
		{"NA from plain digits", PhoneNorthAmerican, "5552345678", "(555) 234-5678", true},
		{"NA invalid input", PhoneNorthAmerican, "+44 20 1234 5678", "", false},

		// International format
		{"International already formatted", PhoneInternational, "+44 20 1234 5678", "+44 20 1234 5678", true},
		{"International from compact", PhoneInternational, "+442012345678", "+44 20 1234 5678", true},
		{"International invalid input", PhoneInternational, "not-a-phone", "", false},

		// Any format
		{"Any format - NA style", PhoneAny, "5552345678", "(555) 234-5678", true},
		{"Any format - international", PhoneAny, "+442012345678", "+44 20 1234 5678", true},
		{"Any format - invalid", PhoneAny, "abc", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, success := FormatPhoneNumber(tt.input, tt.phoneType)

			if success != tt.wantSuccess {
				t.Errorf("FormatPhoneNumber(%q, %v) success = %v, want %v",
					tt.input, tt.phoneType, success, tt.wantSuccess)
			}

			if got != tt.wantOutput {
				t.Errorf("FormatPhoneNumber(%q, %v) = %q, want %q",
					tt.input, tt.phoneType, got, tt.wantOutput)
			}
		})
	}
}

func TestStringError_Phone(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		phoneType PhoneType
		wantError bool
	}{
		{"Valid phone", "(555) 234-5678", PhoneAny, false},
		{"Invalid phone", "123", PhoneAny, true},
		{"Valid E.164", "+15552345678", PhoneE164, false},
		{"Invalid E.164", "5552345678", PhoneE164, true},
		{"Valid North American", "(555) 234-5678", PhoneNorthAmerican, false},
		{"Invalid North American", "(555) 123-4567", PhoneNorthAmerican, true}, // Invalid prefix (starts with 1)
		{"Valid International", "+44 20 7946 0958", PhoneInternational, false},
		{"Invalid International", "20 7946 0958", PhoneInternational, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := String("phone", tt.value)
			e.Phone(tt.phoneType)

			if (e.Get() != nil) != tt.wantError {
				t.Errorf("String(%q).Phone(%v) errors = %v, wantError %v",
					tt.value, tt.phoneType, e.Error(), tt.wantError)
			}
		})
	}
}

func TestIsBasicPhoneFormat(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  bool
	}{
		{"Valid simple", "5551234567", true},
		{"Valid with formatting", "(555) 123-4567", true},
		{"Valid international", "+44 20 1234 5678", true},
		{"Too short", "123456", false},
		{"Too long", "12345678901234567890", false},
		{"With letters", "555-ABC-4567", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBasicPhoneFormat(tt.phone); got != tt.want {
				t.Errorf("isBasicPhoneFormat(%q) = %v, want %v", tt.phone, got, tt.want)
			}
		})
	}
}

func TestCleanPhoneNumber(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  string
	}{
		{"With parentheses", "(555) 123-4567", "5551234567"},
		{"With dashes", "555-123-4567", "5551234567"},
		{"With spaces", "555 123 4567", "5551234567"},
		{"With international code", "+1 (555) 123-4567", "+15551234567"},
		{"Already clean", "5551234567", "5551234567"},
		{"Mixed characters", "Phone: (555) 123-4567", "Phone:5551234567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanPhoneNumber(tt.phone); got != tt.want {
				t.Errorf("cleanPhoneNumber(%q) = %q, want %q", tt.phone, got, tt.want)
			}
		})
	}
}

func TestNAPhoneValidation(t *testing.T) {
	testCases := []struct {
		phone string
		valid bool
	}{
		{"(555) 234-5678", true},    // Changed to valid prefix (234)
		{"555-234-5678", true},      // Changed to valid prefix
		{"+1 (555) 234-5678", true}, // Changed to valid prefix
		{"5552345678", true},        // Changed to valid prefix
		{"123-456-7890", false},     // Invalid area code (starts with 1)
	}

	for _, tc := range testCases {
		result := validateNorthAmericanPhone(tc.phone)
		if result != tc.valid {
			t.Errorf("validateNorthAmericanPhone(%q) = %v, want %v",
				tc.phone, result, tc.valid)
		}
	}
}
