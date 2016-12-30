<a href="https://travis-ci.org/kataras/go-errors"><img src="https://img.shields.io/travis/kataras/go-errors.svg?style=flat-square" alt="Build Status"></a>
<a href="https://github.com/kataras/go-errors/blob/master/LICENSE"><img src="https://img.shields.io/badge/%20license-MIT%20%20License%20-E91E63.svg?style=flat-square" alt="License"></a>
<a href="https://github.com/kataras/go-errors/releases"><img src="https://img.shields.io/badge/%20release%20-%20v0.0.4-blue.svg?style=flat-square" alt="Releases"></a>
<a href="#docs"><img src="https://img.shields.io/badge/%20docs-reference-5272B4.svg?style=flat-square" alt="Read me docs"></a>
<a href="https://kataras.rocket.chat/channel/go-errors"><img src="https://img.shields.io/badge/%20community-chat-00BCD4.svg?style=flat-square" alt="Build Status"></a>
<a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square" alt="Built with GoLang"></a>
<a href="#"><img src="https://img.shields.io/badge/platform-Any--OS-yellow.svg?style=flat-square" alt="Platforms"></a>


This package provides a way to initialize possible errors and handle them with ease.

Installation
------------
The only requirement is the [Go Programming Language](https://golang.org/dl).

```bash
$ go get -u github.com/kataras/go-errors
```


Docs
------------

## New

`New` receives a message format and creates a new Error. Message format, which is created with .New, is never changes.

```go
import "github.com/kataras/go-errors"

var errUserAlreadyJoined = errors.New("User with username: %s was already joined in this room!")
```

## Format

`Format` is like `fmt.Sprintf` but for specific Error, returns a new error with the formatted message.


```go
import "github.com/kataras/go-errors"

var errUserAlreadyJoined = errors.New("User with username: %s was already joined in this room!")

func anything() error {
  return errUserAlreadyJoined.Format("myusername")
  // will return an error with message =
  // User with username: myusername was already joined in this room!
  //
}

```

## Append

`Append` and `AppendErr` adds a message to existing message and returns a new error.

```go
import "github.com/kataras/go-errors"

var errUserAlreadyJoined = errors.New("User with username: %s was already joined in this room!")

func anything() error {
  return errUserAlreadyJoined.Append("Please specify other room.").Format("myusername")
  // will return an error with message =
  // User with username: myusername was already joined in this room!
  // Please specify other room.
  //
}
```
```go
import "github.com/kataras/go-errors"

var errUserAlreadyJoined = errors.New("User with username: %s was already joined in this room!")
var errSpecifyOtherRoom  = errors.New("Please specify other room.")

func anything() error {
  return errUserAlreadyJoined.AppendErr(errSpecifyOtherRoom).Format("myusername")
  // will return an error with message =
  // User with username: myusername was already joined in this room!
  // Please specify other room.
  //
}

```

Use `AppendErr` with go standard error type

```go
import (
  "github.com/kataras/go-errors"
  "fmt"
)

var errUserAlreadyJoined = errors.New("User with username: %s was already joined in this room!")

func anything() error {
  err := fmt.Errorf("Please specify other room") // standard golang error

  return errUserAlreadyJoined.AppendErr(err).Format("myusername")
  // will return an error with message =
  // User with username: myusername was already joined in this room!
  // Please specify other room.
  //
}

```

FAQ
------------
Explore [these questions](https://github.com/kataras/go-errors/issues?go-errors=label%3Aquestion) or navigate to the [community chat][Chat].

Versioning
------------

Current: **v0.0.4**



People
------------
The author of go-errors is [@kataras](https://github.com/kataras).

Contributing
------------
If you are interested in contributing to the go-errors project, please make a PR.

License
------------

This project is licensed under the MIT License.

License can be found [here](LICENSE).

[Travis Widget]: https://img.shields.io/travis/kataras/go-errors.svg?style=flat-square
[Travis]: http://travis-ci.org/kataras/go-errors
[License Widget]: https://img.shields.io/badge/license-MIT%20%20License%20-E91E63.svg?style=flat-square
[License]: https://github.com/kataras/go-errors/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/release-v0.0.4-blue.svg?style=flat-square
[Release]: https://github.com/kataras/go-errors/releases
[Chat Widget]: https://img.shields.io/badge/community-chat-00BCD4.svg?style=flat-square
[Chat]: https://kataras.rocket.chat/channel/go-errors
[ChatMain]: https://kataras.rocket.chat/channel/go-errors
[ChatAlternative]: https://gitter.im/kataras/go-errors
[Report Widget]: https://img.shields.io/badge/report%20card-A%2B-F44336.svg?style=flat-square
[Report]: http://goreportcard.com/report/kataras/go-errors
[Documentation Widget]: https://img.shields.io/badge/documentation-reference-5272B4.svg?style=flat-square
[Documentation]: https://www.gitbook.com/book/kataras/go-errors/details
[Language Widget]: https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square
[Language]: http://golang.org
[Platform Widget]: https://img.shields.io/badge/platform-Any--OS-gray.svg?style=flat-square
