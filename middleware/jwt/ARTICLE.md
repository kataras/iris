# How to use JWT authentication with Iris

In this tutorial, we will learn how to use JWT (JSON Web Token) authentication with [Iris](https://www.iris-go.com/), a fast and simple web framework for Go. JWT is a standard for securely transmitting information between parties as a JSON object. It can be used to authenticate users and protect API endpoints from unauthorized access.

## Prerequisites

To follow this tutorial, you will need:

- Go 1.20 or higher installed on your machine
- A basic understanding of Go and Iris
- A text editor or IDE of your choice (e.g. [VS Code](https://code.visualstudio.com/))

## Creating a new Iris project

First, we will create a new Iris project using the `go mod` command. To do that, run the following commands:

```bash
$ mkdir iris-jwt
$ cd iris-jwt
$ go mod init iris-jwt
$ go get github.com/kataras/iris/v12@latest
```

This will create a new folder called `iris-jwt` with the following files:

```bash
iris-jwt
├── go.mod
└── go.sum
```

The `go.mod` file contains the module name and the dependency on Iris. The `go.sum` file contains the checksums of the dependencies.

Next, we will create a file called `main.go` in the same folder and write some basic code to start an Iris server on port 8080. We will modify this file later to add our JWT logic.

```go
package main

import "github.com/kataras/iris/v12"

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

    "github.com/kataras/iris/v12"
    "github.com/kataras/iris/v12/middleware/jwt"
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

Next, we will create a signer and a verifier for JWT using the Iris middleware/jwt package. The signer will be used to generate and sign JWT tokens with our secret keys and claims. The verifier will be used to verify and validate JWT tokens from the client requests.

We will create two signers and one verifiers, one pair for the access token and one signer for the refresh token. We will also use encryption for our tokens to add an extra layer of security.

```go
package main

import (
    "time"

    "github.com/kataras/iris/v12"
    "github.com/kataras/iris/v12/middleware/jwt"
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

For the sake of simplicity, we will create some mock users in a map and use them to simulate authentication. In a real application, you would use a database or another storage system to store and retrieve your users.

We will also create a login handler that will take a username and password from the client request, check if they match with one of our mock users, and if so, generate an access token and a refresh token for that user and send them back to the client.

```go
package main

import (
    "time"

    "github.com/kataras/iris/v12"
    "github.com/kataras/iris/v12/middleware/jwt"
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
    var user User
    if err := ctx.ReadJSON(&user); err != nil {
        ctx.StopWithError(iris.StatusBadRequest, err)
        return
    }

    // Check if the username and password match with one of our mock users.
    if u, ok := users[user.Username]; !ok || u.Password != user.Password {
        ctx.StopWithStatus(iris.StatusUnauthorized)
        return
    }

    // Generate an access token with the user ID as the subject claim.
    accessClaims := Claims{
        Claims: jwt.Claims{Subject: u.Username},
        UserID: u.ID,
    }
    accessToken, err := accessSigner.Sign(accessClaims)
    if err != nil {
        ctx.StopWithError(iris.StatusInternalServerError, err)
        return
    }

    // Generate a refresh token with the user ID as the subject claim.
    refreshClaims := Claims{
        Claims: jwt.Claims{Subject: u.Username},
        UserID: u.ID,
    }
    refreshToken, err := refreshSigner.Sign(refreshClaims)
    if err != nil {
        ctx.StopWithError(iris.StatusInternalServerError, err)
        return
    }

    /* OR
    tokenPair, err := refreshSigner.NewTokenPair(accessClaims, refreshClaims, refreshExpire)
    // [handle err...]
    ctx.JSON(tokenPair)
    */

    // Send the tokens to the client as JSON.
    ctx.JSON(iris.Map{
        "access_token":  string(accessToken),
        "refresh_token": string(refreshToken),
        "expires_in":    int64(accessExpire.Seconds()),
    })
}
```

## Creating a protected API handler

Next, we will create a protected API handler that will only allow authorized users to access it. We will use the access verifier as a middleware to verify and validate the access token from the client request. If the token is valid, we will extract the user ID from it and send it back to the client as JSON.

```go
package main

import (
    "time"

    "github.com/kataras/iris/v12"
    "github.com/kataras/iris/v12/middleware/jwt"
)

// ...

// Create a protected API handler that will only allow authorized users to access it.
protectedHandler := func(ctx iris.Context) {
    // Get the verified token from the context.
    token := jwt.GetVerifiedToken(ctx) // important step.

    // Get the custom claims from the token.
    var claims Claims
    if err := token.Claims(&claims); err != nil { // important step.
        ctx.StopWithError(iris.StatusInternalServerError, err)
        return
    }

    // Get the user ID from the claims.
    userID := claims.UserID

    // Send the user ID to the client as JSON.
    // This is just an example, you can do whatever you want here.
    ctx.JSON(iris.Map{
        "user_id": userID,
    })
}
```

## Creating a refresh handler

Finally, we will create a refresh handler that will allow users to refresh their access tokens using their refresh tokens. We will use the refresh verifier as a middleware to verify and validate the refresh token from the client request. If the token is valid, we will generate a new access token and a new refresh token for the same user and send them back to the client.

```go
package main

import (
    "time"

    "github.com/kataras/iris/v12"
    "github.com/kataras/iris/v12/middleware/jwt"
)

// ...

// Create a refresh handler that will allow users to refresh their access tokens using their refresh tokens.
refreshHandler := func(ctx iris.Context) {
    // Get the verified token from the context.
    token := jwt.GetVerifiedToken(ctx)

    // Get the custom claims from the token.
    var claims Claims
    if err := token.Claims(&claims); err != nil {
        ctx.StopWithError(iris.StatusInternalServerError, err)
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
        ctx.StopWithError(iris.StatusInternalServerError, err)
        return
    }

    // Generate a new refresh token with the same user ID and username as the subject claim.
    refreshClaims := Claims{
        Claims: jwt.Claims{Subject: username},
        UserID: userID,
    }
    refreshToken, err := refreshSigner.Sign(refreshClaims)
    if err != nil {
        ctx.StopWithError(iris.StatusInternalServerError, err)
        return
    }

    // Send the new tokens to the client as JSON.
    ctx.JSON(iris.Map{
        "access_token":  string(accessToken),
        "refresh_token": string(refreshToken),
        "expires_in":    int64(accessExpire.Seconds()),
    })
}
```

## Registering the handlers and testing the application

Now that we have created all our handlers, we can register them with our Iris application and test our application. We will use the `app.Post` method to register our login, protected, and refresh handlers with different paths. We will also use the `accessVerifier.Verify` method to register our verifiers as middlewares for the rest of the protected handlers.

```go
package main

import (
    "time"

    "github.com/kataras/iris/v12"
    "github.com/kataras/iris/v12/middleware/jwt"
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
    // for next handlers to use through: claims := jwt.Get(ctx).(*Claims).
    app.Use(accessVerifier.Verify(func() interface{} {
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

## Conclusion

In this tutorial, we learned how to use JWT authentication with Iris, a fast and simple web framework for Go. We learned how to create a signer and a verifier for JWT, how to generate and validate JWT tokens, how to protect an API endpoint with JWT, and how to refresh JWT tokens. We also learned how to use the Iris middleware/jwt package, which provides a common Iris handler for JWT authentication.

You can find the complete code for this tutorial on GitHub: https://github.com/kataras/iris/tree/main/_examples/auth/jwt/tutorial and https://github.com/kataras/jwt.

If you want to learn more about Iris, you can visit its official website: https://www.iris-go.com/

If you want to learn more about JWT, you can visit its official website: https://jwt.io/

I hope you enjoyed this tutorial and found it useful. Thank you for reading.😊