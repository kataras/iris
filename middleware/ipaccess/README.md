# IP Access Control Middleware

This middleware decides who reaches your application based on the client's IP address. Allowed and blocked patterns can be exact addresses, CIDR ranges, or regular expressions.

## Installation

To use the IP Access Control middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/ipaccess"
```

## Usage

### IP Whitelisting and Blacklisting

The IP Access Control middleware allows access only to the specified IP patterns and denies access to the specified blocked IP patterns. Requests from IP addresses that do not match any of the allowed patterns or match any of the blocked patterns will be denied with a `403 Forbidden` status.

```go
allowedPatterns := []string{
    "127.0.0.1",          // Exact match.
    "192.168.1.0/24",     // CIDR notation.
    `^10\.0\.\d{1,3}\.\d{1,3}$`, // Regex pattern.
}
blockedPatterns := []string{
    "10.0.0.1",           // Exact match.
    `^172\.16\.\d{1,3}\.\d{1,3}$`, // Regex pattern.
}

ipAccessMiddleware, err := ipaccess.New(allowedPatterns, blockedPatterns)
if err != nil {
    log.Fatalf("Error creating IP access middleware: %v", err)
}
app.Use(ipAccessMiddleware)

// Or, when the patterns are literals or startup configuration and a bad one
// means a broken deployment, let it panic:
app.Use(ipaccess.Must(allowedPatterns, blockedPatterns))
```

## IP Pattern Matching

The middleware supports three types of IP pattern matching:

### Exact Match

An exact match pattern specifies a single IP address that must match exactly.

Example:
```go
"127.0.0.1"
```

### CIDR Notation

CIDR notation specifies a range of IP addresses.

Example:
```go
"192.168.1.0/24"
```

### Regular Expressions

A regular expression matches whatever an exact address and a CIDR range cannot express.

Example:
```go
`^10\.0\.\d{1,3}\.\d{1,3}$`
```

## Example

Here's a complete example of how to use the IP Access Control middleware in your Iris application:

```go
package main

import (
	"log"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/ipaccess"
)

func main() {
	app := iris.New()

	// Define allowed and blocked IP patterns.
	allowedPatterns := []string{
		"127.0.0.1",          // Exact match.
		"192.168.1.0/24",     // CIDR notation.
		`^10\.0\.\d{1,3}\.\d{1,3}$`, // Regex pattern.
	}
	blockedPatterns := []string{
		"10.0.0.1",           // Exact match.
		`^172\.16\.\d{1,3}\.\d{1,3}$`, // Regex pattern.
	}

	// Use the IP Access middleware.
	ipAccessMiddleware := ipaccess.Must(allowedPatterns, blockedPatterns)
	app.Use(ipAccessMiddleware)

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	app.Listen(":8080")
}
```
