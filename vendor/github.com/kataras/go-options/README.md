<p align="center">
  <img src="/logo.jpg" height="400">
  <br/>

 <a href="https://travis-ci.org/kataras/go-options"><img src="https://img.shields.io/travis/kataras/go-options.svg?style=flat-square" alt="Build Status"></a>


 <a href="https://github.com/avelino/awesome-go"><img src="https://img.shields.io/badge/awesome-%E2%9C%93-ff69b4.svg?style=flat-square" alt="Awesome GoLang"></a>
 
 <a href="http://goreportcard.com/report/kataras/go-options"><img src="https://img.shields.io/badge/-A%2B-F44336.svg?style=flat-square" alt="Report A+"></a>


 <a href="https://github.com/kataras/go-options/blob/master/LICENSE"><img src="https://img.shields.io/badge/%20license-MIT%20-E91E63.svg?style=flat-square" alt="License"></a>



 <a href="https://github.com/kataras/go-options/releases"><img src="https://img.shields.io/badge/%20release%20-%20v0.0.1-blue.svg?style=flat-square" alt="Releases"></a>

 <a href="https://godoc.org/github.com/kataras/go-options"><img src="https://img.shields.io/badge/%20docs-reference-5272B4.svg?style=flat-square" alt="Read me docs"></a>

 <a href="https://kataras.rocket.chat/channel/go-options"><img src="https://img.shields.io/badge/%20community-chat-00BCD4.svg?style=flat-square" alt="Chat"></a>

<br/><br/>

Clean APIs for your Go Applications. Inspired by <a href="https://github.com/tmrts/go-patterns/blob/master/idiom/functional-options.md">functional options pattern</a>.

</p>

Quick view
-----------

```go
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

```

Installation
------------
The only requirement is the [Go Programming Language](https://golang.org/dl), at least v1.7.

```bash
$ go get -u github.com/kataras/go-options
```



FAQ
------------

If you'd like to discuss this package, or ask questions about it, feel free to

 * Explore [these questions](https://github.com/kataras/go-options/issues?go-options=label%3Aquestion).
 * Post an issue or  idea [here](https://github.com/kataras/go-options/issues).
 * Navigate to the [Chat][Chat].



Versioning
------------

Current: **v0.0.1**

Read more about Semantic Versioning 2.0.0

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions



People
------------
The author of go-options is [@kataras](https://github.com/kataras).


Contributing
------------
If you are interested in contributing to the go-options project, please make a PR.

License
------------

This project is licensed under the MIT License.

License can be found [here](LICENSE).

[Chat Widget]: https://img.shields.io/badge/community-chat-00BCD4.svg?style=flat-square
[Chat]: https://kataras.rocket.chat/channel/go-options