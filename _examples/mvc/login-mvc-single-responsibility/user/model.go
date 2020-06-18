package user

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Model is our User example model.
type Model struct {
	ID        int64  `json:"id"`
	Firstname string `json:"firstname"`
	Username  string `json:"username"`
	// password is the client-given password
	// which will not be stored anywhere in the server.
	// It's here only for actions like registration and update password,
	// because we caccept a Model instance
	// inside the `DataSource#InsertOrUpdate` function.
	password       string
	HashedPassword []byte    `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
}

// GeneratePassword will generate a hashed password for us based on the
// user's input.
func GeneratePassword(userPassword string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
}

// ValidatePassword will check if passwords are matched.
func ValidatePassword(userPassword string, hashed []byte) (bool, error) {
	if err := bcrypt.CompareHashAndPassword(hashed, []byte(userPassword)); err != nil {
		return false, err
	}
	return true, nil
}
