package main

import "golang.org/x/crypto/bcrypt"

func init() {
	generateSampleUsers()
}

// User represents our User model.
type User struct {
	ID             uint64 `json:"id"`
	Username       string `json:"username"`
	HashedPassword []byte `json:"-"`
}

// Users represents a user database.
// For the sake of the tutorial we use a simple slice of users.
var Users []User

func generateSampleUsers() {
	Users = []User{
		{ID: 1, Username: "vasiliki", HashedPassword: mustGeneratePassword("vasiliki_pass")}, // my grandmother.
		{ID: 2, Username: "kataras", HashedPassword: mustGeneratePassword("kataras_pass")},   // me.
		{ID: 3, Username: "george", HashedPassword: mustGeneratePassword("george_pass")},     // my young brother.
		{ID: 4, Username: "kwstas", HashedPassword: mustGeneratePassword("kwstas_pass")},     // my youngest brother.
	}
}

func fetchUser(username, password string) (User, bool) {
	for _, u := range Users { // our example uses a static slice.
		if u.Username == username {
			// we compare the user input and the stored hashed password.
			ok := ValidatePassword(password, u.HashedPassword)
			if ok {
				return u, true
			}
		}
	}

	return User{}, false
}

// mustGeneratePassword same as GeneratePassword but panics on errors.
func mustGeneratePassword(userPassword string) []byte {
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
