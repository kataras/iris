# Iris Authentication

The [`auth`](https://github.com/kataras/iris/tree/main/auth) package signs users in and decides what they may do, using JSON Web Tokens. Your own struct is the user type, and your own function decides whether a set of credentials is valid.

## Features

- JWT-based authentication and authorization
- Customizable user types
- Support for access and refresh tokens
- Single Sign-On (SSO)
- Secure cookie handling
- Custom error handling
- Token invalidation
- Extensible providers and AWS Cognito support, and more

## Prerequisites

To use the [`auth`](https://github.com/kataras/iris/tree/main/auth) package, you need to have a basic understanding of JWT and how it works. You can refer to the following resources for more information:

- [JSON Web Token (JWT) - RFC 7519](https://tools.ietf.org/html/rfc7519)
- [JSON Web Token (JWT) - Wikipedia](https://en.wikipedia.org/wiki/JSON_Web_Token)

## Installation

To use the [`auth`](https://github.com/kataras/iris/tree/main/auth) package, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/auth"
```

## Configuration

### Single Sign-On (SSO)

Single Sign-On (SSO) is an authentication process that allows a user to access multiple applications with one set of login credentials. This means that once a user logs in to one application, they can access other related applications without having to log in again. SSO improves user experience by reducing the number of times a user has to log in and enhances security by centralizing authentication.

SSO only holds up if the token cannot be read or altered in the browser, which is what the cookie configuration is for. `Hash` authenticates the cookie value with HMAC, so tampering is detected, and `Block` encrypts it, so the contents cannot be read. Set both.

The [`auth`](https://github.com/kataras/iris/tree/main/auth) package requires a configuration file to set up the JWT keys and other settings. The configuration file can be in JSON or YAML format. Here is an example configuration file in YAML:

```yaml
Headers:
  - "Authorization"
  - "X-Authorization"
Cookie:
  Name: "iris_auth_cookie"
  Secure: false
  Hash: "D*G-KaPdSgUkXp2s5v8y/B?E(H+MbQeThWmYq3t6w9z$C&F)J@NcRfUjXn2r4u7x"
  Block: "VkYp3s6v9y$B&E)H@McQfTjWmZq4t7w!"
Keys:
  - ID: IRIS_AUTH_ACCESS
    Alg: EdDSA
    MaxAge: 2h
    Private: |+
      -----BEGIN PRIVATE KEY-----
      MC4CAQAwBQYDK2VwBCIEIFdZWoDdFny5SMnP9Fyfr8bafi/B527EVZh8JJjDTIFO
      -----END PRIVATE KEY-----
    Public: |+
      -----BEGIN PUBLIC KEY-----
      MCowBQYDK2VwAyEAzpgjKSr9E032DX+foiOxq1QDsbzjLxagTN+yVpGWZB4=
      -----END PUBLIC KEY-----
  - ID: IRIS_AUTH_REFRESH
    Alg: EdDSA
    MaxAge: 720h
    Private: |+
      -----BEGIN PRIVATE KEY-----
      MC4CAQAwBQYDK2VwBCIEIHJ1aoIjA2sRp5eqGjGR3/UMucrHbBdBv9p8uwfzZ1KZ
      -----END PRIVATE KEY-----
    Public: |+
      -----BEGIN PUBLIC KEY-----
      MCowBQYDK2VwAyEAsKKAr+kDtfAqwG7cZdoEAfh9jHt9W8qi9ur5AA1KQAQ=
      -----END PUBLIC KEY-----
    EncryptionKey: 0123456789abcdef0123456789abcdef
```

You can also use the `MustGenerateConfiguration` function to generate a configuration file with random keys and `ToYAML` method to convert it to YAML format, this is useful for fast start when you don't know how to generate the necessary keys.

Although it's not yet implemented in a lot of old backends. Iris implemented custom support for EdDSA (Ed25519) which is a modern and secure elliptic-curve signature scheme that offers several advantages over traditional algorithms like RSA and ECDSA.

```
Pros of Ed25519 for Identity Servers:
Security Benefits:

Strong cryptography: Ed25519 provides 128-bit security level with only 256-bit keys
Resistance to side-channel attacks: More resistant to timing attacks compared to RSA
No parameter generation issues: Unlike RSA, no need to worry about weak key generation
Performance Benefits:

Fast signature verification: Much faster than RSA (especially RSA-2048+)
Small signature size: Only 64 bytes vs 256 bytes for RSA-2048
Small key size: 32-byte public keys vs 256+ bytes for RSA
Low memory footprint: Important for high-throughput identity servers
Modern Standards:

RFC 8037 compliant: Standardized for JWT/JWE usage
Future-proof: Designed to resist quantum computing threats better than RSA
```

```go
package main

import (
    "os"

    "github.com/kataras/iris/v14/auth"
)

func main() {
    // Generate a new configuration file
    config := auth.MustGenerateConfiguration()
    yamlContents, err := config.ToYAML()
    if err != nil {
        panic(err)
    }

    // Save the configuration file
    err = os.WriteFile("./auth.yml", yamlContents, 0644)
    if err != nil {
        panic(err)
    }
}
```

## Usage

### Initializing the Auth Package

To initialize the [`auth`](https://github.com/kataras/iris/tree/main/auth) package, you need to load the configuration file and create a new [`auth`](https://github.com/kataras/iris/tree/main/auth) instance:

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/auth"
)

func main() {
    app := iris.New()

    // Load configuration from a file
    authConfig, err := auth.LoadConfiguration("./auth.yml")
    if err != nil {
        panic(err)
    }

    // Initialize a new auth instance
    s, err := auth.New[User](authConfig)
    if err != nil {
        panic(err)
    }

    // Add a custom provider
    s.AddProvider(NewProvider())

    // Register routes
    app.Post("/signin", s.SigninHandler)
    app.Post("/refresh", s.RefreshHandler)
    app.Get("/signout", s.SignoutHandler)
    app.Get("/signout-all", s.SignoutAllHandler)

    app.Listen(":8080")
}
```

### Custom User Type

You can define your own user type and implement the necessary methods for authentication and authorization:

```go
package main

type AccessRole uint16

const (
    InvalidAccessRole AccessRole = 1 << iota
    Read
    Write
    Delete
    Owner  = Read | Write | Delete
    Member = Read | Write
)

type User struct {
    ID    string     `json:"id"`
    Email string     `json:"email"`
    Role  AccessRole `json:"role"`
}

func (u User) GetID() string {
    return u.ID
}

// Allow reports whether this role carries every bit of "role",
// so an Owner is allowed wherever a Member is.
func (r AccessRole) Allow(role AccessRole) bool {
    return r&role == role
}
```

### Custom Provider

You can create a custom provider to handle user authentication and token validation:

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/kataras/iris/v14/auth"
)

type Provider struct {
    dataset []User
    invalidated    map[string]struct{}
    invalidatedAll map[string]int64
    mu             sync.RWMutex
}

func NewProvider() *Provider {
    return &Provider{
        dataset: []User{
            {ID: "id-1", Email: "kataras2006@hotmail.com", Role: Owner},
            {ID: "id-2", Email: "example@example.com", Role: Member},
        },
        invalidated:    make(map[string]struct{}),
        invalidatedAll: make(map[string]int64),
    }
}

func (p *Provider) Signin(ctx context.Context, username, password string) (User, error) {
    for _, user := range p.dataset {
        if user.Email == username {
            return user, nil
        }
    }
    return User{}, fmt.Errorf("user not found")
}

// Refresh is called by the RefreshHandler with the user the token was issued
// for; reload it from storage so a changed role or email reaches the new token.
func (p *Provider) Refresh(ctx context.Context, u User) (User, error) {
    for _, user := range p.dataset {
        if user.ID == u.ID {
            return user, nil
        }
    }
    return User{}, fmt.Errorf("user not found")
}

func (p *Provider) ValidateToken(ctx context.Context, standardClaims auth.StandardClaims, u User) error {
    p.mu.RLock()
    _, tokenBlocked := p.invalidated[standardClaims.ID]
    if !tokenBlocked && standardClaims.OriginID != "" {
        _, tokenBlocked = p.invalidated[standardClaims.OriginID]
    }
    p.mu.RUnlock()

    if tokenBlocked {
        return fmt.Errorf("token was invalidated")
    }

    p.mu.RLock()
    ts, oldUserBlocked := p.invalidatedAll[u.ID]
    p.mu.RUnlock()

    if oldUserBlocked && standardClaims.IssuedAt <= ts {
        return fmt.Errorf("token was invalidated")
    }

    return nil
}

func (p *Provider) InvalidateToken(ctx context.Context, standardClaims auth.StandardClaims, u User) error {
    p.mu.Lock()
    p.invalidated[standardClaims.ID] = struct{}{}
    p.mu.Unlock()
    return nil
}

func (p *Provider) InvalidateTokens(ctx context.Context, u User) error {
    p.mu.Lock()
    p.invalidatedAll[u.ID] = time.Now().Unix()
    p.mu.Unlock()
    return nil
}
```

### Custom Error Handler

You can customize the error handling by implementing the `ErrorHandler` interface and registering it with the `Auth` instance:

```go
package main

import (
    "github.com/kataras/iris/v14/context"
    "github.com/kataras/iris/v14/rest/errors"
)

type CustomErrorHandler struct{}

func (e *CustomErrorHandler) InvalidArgument(ctx *context.Context, err error) {
    errors.InvalidArgument.Detail(ctx, "unable to parse body", err.Error())
}

func (e *CustomErrorHandler) Unauthenticated(ctx *context.Context, err error) {
    errors.Unauthenticated.Err(ctx, err)
}
```

To register the custom error handler:

```go
s.SetErrorHandler(new(CustomErrorHandler))
```

### Token Invalidation

You can invalidate tokens using the `InvalidateToken` and `InvalidateTokens` methods of the provider:

```go
package main

// Invalidate a single token.
func (p *Provider) InvalidateToken(ctx context.Context, standardClaims auth.StandardClaims, u User) error {
    p.mu.Lock()
    p.invalidated[standardClaims.ID] = struct{}{}
    p.mu.Unlock()
    return nil
}

// Invalidate all tokens for this user.
func (p *Provider) InvalidateTokens(ctx context.Context, u User) error {
    p.mu.Lock()
    p.invalidatedAll[u.ID] = time.Now().Unix()
    p.mu.Unlock()
    return nil
}
```

### VerifyHandler

The `VerifyHandler` method verifies and sets the necessary information about the user and the verified token to the Iris Context. You can use it to protect routes and allow access based on user roles:

```go
package main

import (
    "context"
    "fmt"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/auth"
)

// allowRole rejects a verified user whose role does not include "role".
// AccessRole.Allow is defined next to the User type above.
func allowRole(role AccessRole) auth.VerifyUserFunc[User] {
    return func(ctx context.Context, u User) error {
        if !u.Role.Allow(role) {
            return fmt.Errorf("invalid role")
        }
        return nil
    }
}

func main() {
    app := iris.New()

    // Load configuration from a file
    authConfig, err := auth.LoadConfiguration("./auth.yml")
    if err != nil {
        panic(err)
    }

    // Initialize a new auth instance
    s, err := auth.New[User](authConfig)
    if err != nil {
        panic(err)
    }

    // Add a custom provider
    s.AddProvider(NewProvider())

    // Register routes
    app.Post("/signin", s.SigninHandler)
    app.Post("/refresh", s.RefreshHandler)
    app.Get("/signout", s.SignoutHandler)
    app.Get("/signout-all", s.SignoutAllHandler)

    // Protect routes with VerifyHandler
    app.Get("/member", s.VerifyHandler(allowRole(Member)), renderMemberPage(s))
    app.Get("/owner", s.VerifyHandler(allowRole(Owner)), renderOwnerPage(s))

    app.Listen(":8080")
}

func renderMemberPage(s *auth.Auth[User]) iris.Handler {
    return func(ctx iris.Context) {
        user := s.GetUser(ctx)
        ctx.Writef("Hello member: %s\n", user.Email)
    }
}

func renderOwnerPage(s *auth.Auth[User]) iris.Handler {
    return func(ctx iris.Context) {
        user := s.GetUser(ctx)
        ctx.Writef("Hello owner: %s\n", user.Email)
    }
}
```

## Example

Here is a complete example of how to use the [`auth`](https://github.com/kataras/iris/tree/main/auth) package in an Iris application:

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/auth"
)

func main() {
    app := iris.New()

    // Load configuration from a file
    authConfig, err := auth.LoadConfiguration("./auth.yml")
    if err != nil {
        panic(err)
    }

    // Initialize a new auth instance
    s, err := auth.New[User](authConfig)
    if err != nil {
        panic(err)
    }

    // Add a custom provider
    s.AddProvider(NewProvider())

    // Optionally set a custom error handler
    s.SetErrorHandler(new(CustomErrorHandler))

    // Register routes
    app.Post("/signin", s.SigninHandler)
    app.Post("/refresh", s.RefreshHandler)
    app.Get("/signout", s.SignoutHandler)
    app.Get("/signout-all", s.SignoutAllHandler)

    // Protect routes with VerifyHandler
    app.Get("/member", s.VerifyHandler(allowRole(Member)), renderMemberPage(s))
    app.Get("/owner", s.VerifyHandler(allowRole(Owner)), renderOwnerPage(s))

    app.Listen(":8080")
}

func renderMemberPage(s *auth.Auth[User]) iris.Handler {
    return func(ctx iris.Context) {
        user := s.GetUser(ctx)
        ctx.Writef("Hello member: %s\n", user.Email)
    }
}

func renderOwnerPage(s *auth.Auth[User]) iris.Handler {
    return func(ctx iris.Context) {
        user := s.GetUser(ctx)
        ctx.Writef("Hello owner: %s\n", user.Email)
    }
}
```
