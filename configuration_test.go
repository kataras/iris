package iris

import (
	"reflect"
	"testing"
)

// go test -v -run TestConfig*

func TestConfigStatic(t *testing.T) {
	def := DefaultConfiguration()

	api := New(def)
	afterNew := *api.Config

	if !reflect.DeepEqual(def, afterNew) {
		t.Fatalf("Default configuration is not the same after NewFromConfig expected:\n %#v \ngot:\n %#v", def, afterNew)
	}

	afterNew.Charset = "changed"

	if reflect.DeepEqual(def, afterNew) {
		t.Fatalf("Configuration should be not equal, got: %#v", afterNew)
	}

	api = New(Configuration{IsDevelopment: true})

	afterNew = *api.Config

	if api.Config.IsDevelopment == false {
		t.Fatalf("Passing a Configuration field as Option fails, expected IsDevelopment to be true but was false")
	}

	api = New() // empty , means defaults so
	if !reflect.DeepEqual(def, *api.Config) {
		t.Fatalf("Default configuration is not the same after NewFromConfig expected:\n %#v \ngot:\n %#v", def, *api.Config)
	}
}

func TestConfigOptions(t *testing.T) {
	charset := "MYCHARSET"
	dev := true

	api := New(OptionCharset(charset), OptionIsDevelopment(dev))

	if got := api.Config.Charset; got != charset {
		t.Fatalf("Expected configuration Charset to be: %s but got: %s", charset, got)
	}

	if got := api.Config.IsDevelopment; got != dev {
		t.Fatalf("Expected configuration IsDevelopment to be: %#v but got: %#v", dev, got)
	}

	// now check if other default values are setted (should be setted automatically)

	expected := DefaultConfiguration()
	expected.Charset = charset
	expected.IsDevelopment = dev

	has := *api.Config
	if !reflect.DeepEqual(has, expected) {
		t.Fatalf("Default configuration is not the same after New expected:\n %#v \ngot:\n %#v", expected, has)
	}
}

func TestConfigOptionsDeep(t *testing.T) {
	cookiename := "MYSESSIONID"
	charset := "MYCHARSET"
	dev := true
	profilePath := "/mypprof"
	// first session, after charset,dev and profilepath, no canonical order.
	api := New(OptionSessionsCookie(cookiename), OptionCharset(charset), OptionIsDevelopment(dev), OptionProfilePath(profilePath))

	expected := DefaultConfiguration()
	expected.Sessions.Cookie = cookiename
	expected.Charset = charset
	expected.IsDevelopment = dev
	expected.ProfilePath = profilePath

	has := *api.Config

	if !reflect.DeepEqual(has, expected) {
		t.Fatalf("DEEP configuration is not the same after New expected:\n %#v \ngot:\n %#v", expected, has)
	}
}

// ServerConfiguration is independent so make a small test for that
func TestConfigServerOptions(t *testing.T) {
	expected := DefaultServerConfiguration()
	expected.ListeningAddr = "mydomain.com:80"
	expected.RedirectTo = "https://mydomain.com:443"
	expected.Virtual = true

	c := ServerConfiguration{ListeningAddr: expected.ListeningAddr, RedirectTo: expected.RedirectTo, Virtual: expected.Virtual}
	// static config test
	s := newServer(c)

	if !reflect.DeepEqual(s.Config, expected) {
		t.Fatalf("Static Server Configuration not equal after newServer, expected:\n%#v \nwhile got:\n%#v", expected, s.Config)
	}

}
