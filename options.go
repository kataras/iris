package iris

import (
	"os"
	"strings"

	"github.com/kataras/iris/v14/context"

	"gopkg.in/yaml.v3"
)

// Options is the configuration for the Iris Application.
// Includes server & handler timeout, router, request and response options too.
// All fields are optional.
//
// Usage:
//
//	app := iris.New(iris.Options{...})
type Options struct {
	// Name is the application name and the log prefix for
	// that Application instance's Logger.
	Name string `ini:"name" json:"name" yaml:"Name" toml:"Name"`

	// LogLevel determines the level of logging used by the application to output messages.
	// The logger is used mostly during the build state, but may also output debug error messages at runtime,
	// such as when malformed data structures are sent to the client (e.g., Context.JSON/JSONP/XML...).
	// Default: "info".
	// Possible values: "disable", "fatal", "error", "warn", "info", "debug".
	LogLevel string `ini:"log_level" json:"log_level" yaml:"LogLevel" toml:"LogLevel"`

	// Host specifies the trusted domain on which the server should run.
	// It is read back from Options.Host and, per request, from Context.Host;
	// the router uses it for subdomain matching.
	// Example: "iris-go.com".
	Host string `ini:"host" json:"host" yaml:"Host" toml:"Host"`

	// DisableFatalRun keeps Run and Listen from exiting the process when the
	// server fails to start or stops with an error: the failure is logged at
	// error level and returned to the caller instead. Set it in a process that
	// runs more than one server, or in a test that starts a real listener.
	// Default: false, a failed Run logs at fatal level and exits with status 1.
	DisableFatalRun bool `ini:"disable_fatal_run" json:"disable_fatal_run" yaml:"DisableFatalRun" toml:"DisableFatalRun"`

	// Router configures the router for all routes
	Router RouterOptions `ini:"router" json:"router,omitempty" yaml:"Router" toml:"Router"`

	// Context configures request and response handling for all routes, overwrites the defaults.
	//
	// A child Router (Application.Router, Subdomain) copies these when it is
	// created, so a later Application.EditOptions reaches the root Router only:
	// configure the context options before registering any child Router.
	Context context.ContextOptions `ini:"context" json:"context,omitempty" yaml:"Context" toml:"Context"`
}

// SetHost sets the host for the Iris Application,
// it also sets the host for the router.
func (opts *Options) SetHost(host string) {
	if host == "" {
		return
	}

	opts.Host = host
	opts.Router.Host = host
}

var _ Configurator = Options{}

type (
	// Configurator is just an interface which contains a method that accepts the framework instance.
	// It's being used to configure the framework before the server starts.
	Configurator interface {
		Configure(*Application)
	}

	// ConfiguratorFunc is just a type of function which accepts the framework instance.
	// It implements the Configurator interface.
	ConfiguratorFunc func(*Application)
)

// Configure implements the Configurator interface.
func (c ConfiguratorFunc) Configure(app *Application) {
	c(app)
}

// Configure sets the configuration for the Iris Application.
// Unset (zero) fields are filled from the defaults: a partial Options
// literal configures only what it sets, it does not zero the rest.
func (opts Options) Configure(app *Application) { // Called on app.EditOptions too.
	opts.applyDefaults()
	opts.SetHost(opts.Host) // set the host to the router too, must be set in order for safe subdomain handling. See the Application.Run:if rootHost := ... and OnListenerCreated statements too.
	app.options = &opts

	// The root Router only: a child Router copied these when it was created.
	opts.Context.CopyTo(app.RouterBuilder.ContextOptions())
}

// applyDefaults fills every unset (zero) field of opts from DefaultOptions.
// Non-nil empty slices and maps are explicit "none" values and are kept.
func (opts *Options) applyDefaults() {
	if opts.LogLevel == "" {
		opts.LogLevel = DefaultOptions.LogLevel
	}

	opts.Context.FillZero(DefaultOptions.Context)
}

// Bind reads the configuration from the "SERVER_CONFIG" environment variable in form of YAML or
// from a file and sets it to the Options structure.
func (opts *Options) Bind(filename string) error {
	// Try from env first.
	if v := os.Getenv(strings.ToUpper("SERVER_CONFIG")); v != "" {
		return readYAML([]byte(v), opts)
	}

	// Then try from file.
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return readYAML(data, opts)
}

// DefaultOptions returns the default configuration for an Iris Application.
var DefaultOptions = Options{
	LogLevel: "info",
	Router:   RouterOptions{},
	Context:  context.DefaultContextOptions(),
}

// Defaults returns a ConfiguratorFunc which sets the default options for the Iris Application.
// The edits are optional and are applied to the default options.
//
// Usage:
//
//	app := iris.New(iris.Defaults(func(opts *iris.Options) {
//		opts.LogLevel = "debug"
//		opts.Context.RemoteAddrHeaders = []string{"X-Forwarded-For"}
//	}))
func Defaults(edits ...func(*Options)) ConfiguratorFunc {
	return func(app *Application) {
		opts := newDefaultOptions()
		for _, fn := range edits {
			if fn != nil {
				fn(opts)
			}
		}
		opts.Configure(app)
	}
}

func newDefaultOptions() *Options {
	defaultOptionsClone := DefaultOptions
	return &defaultOptionsClone
}

// YAML reads the configuration from a YAML file.
// The file should contain the configuration for the Options structure.
// The file should be in the following format:
//
//	Name: "My Application"
//	LogLevel: "info"
//	Host: "iris-go.com"
//	[...]
//
// The YAML keys are the field names as they are declared in the Options structure
// (they are matched exactly, they are not case-insensitive).
//
// Returns an instance of the Options structure which completes the Configurator interface.
func YAML(optionsFilename string) Options {
	// Read the YAML file.
	data, err := os.ReadFile(optionsFilename)
	if err != nil {
		panic(err)
	}
	// Create an instance of the options structure.
	options := newDefaultOptions()

	if err = readYAML(data, options); err != nil {
		panic(err)
	}

	return *options
}

// readYAML decodes a YAML document into dest, which must be a pointer. It
// takes an "any" and not an *Options because LoadConfiguration decodes a
// service's whole configuration through the same decoder, so the framework has
// one answer to "how is YAML read" rather than two.
func readYAML(data []byte, dest any) error {
	// Unmarshal the YAML data into the structure.
	return yaml.Unmarshal(data, dest)
}
