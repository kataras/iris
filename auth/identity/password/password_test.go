package password

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestGenerateRandomPassword(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		expected int
	}{
		{"minimum_length", 5, 8}, // Should be adjusted to 8
		{"normal_length", 12, 12},
		{"long_length", 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := GenerateRandomPassword(tt.length)
			if err != nil {
				t.Errorf("GenerateRandomPassword() error = %v", err)
				return
			}

			if len(password) != tt.expected {
				t.Errorf("GenerateRandomPassword() length = %d, expected %d", len(password), tt.expected)
			}

			// Verify password meets complexity requirements
			hasUpper, hasLower, hasNumber, hasSymbol := false, false, false, false

			for _, char := range password {
				switch {
				case char >= 'A' && char <= 'Z':
					hasUpper = true
				case char >= 'a' && char <= 'z':
					hasLower = true
				case char >= '0' && char <= '9':
					hasNumber = true
				case char >= '!' && char <= '/' || char >= ':' && char <= '@' || char >= '[' && char <= '`' || char >= '{' && char <= '~':
					hasSymbol = true
				}
			}

			if !hasUpper {
				t.Error("Generated password should contain at least one uppercase letter")
			}
			if !hasLower {
				t.Error("Generated password should contain at least one lowercase letter")
			}
			if !hasNumber {
				t.Error("Generated password should contain at least one number")
			}
			if !hasSymbol {
				t.Error("Generated password should contain at least one symbol")
			}

			t.Logf("Generated password: %s (length: %d)", password, len(password))
		})
	}
}

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"simple_password", "password123"},
		{"complex_password", "ComplexPassword123!@#"},
		{"unicode_password", "пароль123"},
		{"empty_password", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashed, err := HashPassword(tt.password)
			if err != nil {
				t.Errorf("HashPassword() error = %v", err)
				return
			}

			if len(hashed) == 0 {
				t.Error("HashPassword() returned empty hash")
				return
			}

			// Verify the hash can be compared with the original password
			err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte(tt.password))
			if err != nil {
				t.Errorf("Generated hash does not match original password: %v", err)
			}

			t.Logf("Password: %s -> Hash: %s", tt.password, hashed)
		})
	}
}

func TestGenerateSecureRandomInt(t *testing.T) {
	tests := []struct {
		name      string
		max       int
		expectErr bool
	}{
		{"valid_max", 10, false},
		{"max_one", 1, false},
		{"zero_max", 0, true},
		{"negative_max", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range 100 { // Test multiple times for randomness
				result, err := generateSecureRandomInt(tt.max)

				if tt.expectErr {
					if err == nil {
						t.Errorf("generateSecureRandomInt() expected error but got none")
					}
					return
				}

				if err != nil {
					t.Errorf("generateSecureRandomInt() error = %v", err)
					return
				}

				if result < 0 || result >= tt.max {
					t.Errorf("generateSecureRandomInt() = %d, should be in range [0, %d)", result, tt.max)
				}
			}
		})
	}
}

// BenchmarkGenerateRandomPassword benchmarks password generation
func BenchmarkGenerateRandomPassword(b *testing.B) {
	for range b.N {
		_, err := GenerateRandomPassword(12)
		if err != nil {
			b.Errorf("GenerateRandomPassword() error = %v", err)
		}
	}
}

// BenchmarkHashPassword benchmarks password hashing
func BenchmarkHashPassword(b *testing.B) {
	password := "BenchmarkPassword123!"

	for range b.N {
		_, err := HashPassword(password)
		if err != nil {
			b.Errorf("HashPassword() error = %v", err)
		}
	}
}
