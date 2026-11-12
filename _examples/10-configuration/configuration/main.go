// Configuration: one YAML document for the framework and for your own
// settings, secrets from the environment only, and the merge that makes a
// partial configuration safe.
//
// Learn: merge code defaults, a YAML file and the SERVER_CONFIG variable into
// one iris.Options value, and keep secrets in the environment, never in the file.
//
// The program reads config.yml from the working directory (and still starts,
// on defaults, when the file is absent), so run it from this directory.
// ADMIN_PASSWORD is required: without it the program refuses to start.
//
// Run:
//
//	ADMIN_PASSWORD=change-me go run main.go                                   # reads config.yml
//	SERVER_CONFIG='LogLevel: debug' ADMIN_PASSWORD=change-me go run main.go   # the variable overrides the file
//
// Try:
//
//	curl -i localhost:8080/config                        # 200, the effective options and where they came from
//	curl -i -X POST localhost:8080/uploads -d 'hello'    # 200, {"bytes":5}; this group's cap is 10 MB, not the global 1 MB
//
// Next: ../configuration-worker
package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/kataras/iris/v14"
	// rest/errors re-exports the standard errors.Is/As/New, so one import
	// covers both the config checks below and the HTTP error rendering.
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	var app *iris.Application
	if cfg.Source == "defaults" {
		// Nothing external to read, which here is the development case.
		cfg.Name, cfg.LogLevel = "configuration", "debug"

		// iris.Defaults is the code-only path: the exported defaults, edited in
		// place by the functions you pass, wrapped as a Configurator.
		app = iris.New(iris.Defaults(func(opts *iris.Options) {
			opts.Name = cfg.Name
			opts.LogLevel = cfg.LogLevel
		}))
	} else {
		// A document was found. Fields it left out are filled from
		// iris.DefaultOptions inside iris.New. Two refinements: a non-nil
		// empty slice or map means "explicitly none" rather than "unset", and
		// a field whose meaningful value is the zero value cannot be expressed
		// at construction, so set those per request, as /uploads does below.
		app = iris.New(cfg.Options)
	}

	// The configuration actually in effect. Charset is never set by this
	// program or by config.yml: it arrives through the merge.
	app.Get("/config", func(ctx iris.Context) {
		effective := ctx.OptionsReadOnly() // request and response half.

		ctx.JSON(iris.Map{
			"source":              cfg.Source,
			"name":                cfg.Name,
			"log_level":           cfg.LogLevel,
			"env":                 cfg.Env,
			"admin_user":          cfg.AdminUser, // the password is never rendered.
			"charset":             effective.Charset,
			"body_max_memory":     effective.BodyMaxMemory,
			"remote_addr_headers": effective.RemoteAddrHeaders,
		})
	})

	// ctx.Options() hands a middleware a request-scoped copy of the context
	// options, so this group accepts bodies far above the global cap and the
	// change dies with the request.
	uploads := app.Router("/uploads", func(ctx iris.Context) {
		ctx.Options().BodyMaxMemory = 10 * iris.MB
		ctx.Next()
	})
	uploads.Post("/", func(ctx iris.Context) {
		body, err := ctx.Body()
		if err != nil {
			// HandleRequestError keeps the classification: a body past the
			// cap is still a 413, anything else is a 400.
			errors.HandleRequestError(ctx, err)
			return
		}

		ctx.OK(iris.Map{"bytes": len(body)}, nil)
	})

	app.Listen(cfg.Addr)
}

// Config is the whole application configuration: the framework's iris.Options
// inlined, plus this application's own keys. AdminPassword is yaml:"-" on
// purpose: secrets come from the environment, never from a committed file.
type Config struct {
	iris.Options `yaml:",inline"`

	Addr          string `yaml:"Addr"`
	Env           string `yaml:"Env"`
	AdminUser     string `yaml:"AdminUser"`
	AdminPassword string `yaml:"-"`
	Source        string `yaml:"-"` // where the values came from, reported by /config.
}

const configFile = "config.yml"

func loadConfig() (Config, error) {
	// Working defaults first. iris.DefaultOptions is the exported default
	// configuration; anything still at its zero value when the Options reach
	// iris.New is filled from it again, which is what makes partial
	// configuration safe at every layer.
	cfg := Config{
		Options:   iris.DefaultOptions,
		Addr:      ":8080",
		Env:       "development",
		AdminUser: "admin",
		Source:    "defaults",
	}

	// One document carries both halves: yaml:",inline" puts the framework keys
	// (Name, LogLevel, Host, Router, Context) at the top level of the same file
	// as this application's own keys.
	data, err := os.ReadFile(configFile)
	switch {
	case err == nil:
		if err = yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("%s: %w", configFile, err)
		}
		cfg.Source = configFile
	case !errors.Is(err, os.ErrNotExist):
		return cfg, fmt.Errorf("%s: %w", configFile, err)
	}

	// The deployment's overrides for the framework half. (*iris.Options).Bind
	// is environment-first: it reads SERVER_CONFIG as a complete YAML document
	// and falls back to the file it is given only when the variable is empty.
	// Passing no filename here means "the variable or nothing", so a container
	// can retune the framework without mounting anything.
	if err := cfg.Options.Bind(""); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return cfg, fmt.Errorf("SERVER_CONFIG: %w", err)
		}
	} else {
		cfg.Source += " + SERVER_CONFIG"
	}

	// Secrets, always and only from the environment.
	cfg.AdminPassword = os.Getenv("ADMIN_PASSWORD")
	if cfg.AdminPassword == "" {
		return cfg, fmt.Errorf("ADMIN_PASSWORD environment variable is required")
	}

	return cfg, nil
}
