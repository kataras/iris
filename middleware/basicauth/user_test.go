package basicauth

import (
	"errors"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/kataras/iris/v12/context"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

type IUserRepository interface {
	GetByUsernameAndPassword(dest interface{}, username, password string) error
}

// Test a custom implementation of AuthFunc with a user repository.
// This is a usage example of custom AuthFunc implementation.
func UserRepository(repo IUserRepository, newUserPtr func() interface{}) AuthFunc {
	return func(_ *context.Context, username, password string) (interface{}, bool) {
		dest := newUserPtr()
		err := repo.GetByUsernameAndPassword(dest, username, password)
		if err == nil {
			return dest, true
		}

		return nil, false
	}
}

type testUser struct {
	username string
	password string
	email    string // custom field.
}

// GetUsername & Getpassword complete the User interface.
func (u *testUser) GetUsername() string {
	return u.username
}

func (u *testUser) GetPassword() string {
	return u.password
}

type testRepo struct {
	entries []testUser
}

// Implements IUserRepository interface.
func (r *testRepo) GetByUsernameAndPassword(dest interface{}, username, password string) error {
	for _, e := range r.entries {
		if e.username == username && e.password == password {
			*dest.(*testUser) = e
			return nil
		}
	}

	return errors.New("invalid credentials")
}

func TestAllowUserRepository(t *testing.T) {
	repo := &testRepo{
		entries: []testUser{
			{username: "kataras", password: "kataras_pass", email: "kataras2006@hotmail.com"},
		},
	}

	allow := UserRepository(repo, func() interface{} {
		return new(testUser)
	})

	var tests = []struct {
		username string
		password string
		ok       bool
		user     *testUser
	}{
		{
			username: "kataras",
			password: "kataras_pass",
			ok:       true,
			user:     &testUser{username: "kataras", password: "kataras_pass", email: "kataras2006@hotmail.com"},
		},
		{
			username: "makis",
			password: "makis_password",
			ok:       false,
		},
	}

	for i, tt := range tests {
		v, ok := allow(nil, tt.username, tt.password)

		if tt.ok != ok {
			t.Fatalf("[%d] expected: %v but got: %v (username=%s,password=%s)", i, tt.ok, ok, tt.username, tt.password)
		}

		if !ok {
			continue
		}

		u, ok := v.(*testUser)
		if !ok {
			t.Fatalf("[%d] a user should be type of *testUser but got: %#+v (%T)", i, v, v)
		}

		if !reflect.DeepEqual(tt.user, u) {
			t.Fatalf("[%d] expected user:\n%#+v\nbut got:\n%#+v", i, tt.user, u)
		}
	}
}

func TestAllowUsers(t *testing.T) {
	users := []User{
		&testUser{username: "kataras", password: "kataras_pass", email: "kataras2006@hotmail.com"},
	}

	allow := AllowUsers(users)

	var tests = []struct {
		username string
		password string
		ok       bool
		user     *testUser
	}{
		{
			username: "kataras",
			password: "kataras_pass",
			ok:       true,
			user:     &testUser{username: "kataras", password: "kataras_pass", email: "kataras2006@hotmail.com"},
		},
		{
			username: "makis",
			password: "makis_password",
			ok:       false,
		},
	}

	for i, tt := range tests {
		v, ok := allow(nil, tt.username, tt.password)

		if tt.ok != ok {
			t.Fatalf("[%d] expected: %v but got: %v (username=%s,password=%s)", i, tt.ok, ok, tt.username, tt.password)
		}

		if !ok {
			continue
		}

		u, ok := v.(*testUser)
		if !ok {
			t.Fatalf("[%d] a user should be type of *testUser but got: %#+v (%T)", i, v, v)
		}

		if !reflect.DeepEqual(tt.user, u) {
			t.Fatalf("[%d] expected user:\n%#+v\nbut got:\n%#+v", i, tt.user, u)
		}
	}
}

// Test YAML user loading with b-encrypted passwords.
func TestAllowUsersFile(t *testing.T) {
	f, err := ioutil.TempFile("", "*users.yml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	// 	f.WriteString(`
	// - username: kataras
	//   password: kataras_pass
	//   age: 27
	//   role: admin
	// - username: makis
	//   password: makis_password
	// `)
	// This form is supported too, although its features are limited (no custom fields):
	// 	f.WriteString(`
	// kataras: kataras_pass
	// makis: makis_password
	// `)

	var tests = []struct {
		username      string
		password      string // hashed, auto-filled later on.
		inputPassword string
		ok            bool
		user          context.Map
	}{
		{
			username:      "kataras",
			inputPassword: "kataras_pass",
			ok:            true,
			user:          context.Map{"age": 27, "role": "admin"}, // username and password are auto-filled in our tests below.
		},
		{
			username:      "makis",
			inputPassword: "makis_password",
			ok:            true,
			user:          context.Map{},
		},
		{
			username: "invalid",
			password: "invalid_pass",
			ok:       false,
		},
		{
			username: "notvalid",
			password: "",
			ok:       false,
		},
	}

	// Write the tests to the users YAML file.
	var usersToWrite []context.Map
	for _, tt := range tests {
		if tt.ok {
			// store the hashed password.
			tt.password = mustGeneratePassword(t, tt.inputPassword)

			// store and write the username and hashed password.
			tt.user["username"] = tt.username
			tt.user["password"] = tt.password

			// cannot write it as a stream, write it as a slice.
			// enc.Encode(tt.user)
			usersToWrite = append(usersToWrite, tt.user)
		}
		// 	bcrypt.GenerateFromPassword([]byte("kataras_pass"), bcrypt.DefaultCost)
	}

	fileContents, err := yaml.Marshal(usersToWrite)
	if err != nil {
		t.Fatal(err)
	}
	f.Write(fileContents)

	// Build the authentication func.
	allow := AllowUsersFile(f.Name(), BCRYPT)
	for i, tt := range tests {
		v, ok := allow(nil, tt.username, tt.inputPassword)

		if tt.ok != ok {
			t.Fatalf("[%d] expected: %v but got: %v (username=%s,password=%s,user=%#+v)", i, tt.ok, ok, tt.username, tt.inputPassword, v)
		}

		if !ok {
			continue
		}

		if len(tt.user) == 0 { // when username: password form.
			continue
		}

		u, ok := v.(context.Map)
		if !ok {
			t.Fatalf("[%d] a user loaded from external source or file should be alway type of map[string]interface{} but got: %#+v (%T)", i, v, v)
		}

		if expected, got := len(tt.user), len(u); expected != got {
			t.Fatalf("[%d] expected user map length to be equal, expected: %d but got: %d\n%#+v\n%#+v", i, expected, got, tt.user, u)
		}

		for k, v := range tt.user {
			if u[k] != v {
				t.Fatalf("[%d] expected user map %q to be %q but got: %q", i, k, v, u[k])
			}
		}
	}

}

func mustGeneratePassword(t *testing.T, userPassword string) string {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	return string(hashed)
}
