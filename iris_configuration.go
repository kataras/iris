package iris

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

// LoadConfiguration reads a YAML document into T from the environment variable
// named envKey when it is set, otherwise from filename. It is the whole-service
// counterpart of Options.Bind, which reads the framework options only.
//
// The environment variable may hold either the YAML document itself or a path
// to it: a value that names an existing file is read as a file, anything else
// is decoded as the document. A container can therefore pass the configuration
// inline or mount it as a file without the program knowing which.
//
// If T (or *T) implements interface{ BindEnv() } it runs after decoding, so
// environment overrides win over the document. Embed BaseConfiguration to get
// that behaviour for the fields every service shares.
//
// It reads the same YAML decoder Options.Bind and YAML use. It is not a
// generic Options.Bind: that one is hardcoded to the SERVER_CONFIG variable
// and always reads its value as a document, never as a path.
//
// Usage:
//
//	type Configuration struct {
//		iris.BaseConfiguration `yaml:",inline"`
//
//		DatabaseURL string `yaml:"DatabaseURL"`
//	}
//
//	c, err := iris.LoadConfiguration[Configuration]("MY_SERVICE_CONFIG", "configuration.yml")
func LoadConfiguration[T any](envKey, filename string) (T, error) {
	var c T

	data, err := readConfigurationData(envKey, filename)
	if err != nil {
		return c, err
	}

	// The same decoder Options.Bind and iris.YAML use, so a document which
	// loads through one loads through the other.
	if err = readYAML(data, &c); err != nil {
		return c, fmt.Errorf("iris: load configuration: %w", err)
	}

	// Run the environment binder last: the document is the baseline and the
	// environment is the deployment-time override.
	if binder, ok := any(&c).(interface{ BindEnv() }); ok {
		binder.BindEnv()
	} else if binder, ok := any(c).(interface{ BindEnv() }); ok {
		binder.BindEnv()
	}

	return c, nil
}

// readConfigurationData returns the raw YAML document, from the environment
// variable when it holds one, otherwise from the file.
func readConfigurationData(envKey, filename string) ([]byte, error) {
	if envKey != "" {
		if value := os.Getenv(envKey); value != "" {
			if stat, err := os.Stat(value); err == nil && !stat.IsDir() {
				data, err := os.ReadFile(value) // the variable holds a path.
				if err != nil {
					return nil, fmt.Errorf("iris: load configuration: %s: %w", envKey, err)
				}

				return data, nil
			}

			return []byte(value), nil // the variable holds the document.
		}
	}

	if filename == "" {
		if envKey == "" {
			return nil, fmt.Errorf("iris: load configuration: no environment variable and no filename were given")
		}

		return nil, fmt.Errorf("iris: load configuration: the %s environment variable is empty and no filename was given", envKey)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("iris: load configuration: %w", err)
	}

	return data, nil
}

// BaseConfiguration is the part every service's configuration has in common:
// where the server listens, which origin it answers and how the framework
// itself is configured. Embed it, inline, and add the service's own fields:
//
//	type Configuration struct {
//		iris.BaseConfiguration `yaml:",inline"`
//
//		DatabaseURL string `yaml:"DatabaseURL"`
//	}
//
// Load it with LoadConfiguration, which runs BindEnv for you, and listen with
// app.Listen(c.Addr()).
type BaseConfiguration struct {
	// Host is the network interface the server listens on. Empty means every
	// interface. See Addr.
	//
	// It is not Iris.Host: that one is the trusted domain the router matches
	// subdomains against, and Addr never reads it.
	Host string `yaml:"Host" json:"host"`
	// Port is the TCP port the server listens on. Zero means 8080. See Addr.
	Port int `yaml:"Port" json:"port"`
	// AllowOrigin is the origin line for the CORS middleware,
	// as Builder.AllowOrigin accepts it ("*", "none" or a comma-separated list).
	AllowOrigin string `yaml:"AllowOrigin" json:"allow_origin"`
	// EnableCompression reports whether request and response compression is on,
	// as Builder.Compression accepts it.
	EnableCompression bool `yaml:"EnableCompression" json:"enable_compression"`
	// RequestLog reports whether the access log is on,
	// as Builder.LogRequests accepts it.
	RequestLog bool `yaml:"RequestLog" json:"request_log"`
	// Iris is the framework configuration, as iris.New accepts it.
	Iris Options `yaml:"Iris" json:"iris"`
}

// BindEnv overrides the decoded Host, Port and AllowOrigin from the
// environment, so the same document can be deployed unchanged everywhere.
// The variables are HOST, PORT, HTTP_PLATFORM_PORT and ALLOW_ORIGIN.
//
// HTTP_PLATFORM_PORT wins over PORT: a hosting platform that picks the port
// for the process (IIS and Azure App Service do) passes it there, while PORT
// may still carry the value baked into the image.
//
// LoadConfiguration calls this after decoding. Call it yourself only when the
// configuration was built by other means.
func (c *BaseConfiguration) BindEnv() {
	if v := os.Getenv("HOST"); v != "" {
		c.Host = v
	}

	if port, ok := envPort("PORT"); ok {
		c.Port = port
	}

	if port, ok := envPort("HTTP_PLATFORM_PORT"); ok {
		c.Port = port
	}

	if v := os.Getenv("ALLOW_ORIGIN"); v != "" {
		c.AllowOrigin = v
	}
}

// envPort reads a port number from the environment. A value which is not a
// number is ignored: the configuration's own port stays, instead of the
// server starting on a port nobody asked for.
func envPort(key string) (int, bool) {
	v := os.Getenv(key)
	if v == "" {
		return 0, false
	}

	port, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}

	return port, true
}

// Addr returns the "host:port" address for Application.Listen. An empty Host
// listens on every interface (":port") and a zero Port defaults to 8080.
func (c *BaseConfiguration) Addr() string {
	port := c.Port
	if port == 0 {
		port = 8080
	}

	return net.JoinHostPort(c.Host, strconv.Itoa(port))
}
