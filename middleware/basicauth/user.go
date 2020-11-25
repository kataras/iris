package basicauth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/kataras/iris/v12/context"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// ReadFile can be used to customize the way the
// AllowUsersFile function is loading the filename from.
// Example of usage: embedded users.yml file.
// Defaults to the `ioutil.ReadFile` which reads the file from the physical disk.
var ReadFile = ioutil.ReadFile

// User is a partial part of the iris.User interface.
// It's used to declare a static slice of registered User for authentication.
type User interface {
	context.UserGetUsername
	context.UserGetPassword
}

// UserAuthOptions holds optional user authentication options
// that can be given to the builtin Default and Load (and AllowUsers, AllowUsersFile) functions.
type UserAuthOptions struct {
	// Defaults to plain check, can be modified for encrypted passwords,
	// see the BCRYPT optional function.
	ComparePassword func(stored, userPassword string) bool
}

// UserAuthOption is the option function type
// for the Default and Load (and AllowUsers, AllowUsersFile) functions.
//
// See BCRYPT for an implementation.
type UserAuthOption func(*UserAuthOptions)

// BCRYPT it is a UserAuthOption, it compares a bcrypt hashed password with its user input.
// Reports true on success and false on failure.
//
// Useful when the users passwords are encrypted
// using the Provos and Mazières's bcrypt adaptive hashing algorithm.
// See https://www.usenix.org/legacy/event/usenix99/provos/provos.pdf.
//
// Usage:
//  Default(..., BCRYPT) OR
//  Load(..., BCRYPT) OR
//  Options.Allow = AllowUsers(..., BCRYPT) OR
//  OPtions.Allow = AllowUsersFile(..., BCRYPT)
func BCRYPT(opts *UserAuthOptions) {
	opts.ComparePassword = func(stored, userPassword string) bool {
		err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(userPassword))
		return err == nil
	}
}

func toUserAuthOptions(opts []UserAuthOption) (options UserAuthOptions) {
	for _, opt := range opts {
		opt(&options)
	}

	if options.ComparePassword == nil {
		options.ComparePassword = func(stored, userPassword string) bool {
			return stored == userPassword
		}
	}

	return options
}

// AllowUsers is an AuthFunc which authenticates user input based on a (static) user list.
// The "users" input parameter can be one of the following forms:
//  map[string]string e.g. {username: password, username: password...}.
//  []map[string]interface{} e.g. []{"username": "...", "password": "...", "other_field": ...}, ...}.
//  []T which T completes the User interface.
//  []T which T contains at least Username and Password fields.
//
// Usage:
// New(Options{Allow: AllowUsers(..., [BCRYPT])})
func AllowUsers(users interface{}, opts ...UserAuthOption) AuthFunc {
	// create a local user structure to be used in the map copy,
	// takes longer to initialize but faster to serve.
	type user struct {
		password string
		ref      interface{}
	}
	cp := make(map[string]*user)

	v := reflect.Indirect(reflect.ValueOf(users))
	switch v.Kind() {
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i).Interface()
			// MUST contain a username and password.
			username, password, ok := extractUsernameAndPassword(elem)
			if !ok {
				continue
			}

			cp[username] = &user{
				password: password,
				ref:      elem,
			}
		}
	case reflect.Map:
		elem := v.Interface()
		switch m := elem.(type) {
		case map[string]string:
			return userMap(m, opts...)
		case map[string]interface{}:
			username, password, ok := mapUsernameAndPassword(m)
			if !ok {
				break
			}

			cp[username] = &user{
				password: password,
				ref:      m,
			}
		default:
			panic(fmt.Sprintf("unsupported type of map: %T", users))
		}
	default:
		panic(fmt.Sprintf("unsupported type: %T", users))
	}

	options := toUserAuthOptions(opts)

	return func(_ *context.Context, username, password string) (interface{}, bool) {
		if u, ok := cp[username]; ok { // fast map access,
			if options.ComparePassword(u.password, password) {
				return u.ref, true
			}
		}

		return nil, false
	}
}

func userMap(usernamePassword map[string]string, opts ...UserAuthOption) AuthFunc {
	options := toUserAuthOptions(opts)

	return func(_ *context.Context, username, password string) (interface{}, bool) {
		pass, ok := usernamePassword[username]
		return nil, ok && options.ComparePassword(pass, password)
	}
}

// AllowUsersFile is an AuthFunc which authenticates user input based on a (static) user list
// loaded from a file on initialization.
//
// Example Code:
//  New(Options{Allow: AllowUsersFile("users.yml", BCRYPT)})
// The users.yml file looks like the following:
//  - username: kataras
//    password: kataras_pass
//    age: 27
//    role: admin
//  - username: makis
//    password: makis_password
//    ...
func AllowUsersFile(jsonOrYamlFilename string, opts ...UserAuthOption) AuthFunc {
	var (
		usernamePassword map[string]string
		// no need to support too much forms, this would be for:
		// "$username": { "password": "$pass", "other_field": ...}
		userList []map[string]interface{}
	)

	if err := decodeFile(jsonOrYamlFilename, &usernamePassword, &userList); err != nil {
		panic(err)
	}

	if len(usernamePassword) > 0 {
		// JSON Form: { "$username":"$pass", "$username": "$pass" }
		// YAML Form: $username: $pass
		// 			  $username: $pass
		return userMap(usernamePassword, opts...)
	}

	if len(userList) > 0 {
		// JSON Form: [{"username": "$username", "password": "$pass", "other_field": ...}, {"username": ...}, ... ]
		// YAML Form:
		// - username: $username
		//   password: $password
		//   other_field: ...
		return AllowUsers(userList, opts...)
	}

	panic("malformed document file: " + jsonOrYamlFilename)
}

func decodeFile(src string, dest ...interface{}) error {
	data, err := ReadFile(src)
	if err != nil {
		return err
	}

	// We use unmarshal instead of file decoder
	// as we may need to read it more than once (dests, see below).
	var (
		unmarshal func(data []byte, v interface{}) error
		ext       string
	)

	if idx := strings.LastIndexByte(src, '.'); idx > 0 {
		ext = src[idx:]
	}

	switch ext {
	case "", ".json":
		unmarshal = json.Unmarshal
	case ".yml", ".yaml":
		unmarshal = yaml.Unmarshal
	default:
		return fmt.Errorf("unexpected file extension: %s", ext)
	}

	var (
		ok      bool
		lastErr error
	)

	for _, d := range dest {
		if err = unmarshal(data, d); err == nil {
			ok = true
		} else {
			lastErr = err
		}
	}

	if !ok {
		return lastErr
	}

	return nil // if at least one is succeed we are ok.
}

func extractUsernameAndPassword(s interface{}) (username, password string, ok bool) {
	if s == nil {
		return
	}

	switch u := s.(type) {
	case User:
		username = u.GetUsername()
		password = u.GetPassword()
		ok = username != "" && password != ""
		return
	case map[string]interface{}:
		return mapUsernameAndPassword(u)
	default:
		b, err := json.Marshal(u)
		if err != nil {
			return
		}

		var m map[string]interface{}
		if err = json.Unmarshal(b, &m); err != nil {
			return
		}

		return mapUsernameAndPassword(m)
	}
}

func mapUsernameAndPassword(m map[string]interface{}) (username, password string, ok bool) {
	// type of username: password.
	if len(m) == 1 {
		for username, v := range m {
			if password, ok := v.(string); ok {
				ok := username != "" && password != ""
				return username, password, ok
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
			ok = true
			break
		}
	}

	return
}
