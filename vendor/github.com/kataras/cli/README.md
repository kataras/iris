<p align="center">
 <a href="https://github.com/kataras/cli"><img  width="600"  src="https://github.com/kataras/cli/raw/master/logo.png"></a>
 <br/>

 <a href="https://travis-ci.org/kataras/cli"><img src="https://img.shields.io/travis/kataras/cli.svg?style=flat-square" alt="Build Status"></a>
 <a href="https://github.com/kataras/cli/blob/master/LICENSE"><img src="https://img.shields.io/badge/%20license-MIT%20%20License%20-E91E63.svg?style=flat-square" alt="License"></a>
 <a href="https://github.com/kataras/cli/releases"><img src="https://img.shields.io/badge/%20release%20-%200.0.4-blue.svg?style=flat-square" alt="Releases"></a>
 <a href="https://godoc.org/github.com/kataras/cli"><img src="https://img.shields.io/badge/%20docs-reference-5272B4.svg?style=flat-square" alt="Read me docs"></a>
 <br/>
 <a href="https://kataras.rocket.chat/channel/cli"><img src="https://img.shields.io/badge/%20community-chat-00BCD4.svg?style=flat-square" alt="Build Status"></a>
 <a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square" alt="Built with GoLang"></a>
 <a href="#"><img src="https://img.shields.io/badge/platform-All-yellow.svg?style=flat-square" alt="Platforms"></a>

<br/><br/>
Build command line interfaced applications fast and easy.
<br/>
Ideally suited for novice Developers.


</p>

Quick view
-----------

```go
import "github.com/kataras/cli"

func main() {
  cli.NewApp("httpserver", "converts current directory into http server", "0.0.1").
  Flag("directory", "C:/users/myfiles", "specify a directory to convert").
  Run(func(cli.Flags) error {
    return nil
  })
}


```

Features
------------
- Simple to use, create a working app in one line
- Easy API, no need to read any docs, just type `cli.` via your favorite editor and your hand will select the correct function to do the job
- Auto command naming alias
- App has commands, each command can have subcommands, the same commands and their flags can be registered and used over multiple Apps
- Understands the go types automatically, no more `*string`
- Monitor, optionally, each command through App's action listener  
- Help command automation
- Share app's screen and output with any type of `io.Writer`

Installation
------------
The only requirement is the [Go Programming Language](https://golang.org/dl).

```bash
$ go get -u github.com/kataras/cli
```

Getting started
------------

```go
// NewApp creates and returns a new cli App instance
//
// example:
// app := cli.NewApp("iris", "Command line tool for Iris web framework", "0.0.1")
NewApp(name string, description string, version string) *App
```


```go

package main

import (
	"fmt"
	"github.com/kataras/cli"
)

func main() {
	app := cli.NewApp("httpserver", "converts current directory into http server", "0.0.1")

    // $executable -d, $executable --directory $the_dir
	app.Flag("directory", "C:/users/myfiles", "specify a current working directory")

    //  $executable listen
	listenCommand := cli.Command("listen", "starts the server")

    // $executable listen -h, $executable listen --host $the_host
	listenCommand.Flag("host", "127.0.0.1", "specify an address listener")   
    // $executable listen -p, $executable listen --port $the_port   
	listenCommand.Flag("port", 8080, "specify a port to listen")   
    // $executable listen -d, $executable listen --dir $the_dir     
	listenCommand.Flag("dir", "", "current working directory")    
    // $executable listen -r , $executable listen --req $the_req              
	listenCommand.Flag("req", nil, "a required flag because nil default given")

	listenCommand.Action(listen)

	app.Command(listenCommand) //register the listenCommand to the app.

	app.Run(run)
}

// httpserver -d C:/web/site
func run(args cli.Flags) error {
  // if the app has flags then 'run' will do its job as action, not as monitor
  fmt.Printf("Executing from global app's flag -d/-directory = %s\n ", args.String("directory"))

  // you can also run a command by code, listenCommand.Execute()
  return nil
}

// httpserver listen -h mydomain.com
func listen(args cli.Flags) error {
  fmt.Printf("Executing from command listen with Host: %s and Port: %d \n",
    args.String("host"), args.Int("port"))
  return nil
}

```
> Note that: --help (or -help, help, -h) global flag is automatically used and displays help message.


FAQ
------------

If you'd like to discuss this package, or ask questions about it, feel free to

 * Explore [these questions](https://github.com/kataras/cli/issues?cli=label%3Aquestion).
 * Post an issue or  idea [here](https://github.com/kataras/cli/issues).
 * Navigate to the [Chat][Chat].



Versioning
------------

Current: **0.0.4**

Read more about Semantic Versioning 2.0.0

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions



People
------------
The author of cli is [@kataras](https://github.com/kataras).


Contributing
------------
If you are interested in contributing to the cli project, please make a PR.

License
------------

This project is licensed under the MIT License.

License can be found [here](LICENSE).

[Travis Widget]: https://img.shields.io/travis/kataras/cli.svg?style=flat-square
[Travis]: http://travis-ci.org/kataras/cli
[License Widget]: https://img.shields.io/badge/license-MIT%20%20License%20-E91E63.svg?style=flat-square
[License]: https://github.com/kataras/cli/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/release-0.0.4-blue.svg?style=flat-square
[Release]: https://github.com/kataras/cli/releases
[Chat Widget]: https://img.shields.io/badge/community-chat-00BCD4.svg?style=flat-square
[Chat]: https://kataras.rocket.chat/channel/cli
[ChatMain]: https://kataras.rocket.chat/channel/cli
[ChatAlternative]: https://gitter.im/kataras/cli
[Report Widget]: https://img.shields.io/badge/report%20card-A%2B-F44336.svg?style=flat-square
[Report]: http://goreportcard.com/report/kataras/cli
[Documentation Widget]: https://img.shields.io/badge/docs-reference-5272B4.svg?style=flat-square
[Documentation]: https://godoc.org/github.com/kataras/cli
[Language Widget]: https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square
[Language]: http://golang.org
[Platform Widget]: https://img.shields.io/badge/platform-Any--OS-yellow.svg?style=flat-square
