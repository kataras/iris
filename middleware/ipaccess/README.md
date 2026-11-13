# IP Access Control Middleware

This middleware decides who reaches your application based on the client's IP address. Allowed and blocked patterns can be exact addresses, CIDR ranges, or regular expressions.

The client's IP address is the one `ctx.RemoteAddr()` returns: the address of the connection or, when `Options.Context.RemoteAddrHeaders` names a header, the address a proxy wrote in it. Read [Client Address and Proxies](#client-address-and-proxies) before you put the middleware behind a proxy.

## Installation

To use the IP Access Control middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/ipaccess"
```

## Usage

### IP Whitelisting and Blacklisting

The IP Access Control middleware allows access only to the specified IP patterns and denies access to the specified blocked IP patterns. Requests from IP addresses that do not match any of the allowed patterns or match any of the blocked patterns will be denied with a `403 Forbidden` status. With no allowed patterns at all, every address that is not blocked gets through. The address that is matched is `ctx.RemoteAddr()`.

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

## Client Address and Proxies

The middleware decides on `ctx.RemoteAddr()`. With no `Options.Context.RemoteAddrHeaders`, which is the default, that is the address of the connection, and no header is read whatever the client sends.

Behind a reverse proxy the connection is the proxy's, so you name the header the proxy writes, `X-Forwarded-For` for example, in `RemoteAddrHeaders`. A proxy appends the address it got the request from, so the header is read from the right: the first public address is the client's and private addresses are skipped as your own proxies. When no header gives a public address, the address of the connection is matched. A client on the internet that sends `X-Forwarded-For: 203.0.113.7` does not pass an allowlist of `203.0.113.0/24` that way, because the proxy appends the client's own public address to the right of it.

Iris has no list of trusted proxies yet, so the address is only as good as the proxy in front of the application. Check these before you rely on it:

- The server has to be reachable through the proxy only. A client that connects to it directly writes the whole header, and with it the address that is matched.
- Name only headers your proxy writes. One that it passes on as the client sent it is the client's, and it is read when the headers named before it give no public address.
- A client inside your network is appended as a private address, which is skipped like one of your proxies. The entry on its left is read then, and the client wrote that one: `X-Forwarded-For: 203.0.113.7, 10.0.0.5` passes an allowlist of `203.0.113.0/24`. A proxy that overwrites the header closes this.
- A private address in the header never matches. The patterns at the top of this page are private ranges: when an internal proxy reports a client as `192.168.1.20`, `192.168.1.0/24` does not match it, and the address of the connection, the proxy, is matched in its place. Put the proxy's address in the patterns. Or set `RemoteAddrHeadersForce`, which takes the first address of a header that has no public one (an entry the client can write), or an empty, non-nil `RemoteAddrPrivateSubnets`, which takes the right-most one.
- A proxy with a public address in front of your own, a CDN for example, is the address that is matched, and the client it got the request from is not read. Add the CDN's IPv4 range to `RemoteAddrPrivateSubnets` to have it skipped.

The doc comment of `ipaccess.New` lists the remaining cases, and [SECURITY.md](../../SECURITY.md) describes how `RemoteAddr()` reads a header.

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
