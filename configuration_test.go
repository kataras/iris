// Black-box Testing
package iris_test

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	. "gopkg.in/kataras/iris.v6"
)

// $ go test -v -run TestConfiguration*

func TestConfigurationStatic(t *testing.T) {
	def := DefaultConfiguration()

	app := New(def)
	afterNew := *app.Config

	if !reflect.DeepEqual(def, afterNew) {
		t.Fatalf("Default configuration is not the same after NewFromConfig expected:\n %#v \ngot:\n %#v", def, afterNew)
	}

	afterNew.Charset = "changed"

	if reflect.DeepEqual(def, afterNew) {
		t.Fatalf("Configuration should be not equal, got: %#v", afterNew)
	}

	app = New(Configuration{DisableBanner: true})

	afterNew = *app.Config

	if app.Config.DisableBanner == false {
		t.Fatalf("Passing a Configuration field as Option fails, expected DisableBanner to be true but was false")
	}

	app = New() // empty , means defaults so
	if !reflect.DeepEqual(def, *app.Config) {
		t.Fatalf("Default configuration is not the same after NewFromConfig expected:\n %#v \ngot:\n %#v", def, *app.Config)
	}
}

func TestConfigurationOptions(t *testing.T) {
	charset := "MYCHARSET"
	disableBanner := true

	app := New(OptionCharset(charset), OptionDisableBanner(disableBanner))

	if got := app.Config.Charset; got != charset {
		t.Fatalf("Expected configuration Charset to be: %s but got: %s", charset, got)
	}

	if got := app.Config.DisableBanner; got != disableBanner {
		t.Fatalf("Expected configuration DisableBanner to be: %#v but got: %#v", disableBanner, got)
	}

	// now check if other default values are setted (should be setted automatically)

	expected := DefaultConfiguration()
	expected.Charset = charset
	expected.DisableBanner = disableBanner

	has := *app.Config
	if !reflect.DeepEqual(has, expected) {
		t.Fatalf("Default configuration is not the same after New expected:\n %#v \ngot:\n %#v", expected, has)
	}
}

func TestConfigurationOptionsDeep(t *testing.T) {
	charset := "MYCHARSET"
	disableBanner := true
	vhost := "mydomain.com"
	// first charset,disableBanner and profilepath, no canonical order.
	app := New(OptionCharset(charset), OptionDisableBanner(disableBanner), OptionVHost(vhost))

	expected := DefaultConfiguration()
	expected.Charset = charset
	expected.DisableBanner = disableBanner
	expected.VHost = vhost

	has := *app.Config

	if !reflect.DeepEqual(has, expected) {
		t.Fatalf("DEEP configuration is not the same after New expected:\n %#v \ngot:\n %#v", expected, has)
	}
}

func TestConfigurationYAML(t *testing.T) {
	// create the key and cert files on the fly, and delete them when this test finished
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
  VHost: iris-go.com
  VScheme: https://
  ReadTimeout: 0
  WriteTimeout: 5s
  MaxHeaderBytes: 8096
  CheckForUpdates: true
  DisablePathCorrection: false
  EnablePathEscape: false
  FireMethodNotAllowed: true
  DisableBanner: true
  DisableBodyConsumptionOnUnmarshal: true
  TimeFormat: Mon, 01 Jan 2006 15:04:05 GMT
  Charset: UTF-8
  Gzip: true

  `
	yamlFile.WriteString(yamlConfigurationContents)
	filename := yamlFile.Name()
	app := New(YAML(filename))

	c := app.Config

	if expected := "iris-go.com"; c.VHost != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected VHost %s but got %s", expected, c.VHost)
	}

	if expected := "https://"; c.VScheme != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected VScheme %s but got %s", expected, c.VScheme)
	}

	if expected := 0; c.ReadTimeout != time.Duration(expected) {
		t.Fatalf("error on TestConfigurationYAML: Expected ReadTimeout %s but got %s", expected, c.ReadTimeout)
	}

	if expected := time.Duration(5 * time.Second); c.WriteTimeout != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected WriteTimeout %s but got %s", expected, c.WriteTimeout)
	}

	if expected := 8096; c.MaxHeaderBytes != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected MaxHeaderBytes %s but got %s", expected, c.MaxHeaderBytes)
	}

	if expected := true; c.CheckForUpdates != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected checkForUpdates %v but got %v", expected, c.CheckForUpdates)
	}

	if expected := false; c.DisablePathCorrection != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected DisablePathCorrection %v but got %v", expected, c.DisablePathCorrection)
	}

	if expected := false; c.EnablePathEscape != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected EnablePathEscape %v but got %v", expected, c.EnablePathEscape)
	}

	if expected := true; c.FireMethodNotAllowed != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected FireMethodNotAllowed %v but got %v", expected, c.FireMethodNotAllowed)
	}

	if expected := true; c.DisableBanner != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected DisableBanner %v but got %v", expected, c.DisableBanner)
	}

	if expected := true; c.DisableBodyConsumptionOnUnmarshal != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected DisableBodyConsumptionOnUnmarshal %v but got %v",
			expected, c.DisableBodyConsumptionOnUnmarshal)
	}

	if expected := "Mon, 01 Jan 2006 15:04:05 GMT"; c.TimeFormat != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected TimeFormat %s but got %s", expected, c.TimeFormat)
	}

	if expected := "UTF-8"; c.Charset != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected Charset %s but got %s", expected, c.Charset)
	}

	if expected := true; c.Gzip != expected {
		t.Fatalf("error on TestConfigurationYAML: Expected != %v but got %v", expected, c.Gzip)
	}

}
