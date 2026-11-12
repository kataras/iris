# ModRevision Middleware

Mount this middleware on an endpoint and it reports the server name, the environment, and the build that is running: revision, build time, and optionally the developer. It makes a good `/health` handler, and it answers the question "which commit is actually deployed" without a shell on the box.

## Features

- **Server Name Display**: Shows the server name.
- **Environment Display**: Shows the environment in which the server is running.
- **Build Information**: Displays the build time and revision.
- **Customizable Time Format**: Allows customization of the build time display format.
- **Developer Information**: Optionally displays the developer's name.
- **Cloud Detection**: Automatically detects if the application is running in a cloud environment and adjusts the output accordingly.

## Installation

To use the `modrevision` middleware, you need to import it in your Iris application:

```go
import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/modrevision"
)
```

## Usage

### Example Code

Here is an example of how to use the `modrevision` middleware with all options:

```go
package main

import (
    "time"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/modrevision"
)

func main() {
    app := iris.New()
    app.Get("/health", modrevision.New(modrevision.Options{
        ServerName:   "Iris Server",
        Env:          "development",
        Developer:    "kataras",
        UnixTime:     false,
        TimeLocation: time.FixedZone("Greece/Athens", 7200),
        // CloudDetection: true,
    }))

    app.Listen(":8080")
}
```

```sh
go build -o main .
./main
```

When you access the `/health` endpoint, you will see an output similar to the following:

```
Iris Server (development)

>>>> build
          revision        2a24c2dcbc8f0485521afcd5d7b675087b5073af
          buildtime       2024-11-17 08:19:24 +0200 Greece/Athens
          developer       kataras

OK
```
