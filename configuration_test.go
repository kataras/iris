package iris

import (
	"os"
	"reflect"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// $ go test -v -run TestConfiguration*

func TestConfigurationStatic(t *testing.T) {
	def := DefaultConfiguration()

	app := New()
	afterNew := *app.config

	if !reflect.DeepEqual(def, afterNew) {
		t.Fatalf("Default configuration is not the same after New expected:\n %#v \ngot:\n %#v", def, afterNew)
	}

	afterNew.Charset = "changed"

	if reflect.DeepEqual(def, afterNew) {
		t.Fatalf("Configuration should be not equal, got: %#v", afterNew)
	}

	app = New().Configure(WithConfiguration(Configuration{DisableBodyConsumptionOnUnmarshal: true}))

	afterNew = *app.config

	if !app.config.DisableBodyConsumptionOnUnmarshal {
		t.Fatalf("Passing a Configuration field as Option fails, expected DisableBodyConsumptionOnUnmarshal to be true but was false")
	}

	app = New() // empty , means defaults so
	if !reflect.DeepEqual(def, *app.config) {
		t.Fatalf("Default configuration is not the same after New expected:\n %#v \ngot:\n %#v", def, *app.config)
	}
}

func TestConfigurationOptions(t *testing.T) {
	charset := "MYCHARSET"
	disableBodyConsumptionOnUnmarshal := true
	disableBanner := true

	app := New().Configure(WithCharset(charset), WithoutBodyConsumptionOnUnmarshal, WithoutBanner)

	if got := app.config.Charset; got != charset {
		t.Fatalf("Expected configuration Charset to be: %s but got: %s", charset, got)
	}

	if got := app.config.DisableBodyConsumptionOnUnmarshal; got != disableBodyConsumptionOnUnmarshal {
		t.Fatalf("Expected configuration DisableBodyConsumptionOnUnmarshal to be: %#v but got: %#v", disableBodyConsumptionOnUnmarshal, got)
	}

	if got := app.config.DisableStartupLog; got != disableBanner {
		t.Fatalf("Expected configuration DisableStartupLog to be: %#v but got: %#v", disableBanner, got)
	}

	// now check if other default values are set (should be set automatically)

	expected := DefaultConfiguration()
	expected.Charset = charset
	expected.DisableBodyConsumptionOnUnmarshal = disableBodyConsumptionOnUnmarshal
	expected.DisableStartupLog = disableBanner

	has := *app.config
	if !reflect.DeepEqual(has, expected) {
		t.Fatalf("Default configuration is not the same after New expected:\n %#v \ngot:\n %#v", expected, has)
	}
}

func TestConfigurationOptionsDeep(t *testing.T) {
	charset := "MYCHARSET"

	app := New().Configure(WithCharset(charset), WithoutBodyConsumptionOnUnmarshal, WithoutBanner)

	expected := DefaultConfiguration()
	expected.Charset = charset
	expected.DisableBodyConsumptionOnUnmarshal = true
	expected.DisableStartupLog = true

	has := *app.config

	if !reflect.DeepEqual(has, expected) {
		t.Fatalf("DEEP configuration is not the same after New expected:\n %#v \ngot:\n %#v", expected, has)
	}
}

func createGlobalConfiguration(t *testing.T) {
	filename := homeConfigurationFilename(".yml")
	c, err := parseYAML(filename)
	if err != nil {
		// this error will be occurred the first time that the configuration
		// file doesn't exist.
		// Create the YAML-ONLY global configuration file now using the default configuration 'c'.
		// This is useful when we run multiple iris servers that share the same
		// configuration, even with custom values at its "Other" field.
		out, err := yaml.Marshal(&c)

		if err == nil {
			err = os.WriteFile(filename, out, os.FileMode(0666))
		}
		if err != nil {
			t.Fatalf("error while writing the global configuration field at: %s. Trace: %v\n", filename, err)
		}
	}
}

func TestConfigurationGlobal(t *testing.T) {
	t.Cleanup(func() {
		os.Remove(homeConfigurationFilename(".yml"))
	})

	createGlobalConfiguration(t)

	testConfigurationGlobal(t, WithGlobalConfiguration)
	testConfigurationGlobal(t, WithConfiguration(YAML(globalConfigurationKeyword)))
}

func testConfigurationGlobal(t *testing.T, c Configurator) {
	app := New().Configure(c)

	if has, expected := *app.config, DefaultConfiguration(); !reflect.DeepEqual(has, expected) {
		t.Fatalf("global configuration (which should be defaulted) is not the same with the default one:\n %#v \ngot:\n %#v", has, expected)
	}
}

func TestConfigurationYAML(t *testing.T) {
	yamlFile, ferr := os.CreateTemp("", "configuration.yml")

	if ferr != nil {
		t.Fatal(ferr)
	}

	defer func() {
		yamlFile.Close()
		time.Sleep(50 * time.Millisecond)
		os.Remove(yamlFile.Name())
	}()

	yamlConfigurationContents := `
DisablePathCorrection: false
DisablePathCorrectionRedirection: true
EnablePathIntelligence: true
EnablePathEscape: false
FireMethodNotAllowed: true
EnableOptimizations: true
DisableBodyConsumptionOnUnmarshal: true
TimeFormat: "Mon, 02 Jan 2006 15:04:05 GMT"
Charset: "utf-8"
RemoteAddrHeaders:
  - X-Real-Ip
  - X-Forwarded-For
  - CF-Connecting-IP
HostProxyHeaders:
  X-Host: true
SSLProxyHeaders:
  X-Forwarded-Proto: https
Other:
  MyServerName: "Iris: https://github.com/kataras/iris"
`
	yamlFile.WriteString(yamlConfigurationContents)
	filename := yamlFile.Name()
	app := New().Configure(WithConfiguration(YAML(filename)))

	c := app.config

	if expected := false; c.DisablePathCorrection != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected DisablePathCorrection %v but got %v", expected, c.DisablePathCorrection)
	}

	if expected := true; c.DisablePathCorrectionRedirection != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected DisablePathCorrectionRedirection %v but got %v", expected, c.DisablePathCorrectionRedirection)
	}

	if expected := true; c.EnablePathIntelligence != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected EnablePathIntelligence %v but got %v", expected, c.EnablePathIntelligence)
	}

	if expected := false; c.EnablePathEscape != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected EnablePathEscape %v but got %v", expected, c.EnablePathEscape)
	}

	if expected := true; c.EnableOptimizations != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected EnableOptimizations %v but got %v", expected, c.EnablePathEscape)
	}

	if expected := true; c.FireMethodNotAllowed != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected FireMethodNotAllowed %v but got %v", expected, c.FireMethodNotAllowed)
	}

	if expected := true; c.DisableBodyConsumptionOnUnmarshal != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected DisableBodyConsumptionOnUnmarshal %v but got %v", expected, c.DisableBodyConsumptionOnUnmarshal)
	}

	if expected := "Mon, 02 Jan 2006 15:04:05 GMT"; c.TimeFormat != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected TimeFormat %s but got %s", expected, c.TimeFormat)
	}

	if expected := "utf-8"; c.Charset != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected Charset %s but got %s", expected, c.Charset)
	}

	if len(c.RemoteAddrHeaders) == 0 {
		t.Fatalf("error on TestConfigurationYAML: Expected RemoteAddrHeaders to be filled")
	}

	expectedRemoteAddrHeaders := []string{
		"X-Real-Ip",
		"X-Forwarded-For",
		"CF-Connecting-IP",
	}

	if expected, got := len(c.RemoteAddrHeaders), len(expectedRemoteAddrHeaders); expected != got {
		t.Fatalf("error on TestConfigurationYAML: Expected RemoteAddrHeaders' len(%d) and got(%d), len is not the same", expected, got)
	}

	for i, v := range c.RemoteAddrHeaders {
		if expected, got := expectedRemoteAddrHeaders[i], v; expected != got {
			t.Fatalf("error on TestConfigurationYAML: Expected RemoteAddrHeaders[%d] = %s but got %s", i, expected, got)
		}
	}

	expectedHostProxyHeaders := map[string]bool{
		"X-Host": true,
	}

	if expected, got := len(c.HostProxyHeaders), len(expectedHostProxyHeaders); expected != got {
		t.Fatalf("error on TestConfigurationYAML: Expected HostProxyHeaders' len(%d) and got(%d), len is not the same", expected, got)
	}

	for k, v := range c.HostProxyHeaders {
		if expected, got := expectedHostProxyHeaders[k], v; expected != got {
			t.Fatalf("error on TestConfigurationYAML: Expected HostProxyHeaders[%s] = %t but got %t", k, expected, got)
		}
	}

	expectedSSLProxyHeaders := map[string]string{
		"X-Forwarded-Proto": "https",
	}

	if expected, got := len(c.SSLProxyHeaders), len(c.SSLProxyHeaders); expected != got {
		t.Fatalf("error on TestConfigurationYAML: Expected SSLProxyHeaders' len(%d) and got(%d), len is not the same", expected, got)
	}

	for k, v := range c.SSLProxyHeaders {
		if expected, got := expectedSSLProxyHeaders[k], v; expected != got {
			t.Fatalf("error on TestConfigurationYAML: Expected SSLProxyHeaders[%s] = %s but got %s", k, expected, got)
		}
	}

	if len(c.Other) == 0 {
		t.Fatalf("error on TestConfigurationYAML: Expected Other to be filled")
	}

	if expected, got := "Iris: https://github.com/kataras/iris", c.Other["MyServerName"]; expected != got {
		t.Fatalf("error on TestConfigurationYAML: Expected Other['MyServerName'] %s but got %s", expected, got)
	}
}

func TestConfigurationTOML(t *testing.T) {
	tomlFile, ferr := os.CreateTemp("", "configuration.toml")

	if ferr != nil {
		t.Fatal(ferr)
	}

	defer func() {
		tomlFile.Close()
		time.Sleep(50 * time.Millisecond)
		os.Remove(tomlFile.Name())
	}()

	tomlConfigurationContents := `
DisablePathCorrectionRedirection = true
EnablePathEscape = false
FireMethodNotAllowed = true
EnableOptimizations = true
DisableBodyConsumptionOnUnmarshal = true
TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
Charset = "utf-8"

RemoteAddrHeaders = ["X-Real-Ip", "X-Forwarded-For", "CF-Connecting-IP"]

[Other]
	# Indentation (tabs and/or spaces) is allowed but not required
	MyServerName = "Iris: https://github.com/kataras/iris"

`
	tomlFile.WriteString(tomlConfigurationContents)
	filename := tomlFile.Name()
	app := New().Configure(WithConfiguration(TOML(filename)))

	c := app.config

	if expected := false; c.DisablePathCorrection != expected {
		t.Fatalf("error on TestConfigurationTOML: Expected DisablePathCorrection %v but got %v", expected, c.DisablePathCorrection)
	}

	if expected := true; c.DisablePathCorrectionRedirection != expected {
		t.Fatalf("error on TestConfigurationTOML: Expected DisablePathCorrectionRedirection %v but got %v", expected, c.DisablePathCorrectionRedirection)
	}

	if expected := false; c.EnablePathEscape != expected {
		t.Fatalf("error on TestConfigurationTOML: Expected EnablePathEscape %v but got %v", expected, c.EnablePathEscape)
	}

	if expected := true; c.EnableOptimizations != expected {
		t.Fatalf("error on TestConfigurationTOML: Expected EnableOptimizations %v but got %v", expected, c.EnablePathEscape)
	}

	if expected := true; c.FireMethodNotAllowed != expected {
		t.Fatalf("error on TestConfigurationTOML: Expected FireMethodNotAllowed %v but got %v", expected, c.FireMethodNotAllowed)
	}

	if expected := true; c.DisableBodyConsumptionOnUnmarshal != expected {
		t.Fatalf("error on TestConfigurationTOML: Expected DisableBodyConsumptionOnUnmarshal %v but got %v", expected, c.DisableBodyConsumptionOnUnmarshal)
	}

	if expected := "Mon, 02 Jan 2006 15:04:05 GMT"; c.TimeFormat != expected {
		t.Fatalf("error on TestConfigurationTOML: Expected TimeFormat %s but got %s", expected, c.TimeFormat)
	}

	if expected := "utf-8"; c.Charset != expected {
		t.Fatalf("error on TestConfigurationTOML: Expected Charset %s but got %s", expected, c.Charset)
	}

	if len(c.RemoteAddrHeaders) == 0 {
		t.Fatalf("error on TestConfigurationTOML: Expected RemoteAddrHeaders to be filled")
	}

	expectedRemoteAddrHeaders := []string{
		"X-Real-Ip",
		"X-Forwarded-For",
		"CF-Connecting-IP",
	}

	if expected, got := len(c.RemoteAddrHeaders), len(expectedRemoteAddrHeaders); expected != got {
		t.Fatalf("error on TestConfigurationTOML: Expected RemoteAddrHeaders' len(%d) and got(%d), len is not the same", expected, got)
	}

	for i, got := range c.RemoteAddrHeaders {
		if expected := expectedRemoteAddrHeaders[i]; expected != got {
			t.Fatalf("error on TestConfigurationTOML: Expected RemoteAddrHeaders[%d] = %s but got %s", i, expected, got)
		}
	}

	if len(c.Other) == 0 {
		t.Fatalf("error on TestConfigurationTOML: Expected Other to be filled")
	}

	if expected, got := "Iris: https://github.com/kataras/iris", c.Other["MyServerName"]; expected != got {
		t.Fatalf("error on TestConfigurationTOML: Expected Other['MyServerName'] %s but got %s", expected, got)
	}
}
