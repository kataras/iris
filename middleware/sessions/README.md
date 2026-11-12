# Sessions

The Sessions middleware keeps per-visitor state between requests. Session data can live in a signed cookie, in PostgreSQL, or in Redis, and the same API covers flash messages, expiration and session id rotation.

For background on how the browser side works, see [RFC 6265: HTTP State Management Mechanism](https://tools.ietf.org/html/rfc6265) and the [Wikipedia article on sessions](https://en.wikipedia.org/wiki/Session_(computer_science)).

## Table of Contents

- Installation
- [Getting Started](#getting-started)
- [Defining Sessions](#defining-sessions)
  - [Basic Usage](#basic-usage)
  - [Session Options](#session-options)
  - [Flash Messages](#flash-messages)
- [Database Integrations](#database-integrations)
  - PostgreSQL
  - Redis
  - Cookie-Based
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- References

## Installation

To use the Sessions middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/sessions"
```

## Getting Started

Here's a basic example of using the Sessions middleware with Cookies as the session backend:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/sessions"
	"github.com/kataras/iris/v14/rest/errors"
)

// UserSession represents the actual session data for a user.
// You can define any structure that suits your application's needs.
type UserSession struct {
	ID       string
	Username string
}

var (
    // 64-character hash key and 32-character block key.
    // These are used to encrypt and decrypt the session data.
    // You should generate your own random keys for production use.
	hashKey  = []byte("D*G-KaPdSgUkXp2s5v8y/B?E(H+MbQeThWmYq3t6w9z$C&F)J@NcRfUjXn2r4u7x")
	blockKey = []byte("VkYp3s6v9y$B&E)H@McQfTjWmZq4t7w!")
)

func main() {
	app := iris.New()

	sessionOptions := sessions.NewOptions(hashKey, blockKey)
	sessionOptions.MaxAge = 1 * time.Hour

	sessionDB := sessions.NewCookieDatabase[UserSession](sessionOptions)
	// If you want to use the same cookie name for all sessions (most common usage)
	// then you can use the `Single` function as it doesn't require you to provide
	// a cookie name each time you want to retrieve the session.
	// sessionDB := sessions.Single(
	// 	sessions.NewCookieDatabase[UserSession](sessionOptions),
	// 	"my_session",
	// )
    // And then retrieve the session as follows:
    // session, err := sessionDB.Get(ctx)

	app.Get("/set", func(ctx iris.Context) {
		session, err := sessionDB.Get(ctx, "my_session")
		if err != nil {
			errors.HandleError(ctx, err)
			return
		}

		session.Set(UserSession{
			ID:       "1",
			Username: "kataras",
		})
		if err := session.Commit(ctx); err != nil {
			errors.HandleError(ctx, err)
			return
		}

		ctx.WriteString("session set")
	})

	app.Get("/get", func(ctx iris.Context) {
		session, err := sessionDB.Get(ctx, "my_session")
		if err != nil {
			errors.HandleError(ctx, err)
			return
		}

		ctx.JSON(session.Data)
	})

	app.Listen(":8080")
}
```

> The `sessions.Single` works with all session databases, not just the cookie-based one.

Using the `sessions.Handler` middleware and `sessions.Get` function is also possible when you want to manage a single session per request for routes that the Handler is registered to:

```go
sessionMiddleware := sessions.Handler(sessionDB, "my_session")
app.Use(sessionMiddleware)
// OR per-route: app.Get("/my-route", sessionMiddleware, myHandler)
// OR per-group: app.Router("/my-group", sessionMiddleware)
```

Using the `sessions.Get` function:

```go
app.Get("/get", func(ctx iris.Context) {
	session := sessions.Get[UserSession](ctx)
	ctx.JSON(session.Data)
})

// Sessions are automically saved on the end of each request because we registered the sessions Handler,
// but you can manually commit them as well.
app.Get("/set", func(ctx iris.Context) {
	session := sessions.Get[UserSession](ctx)
	session.Set(UserSession{ // or session.Data = UserSession{...}
		ID:       "1",
		Username: "kataras",
	})
})
```

> TIP: You can also use the `sessions.ReadOnlyHandler` middleware to avoid writing to the session on each request.

## Defining Sessions

### Basic Usage

To define a session, you need to create a session database and configure session options. The session database can be backed by PostgreSQL, Redis, or cookies.

### Session Options

The `Options` struct defines the configuration settings for a session. These settings determine how the session cookies are managed and stored.

`sessions.NewOptions` returns an `Options` **value**, and the database constructors take one by value: assign to its fields before you pass it, and a later change to your local copy does not reach the database. The `DataCodecs` slice is the one shared part, by design, because it holds the keys.

```go
type Options struct {
	Path        string
	Domain      string
	MaxAge      time.Duration
	SameSite    http.SameSite
	Partitioned bool 
	Secure      bool
}
```

### Flash Messages

Flash messages are temporary messages that are intended to be displayed only once. They are stored in the session and are removed after being retrieved.

```go
app.Get("/set-flash", func(ctx iris.Context) {
	session, err := sessionDB.Get(ctx, "my_session")
	if err != nil {
		errors.HandleError(ctx, err)
		return
	}

	session.SetFlash("notice", "This is a flash message!")
	if err := session.Commit(ctx); err != nil {
		errors.HandleError(ctx, err)
		return
	}

	ctx.WriteString("Flash message set")
})

app.Get("/get-flash", func(ctx iris.Context) {
	session, err := sessionDB.Get(ctx, "my_session")
	if err != nil {
		errors.HandleError(ctx, err)
		return
	}

	message, exists := session.GetFlash("notice")
	if !exists {
		ctx.WriteString("No flash message")
		return
	}

	ctx.WriteString("Flash message: " + message)
})
```

## Database Integrations

### PostgreSQL

To use PostgreSQL as the session backend, you need to create a PostgreSQL connection and initialize the session database.

```go
import "github.com/jackc/pgx/v5"
// [...]

connString := "postgres://username:password@localhost:5432/database_name"
conn, err := pgx.Connect(context.Background(), connString)
if err != nil {
	fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
	os.Exit(1)
}
defer conn.Close(context.Background())

sessionDB := sessions.NewPostgresDatabase[UserSession](sessions.DefaultTableName, conn, sessionOptions)
// Optionally, enable the shift expiration feature.
// It will update the session's expiration to NOW+Options.MaxAge on each commit.
sessionDB.ShiftExpiration(true)
// Start the cleanup scheduler, optionally.
errCh := sessionDB.StartCleanupScheduler(context.Background(), 4*time.Hour) // cleanup expired sessions every 4 hours.
go func() {
    for err := range errCh {
        fmt.Printf("error during cleanup: %v\n", err)
    }
}()
```

### Redis

To use Redis as the session backend, you need to create a Redis client and initialize the session database.

```go
import "github.com/redis/go-redis/v9"
// [...]

redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
defer redisClient.Close()

sessionDB := sessions.NewRedisDatabase[UserSession](redisClient, sessionOptions)
```

### Cookie-Based

To use cookies as the session backend, you need to initialize the session database with cookie options.

```go
sessionDB := sessions.NewCookieDatabase[UserSession](sessionOptions)
```

## Error Handling

Handle errors gracefully by checking for errors when retrieving or committing sessions. `errors.HandleError` (from `github.com/kataras/iris/v14/rest/errors`) is what the rest of the framework uses: a session failure is the server's, so the client gets the generic 500 envelope while the cause goes to the logger and the collector.

```go
session, err := sessionDB.Get(ctx, "my_session")
if err != nil {
	errors.HandleError(ctx, err)
	return
}

if err := session.Commit(ctx); err != nil {
	errors.HandleError(ctx, err)
	return
}
```

## Best Practices

- **Use Proper Session Options**: Configure session options to match your application's security and performance requirements.
- **Validate Session Data**: Always validate session data to prevent unexpected behavior.
- **Handle Errors Gracefully**: Provide meaningful error messages and use centralized error handling.
- **Use Flash Messages**: Keep one-off notifications and alerts in flash messages, not in the session proper.

## Secure Defaults (v14)

`sessions.NewOptions` now returns cookies that are `Secure` and `SameSite=Lax` by default. The `Secure` flag keeps the cookie off plaintext HTTP and `SameSite=Lax` mitigates CSRF. For local development over plain HTTP, opt out:

```go
opts := sessions.NewOptions(secret)
opts.Secure = false                   // allow over HTTP
opts.SameSite = http.SameSiteNoneMode // browsers require Secure for None
```

To defend against session fixation, rotate the session id after a privilege change (for example right after login) with `Session.RegenerateID`:

```go
session.Set(user)
if err := session.RegenerateID(ctx); err != nil {
    // handle error
}
```

`RegenerateID` is a no-op for cookie-based sessions, whose data lives in the signed cookie.
