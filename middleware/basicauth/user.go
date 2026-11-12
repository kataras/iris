package basicauth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/kataras/iris/v14/context"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// ReadFile can be used to customize the way the
// AllowUsersFile function is loading the filename from.
// Example of usage: embedded users.yml file.
// Defaults to the `os.ReadFile` which reads the file from the physical disk.
var ReadFile = os.ReadFile

// User can be implemented by custom struct values
// to provide the username and the password as
// basic authentication credentials for a user list.
//
// Look AllowUsers package-level function and the Options.Allow field.
type User interface {
	context.UserGetUsername
	context.UserGetPassword
}

// UserAuthOptions holds optional user authentication options
// that can be given to the builtin Default and Load (and AllowUsers, AllowUsersMap, AllowUsersFile) functions.
type UserAuthOptions[U any] struct {
	// ComparePassword compares the stored password of a user with the user input.
	// Defaults to a constant-time equality check, can be modified for encrypted passwords,
	// see the BCRYPT optional function.
	ComparePassword func(stored, userPassword string) bool
	// Credentials returns the username and the stored password of a user list element.
	// When nil, AllowUsers reads them from the User interface methods
	// or from the Username and Password struct fields (or their json tags).
	// See the Credentials optional function.
	Credentials func(user U) (username, password string)

	// decoyPassword is what an unknown username is compared against, so the
	// request costs the same whether or not the user exists and the endpoint
	// does not enumerate valid usernames by response time. Nobody can
	// authenticate with it, the username lookup has already failed.
	decoyPassword string
}

// UserAuthOption is the option function type
// for the Default and Load (and AllowUsers, AllowUsersMap, AllowUsersFile) functions.
//
// See BCRYPT and Credentials for implementations.
type UserAuthOption[U any] func(*UserAuthOptions[U])

// BCRYPT it is a UserAuthOption, it compares a bcrypt hashed password with its user input.
// Reports true on success and false on failure.
//
// Useful when the users passwords are encrypted
// using the Provos and Mazières's bcrypt adaptive hashing algorithm.
// See https://www.usenix.org/legacy/event/usenix99/provos/provos.pdf.
//
// Usage:
//
//	Default(..., BCRYPT) OR
//	Load(..., BCRYPT) OR
//	Options.Allow = AllowUsers(..., BCRYPT) OR
//	Options.Allow = AllowUsersFile[Map](..., BCRYPT)
func BCRYPT[U any](opts *UserAuthOptions[U]) {
	opts.ComparePassword = func(stored, userPassword string) bool {
		err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(userPassword))
		return err == nil
	}
	// A real bcrypt hash, so an absent username costs a full bcrypt comparison
	// just like a present one. Generated once at configuration time from a value
	// nobody can authenticate with.
	opts.decoyPassword = bcryptDecoy()
}

// bcryptDecoy returns a bcrypt hash of an unguessable value, computed once.
var bcryptDecoy = sync.OnceValue(func() string {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		// crypto/rand does not fail in practice; a fixed value still costs a full
		// bcrypt comparison, which is all this is for.
		secret = []byte(plainDecoyPassword)
	}

	hashed, err := bcrypt.GenerateFromPassword(secret, bcrypt.DefaultCost)
	if err != nil {
		return "$2a$10$" + strings.Repeat("x", 53)
	}

	return string(hashed)
})

// plainDecoyPassword is the default decoy, see UserAuthOptions.decoyPassword.
const plainDecoyPassword = "iris-basicauth-decoy-password-value"

// Credentials is a UserAuthOption which tells AllowUsers how to read
// the username and the stored password out of a user list element,
// instead of relying on the User interface or on the struct field names.
//
// Usage:
//
//	AllowUsers(members, Credentials(func(m Member) (string, string) {
//		return m.Email, m.Hash
//	}), BCRYPT)
func Credentials[U any](fn func(user U) (username, password string)) UserAuthOption[U] {
	return func(opts *UserAuthOptions[U]) {
		opts.Credentials = fn
	}
}

// constantTimeComparePassword is the default ComparePassword:
// a plain equality check that takes the same time whether or not the values match.
func constantTimeComparePassword(stored, userPassword string) bool {
	return subtle.ConstantTimeCompare([]byte(stored), []byte(userPassword)) == 1
}

func toUserAuthOptions[U any](opts []UserAuthOption[U]) (options UserAuthOptions[U]) {
	for _, opt := range opts {
		opt(&options)
	}

	if options.ComparePassword == nil {
		options.ComparePassword = constantTimeComparePassword
	}

	if options.decoyPassword == "" {
		// ConstantTimeCompare is already constant-time for equal lengths, but the
		// decoy keeps the code path identical for a present and an absent user.
		options.decoyPassword = plainDecoyPassword
	}

	return options
}

// credentials returns the username and stored password of a user list element,
// using the Credentials option when set and the builtin extraction otherwise.
func (options UserAuthOptions[U]) credentials(u U) (username, password string, ok bool) {
	if options.Credentials != nil {
		username, password = options.Credentials(u)
		return username, password, username != "" && password != ""
	}

	return extractUsernameAndPassword(u)
}

// AllowUsers is an AuthFunc which authenticates user input based on a (static) user list.
// The credentials of each element are resolved once, when AllowUsers is called,
// so serving a request costs a map lookup and a password comparison.
// The element type U can be:
//
//	a type which completes the User interface,
//	a struct (or pointer to struct) with Username and Password string fields,
//	or with fields tagged `json:"username"` and `json:"password"`, embedded structs included,
//	a Map (map[string]any) with "username" and "password" keys,
//	any other type, when the Credentials option is given.
//
// Elements without both a username and a password are skipped.
// The authenticated user returned to the handlers is the matching list element.
//
// Usage:
//
//	New(Options[MyUser]{Allow: AllowUsers(users, [BCRYPT])})
func AllowUsers[U any](users []U, opts ...UserAuthOption[U]) AuthFunc[U] {
	options := toUserAuthOptions(opts)

	// create a local user structure to be used in the map copy,
	// takes longer to initialize but faster to serve.
	type entry struct {
		password string
		user     U
	}
	cp := make(map[string]entry, len(users))

	for _, u := range users {
		// MUST contain a username and password.
		username, password, ok := options.credentials(u)
		if !ok {
			continue
		}

		cp[username] = entry{password: password, user: u}
	}

	return func(_ *context.Context, username, password string) (U, bool) {
		var zero U

		e, ok := cp[username] // fast map access,
		if !ok {
			// Compare against a decoy so an unknown username costs the same as a
			// known one. See decoyPassword.
			options.ComparePassword(options.decoyPassword, password)
			return zero, false
		}

		if options.ComparePassword(e.password, password) {
			return e.user, true
		}

		return zero, false
	}
}

// AllowUsersMap is an AuthFunc which authenticates user input based on
// a (static) username:password map. The map is copied, later changes to it are not seen.
// The authenticated user returned to the handlers is a *context.SimpleUser
// holding the username and the password the client sent, with its ID set to the username
// and its Authorization to "Basic Authentication".
//
// Usage:
//
//	New(Options[*iris.SimpleUser]{Allow: AllowUsersMap(map[string]string{"admin": "admin"}, [BCRYPT])})
func AllowUsersMap(users map[string]string, opts ...UserAuthOption[*context.SimpleUser]) AuthFunc[*context.SimpleUser] {
	options := toUserAuthOptions(opts)
	usernamePassword := maps.Clone(users)

	return func(_ *context.Context, username, password string) (*context.SimpleUser, bool) {
		stored, ok := usernamePassword[username]
		if !ok {
			options.ComparePassword(options.decoyPassword, password)
			return nil, false
		}

		if options.ComparePassword(stored, password) {
			return newSimpleUser(username, password), true
		}

		return nil, false
	}
}

// newSimpleUser builds the user of the username:password forms.
// The AuthorizedAt field is filled by the middleware when MaxAge is set.
func newSimpleUser(username, password string) *context.SimpleUser {
	return &context.SimpleUser{
		Authorization: authorizationType,
		ID:            username,
		Username:      username,
		Password:      password,
	}
}

// AllowUsersFile is an AuthFunc which authenticates user input based on a (static) user list
// loaded from a JSON or YAML file on initialization. The file is decoded into a []U,
// so U can be any struct with the matching json/yaml tags, or Map to keep every field as is.
// The username and password are then resolved as documented on AllowUsers.
//
// Example Code:
//
//	New(Options[Map]{Allow: AllowUsersFile[Map]("users.yml", BCRYPT)})
//	New(Options[Member]{Allow: AllowUsersFile[Member]("users.yml", BCRYPT)})
//
// The users.yml file looks like the following:
//   - username: kataras
//     password: kataras_pass
//     age: 27
//     role: admin
//   - username: makis
//     password: makis_password
//     ...
//
// The short form below is supported as well, when U is Map or *context.SimpleUser:
//
//	kataras: kataras_pass
//	makis: makis_password
//
// It panics when the file cannot be read or decoded.
func AllowUsersFile[U any](jsonOrYamlFilename string, opts ...UserAuthOption[U]) AuthFunc[U] {
	users, err := loadUsersFile[U](jsonOrYamlFilename)
	if err != nil {
		panic(err)
	}

	return AllowUsers(users, opts...)
}

// loadUsersFile decodes a user list file into a []U.
// It tries the list form first and then the username: password form.
func loadUsersFile[U any](src string) ([]U, error) {
	data, err := ReadFile(src)
	if err != nil {
		return nil, err
	}

	// We use unmarshal instead of a file decoder
	// as we may need to decode the data more than once (see below).
	var unmarshal func(data []byte, v any) error

	switch ext := filepath.Ext(src); ext {
	case "", ".json":
		unmarshal = json.Unmarshal
	case ".yml", ".yaml":
		unmarshal = yaml.Unmarshal
	default:
		return nil, fmt.Errorf("unexpected file extension: %s", ext)
	}

	// JSON Form: [{"username": "$username", "password": "$pass", "other_field": ...}, {"username": ...}, ... ]
	// YAML Form:
	// - username: $username
	//   password: $password
	//   other_field: ...
	var list []U
	listErr := unmarshal(data, &list)
	if listErr == nil && len(list) > 0 {
		return list, nil
	}

	// JSON Form: { "$username":"$pass", "$username": "$pass" }
	// YAML Form: $username: $pass
	// 			  $username: $pass
	var pairs map[string]string
	if err = unmarshal(data, &pairs); err != nil || len(pairs) == 0 {
		if listErr == nil {
			listErr = err
		}
		return nil, fmt.Errorf("malformed document file: %s: %w", src, listErr)
	}

	list = make([]U, 0, len(pairs))
	for username, password := range pairs {
		u, ok := pairToUser[U](username, password)
		if !ok {
			return nil, fmt.Errorf("document file: %s: the username: password form requires a Map or *context.SimpleUser user type, got %T", src, u)
		}
		list = append(list, u)
	}

	return list, nil
}

// pairToUser builds a U out of a username: password file entry.
// Only Map and context.SimpleUser carry no other information, so only they are supported.
func pairToUser[U any](username, password string) (U, bool) {
	var u U
	switch p := any(&u).(type) {
	case *Map:
		*p = Map{"username": username, "password": password}
	case **context.SimpleUser:
		*p = newSimpleUser(username, password)
	case *context.SimpleUser:
		*p = *newSimpleUser(username, password)
	default:
		return u, false
	}

	return u, true
}

// extractUsernameAndPassword reads the credentials out of a user list element.
// It accepts, in this order: a User implementation, a map[string]any and
// a struct (or pointer to struct) with Username and Password fields.
// Struct fields are matched by name ("Username", "Password") or by their json tag
// ("username", "password"), and embedded structs are searched too.
func extractUsernameAndPassword(s any) (username, password string, ok bool) {
	switch u := s.(type) {
	case nil:
		return "", "", false
	case User:
		username, password = u.GetUsername(), u.GetPassword()
		return username, password, username != "" && password != ""
	case map[string]any:
		return mapUsernameAndPassword(u)
	default:
		return structUsernameAndPassword(reflect.ValueOf(s))
	}
}

// structUsernameAndPassword walks the exported fields of a struct value
// (dereferencing pointers and descending into embedded structs)
// and returns the string values of its username and password fields.
func structUsernameAndPassword(v reflect.Value) (username, password string, ok bool) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return "", "", false
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return "", "", false
	}

	var usernameFound, passwordFound bool
	for field, value := range v.Fields() {
		if field.Anonymous {
			// An explicitly tagged embedded field is a regular field, do not descend.
			if _, tagged := field.Tag.Lookup("json"); !tagged {
				if u, p, found := structUsernameAndPassword(value); found {
					if !usernameFound {
						username, usernameFound = u, true
					}
					if !passwordFound {
						password, passwordFound = p, true
					}
					if usernameFound && passwordFound {
						break
					}
				}
				continue
			}
		}

		if !field.IsExported() || value.Kind() != reflect.String {
			continue
		}

		switch fieldKey(field) {
		case "username":
			if !usernameFound {
				username, usernameFound = value.String(), true
			}
		case "password":
			if !passwordFound {
				password, passwordFound = value.String(), true
			}
		}

		if usernameFound && passwordFound {
			break
		}
	}

	ok = username != "" && password != ""
	return username, password, ok
}

// fieldKey returns the lower-cased name a struct field is matched by:
// the json tag name when present, the field name otherwise.
func fieldKey(field reflect.StructField) string {
	if tag, found := field.Tag.Lookup("json"); found {
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			return ""
		}
		if name != "" {
			return strings.ToLower(name)
		}
	}

	return strings.ToLower(field.Name)
}

func mapUsernameAndPassword(m map[string]any) (username, password string, ok bool) {
	// Single entry of the form username: password.
	if len(m) == 1 {
		for k, v := range m {
			if pass, isString := v.(string); isString {
				return k, pass, k != "" && pass != ""
			}
		}
	}

	var usernameFound, passwordFound bool
	for k, v := range m {
		switch k {
		case "username", "Username":
			username, usernameFound = v.(string)
		case "password", "Password":
			password, passwordFound = v.(string)
		}

		if usernameFound && passwordFound {
			return username, password, true
		}
	}

	return "", "", false
}
