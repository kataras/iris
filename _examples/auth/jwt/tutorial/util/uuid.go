package util

import "github.com/google/uuid"

// MustGenerateUUID returns a new v4 UUID or panics.
func MustGenerateUUID() string {
	id, err := GenerateUUID()
	if err != nil {
		panic(err)
	}

	return id
}

// GenerateUUID returns a new v4 UUID.
func GenerateUUID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return id.String(), nil
}
