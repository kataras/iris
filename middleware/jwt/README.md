# How to use JWT authentication with Iris

A JSON Web Token carries signed claims about a user, so a server can trust a request without keeping session state. This tutorial builds JWT authentication into an [Iris](https://www.iris-go.com/) application: a login endpoint that issues tokens, a protected endpoint that requires one, and a refresh endpoint that exchanges an expiring token for a new pair.

## Prerequisites

To follow this tutorial, you will need:

- Go 1.27 or higher installed on your machine
- A basic understanding of Go and Iris
- A text editor or IDE of your choice (e.g. [VS Code](https://code.visualstudio.com/))

## Creating a new Iris project

Start a new Iris project with `go mod`. Run the following commands:

```bash
$ mkdir iris-jwt
$ cd iris-jwt
$ go mod init iris-jwt
$ go get github.com/kataras/iris/v14@latest
```

This will create a new folder called `iris-jwt` with the following files:

```bash
iris-jwt
├── go.mod
└── go.sum
```

The `go.mod` file contains the module name and the dependency on Iris. The `go.sum` file contains the checksums of the dependencies.

Create a file called `main.go` in the same folder with enough code to start a server on port 8080. The JWT logic goes into this same file as the tutorial continues.

```go
package main

import "github.com/kataras/iris/v14"

func main() {
    // Create a new Iris application.
    app := iris.New()

    // Register a simple GET handler at the root path.
    app.Get("/", func(ctx iris.Context) {
        ctx.WriteString("Hello, world!")
    })

    // Start the server at http://localhost:8080.
    app.Listen(":8080")
}
```

To run the server, run the following command:

```bash
$ go run main.go
```

You should see something like this in your terminal:

```bash
Now listening on: http://localhost:8080
Application started. Press CTRL+C to shut down.
```

You can also visit http://localhost:8080 in your browser and see the message "Hello, world!".

## Defining the user model and the JWT claims

Now, we will define a simple user model and a custom JWT claims struct in our `main.go` file. The user model will represent the data of our users, such as username and password. The JWT claims struct will contain the information that we want to store in our JWT tokens, such as user ID and expiration time.

We will also define some constants for our secret keys and token expiration durations.

```go
package main

import (
    "time"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/jwt"
)

// User is a simple user model.
type User struct {
    ID       int64  `json:"id"`
    Username string `json:"username"`
    Password string `json:"password"`
}

// Claims is a custom JWT claims struct.
type Claims struct {
    jwt.Claims // embeds standard claims iat, exp and sub.
    UserID     int64  `json:"user_id"`
}

// Define some constants for our secret keys and token expiration durations.
const (
    accessSecret  = "my-access-secret"
    refreshSecret = "my-refresh-secret"
    accessExpire  = 15 * time.Minute
    refreshExpire = 24 * time.Hour
)
```

## Creating a signer and a verifier for JWT

The `middleware/jwt` package gives you a signer and a verifier. The signer generates and signs tokens from your secret key and claims; the verifier checks the token a client sends and rejects anything that does not hold up.

Create two signers, one for the access token and one for the refresh token, and one verifier for the access token. Encrypting the tokens on top of signing them keeps the claims unreadable to whoever holds the token.

```go
package main

import (
    "time"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/jwt"
)

// ...

// Create a signer for the access token.
accessSigner := jwt.NewSigner(jwt.HS256, accessSecret, accessExpire).
    // Use encryption for the access token.
    WithEncryption([]byte("my-access-encryption-key"), nil)

// Create a signer for the refresh token.
refreshSigner := jwt.NewSigner(jwt.HS256, refreshSecret, refreshExpire).
    // Use encryption for the refresh token.
    WithEncryption([]byte("my-refresh-encryption-key"), nil)

// Create a verifier for the access token.
accessVerifier := jwt.NewVerifier(jwt.HS256, accessSecret).
    // Use decryption for the access token.
    WithDecryption([]byte("my-access-encryption-key"), nil)
    // Use a blocklist to revoke tokens.
    // .WithDefaultBlocklist()
```

## Creating some mock users and a login handler

To keep the example short, the users live in a map. A real application would read them from a database.

The login handler takes a username and password from the request, checks them against that map, and on a match signs an access token and a refresh token and returns both.

```go
package main

import (
    "time"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/jwt"
    "github.com/kataras/iris/v14/rest/errors"
)

// ...

// Create some mock users in a map.
users := map[string]User{
    "alice": {ID: 1, Username: "alice", Password: "1234"},
    "bob":   {ID: 2, Username: "bob", Password: "5678"},
}

// Create a login handler that will generate JWT tokens for authenticated users.
loginHandler := func(ctx iris.Context) {
    // Get the username and password from the request body.
    user, ok := ctx.BindJSON[User]()
    if !ok {
        return // ctx.BindJSON wrote the 400 itself.
    }

    // Check if the username and password match with one of our mock users.
    u, found := users[user.Username]
    if !found || u.Password != user.Password {
        errors.Unauthenticated.Message(ctx, "invalid credentials")
        return
    }

    // Generate an access token with the user ID as the subject claim.
    accessClaims := Claims{
        Claims: jwt.Claims{Subject: u.Username},
        UserID: u.ID,
    }
    accessToken, err := accessSigner.Sign(accessClaims)
    if err != nil {
        errors.HandleError(ctx, err)
        return
    }

    // Generate a refresh token with the user ID as the subject claim.
    refreshClaims := Claims{
        Claims: jwt.Claims{Subject: u.Username},
        UserID: u.ID,
    }
    refreshToken, err := refreshSigner.Sign(refreshClaims)
    if err != nil {
        errors.HandleError(ctx, err)
        return
    }

    /* OR
    tokenPair, err := refreshSigner.NewTokenPair(accessClaims, refreshClaims, refreshExpire)
    // [handle err...]
    ctx.JSON(tokenPair)
    */

    // Send the tokens to the client as JSON.
    ctx.OK(iris.Map{
        "access_token":  string(accessToken),
        "refresh_token": string(refreshToken),
        "expires_in":    int64(accessExpire.Seconds()),
    }, nil)
}
```

## Creating a protected API handler

A protected endpoint puts the access verifier in front of the handler as middleware. A request without a valid token never reaches the handler. When the token is valid, the handler reads the user ID out of the claims and answers with it.

```go
package main

import (
    "time"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/jwt"
    "github.com/kataras/iris/v14/rest/errors"
)

// ...

// Create a protected API handler that will only allow authorized users to access it.
protectedHandler := func(ctx iris.Context) {
    // Get the verified token from the context.
    token := jwt.GetVerifiedToken(ctx) // important step.

    // Get the custom claims from the token.
    var claims Claims
    if err := token.Claims(&claims); err != nil { // important step.
        errors.HandleError(ctx, err)
        return
    }

    // Get the user ID from the claims.
    userID := claims.UserID

    // Send the user ID to the client as JSON.
    // This is just an example, you can do whatever you want here.
    ctx.OK(iris.Map{
        "user_id": userID,
    }, nil)
}
```

## Creating a refresh handler

Access tokens are short-lived on purpose, so a client needs a way to get a fresh one without asking the user to log in again. The refresh handler verifies the refresh token and, if it holds up, signs a new access and refresh token for the same user.

```go
package main

import (
    "time"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/jwt"
    "github.com/kataras/iris/v14/rest/errors"
)

// ...

// Create a refresh handler that will allow users to refresh their access tokens using their refresh tokens.
refreshHandler := func(ctx iris.Context) {
    // Get the verified token from the context.
    token := jwt.GetVerifiedToken(ctx)

    // Get the custom claims from the token.
    var claims Claims
    if err := token.Claims(&claims); err != nil {
        errors.HandleError(ctx, err)
        return
    }

    // Get the user ID and username from the claims.
    userID := claims.UserID
    username := claims.Subject

    // Generate a new access token with the same user ID and username as the subject claim.
    accessClaims := Claims{
        Claims: jwt.Claims{Subject: username},
        UserID: userID,
    }
    accessToken, err := accessSigner.Sign(accessClaims)
    if err != nil {
        errors.HandleError(ctx, err)
        return
    }

    // Generate a new refresh token with the same user ID and username as the subject claim.
    refreshClaims := Claims{
        Claims: jwt.Claims{Subject: username},
        UserID: userID,
    }
    refreshToken, err := refreshSigner.Sign(refreshClaims)
    if err != nil {
        errors.HandleError(ctx, err)
        return
    }

    // Send the new tokens to the client as JSON.
    ctx.OK(iris.Map{
        "access_token":  string(accessToken),
        "refresh_token": string(refreshToken),
        "expires_in":    int64(accessExpire.Seconds()),
    }, nil)
}
```

## Registering the handlers and testing the application

Now that we have created all our handlers, we can register them with our Iris application and test our application. We will use the `app.Post` method to register our login, protected, and refresh handlers with different paths. We will also use the `accessVerifier.Verify` method to register our verifiers as middlewares for the rest of the protected handlers.

```go
package main

import (
    "time"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/jwt"
)

// ...

func main() {
    // Create a new Iris application.
    app := iris.New()

    // Register our login handler at /login path.
    app.Post("/login", loginHandler)

    // Register a single protected handler at /protected path.
    // This handler verifies the request manually as we've seen above.
    app.Post("/protected", protectedHandler)

    // Register our refresh handler at /refresh path with refresh signer.
    app.Post("/refresh", refreshHandler)

    // You can also register the pre-defined jwt middleware for all protected routes
    // which performs verification automatically and set the custom Claims to the Context
    // for next handlers to use through: claims, ok := jwt.Get[*Claims](ctx).
    app.Use(accessVerifier.Verify(func() any {
		return new(Claims)
	}))

    // [more routes...]

    // Start the server at http://localhost:8080.
    app.Listen(":8080")
}
```

To test our application, we can use a tool like curl or Postman to send HTTP requests to our server. Here are some examples of how to do that:

- To login as Alice and get the access token and the refresh token, we can send a POST request to http://localhost:8080/login with the following JSON body:

```json
{
    "username": "alice",
    "password": "1234"
}
```

We should get a response like this:

```json
{
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzYwNjQ3MzEsImlhdCI6MTYzNjA2MzIzMSwic3ViIjoiYWxpY2UiLCJ1c2VySWQiOjF9.8f7a7b8f7a7b8f7a7b8f7a7b8f7a7b8f7a7b8f7a7b8f7a7b8f7a7b8f",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzYxNTAzMzEsImlhdCI6MTYzNjA2MzIzMSwic3ViIjoiYWxpY2UiLCJ1c2VySWQiOjF9.9f8a8b9f8a8b9f8a8b9f8a8b9f8a8b9f8a8b9f8a8b9f8a8b9f8a8b9f",
    "expires_in": 900
}
```

- To access the protected API endpoint with the access token, we can send a POST request to http://localhost:8080/protected with the following header:

```bash
Authorization: Bearer <access-token>
```

We should get a response like this:

```json
{
    "user_id": 1
}
```

- To refresh the access token with the refresh token, we can send a POST request to http://localhost:8080/refresh with the following header:

```bash
Authorization: Bearer <refresh-token>
```

We should get a response like this:

```json
{
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzYwNjQ4NTMsImlhdCI6MTYzNjA2MzM1Mywic3ViIjoiYWxpY2UiLCJ1c2VySWQiOjF9.9g7a7c9g7a7c9g7a7c9g7a7c9g7a7c9g7a7c9g7a7c9g7a7c9g7a7c9g",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzYxNTA0NTMsImlhdCI6MTYzNjA2MzM1Mywic3ViIjoiYWxpY2UiLCJ1c2VySWQiOjF9.aG8aBbaG8aBbaG8aBbaG8aBbaG8aBbaG8aBbaG8aBbaG8aBbaG8aBbaG",
    "expires_in": 900
}
```

## Token Extraction Defaults (v14)

As of v14, `jwt.NewVerifier` reads the token from the `Authorization` header only. The `?token=` query-parameter extractor is no longer enabled by default, because tokens in URLs leak into access logs, proxy logs, browser history and Referer headers. Opt back in only when a header cannot be set (for example browser file downloads):

```go
verifier := jwt.NewVerifier(jwt.HS256, key).WithQueryToken()
```
