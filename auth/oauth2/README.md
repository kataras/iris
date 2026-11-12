# OAuth2 Documentation

This package runs an OAuth 2.0 server on Iris, and covers the standard flows on the client side. What follows is how to configure the server, then how to drive each flow from a client.

## Server-Side Overview

Below is a full example of how to create and configure an OAuth2 server based on the attached code. We'll break it down into parts.

### 1. Defining the User Type and Provider

```go
package main

import (
	"context"
	"fmt"
	"sync"
	"uuid"

	"github.com/kataras/iris/v14/auth"
)

// User holds the user information.
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// Provider is a mock user data store.
type Provider struct {
	dataset []User
	mu      sync.RWMutex
}

func NewProvider() *Provider {
	return &Provider{
		dataset: []User{
			{
				ID:    uuid.New().String(),
				Email: "kataras2006@hotmail.com",
			},
			{
				ID:    uuid.New().String(),
				Email: "example@example.com",
			},
		},
	}
}

func (p *Provider) Signin(ctx context.Context, username, password string) (User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Simple example: check only username (email). Password not validated here.
	for _, user := range p.dataset {
		if user.Email == username {
			return user, nil
		}
	}

	return User{}, fmt.Errorf("user not found")
}

func (p *Provider) ValidateToken(ctx context.Context, claims auth.StandardClaims, u User) error {
	return nil
}

func (p *Provider) InvalidateToken(ctx context.Context, claims auth.StandardClaims, u User) error {
	return nil
}

func (p *Provider) InvalidateTokens(ctx context.Context, u User) error {
	return nil
}
```

1. We define a simple `User` struct to represent an authenticated user.
2. `Provider` implements the Iris auth interfaces to handle user sign-in and token validation.

### 2. Loading the Auth Configuration (auth.yml example)

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

These keys and cookies allow Iris to sign and store tokens securely.

### 3. Main Server Setup

```go
package main

import (
	"context"
	"fmt"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/auth/oauth2"
	"github.com/kataras/iris/v14/middleware/view"

	"github.com/kataras/blocks"
)

func main() {
	app := iris.New()

	// Load templates from ./views directory.
	templates := blocks.New("./views")
	if err := templates.Load(); err != nil {
		panic(err)
	}
	app.UseRouter(view.Handler(templates))

	// 1) Load Auth configuration from auth.yml
	authConfig, err := auth.LoadConfiguration("./auth.yml")
	if err != nil {
		panic(err)
	}
	s, err := auth.New[User](authConfig)
	if err != nil {
		panic(err)
	}
	s.AddProvider(NewProvider())

	// 2) Define routes:
	app.Get("/signin", renderSigninForm)
	app.Post("/signin", s.SigninHandler)
	app.Get("/consent", renderConsentForm())

	// 3) Create the OAuth2 server instance:
	// The client store keeps the registered clients. The in-memory one is
	// enough for this example; use oauth2.NewPostgresClientStore("clients", pool)
	// with a pgxpool.Pool to persist them.
	oauth2Server := oauth2.New(s,
		oauth2.WithClientStore(oauth2.NewMemoryClientStore()),
		oauth2.WithLoginPageURL("/signin"),
		oauth2.WithConsentPageURL("/consent"),
		oauth2.WithDeviceInterval(5),
	)

	// 4) Register a mock client:
	myClient, err := oauth2Server.RegisterClient(context.Background(), oauth2.ClientRegistration{
		Name:            "Example-Client-Name",
		Website:         "https://www.iris-go.com",
		LogoURL:         "https://www.iris-go.com/images/logo.png",
		RedirectURI:     "http://localhost:8081/callback",
		VerificationURI: "http://localhost:8081/device_verification",
		GrantTypes:      []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeDeviceCode, oauth2.GrantTypeRefreshToken, oauth2.GrantTypePassword},
		Scopes:          []string{"profile", "email:*"}, // allowed scopes and resource-action policy example.
		CodeMaxAge:      3600,   // max age for codes, in seconds.
		AccessMaxAge:    3600,   // max age for access tokens, in seconds.
		RefreshMaxAge:   604800, // max age for refresh tokens, in seconds.
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("ClientID:     %s\nClientSecret: %s\n", myClient.ID, myClient.Secret)

	// 5) Attach the OAuth2 routes under /oauth2:
	app.Router("/oauth2", oauth2Server)


	// Register the VerifyHandler to protect the routes.
	userRouter := app.Router("/user", s.VerifyHandler())                  
	// AllowScopes is a middleware that checks if the user has the required scopes.
	userRouter.Get("/profile", oauth2.AllowScopes("profile"), getProfile)
	// This will fire a 403 Forbidden if the user does not have both "profile" and "email" scopes
	// or it's not directly logged-in to the application.
	userRouter.Get("/email", oauth2.AllowScopes("profile", "email:read"), getEmail)

	app.Listen(":8080")
}

func getProfile(ctx iris.Context) {
	clientClaims, err := oauth2.GetClientClaims(ctx) // example of getting the oauth2 client claims.
	if err == nil {
		ctx.Writef("Request made by client: %#+v\n", clientClaims)
	}

	user := auth.GetUser[User](ctx) // example of getting the custom user claims.
	ctx.WriteString(user.ID)
}

func getEmail(ctx iris.Context) {
	clientClaims, err := oauth2.GetClientClaims(ctx)
	if err == nil {
		ctx.Writef("Request made by client: %#+v\n", clientClaims)
	}

	user := auth.GetUser[User](ctx)
	ctx.WriteString(user.Email)
}

func renderSigninForm(ctx iris.Context) {
	// Renders a basic sign-in form found in ./views/signin.html
	if err := view.Render(ctx, "signin", "main", iris.Map{"Title": "Signin Page"}); err != nil {
		ctx.HTML(fmt.Sprintf("<h3>%s</h3>", err.Error()))
	}
}

func renderConsentForm() iris.Handler {
	type Request struct {
		ClientID      string `url:"client_id"`
		ClientName    string `url:"client_name"`
		ClientWebsite string `url:"client_website"`
		ClientLogoURL string `url:"client_logo_url"`
		State         string `url:"state"`
		Scope         string `url:"scope"`
	}

	return func(ctx iris.Context) {
		var req Request
		if err := ctx.ReadQuery(&req); err != nil {
			ctx.HTML(fmt.Sprintf("<h3>Error reading query: %s</h3>", err.Error()))
			return
		}

		tmplData := struct {
			Request
			Title  string
			Scopes []string
		}{
			Request: req,
			Title:   "Consent Page",
			Scopes:  oauth2.SplitScopes(req.Scope),
		}

		if err := view.Render(ctx, "oauth2/consent", "main", tmplData); err != nil {
			ctx.HTML(fmt.Sprintf("<h3>%s</h3>", err.Error()))
		}
	}
}
```

#### Explanation:

1. We initialize a simple Iris application.  
2. We load templates and use the View middleware so we can render forms.  
3. We load the “auth.yml” to configure signing keys and cookies.  
4. We create an `auth.Auth[User]` instance, adding our custom `Provider`.  
5. We create a new `oauth2.Server[User]` to handle OAuth flows. We set custom `LoginPageURL` and `ConsentPageURL`. 
6. We register a mock client with `RegisterClient()`, you can do the same through the client's store as well.  
7. Finally, we map `oauth2Server` to the route prefix `/oauth2`. This automatically registers the necessary authorization/token/device endpoints; /oauth2/auth, /oauth2/token and /oauth2/device/code routes.

## Client-Side Usage

Below are quick examples for each OAuth2 flow using just the "golang.org/x/oauth2" and "net/http" packages.

### Authorization Code Flow

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

var authConfig = &oauth2.Config{
	ClientID:     "my-client-id",
	ClientSecret: "my-secret",
	Endpoint: oauth2.Endpoint{
		AuthURL:  "http://localhost:8080/oauth2/authorize",
		TokenURL: "http://localhost:8080/oauth2/token",
	},
	RedirectURL: "http://localhost:8081/callback",
	Scopes:      []string{"profile"},
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1) Redirect for authorization:
		url := authConfig.AuthCodeURL("random-state", oauth2.AccessTypeOffline)
		http.Redirect(w, r, url, http.StatusFound)
	})

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// 2) Exchange code for token:
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		token, err := authConfig.Exchange(context.Background(), code, oauth2.SetAuthURLParam("state", state))
		if err != nil {
			http.Error(w, "Token exchange error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Access Token: %s\n", token.AccessToken)
	})

	log.Println("Client listening on http://localhost:8081/")
	http.ListenAndServe(":8081", nil)
}
```

Steps to try it:

1. Run the OAuth2 server on :8080.  
2. Run this client on :8081.  
3. Go to http://localhost:8081. It redirects to the server’s `authorize` endpoint.  
4. Login and approve if necessary; you’ll return with a code to `/callback`, which is exchanged for a token.

### Device Code Flow

```go
package main

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/oauth2"
	devtools "golang.org/x/oauth2/device"
)

func main() {
	devConfig := &oauth2.Config{
		ClientID:     "my-client-id",
		ClientSecret: "my-secret",
		Endpoint: oauth2.Endpoint{
            AuthURL:       "http://localhost:8080/oauth2/authorize",
            TokenURL:      "http://localhost:8080/oauth2/token",
            DeviceAuthURL: "http://localhost:8080/oauth2/device/code",
		},
		Scopes: []string{"profile"},
	}

	deviceClient := devtools.Config{Config: *devConfig}
	ctx := context.Background()

	// 1) Request deviceauth
	deviceAuth, err := deviceClient.DeviceAuth(ctx)
	if err != nil {
		log.Fatalln("DeviceAuth Error:", err)
	}

	// Show user instructions
	fmt.Println("Go to:", deviceAuth.VerificationURI)
	fmt.Println("Enter Code:", deviceAuth.UserCode)

	// 2) Poll for token
	token, err := deviceClient.DeviceAccessToken(ctx, deviceAuth)
	if err != nil {
		log.Fatalln("DeviceAccessToken Error:", err)
	}

	fmt.Println("Access Token:", token.AccessToken)
}
```

### Password Grant Flow

```go
package main

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/oauth2"
    "golang.org/x/oauth2/authhandler" // or your own password credential helper
)

func main() {
	passConfig := &oauth2.Config{
		ClientID:     "my-client-id",
		ClientSecret: "my-secret",
		Endpoint: oauth2.Endpoint{
            AuthURL:       "http://localhost:8080/oauth2/authorize",
            TokenURL:      "http://localhost:8080/oauth2/token",
            DeviceAuthURL: "http://localhost:8080/oauth2/device/code",
		},
		Scopes: []string{"profile"},
	}

	ctx := context.Background()

	// Suppose a username/password is given:
	username := "kataras2006@hotmail.com"
	password := "admin_$@#@1239182"

	token, err := authhandler.PasswordCredentialsToken(ctx, passConfig, username, password)
	if err != nil {
		log.Fatalln("Failed to obtain token:", err)
	}

	fmt.Println("Access Token:", token.AccessToken)
}
```

### Refresh Token Flow

Refresh tokens are automatically handled by the [client's OAuth2](https://github.com/kataras/iris/tree/main/client) package option but here is how you can handle refreshes manually:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/oauth2"
)

func main() {
	config := &oauth2.Config{
		ClientID:     "my-client-id",
		ClientSecret: "my-secret",
		Endpoint: oauth2.Endpoint{
			TokenURL: "http://localhost:8080/oauth2/token",
		},
		Scopes: []string{"profile"},
	}

	ctx := context.Background()

	// Pretend we already have a token with a refresh token:
	oldToken := &oauth2.Token{
		AccessToken:  "the-old-access-token",
		RefreshToken: "the-refresh-token",
	}

	// 1) Create a TokenSource
	ts := config.TokenSource(ctx, oldToken)

	// 2) Request a new token (the library automatically handles refresh grant)
	newToken, err := ts.Token()
	if err != nil {
		log.Fatalln("Refresh Token Error:", err)
	}

	fmt.Println("New Access Token:", newToken.AccessToken)
}
```


## Advanced Policy Engine and Open Policy Agent (OPA)

**Open Policy Agent (OPA)** is an open-source, general-purpose policy engine. It moves the authorization decision out of your application code and into policies you write once and apply across services.

### Key Concepts

1. **Policy**: A set of rules written in Rego, OPA’s policy language, that define what actions are allowed or denied.
2. **Data**: JSON documents that provide context for policy decisions.
3. **Query**: A request to evaluate a policy against a set of input data.

### Benefits of Using OPA

- **Centralized Policy Management**: Manage all policies in one place.
- **Flexibility**: Write complex policies using Rego.
- **Decoupling**: Separate policy logic from application code.
- **Consistency**: Enforce policies consistently across different services.

### Example: Integrating OPA with OAuth2

#### 1. Define Policies in Rego

Create a policy file (e.g., `policy.rego`) that defines the rules for access control.

```rego
package authz

default allow = false

allow {
    input.client_id == "example-client"
    input.resource == "users"
    input.action == "read"
}

allow {
    input.client_id == "example-client"
    input.resource == "orders"
    input.action == "write"
}
```

#### 2. Load Policies into OPA

Start OPA with the policy file.

```sh
opa run --server --addr :8181 policy.rego
```


#### 3. Initialize the OPA Client

```go
import "github.com/kataras/iris/v14/auth/oauth2/opa"
```

```go
opaClient := opa.NewClient("http://localhost:8181/v1/data/authz/allow")
```

#### 4. Usage in Routes

Use the OPA Client to define new policies and the `AllowPolicies` middleware in your routes (or group of routes) to enforce policies.

```go
app.Get("/api/users",
    oauth2.AllowPolicies(opaClient.NewPolicy("uesrs", "read")),
    func(ctx iris.Context) {
        ctx.JSON(iris.Map{"message": "List of users..."})
    })

app.Post("/api/orders",
    oauth2.AllowPolicies(opaClient.NewPolicy("orders", "write")),
    func(ctx iris.Context) {
        ctx.JSON(iris.Map{"message": "Order created..."})
    })
```

With OPA in front of your OAuth2 server, an access rule lives in one policy instead of being spread across handlers, and every service that consults it decides the same way.
