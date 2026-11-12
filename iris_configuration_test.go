package iris_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kataras/iris/v14"
)

type testServiceConfiguration struct {
	iris.BaseConfiguration `yaml:",inline"`

	DatabaseURL string `yaml:"DatabaseURL"`
}

const testConfigurationYAML = `Host: "127.0.0.1"
Port: 9090
AllowOrigin: "*"
EnableCompression: true
RequestLog: true
DatabaseURL: "postgres://localhost/db"
Iris:
  Name: "test-service"
  LogLevel: "debug"
`

// clearConfigurationEnv makes a case independent of the machine it runs on:
// BindEnv reads HOST, PORT, HTTP_PLATFORM_PORT and ALLOW_ORIGIN, and a runner
// with PORT set would otherwise fail every assertion about the document.
// An empty value is treated as unset by BindEnv.
func clearConfigurationEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{"TEST_SERVICE_CONFIG", "HOST", "PORT", "HTTP_PLATFORM_PORT", "ALLOW_ORIGIN"} {
		t.Setenv(key, "")
	}
}

func writeTempConfiguration(t *testing.T, contents string) string {
	t.Helper()

	filename := filepath.Join(t.TempDir(), "configuration.yml")
	if err := os.WriteFile(filename, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	return filename
}

func expectTestConfiguration(t *testing.T, c testServiceConfiguration) {
	t.Helper()

	if expected, got := "127.0.0.1", c.Host; expected != got {
		t.Fatalf("Host: expected: %s but got: %s", expected, got)
	}

	if expected, got := 9090, c.Port; expected != got {
		t.Fatalf("Port: expected: %d but got: %d", expected, got)
	}

	if expected, got := "*", c.AllowOrigin; expected != got {
		t.Fatalf("AllowOrigin: expected: %s but got: %s", expected, got)
	}

	if !c.EnableCompression || !c.RequestLog {
		t.Fatalf("EnableCompression and RequestLog: expected both true but got: %v, %v", c.EnableCompression, c.RequestLog)
	}

	if expected, got := "postgres://localhost/db", c.DatabaseURL; expected != got {
		t.Fatalf("DatabaseURL: expected: %s but got: %s", expected, got)
	}

	if expected, got := "test-service", c.Iris.Name; expected != got {
		t.Fatalf("Iris.Name: expected: %s but got: %s", expected, got)
	}

	if expected, got := "debug", c.Iris.LogLevel; expected != got {
		t.Fatalf("Iris.LogLevel: expected: %s but got: %s", expected, got)
	}
}

func TestLoadConfigurationFromFile(t *testing.T) {
	clearConfigurationEnv(t)

	filename := writeTempConfiguration(t, testConfigurationYAML)

	c, err := iris.LoadConfiguration[testServiceConfiguration]("TEST_SERVICE_CONFIG", filename)
	if err != nil {
		t.Fatal(err)
	}

	expectTestConfiguration(t, c)
}

func TestLoadConfigurationFromEnvironmentText(t *testing.T) {
	clearConfigurationEnv(t)
	t.Setenv("TEST_SERVICE_CONFIG", testConfigurationYAML)

	c, err := iris.LoadConfiguration[testServiceConfiguration]("TEST_SERVICE_CONFIG", "does-not-exist.yml")
	if err != nil {
		t.Fatal(err)
	}

	expectTestConfiguration(t, c)
}

func TestLoadConfigurationFromEnvironmentPath(t *testing.T) {
	clearConfigurationEnv(t)

	filename := writeTempConfiguration(t, testConfigurationYAML)
	t.Setenv("TEST_SERVICE_CONFIG", filename)

	c, err := iris.LoadConfiguration[testServiceConfiguration]("TEST_SERVICE_CONFIG", "does-not-exist.yml")
	if err != nil {
		t.Fatal(err)
	}

	expectTestConfiguration(t, c)
}

func TestLoadConfigurationMissingFile(t *testing.T) {
	clearConfigurationEnv(t)

	if _, err := iris.LoadConfiguration[testServiceConfiguration]("TEST_SERVICE_CONFIG", "does-not-exist.yml"); err == nil {
		t.Fatal("expected an error for a missing configuration file")
	}
}

// The environment overrides run after the document is decoded, so a container's
// HOST, PORT and ALLOW_ORIGIN win over the file.
func TestLoadConfigurationBindEnvPrecedence(t *testing.T) {
	clearConfigurationEnv(t)

	filename := writeTempConfiguration(t, testConfigurationYAML)

	t.Setenv("HOST", "0.0.0.0")
	t.Setenv("PORT", "7000")
	t.Setenv("ALLOW_ORIGIN", "https://iris-go.com")

	c, err := iris.LoadConfiguration[testServiceConfiguration]("TEST_SERVICE_CONFIG", filename)
	if err != nil {
		t.Fatal(err)
	}

	if expected, got := "0.0.0.0", c.Host; expected != got {
		t.Fatalf("Host: expected: %s but got: %s", expected, got)
	}

	if expected, got := 7000, c.Port; expected != got {
		t.Fatalf("Port: expected: %d but got: %d", expected, got)
	}

	if expected, got := "https://iris-go.com", c.AllowOrigin; expected != got {
		t.Fatalf("AllowOrigin: expected: %s but got: %s", expected, got)
	}
}

// HTTP_PLATFORM_PORT is the port the hosting platform reserved for the process;
// it wins over PORT.
func TestLoadConfigurationBindEnvPlatformPort(t *testing.T) {
	clearConfigurationEnv(t)

	filename := writeTempConfiguration(t, testConfigurationYAML)

	t.Setenv("PORT", "7000")
	t.Setenv("HTTP_PLATFORM_PORT", "6000")

	c, err := iris.LoadConfiguration[testServiceConfiguration]("TEST_SERVICE_CONFIG", filename)
	if err != nil {
		t.Fatal(err)
	}

	if expected, got := 6000, c.Port; expected != got {
		t.Fatalf("Port: expected: %d but got: %d", expected, got)
	}
}

// A port which is not a number is ignored: the document's port stays, instead
// of the server starting somewhere nobody asked for.
func TestLoadConfigurationBindEnvIgnoresBadPort(t *testing.T) {
	clearConfigurationEnv(t)

	filename := writeTempConfiguration(t, testConfigurationYAML)

	t.Setenv("PORT", "not-a-port")
	t.Setenv("HTTP_PLATFORM_PORT", "also-not-a-port")

	c, err := iris.LoadConfiguration[testServiceConfiguration]("TEST_SERVICE_CONFIG", filename)
	if err != nil {
		t.Fatal(err)
	}

	if expected, got := 9090, c.Port; expected != got {
		t.Fatalf("Port: expected: %d but got: %d", expected, got)
	}
}

func TestBaseConfigurationAddr(t *testing.T) {
	tests := []struct {
		host     string
		port     int
		expected string
	}{
		{"127.0.0.1", 9090, "127.0.0.1:9090"},
		{"", 9090, ":9090"},
		{"", 0, ":8080"},
		{"localhost", 0, "localhost:8080"},
	}

	for _, tt := range tests {
		c := iris.BaseConfiguration{Host: tt.host, Port: tt.port}
		if got := c.Addr(); got != tt.expected {
			t.Fatalf("Addr: host: %q port: %d: expected: %s but got: %s", tt.host, tt.port, tt.expected, got)
		}
	}
}
