# API Key Middleware for Iris

This middleware guards routes with a shared secret that the caller presents on every request. It is how services, cron jobs, webhook senders and CLI tools authenticate to an API when there is no user to sign in and no session to keep.

A key arrives on a header (`Authorization: Bearer <key>` by default) or, when you enable it, on a query parameter. It is compared against the configured keys in constant time, or resolved through a `Lookup` function when the keys live in a database. The principal that matched is stored on the Context, so a handler and the access log can name the caller.

The default is to deny. A middleware with no keys and no `Lookup` rejects every request instead of quietly letting everyone through, and logs a warning so the missing configuration is visible.

## Installation

```go
import "github.com/kataras/iris/v14/middleware/apikey"
```

## Usage

```go
package main

import (
	"os"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/apikey"
)

func main() {
	app := iris.New()

	guard := apikey.New(apikey.Options{
		Keys: map[string]string{
			"reporting": os.Getenv("REPORTING_API_KEY"),
			"ops":       os.Getenv("OPS_API_KEY"),
		},
	})

	app.Get("/orders", guard, func(ctx iris.Context) {
		principal, _ := apikey.Get(ctx) // "reporting" or "ops".
		ctx.JSON(iris.Map{"called_by": principal})
	})

	app.Listen(":8080")
}
```

The caller sends:

```sh
curl -H "Authorization: Bearer $REPORTING_API_KEY" http://localhost:8080/orders
```

## Options

| Option | Type | Default | Description |
| ------ | ---- | ------- | ----------- |
| `Header` | `string` | `"Authorization"` | The request header that carries the key. The `Authorization` header is redacted by default in proxy, CDN and log-scrubber configurations, which a custom header is not, so override it only for a caller that cannot set it. |
| `Scheme` | `string` | `"Bearer"` | The auth-scheme prefix on the header value, matched case-insensitively per RFC 7235. Set it to `apikey.NoScheme` (`"-"`) to accept a bare value, which is what a custom header such as `X-API-Key` carries. |
| `Query` | `string` | `""` (off) | A query parameter to read the key from when the header is absent. A key in a URL ends up in browser history, proxy logs, referrers and, when `middleware/accesslog` is registered, in your own access log, which records every query argument. Its `Redact` hook covers request and response bodies only, so drop or mask the parameter in a custom accesslog formatter (`ac.SetFormatter`) before you enable this, and enable it only for a caller that cannot set a header at all. |
| `Keys` | `map[string]string` | `nil` | Principal name to secret, or to the secret's digest when `Hash` is set. One entry per consumer, so a leak can be traced and one key revoked without breaking the others. An entry with an empty secret is dropped. Give every consumer its own secret: two principals sharing one authenticate each other's requests, and which of the two is reported then depends on map iteration order. |
| `Hash` | `func(string) string` | `nil` | Applied to the presented key before it is compared, so `Keys` hold `Hash(secret)` and the configuration never holds a usable credential. Use `apikey.SHA256`. |
| `Lookup` | `func(ctx iris.Context, key string) (principal string, ok bool, err error)` | `nil` | Resolves a key that none of the `Keys` matched, for keys that live in a database. An error is a 500, not a 401: a store that is down must not read as an invalid key. With `Hash` set it receives `Hash(key)`, so a store of hashed keys must not hash again. |
| `AllowAnonymous` | `bool` | `false` | Lets every request through when there is neither a key nor a `Lookup`, for local development. Ignored as soon as one of them exists. Never true in production. |
| `ErrorHandler` | `func(ctx iris.Context, err error)` | `nil` | Renders a rejection. The default renders the `rest/errors` `Unauthenticated` envelope with the reason as its message. |

## Behaviour

| Request | Configuration | Result |
| ------- | ------------- | ------ |
| A key that matches one of `Keys` | any | The next handler runs, `apikey.Get` returns that key's principal |
| A key that matches nothing, `Lookup` resolves it | `Lookup` set | The next handler runs with the principal `Lookup` reported |
| A key that matches nothing | any | 401, `ErrInvalid` |
| No key, or a value the scheme does not fit | any | 401, `ErrMissing` |
| Two `Authorization` headers | any | 401: two credentials on one request is not a valid presentation of either |
| A key on the query parameter | `Query` set | The next handler runs; the query carries a bare key, never a scheme prefix |
| Any request | no `Keys`, no `Lookup`, `AllowAnonymous` false | 401, and a warning logged on the first request |
| Any request | no `Keys`, no `Lookup`, `AllowAnonymous` true | The next handler runs, and a loud warning is logged on the first request |
| Any request | `Lookup` returns an error | 500 through the `rest/errors` pipeline |

The comparison runs over every configured key with no early exit and no branch on the result. Both sides are reduced to a 32-byte SHA-256 digest first, so every comparison is over the same number of bytes: `subtle.ConstantTimeCompare` returns immediately when two lengths differ, and comparing the raw values would make a presented key whose length matches a configured one measurably slower than one that matches none. Neither which key nor how long it is can be read out of the time a rejection takes.

## The principal and the user

On success the middleware stores the principal name and, when nothing else has set a user yet, a `context.SimpleUser` whose `ID` and `Username` are that principal. Code written against `ctx.User()` therefore sees the caller without knowing where the credential came from.

```go
principal, ok := apikey.Get(ctx)  // "reporting", true
id, _ := ctx.User().GetID()       // "reporting"
```

An existing user is never replaced, so a route that already ran `auth` or `basicauth` keeps its own.

## Hashed keys

With `Hash` set, `Keys` hold the digest instead of the secret. The configuration file, the environment variable and the deployment manifest then never hold a credential that works.

```go
apikey.New(apikey.Options{
	Hash: apikey.SHA256,
	Keys: map[string]string{
		// apikey.SHA256("the-secret-you-handed-out")
		"reporting": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
	},
})
```

A shared secret is a high-entropy random string, not a password, so a plain digest is the right hash here: there is nothing to brute-force.

## Keys in a database

`Lookup` is consulted for a key that none of the static `Keys` matched, so a handful of bootstrap keys in the configuration and the rest in a table work together.

```go
apikey.New(apikey.Options{
	Lookup: func(ctx iris.Context, key string) (string, bool, error) {
		row, err := tokens.FindByHash(ctx, apikey.SHA256(key))
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil // not a match, a 401
		}
		if err != nil {
			return "", false, err // the store is down, a 500
		}

		return row.Name, !row.RevokedAt.Valid, nil
	},
})
```

Hash the key before you use it as a lookup value, so the table holds digests and not credentials.

## Telling the two failures apart

`ErrMissing` and `ErrInvalid` are the reasons wrapped into the rejection, for an `ErrorHandler` that treats them differently.

```go
apikey.New(apikey.Options{
	Keys: keys,
	ErrorHandler: func(ctx iris.Context, err error) {
		if errors.Is(err, apikey.ErrInvalid) {
			metrics.BadAPIKey.Inc() // someone is presenting a key that does not work
		}

		errors.Unauthenticated.Err(ctx, err)
	},
})
```
