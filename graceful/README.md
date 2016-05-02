## Package information

Enables graceful shutdown.

[Original repository](https://github.com/iris-contrib/graceful)


## Usage

```go
package main

import (
    "github.com/kataras/iris/graceful"
    "github.com/kataras/iris"
    "time"
)

func main() {
    api := iris.New()
    api.Get("/", func(c *iris.Context) {
        c.Write("Welcome to the home page!")
    })

    graceful.Run(":3001", time.Duration(10)*time.Second, api)
}

```
