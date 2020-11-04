package util

import "golang.org/x/crypto/bcrypt"

// MustGeneratePassword same as GeneratePassword but panics on errors.
func MustGeneratePassword(userPassword string) []byte {
	hashed, err := GeneratePassword(userPassword)
	if err != nil {
		panic(err)
	}

	return hashed
}

// GeneratePassword will generate a hashed password for us based on the
// user's input.
func GeneratePassword(userPassword string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
}

// ValidatePassword will check if passwords are matched.
func ValidatePassword(userPassword string, hashed []byte) bool {
	err := bcrypt.CompareHashAndPassword(hashed, []byte(userPassword))
	return err == nil
}
