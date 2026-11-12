package basicauth

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kataras/iris/v14/context"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

type IUserRepository interface {
	GetByUsernameAndPassword(dest any, username, password string) error
}

// UserRepository is a usage example of a custom, typed AuthFunc
// backed by a user repository (e.g. a database).
func UserRepository[U any](repo IUserRepository, newUserPtr func() *U) AuthFunc[*U] {
	return func(_ *context.Context, username, password string) (*U, bool) {
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

// GetUsername & GetPassword complete the User interface.
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
func (r *testRepo) GetByUsernameAndPassword(dest any, username, password string) error {
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

	allow := UserRepository(repo, func() *testUser {
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
		u, ok := allow(nil, tt.username, tt.password)

		if tt.ok != ok {
			t.Fatalf("[%d] expected: %v but got: %v (username=%s,password=%s)", i, tt.ok, ok, tt.username, tt.password)
		}

		if !ok {
			continue
		}

		if !reflect.DeepEqual(tt.user, u) {
			t.Fatalf("[%d] expected user:\n%#+v\nbut got:\n%#+v", i, tt.user, u)
		}
	}
}

func TestAllowUsers(t *testing.T) {
	users := []*testUser{
		{username: "kataras", password: "kataras_pass", email: "kataras2006@hotmail.com"},
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
		u, ok := allow(nil, tt.username, tt.password)

		if tt.ok != ok {
			t.Fatalf("[%d] expected: %v but got: %v (username=%s,password=%s)", i, tt.ok, ok, tt.username, tt.password)
		}

		if !ok {
			continue
		}

		if !reflect.DeepEqual(tt.user, u) {
			t.Fatalf("[%d] expected user:\n%#+v\nbut got:\n%#+v", i, tt.user, u)
		}
	}
}

// A user list through the User interface list type itself.
func TestAllowUsersInterfaceSlice(t *testing.T) {
	users := []User{
		&testUser{username: "kataras", password: "kataras_pass"},
		&testUser{username: "makis", password: "makis_pass"},
	}

	allow := AllowUsers(users)

	u, ok := allow(nil, "makis", "makis_pass")
	if !ok {
		t.Fatal("expected makis to be allowed")
	}
	if u.GetUsername() != "makis" {
		t.Fatalf("expected the matching list element but got %#+v", u)
	}
	if _, ok = allow(nil, "kataras", "makis_pass"); ok {
		t.Fatal("expected a wrong password to be rejected")
	}
}

func TestAllowUsersCredentialsOption(t *testing.T) {
	type member struct {
		Email string
		Hash  string
	}

	members := []member{
		{Email: "kataras@example.com", Hash: mustGeneratePassword(t, "kataras_pass")},
	}

	allow := AllowUsers(members, Credentials(func(m member) (string, string) {
		return m.Email, m.Hash
	}), BCRYPT)

	m, ok := allow(nil, "kataras@example.com", "kataras_pass")
	if !ok {
		t.Fatal("expected the member to be allowed")
	}
	if m != members[0] {
		t.Fatalf("expected the matching member but got %#+v", m)
	}
	if _, ok = allow(nil, "kataras@example.com", "wrong"); ok {
		t.Fatal("expected a wrong password to be rejected")
	}
}

// AllowUsersMap builds a *context.SimpleUser per request with the identity fields filled in.
func TestAllowUsersMap(t *testing.T) {
	users := map[string]string{"kataras": mustGeneratePassword(t, "kataras_pass")}
	allow := AllowUsersMap(users, BCRYPT)

	// The map is copied on creation.
	users["late"] = "late_pass"

	u, ok := allow(nil, "kataras", "kataras_pass")
	if !ok {
		t.Fatal("expected kataras to be allowed")
	}
	if u.Username != "kataras" || u.Password != "kataras_pass" || u.ID != "kataras" || u.Authorization != authorizationType {
		t.Fatalf("unexpected user: %#+v", u)
	}
	if _, ok = allow(nil, "kataras", "wrong"); ok {
		t.Fatal("expected a wrong password to be rejected")
	}
	if _, ok = allow(nil, "late", "late_pass"); ok {
		t.Fatal("expected changes to the source map to be ignored")
	}
}

// Users given as plain structs are matched by field name or by json tag,
// including fields promoted from embedded structs.
func TestAllowUsersStructTags(t *testing.T) {
	type credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	type taggedUser struct {
		Name   string `json:"username"`
		Secret string `json:"password"`
		Email  string `json:"email"`
	}

	type embeddedUser struct {
		credentials
		Email string
	}

	type ptrEmbeddedUser struct {
		*credentials
		Role string
	}

	// allowFuncs erases the user type so the table below can hold every form.
	allowFuncs := map[string]func(username, password string) bool{
		"json tags":                      erase(AllowUsers([]taggedUser{{Name: "kataras", Secret: "kataras_pass", Email: "k@example.com"}})),
		"embedded struct":                erase(AllowUsers([]embeddedUser{{credentials{"kataras", "kataras_pass"}, "k@example.com"}})),
		"embedded pointer":               erase(AllowUsers([]ptrEmbeddedUser{{&credentials{"kataras", "kataras_pass"}, "admin"}})),
		"pointer elements":               erase(AllowUsers([]*taggedUser{{Name: "kataras", Secret: "kataras_pass"}})),
		"nil pointer element is skipped": erase(AllowUsers([]*taggedUser{nil, {Name: "kataras", Secret: "kataras_pass"}})),
		"map elements":                   erase(AllowUsers([]Map{{"username": "kataras", "password": "kataras_pass", "role": "admin"}})),
		"context.SimpleUser elements":    erase(AllowUsers([]*context.SimpleUser{{Username: "kataras", Password: "kataras_pass"}})),
	}

	for name, allow := range allowFuncs {
		t.Run(name, func(t *testing.T) {
			if !allow("kataras", "kataras_pass") {
				t.Fatal("expected the user to be allowed")
			}
			if allow("kataras", "wrong") {
				t.Fatal("expected a wrong password to be rejected")
			}
			if allow("unknown", "kataras_pass") {
				t.Fatal("expected an unknown username to be rejected")
			}
		})
	}

	t.Run("map elements keep their fields", func(t *testing.T) {
		allow := AllowUsers([]Map{{"username": "kataras", "password": "kataras_pass", "role": "admin"}})
		u, ok := allow(nil, "kataras", "kataras_pass")
		if !ok || u["role"] != "admin" {
			t.Fatalf("expected the map user to be allowed, got %#+v (ok=%v)", u, ok)
		}
	})

	t.Run("unusable elements are skipped", func(t *testing.T) {
		allow := AllowUsers([]int{1, 2})
		if _, ok := allow(nil, "1", "2"); ok {
			t.Fatal("expected no user to match")
		}
	})
}

func erase[U any](allow AuthFunc[U]) func(username, password string) bool {
	return func(username, password string) bool {
		_, ok := allow(nil, username, password)
		return ok
	}
}

// Test YAML user loading with bcrypt-encrypted passwords.
func TestAllowUsersFile(t *testing.T) {
	var tests = []struct {
		username      string
		password      string // hashed, auto-filled later on.
		inputPassword string
		ok            bool
		user          Map
	}{
		{
			username:      "kataras",
			inputPassword: "kataras_pass",
			ok:            true,
			user:          Map{"age": 27, "role": "admin"}, // username and password are auto-filled in our tests below.
		},
		{
			username:      "makis",
			inputPassword: "makis_password",
			ok:            true,
			user:          Map{},
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
	var usersToWrite []Map
	for _, tt := range tests {
		if tt.ok {
			// store the hashed password.
			tt.password = mustGeneratePassword(t, tt.inputPassword)

			// store and write the username and hashed password.
			tt.user["username"] = tt.username
			tt.user["password"] = tt.password

			usersToWrite = append(usersToWrite, tt.user)
		}
	}

	fileContents, err := yaml.Marshal(usersToWrite)
	if err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(t.TempDir(), "users.yml")
	if err = os.WriteFile(filename, fileContents, 0o600); err != nil {
		t.Fatal(err)
	}

	// Build the authentication func.
	allow := AllowUsersFile[Map](filename, BCRYPT)
	for i, tt := range tests {
		u, ok := allow(nil, tt.username, tt.inputPassword)

		if tt.ok != ok {
			t.Fatalf("[%d] expected: %v but got: %v (username=%s,password=%s,user=%#+v)", i, tt.ok, ok, tt.username, tt.inputPassword, u)
		}

		if !ok {
			continue
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

	// The same file decoded into a typed user.
	type member struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Age      int    `yaml:"age"`
		Role     string `yaml:"role"`
	}

	typed := AllowUsersFile[member](filename, BCRYPT)
	m, ok := typed(nil, "kataras", "kataras_pass")
	if !ok {
		t.Fatal("expected kataras to be allowed through the typed loader")
	}
	if m.Age != 27 || m.Role != "admin" {
		t.Fatalf("unexpected typed user: %#+v", m)
	}
}

func TestAllowUsersFileJSON(t *testing.T) {
	writeTemp := func(t *testing.T, name, contents string) string {
		t.Helper()
		filename := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(filename, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
		return filename
	}

	t.Run("username password map as Map", func(t *testing.T) {
		name := writeTemp(t, "users.json", `{"kataras": "kataras_pass", "makis": "makis_pass"}`)
		allow := AllowUsersFile[Map](name)

		u, ok := allow(nil, "makis", "makis_pass")
		if !ok {
			t.Fatal("expected makis to be allowed")
		}
		if u["username"] != "makis" || u["password"] != "makis_pass" {
			t.Fatalf("unexpected map user: %#+v", u)
		}
		if _, ok := allow(nil, "makis", "kataras_pass"); ok {
			t.Fatal("expected a wrong password to be rejected")
		}
	})

	t.Run("username password map as SimpleUser", func(t *testing.T) {
		name := writeTemp(t, "users.json", `{"kataras": "kataras_pass"}`)
		allow := AllowUsersFile[*context.SimpleUser](name)

		u, ok := allow(nil, "kataras", "kataras_pass")
		if !ok {
			t.Fatal("expected kataras to be allowed")
		}
		if u.Username != "kataras" || u.Password != "kataras_pass" || u.ID != "kataras" {
			t.Fatalf("unexpected user: %#+v", u)
		}
	})

	t.Run("username password map as custom type panics", func(t *testing.T) {
		type member struct{ Username, Password string }
		name := writeTemp(t, "users.json", `{"kataras": "kataras_pass"}`)
		defer func() {
			if recover() == nil {
				t.Fatal("expected a panic: the short form cannot fill a custom type")
			}
		}()
		AllowUsersFile[member](name)
	})

	t.Run("user list", func(t *testing.T) {
		name := writeTemp(t, "users.json", `[{"username": "kataras", "password": "kataras_pass", "role": "admin"}]`)
		allow := AllowUsersFile[Map](name)

		u, ok := allow(nil, "kataras", "kataras_pass")
		if !ok {
			t.Fatal("expected kataras to be allowed")
		}
		if u["role"] != "admin" {
			t.Fatalf("expected role admin but got %v", u["role"])
		}
	})

	t.Run("typed user list", func(t *testing.T) {
		type member struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		name := writeTemp(t, "users.json", `[{"username": "kataras", "password": "kataras_pass", "role": "admin"}]`)
		allow := AllowUsersFile[member](name)

		u, ok := allow(nil, "kataras", "kataras_pass")
		if !ok {
			t.Fatal("expected kataras to be allowed")
		}
		if u.Role != "admin" {
			t.Fatalf("expected role admin but got %q", u.Role)
		}
	})

	t.Run("unsupported extension", func(t *testing.T) {
		name := writeTemp(t, "users.toml", `kataras = "kataras_pass"`)
		defer func() {
			if recover() == nil {
				t.Fatal("expected a panic for an unsupported file extension")
			}
		}()
		AllowUsersFile[Map](name)
	})

	t.Run("missing file", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected a panic for a missing file")
			}
		}()
		AllowUsersFile[Map](filepath.Join(t.TempDir(), "missing.yml"))
	})

	t.Run("malformed file", func(t *testing.T) {
		name := writeTemp(t, "users.json", `"just a string"`)
		defer func() {
			if recover() == nil {
				t.Fatal("expected a panic for a malformed file")
			}
		}()
		AllowUsersFile[Map](name)
	})
}

// TestDefaultComparePassword locks the behavior of the default (non-BCRYPT)
// password comparator, which uses a constant-time comparison. Correctness must
// be identical to a plain equality check, including for length-mismatched input
// (crypto/subtle reports inequality for differing lengths).
func TestDefaultComparePassword(t *testing.T) {
	compare := toUserAuthOptions[Map](nil).ComparePassword

	var tests = []struct {
		stored string
		input  string
		ok     bool
	}{
		{"kataras_pass", "kataras_pass", true},
		{"kataras_pass", "Kataras_pass", false},  // case-sensitive
		{"kataras_pass", "kataras_pas", false},   // shorter input
		{"kataras_pass", "kataras_passs", false}, // longer input
		{"", "", true},
		{"kataras_pass", "", false},
	}

	for i, tt := range tests {
		if got := compare(tt.stored, tt.input); got != tt.ok {
			t.Fatalf("[%d] ComparePassword(%q, %q) = %v, want %v", i, tt.stored, tt.input, got, tt.ok)
		}
	}
}

// The BCRYPT option must compare against a real bcrypt hash on an unknown
// username too, otherwise the missing user is answered in microseconds.
func TestBCRYPTDecoyIsABcryptHash(t *testing.T) {
	options := toUserAuthOptions([]UserAuthOption[Map]{BCRYPT[Map]})
	if !strings.HasPrefix(options.decoyPassword, "$2") {
		t.Fatalf("expected a bcrypt hash as the decoy but got %q", options.decoyPassword)
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
