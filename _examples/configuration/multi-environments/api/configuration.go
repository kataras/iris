package api

import (
	"os"

	"github.com/kataras/iris/v12"
	"gopkg.in/yaml.v3"
)

type Configuration struct {
	Host              string `yaml:"Host"`
	Port              int    `yaml:"Port"`
	EnableCompression bool   `yaml:"EnableCompression"`
	AllowOrigin       string `yaml:"AllowOrigin"`
	// Iris specific configuration.
	Iris iris.Configuration `yaml:"Iris"`
}

// BindFile binds the yaml file's contents to this Configuration.
func (c *Configuration) BindFile(filename string) error {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(contents, c)
}
