Rizla builds, runs and monitors your Go Applications with ease.

[![Travis Widget]][Travis] [![Release Widget]][Release] [![Report Widget]][Report] [![License Widget]][License] [![Chat Widget]][Chat]

# Get Started

```bash
$ rizla main.go #single project monitoring
$ rizla C:/myprojects/project1/main.go C:/myprojects/project2/main.go #multi projects monitoring
```

Want to use it from your project's source code? easy
```sh
$ cat from_code_simple.go
```
```go
package main

import (
	"github.com/kataras/rizla/rizla"
)

func main() {
  // Build, run & start monitoring the projects
  rizla.Run("C:/iris-project/main.go", "C:/otherproject/main.go")
}
```

```sh
$ cat from_code_pro.go
```
```go
package main

import (
	"github.com/kataras/rizla/rizla"
	"path/filepath"
	"runtime"
	"time"
	"os"
)

func main() {


  // Create a new project by the main source file
  project := rizla.NewProject("C:/myproject/main.go")

  // The below are optional

  // Optional, set the messages/logs output destination of our app project,
  // let's set them to their defaults
  project.Out = rizla.NewPrinter(os.Stdout)
  project.Err = rizla.NewPrinter(os.Stderr)

  project.Name = "My super project"
  // Allow reload every 3 seconds or more no less
  project.AllowReloadAfter = time.Duration(3) * time.Second
  // Custom subdirectory matcher, for the watcher, return true to include this folder to the watcher
  // the default is:
  project.Watcher = func(absolutePath string) bool {
     	base := filepath.Base(abs)
		return !(base == ".git" || base == "node_modules" || base == "vendor")
  }
  // Custom file matcher on runtime (file change), return true to reload when a file with this file name changed
  // the default is:
  project.Matcher = func(filename string) bool {
		isWindows = runtime.GOOS == "windows"
		goExt     = ".go"
		return (filepath.Ext(fullname) == goExt) ||
		(!isWindows && strings.Contains(fullname, goExt))
  }
  // Add arguments, these will be used from the executable file
  project.Args = []string{"-myargument","the value","-otherargument","a value"}
  // Custom callback before reload, the default is:
  project.OnReload = func(string) {
		fromproject := ""
		if p.Name != "" {
			fromproject = "From project '" + project.Name + "': "
		}
		project.Out.Infof("\n%sA change has been detected, reloading now...", fromproject)
   }
   // Custom callback after reload, the default is:
   project.OnReloaded = func(string) {
 		project.Out.Successf("ready!\n")
   }

  // End of optional

  // Add the project to the rizla container
  rizla.Add(project)
  //  Build, run & start monitoring the project(s)
  rizla.Run()
}
```


> That's all!

Installation
------------
The only requirement is the [Go Programming Language](https://golang.org/dl)

`$ go get -u github.com/kataras/rizla`

FAQ
------------
Ask questions and get real-time answer from the [Chat][CHAT].


Features
------------
- Super easy - is created for everyone.
- You can use it either as command line tool either as part of your project's source code!
- Multi-Monitoring - Supports monitoring of unlimited projects
- No forever loops with filepath.Walk, rizla uses the Operating System's signals to fire a reload.


People
------------
If you'd like to discuss this package, or ask questions about it, feel free to [Chat][CHAT].

The author of rizla is [@kataras](https://github.com/kataras).


Versioning
------------

Current: **v0.0.6**

[HISTORY](https://github.com/kataras/rizla/blob/master/HISTORY.md) file is your best friend!

Read more about Semantic Versioning 2.0.0

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions


Todo
------------

 - [ ] Tests
 - [ ] Provide full examples.

Third-Party Licenses
------------

Third-Party Licenses can be found [here](THIRDPARTY-LICENSE.md)


License
------------

This project is licensed under the MIT License.

License can be found [here](LICENSE).

[Travis Widget]: https://img.shields.io/travis/kataras/rizla.svg?style=flat-square
[Travis]: http://travis-ci.org/kataras/rizla
[License Widget]: https://img.shields.io/badge/license-MIT%20%20License%20-E91E63.svg?style=flat-square
[License]: https://github.com/kataras/rizla/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/release-v0.0.6-blue.svg?style=flat-square
[Release]: https://github.com/kataras/rizla/releases
[Chat Widget]: https://img.shields.io/badge/community-chat-00BCD4.svg?style=flat-square
[Chat]: https://kataras.rocket.chat/channel/rizla
[ChatMain]: https://kataras.rocket.chat/channel/rizla
[ChatAlternative]: https://gitter.im/kataras/rizla
[Report Widget]: https://img.shields.io/badge/report%20card-A%2B-F44336.svg?style=flat-square
[Report]: http://goreportcard.com/report/kataras/rizla
[Language Widget]: https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square
[Language]: http://golang.org
[Platform Widget]: https://img.shields.io/badge/platform-Any--OS-gray.svg?style=flat-square
