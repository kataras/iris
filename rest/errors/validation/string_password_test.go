package validation

import (
	"strings"
	"testing"
)

func TestStrongPassword(t *testing.T) {
	tests := []struct {
		name           string
		password       string
		requiredLevel  PasswordStrengthLevel
		expectedError  bool
		expectedPhrase string
	}{
		// Tests for too short passwords
		{
			name:           "Empty password",
			password:       "",
			requiredLevel:  PasswordWeak,
			expectedError:  true,
			expectedPhrase: "too short",
		},
		{
			name:           "Password less than 4 chars",
			password:       "abc",
			requiredLevel:  PasswordVeryWeak,
			expectedError:  true,
			expectedPhrase: "too short",
		},

		// Tests for PasswordVeryWeak level
		{
			name:          "Very weak password - meets requirements",
			password:      "1234",
			requiredLevel: PasswordVeryWeak,
			expectedError: false,
		},
		{
			name:          "Very weak password - exceeds requirements",
			password:      "password1",
			requiredLevel: PasswordVeryWeak,
			expectedError: false,
		},

		// Tests for PasswordWeak level
		{
			name:           "Weak level - too short",
			password:       "pass",
			requiredLevel:  PasswordWeak,
			expectedError:  true,
			expectedPhrase: "at least 8 characters",
		},
		{
			name:          "Weak level - meets requirements",
			password:      "password",
			requiredLevel: PasswordWeak,
			expectedError: false,
		},
		{
			name:          "Weak level - exceeds requirements",
			password:      "Password1",
			requiredLevel: PasswordWeak,
			expectedError: false,
		},

		// Tests for PasswordMedium level
		{
			name:           "Medium level - too short",
			password:       "pass",
			requiredLevel:  PasswordMedium,
			expectedError:  true,
			expectedPhrase: "at least 8 characters",
		},
		{
			name:           "Medium level - not enough categories",
			password:       "password",
			requiredLevel:  PasswordMedium,
			expectedError:  true,
			expectedPhrase: "at least 2 character types",
		},
		{
			name:          "Medium level - meets requirements (lowercase + numbers)",
			password:      "password123",
			requiredLevel: PasswordMedium,
			expectedError: false,
		},
		{
			name:          "Medium level - meets requirements (lowercase + uppercase)",
			password:      "passwordABC",
			requiredLevel: PasswordMedium,
			expectedError: false,
		},
		{
			name:          "Medium level - exceeds requirements",
			password:      "Password1!",
			requiredLevel: PasswordMedium,
			expectedError: false,
		},

		// Tests for PasswordStrong level
		{
			name:           "Strong level - too short",
			password:       "Pass1!",
			requiredLevel:  PasswordStrong,
			expectedError:  true,
			expectedPhrase: "at least 10 characters",
		},
		{
			name:           "Strong level - not enough categories",
			password:       "passwordAB",
			requiredLevel:  PasswordStrong,
			expectedError:  true,
			expectedPhrase: "at least 3 character types",
		},
		{
			name:          "Strong level - meets requirements",
			password:      "Password123",
			requiredLevel: PasswordStrong,
			expectedError: false,
		},
		{
			name:          "Strong level - exceeds requirements",
			password:      "Password123!",
			requiredLevel: PasswordStrong,
			expectedError: false,
		},

		// Tests for PasswordVeryStrong level
		{
			name:           "Very strong level - too short",
			password:       "Pass1!",
			requiredLevel:  PasswordVeryStrong,
			expectedError:  true,
			expectedPhrase: "at least 12 characters",
		},
		{
			name:           "Very strong level - missing uppercase",
			password:       "password123!@#",
			requiredLevel:  PasswordVeryStrong,
			expectedError:  true,
			expectedPhrase: "uppercase letters",
		},
		{
			name:           "Very strong level - missing lowercase",
			password:       "PASSWORD123!@#",
			requiredLevel:  PasswordVeryStrong,
			expectedError:  true,
			expectedPhrase: "lowercase letters",
		},
		{
			name:           "Very strong level - missing numbers",
			password:       "Password!@#$%^",
			requiredLevel:  PasswordVeryStrong,
			expectedError:  true,
			expectedPhrase: "numbers",
		},
		{
			name:           "Very strong level - missing special characters",
			password:       "Password12345",
			requiredLevel:  PasswordVeryStrong,
			expectedError:  true,
			expectedPhrase: "special characters",
		},
		{
			name:          "Very strong level - meets requirements",
			password:      "Password123!@#",
			requiredLevel: PasswordVeryStrong,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StrongPassword(tt.password, tt.requiredLevel)

			if tt.expectedError && result == "" {
				t.Errorf("Expected error message, but got no error")
			}

			if !tt.expectedError && result != "" {
				t.Errorf("Expected no error, but got: %s", result)
			}

			if tt.expectedError && tt.expectedPhrase != "" && !strings.Contains(result, tt.expectedPhrase) {
				t.Errorf("Expected error message to contain '%s', but got: '%s'", tt.expectedPhrase, result)
			}
		})
	}
}

func TestPasswordValidation(t *testing.T) {
	// Test the StringError.Password method
	tests := []struct {
		name          string
		password      string
		requiredLevel PasswordStrengthLevel
		expectedError bool
	}{
		{
			name:          "Valid password",
			password:      "Password123!",
			requiredLevel: PasswordStrong,
			expectedError: false,
		},
		{
			name:          "Invalid password",
			password:      "pass",
			requiredLevel: PasswordMedium,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := String("password", tt.password).Password(tt.requiredLevel)

			if tt.expectedError && err.Get() == nil {
				t.Errorf("Expected validation error, but got none")
			}

			if !tt.expectedError && err.Get() != nil {
				t.Errorf("Expected no validation error, but got: %v", err.Get())
			}
		})
	}
}

func TestAnalyzePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     PasswordStrength
	}{
		{
			name:     "Empty password",
			password: "",
			want: PasswordStrength{
				HasUpper:   false,
				HasLower:   false,
				HasNumber:  false,
				HasSpecial: false,
				Length:     0,
			},
		},
		{
			name:     "Lowercase only",
			password: "password",
			want: PasswordStrength{
				HasUpper:   false,
				HasLower:   true,
				HasNumber:  false,
				HasSpecial: false,
				Length:     8,
			},
		},
		{
			name:     "Mixed case",
			password: "Password",
			want: PasswordStrength{
				HasUpper:   true,
				HasLower:   true,
				HasNumber:  false,
				HasSpecial: false,
				Length:     8,
			},
		},
		{
			name:     "With numbers",
			password: "Password123",
			want: PasswordStrength{
				HasUpper:   true,
				HasLower:   true,
				HasNumber:  true,
				HasSpecial: false,
				Length:     11,
			},
		},
		{
			name:     "With special characters",
			password: "Password123!@#",
			want: PasswordStrength{
				HasUpper:   true,
				HasLower:   true,
				HasNumber:  true,
				HasSpecial: true,
				Length:     14,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzePassword(tt.password)
			if got.HasUpper != tt.want.HasUpper ||
				got.HasLower != tt.want.HasLower ||
				got.HasNumber != tt.want.HasNumber ||
				got.HasSpecial != tt.want.HasSpecial ||
				got.Length != tt.want.Length {
				t.Errorf("analyzePassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPasswordLevel(t *testing.T) {
	tests := []struct {
		name     string
		strength PasswordStrength
		want     PasswordStrengthLevel
	}{
		{
			name: "Too short",
			strength: PasswordStrength{
				Length: 3,
			},
			want: "",
		},
		{
			name: "Very weak",
			strength: PasswordStrength{
				HasLower: true,
				Length:   4,
			},
			want: PasswordVeryWeak,
		},
		{
			name: "Weak",
			strength: PasswordStrength{
				HasLower: true,
				Length:   8,
			},
			want: PasswordWeak,
		},
		{
			name: "Medium",
			strength: PasswordStrength{
				HasLower: true,
				HasUpper: true,
				Length:   8,
			},
			want: PasswordMedium,
		},
		{
			name: "Strong",
			strength: PasswordStrength{
				HasLower:  true,
				HasUpper:  true,
				HasNumber: true,
				Length:    10,
			},
			want: PasswordStrong,
		},
		{
			name: "Very strong",
			strength: PasswordStrength{
				HasLower:   true,
				HasUpper:   true,
				HasNumber:  true,
				HasSpecial: true,
				Length:     12,
			},
			want: PasswordVeryStrong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPasswordLevel(tt.strength); got != tt.want {
				t.Errorf("getPasswordLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
