package iris

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
)

// $ go test -v -run TestConfiguration*

func TestConfigurationStatic(t *testing.T) {
	def := DefaultConfiguration()

	app := New()
	afterNew := *app.config

	if !reflect.DeepEqual(def, afterNew) {
		t.Fatalf("Default configuration is not the same after NewFromConfig expected:\n %#v \ngot:\n %#v", def, afterNew)
	}

	afterNew.Charset = "changed"

	if reflect.DeepEqual(def, afterNew) {
		t.Fatalf("Configuration should be not equal, got: %#v", afterNew)
	}

	app = New().Configure(WithConfiguration(Configuration{DisableBodyConsumptionOnUnmarshal: true}))

	afterNew = *app.config

	if app.config.DisableBodyConsumptionOnUnmarshal == false {
		t.Fatalf("Passing a Configuration field as Option fails, expected DisableBodyConsumptionOnUnmarshal to be true but was false")
	}

	app = New() // empty , means defaults so
	if !reflect.DeepEqual(def, *app.config) {
		t.Fatalf("Default configuration is not the same after NewFromConfig expected:\n %#v \ngot:\n %#v", def, *app.config)
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

	// now check if other default values are setted (should be setted automatically)

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

	app := New().Configure(WithCharset(charset), WithoutBodyConsumptionOnUnmarshal, WithoutBanner, WithoutVersionChecker)

	expected := DefaultConfiguration()
	expected.Charset = charset
	expected.DisableBodyConsumptionOnUnmarshal = true
	expected.DisableStartupLog = true
	expected.DisableVersionChecker = true

	has := *app.config

	if !reflect.DeepEqual(has, expected) {
		t.Fatalf("DEEP configuration is not the same after New expected:\n %#v \ngot:\n %#v", expected, has)
	}
}

func TestConfigurationYAML(t *testing.T) {
	yamlFile, ferr := ioutil.TempFile("", "configuration.yml")

	if ferr != nil {
		t.Fatal(ferr)
	}

	defer func() {
		yamlFile.Close()
		time.Sleep(50 * time.Millisecond)
		os.Remove(yamlFile.Name())
	}()

	yamlConfigurationContents := `
DisableVersionChecker: true
DisablePathCorrection: false
EnablePathEscape: false
FireMethodNotAllowed: true
EnableOptimizations: true
DisableBodyConsumptionOnUnmarshal: true
TimeFormat: Mon, 01 Jan 2006 15:04:05 GMT
Charset: UTF-8

`
	yamlFile.WriteString(yamlConfigurationContents)
	filename := yamlFile.Name()
	app := New().Configure(WithConfiguration(YAML(filename)))

	c := app.config

	if expected := true; c.DisableVersionChecker != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected DisableVersionChecker %v but got %v", expected, c.DisableVersionChecker)
	}

	if expected := false; c.DisablePathCorrection != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected DisablePathCorrection %v but got %v", expected, c.DisablePathCorrection)
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

	if expected := "Mon, 01 Jan 2006 15:04:05 GMT"; c.TimeFormat != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected TimeFormat %s but got %s", expected, c.TimeFormat)
	}

	if expected := "UTF-8"; c.Charset != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected Charset %s but got %s", expected, c.Charset)
	}
}

func TestConfigurationTOML(t *testing.T) {
	tomlFile, ferr := ioutil.TempFile("", "configuration.toml")

	if ferr != nil {
		t.Fatal(ferr)
	}

	defer func() {
		tomlFile.Close()
		time.Sleep(50 * time.Millisecond)
		os.Remove(tomlFile.Name())
	}()

	tomlConfigurationContents := `
DisableVersionChecker = true
EnablePathEscape = false
FireMethodNotAllowed = true
EnableOptimizations = true
DisableBodyConsumptionOnUnmarshal = true
TimeFormat = "Mon, 01 Jan 2006 15:04:05 GMT"
Charset = "UTF-8"

[Other]
	# Indentation (tabs and/or spaces) is allowed but not required
	MyServerName = "Iris"

`
	tomlFile.WriteString(tomlConfigurationContents)
	filename := tomlFile.Name()
	app := New().Configure(WithConfiguration(TOML(filename)))

	c := app.config

	if expected := true; c.DisableVersionChecker != expected {
		t.Fatalf("error on TestConfigurationTOML: Expected DisableVersionChecker %v but got %v", expected, c.DisableVersionChecker)
	}

	if expected := false; c.DisablePathCorrection != expected {
		t.Fatalf("error on TestConfigurationTOML: Expected DisablePathCorrection %v but got %v", expected, c.DisablePathCorrection)
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

	if expected := "Mon, 01 Jan 2006 15:04:05 GMT"; c.TimeFormat != expected {
		t.Fatalf("error on TestConfigurationTOML: Expected TimeFormat %s but got %s", expected, c.TimeFormat)
	}

	if expected := "UTF-8"; c.Charset != expected {
		t.Fatalf("error on TestConfigurationTOML: Expected Charset %s but got %s", expected, c.Charset)
	}
}
