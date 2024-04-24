//go:build go1.18
// +build go1.18

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/kataras/jwt"
	"gopkg.in/yaml.v3"
)

const (
	// The JWT Key ID for access tokens.
	KIDAccess = "IRIS_AUTH_ACCESS"
	// The JWT Key ID for refresh tokens.
	KIDRefresh = "IRIS_AUTH_REFRESH"
)

type (
	// Configuration holds the necessary information for Iris Auth & Single-Sign-On feature.
	//
	// See the `New` package-level function.
	Configuration struct {
		// The authorization header keys that server should read the access token from.
		//
		// Defaults to:
		// - Authorization
		// - X-Authorization
		Headers []string `json:"headers" yaml:"Headers" toml:"Headers" ini:"Headers"`
		// Cookie optional configuration.
		// A Cookie.Name holds the access token value fully encrypted.
		Cookie CookieConfiguration `json:"cookie" yaml:"Cookie" toml:"Cookie" ini:"cookie"`
		// Keys MUST define the jwt keys configuration for access,
		// and optionally, for refresh tokens signing and verification.
		Keys jwt.KeysConfiguration `json:"keys" yaml:"Keys" toml:"Keys" ini:"keys"`
	}

	// CookieConfiguration holds the necessary information for cookie client storage.
	CookieConfiguration struct {
		// Name defines the cookie's name.
		Name string `json:"cookie" yaml:"Name" toml:"Name" ini:"name"`
		// Secure if true then "; Secure" is appended to the Set-Cookie header.
		// By setting the secure to true, the web browser will prevent the
		// transmission of a cookie over an unencrypted channel.
		//
		// Defaults to false but it's true when the request is under iris.Context.IsSSL().
		Secure bool `json:"secure" yaml:"Secure" toml:"Secure" ini:"secure"`
		// Hash is optional, it is used to authenticate cookie value using HMAC.
		// It is recommended to use a key with 32 or 64 bytes.
		Hash string `json:"hash" yaml:"Hash" toml:"Hash" ini:"hash"`
		// Block is optional, used to encrypt cookie value.
		// The key length must correspond to the block size
		// of the encryption algorithm. For AES, used by default, valid lengths are
		// 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
		Block string `json:"block" yaml:"Block" toml:"Block" ini:"block"`
	}
)

func (c *Configuration) validate() (jwt.Keys, error) {
	if len(c.Headers) == 0 {
		return nil, fmt.Errorf("auth: configuration: headers slice is empty")
	}

	if c.Cookie.Name != "" {
		if c.Cookie.Hash == "" || c.Cookie.Block == "" {
			return nil, fmt.Errorf("auth: configuration: cookie block and cookie hash are required for security reasons when cookie is used")
		}
	}

	keys, err := c.Keys.Load()
	if err != nil {
		return nil, fmt.Errorf("auth: configuration: %w", err)
	}

	if _, ok := keys[KIDAccess]; !ok {
		return nil, fmt.Errorf("auth: configuration: %s access token is missing from the configuration", KIDAccess)
	}

	// Let's keep refresh optional.
	// if _, ok := keys[KIDRefresh]; !ok {
	// 	return nil, fmt.Errorf("auth: configuration: %s refresh token is missing from the configuration", KIDRefresh)
	// }
	return keys, nil
}

// BindRandom binds the "c" configuration to random values for keys and cookie security.
// Keys will not be persisted between restarts,
// a more persistent storage should be considered for production applications,
// see BindFile method and LoadConfiguration/MustLoadConfiguration package-level functions.
func (c *Configuration) BindRandom() error {
	accessPublic, accessPrivate, err := jwt.GenerateEdDSA()
	if err != nil {
		return err
	}

	refreshPublic, refreshPrivate, err := jwt.GenerateEdDSA()
	if err != nil {
		return err
	}

	*c = Configuration{
		Headers: []string{
			"Authorization",
			"X-Authorization",
		},
		Cookie: CookieConfiguration{
			Name:   "iris_auth_cookie",
			Secure: false,
			Hash:   string(securecookie.GenerateRandomKey(64)),
			Block:  string(securecookie.GenerateRandomKey(32)),
		},
		Keys: jwt.KeysConfiguration{
			{
				ID:      KIDAccess,
				Alg:     jwt.EdDSA.Name(),
				MaxAge:  2 * time.Hour,
				Public:  string(accessPublic),
				Private: string(accessPrivate),
			},
			{
				ID:            KIDRefresh,
				Alg:           jwt.EdDSA.Name(),
				MaxAge:        720 * time.Hour,
				Public:        string(refreshPublic),
				Private:       string(refreshPrivate),
				EncryptionKey: string(jwt.MustGenerateRandom(32)),
			},
		},
	}

	return nil
}

// BindFile binds a filename (fullpath) to "c" Configuration.
// The file format is either JSON or YAML and it should be suffixed
// with .json or .yml/.yaml.
func (c *Configuration) BindFile(filename string) error {
	switch filepath.Ext(filename) {
	case ".json":
		contents, err := os.ReadFile(filename)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				generatedConfig := MustGenerateConfiguration()
				if generatedYAML, gErr := generatedConfig.ToJSON(); gErr == nil {
					err = fmt.Errorf("%w: example:\n\n%s", err, generatedYAML)
				}
			}
			return err
		}

		return json.Unmarshal(contents, c)
	default:
		contents, err := os.ReadFile(filename)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				generatedConfig := MustGenerateConfiguration()
				if generatedYAML, gErr := generatedConfig.ToYAML(); gErr == nil {
					err = fmt.Errorf("%w: example:\n\n%s", err, generatedYAML)
				}
			}
			return err
		}

		return yaml.Unmarshal(contents, c)
	}
}

// ToYAML returns the "c" Configuration's contents as raw yaml byte slice.
func (c *Configuration) ToYAML() ([]byte, error) {
	return yaml.Marshal(c)
}

// ToJSON returns the "c" Configuration's contents as raw json byte slice.
func (c *Configuration) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// MustGenerateConfiguration calls the Configuration's BindRandom
// method and returns the result. It panics on errors.
func MustGenerateConfiguration() (c Configuration) {
	if err := c.BindRandom(); err != nil {
		panic(err)
	}

	return
}

// MustLoadConfiguration same as LoadConfiguration package-level function
// but it panics on error.
func MustLoadConfiguration(filename string) Configuration {
	c, err := LoadConfiguration(filename)
	if err != nil {
		panic(err)
	}

	return c
}

// LoadConfiguration reads a filename (fullpath)
// and returns a Configuration binded to the contents of the given filename.
// See Configuration.BindFile method too.
func LoadConfiguration(filename string) (c Configuration, err error) {
	err = c.BindFile(filename)
	return
}
