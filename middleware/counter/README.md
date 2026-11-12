# Counter Middleware for Iris

This middleware counts things: requests served, users connected, or any other number you want to keep. Increments and decrements are atomic, the counter works with any of Go's numeric types, and you can subscribe to a channel to be told when the value changes or crosses a limit.

## Features of Iris Counter Middleware

The Iris counter middleware offers a variety of features to handle numerical counting efficiently. These features include:

1. **Atomic Operations**: Ensures thread-safe increment and decrement operations.
2. **Support for Various Numerical Types**: Supports all basic numerical types including integers and floats.
3. **Notifications**: Provides a channel to listen for counter changes.
4. **Global Counter**: A shared counter instance that can be used across multiple Iris applications.
5. **Customizable Limits**: Allows setting minimum and maximum values for the counter.

## Installation

To use the Iris counter middleware, you need to have the Iris framework installed. You can install Iris using the following command:

```sh
go get github.com/kataras/iris/v14@latest
```

## Usage

### Basic Setup

To use the counter middleware, you need to import the necessary packages and configure the middleware in your Iris application. Here is a basic example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/counter"
)

func main() {
	app := iris.New()
	c := counter.NewCounter[uint64]()
	app.Use(counter.RequestCounter(c))

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	app.Listen(":8080")
}
```

### Incrementing and Decrementing the Counter

The counter middleware provides methods to increment and decrement the counter. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/counter"
)

func main() {
	app := iris.New()
	c := counter.NewCounter[uint64]()
	app.Use(counter.RequestCounter(c))

	app.Get("/", func(ctx iris.Context) {
		currentCount := c.Get()
		ctx.Writef("Current Count: %d", currentCount)
	})

	app.Listen(":8080")
}
```

### Using Notifications

The counter middleware provides a channel to listen for counter changes. Here is an example:

```go
package main

import (
	"log"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/counter"
)

func main() {
	app := iris.New()
	c := counter.NewCounter[uint64]()
	app.Use(counter.RequestCounter(c))

	go func() {
		for notif := range c.Notifications() {
			log.Printf("Counter value changed from %d to %d", notif.OldValue, notif.NewValue)
		}
	}()

	app.Get("/", func(ctx iris.Context) {
		currentCount := c.Get()
		ctx.Writef("Current Count: %d", currentCount)
	})

	app.Listen(":8080")
}
```

### Using the Global Counter

The Iris counter middleware provides a global counter instance that can be used across multiple Iris applications. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/counter"
)

func main() {
	app := iris.New()
	app.Use(counter.RequestCounter(counter.GlobalCounter))

	app.Get("/", func(ctx iris.Context) {
		currentCount := counter.GlobalCounter.Get()
		ctx.Writef("Current Count: %d", currentCount)
	})

	app.Listen(":8080")
}
```

### Customizing Limits

You can set custom minimum and maximum values for the counter. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/counter"
)

func main() {
	app := iris.New()
	c := counter.NewCounter[int]()
	c.Min = 0
	c.Max = 100

	app.Use(counter.RequestCounter(c))

	app.Get("/", func(ctx iris.Context) {
		currentCount := c.Get()
		ctx.Writef("Current Count: %d", currentCount)
	})

	app.Listen(":8080")
}
```
