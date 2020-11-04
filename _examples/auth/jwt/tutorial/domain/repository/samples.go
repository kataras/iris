package repository

import (
	"fmt"

	"myapp/domain/model"
)

// GenerateSamples generates data samples.
func GenerateSamples(userRepo UserRepository, todoRepo TodoRepository) error {
	// Create users.
	for _, username := range []string{"vasiliki", "george", "kwstas"} {
		// My grandmother.
		// My young brother.
		// My youngest brother.
		password := fmt.Sprintf("%s_pass", username)
		if _, err := userRepo.Create(username, password); err != nil {
			return err
		}
	}

	// Create a user with admin role.
	if _, err := userRepo.Create("admin", "admin", model.Admin); err != nil {
		return err
	}

	// Create two todos per user.
	users, err := userRepo.GetAll()
	if err != nil {
		return err
	}

	for i, u := range users {
		for j := 0; j < 2; j++ {
			title := fmt.Sprintf("%s todo %d:%d title", u.Username, i, j)
			body := fmt.Sprintf("%s todo %d:%d body", u.Username, i, j)
			_, err := todoRepo.Create(u.ID, title, body)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
