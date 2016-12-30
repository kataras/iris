package main

import (
	"fmt"
	"github.com/kataras/go-options"
)

type appOptions struct {
	Name        string
	MaxRequests int
	DevMode     bool
	EnableLogs  bool
}

type app struct {
	options appOptions
}

func newApp(setters ...options.OptionSetter) *app {
	opts := appOptions{Name: "My App's Default Name"} // any default options here
	options.Struct(&opts, setters...)                 // convert any dynamic options to the appOptions struct, fills non-default options from the setters

	app := &app{options: opts}

	return app
}

// Name sets the appOptions.Name field
// pass an (optionall) option via static func
func Name(val string) options.OptionSetter {
	return options.Option("Name", val)
}

// Dev sets the appOptions.DevMode & appOptions.EnableLogs field
// pass an (optionall) option via static func
func Dev(val bool) options.OptionSetter {
	return options.Options{"DevMode": val, "EnableLogs": val}
}

// and so on...

func passDynamicOptions() {
	myApp := newApp(options.Options{"MaxRequests": 17, "DevMode": true})

	fmt.Printf("passDynamicOptions: %#v\n", myApp.options)
}

func passDynamicOptionsAlternative() {
	myApp := newApp(options.Option("MaxRequests", 17), options.Option("DevMode", true))

	fmt.Printf("passDynamicOptionsAlternative: %#v\n", myApp.options)
}

func passFuncsOptions() {
	myApp := newApp(Name("My name"), Dev(true))

	fmt.Printf("passFuncsOptions: %#v\n", myApp.options)
}

// go run $GOPATH/github.com/kataras/go-options/example/main.go
func main() {
	passDynamicOptions()
	passDynamicOptionsAlternative()
	passFuncsOptions()
}
