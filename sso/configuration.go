//go:build go1.18

package sso

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
	KIDAccess  = "IRIS_SSO_ACCESS"
	KIDRefresh = "IRIS_SSO_REFRESH"
)

type (
	Configuration struct {
		Cookie CookieConfiguration `json:"cookie" yaml:"Cookie" toml:"Cookie" ini:"cookie"`
		// keep it to always renew the refresh token. RefreshStrategy string                `json:"refresh_strategy" yaml:"RefreshStrategy" toml:"RefreshStrategy" ini:"refresh_strategy"`
		Keys jwt.KeysConfiguration `json:"keys" yaml:"Keys" toml:"Keys" ini:"keys"`
	}

	CookieConfiguration struct {
		Name  string `json:"cookie" yaml:"Name" toml:"Name" ini:"name"`
		Hash  string `json:"hash" yaml:"Hash" toml:"Hash" ini:"hash"`
		Block string `json:"block" yaml:"Block" toml:"Block" ini:"block"`
	}
)

func (c *Configuration) validate() (jwt.Keys, error) {
	if c.Cookie.Name != "" {
		if c.Cookie.Hash == "" || c.Cookie.Block == "" {
			return nil, fmt.Errorf("cookie block and cookie hash are required for security reasons when cookie is used")
		}
	}

	keys, err := c.Keys.Load()
	if err != nil {
		return nil, fmt.Errorf("sso: %w", err)
	}

	if _, ok := keys[KIDAccess]; !ok {
		return nil, fmt.Errorf("sso: %s access token is missing from the configuration", KIDAccess)
	}

	// Let's keep refresh optional.
	// if _, ok := keys[KIDRefresh]; !ok {
	// 	return nil, fmt.Errorf("sso: %s refresh token is missing from the configuration", KIDRefresh)
	// }
	return keys, nil
}

// BindRandom binds the "c" configuration to random values for keys and cookie security.
// Keys will not be persisted between restarts,
// a more persistent storage should be considered for production applications.
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
		Cookie: CookieConfiguration{
			Name:  "iris_sso",
			Hash:  string(securecookie.GenerateRandomKey(64)),
			Block: string(securecookie.GenerateRandomKey(32)),
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

func (c *Configuration) ToYAML() ([]byte, error) {
	return yaml.Marshal(c)
}

func (c *Configuration) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

func MustGenerateConfiguration() (c Configuration) {
	if err := c.BindRandom(); err != nil {
		panic(err)
	}

	return
}

func LoadConfiguration(filename string) (c Configuration, err error) {
	err = c.BindFile(filename)
	return
}

func MustLoadConfiguration(filename string) Configuration {
	c, err := LoadConfiguration(filename)
	if err != nil {
		panic(err)
	}

	return c
}
