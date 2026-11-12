# Authentication SDK

The `auth/` package provides a complete, production-ready authentication and authorization system for Iris applications.

## Architecture

Three main components:

1. **Core Auth** (`auth/`) - Generic JWT signer/verifier with provider interface
2. **Identity Server** (`auth/identity/`) - Complete user management with PostgreSQL
3. **OAuth2 Server** (`auth/oauth2/`) - Full OAuth2 2.0 implementation

## Core Auth Package

### JWT Authentication

- **Algorithm**: EdDSA (Ed25519) - modern elliptic-curve cryptography
- **Access tokens**: 2-hour lifetime (short-lived, stateless)
- **Refresh tokens**: 30-day lifetime (encrypted, supports rotation)

### Configuration

```yaml
Headers:
  - "Authorization"
  - "X-Authorization"
Cookie:
  Name: "iris_auth_cookie"
  Secure: true
  Hash: "64-character-hash-key"       # HMAC authentication (512-bit)
  Block: "32-character-block-key"     # AES encryption (256-bit)
Keys:
  - ID: IRIS_AUTH_ACCESS
    Alg: EdDSA
    MaxAge: 2h
    Private: "-----BEGIN PRIVATE KEY-----..."
    Public: "-----BEGIN PUBLIC KEY-----..."
  - ID: IRIS_AUTH_REFRESH
    Alg: EdDSA
    MaxAge: 720h
    EncryptionKey: "hex-encoded-16-bytes"
    Private: "-----BEGIN PRIVATE KEY-----..."
    Public: "-----BEGIN PUBLIC KEY-----..."
```

Configuration methods:
- `config.BindRandom()` - Generate random keys (development/testing)
- `config.BindFile("auth.yml")` - Load from YAML or JSON file
- `config.ToYAML()` / `config.ToJSON()` - Export configuration
- `config.Clone()` - Deep copy

### Provider Interface

Implement custom authentication logic:

```go
type Provider[T User] interface {
    Signin(ctx context.Context, username, password string) (T, error)
    Refresh(ctx context.Context, t T) (T, error)
    ValidateToken(ctx context.Context, claims StandardClaims, t T) error
    InvalidateToken(ctx context.Context, claims StandardClaims, t T) error
    InvalidateTokens(ctx context.Context, t T) error
}
```

Optional enhancement interfaces:
- `Transformer[T]` - Convert JWT claims to custom user type
- `ClaimsProvider` - Customize token claim content
- `ErrorHandler` - Custom HTTP error responses

### Auth[T] Handlers

```go
auth, err := auth.New[User](config)
auth.AddProvider(&MyProvider{})

// Register handlers
app.Post("/signin", auth.SigninHandler)
app.Post("/refresh", auth.RefreshHandler)
app.Post("/signout", auth.SignoutHandler)
app.Post("/signout-all", auth.SignoutAllHandler)
app.Get("/.well-known/jwks.json", auth.JWKSHandler())

// Verify middleware
app.Use(auth.VerifyHandler())

// With custom verification (VerifyUserFunc takes a standard context.Context)
app.Use(auth.VerifyHandler(func(ctx context.Context, user User) error {
    if !user.Active {
        return errors.PermissionDenied
    }
    return nil
}))
```

### Context Helpers

```go
// Get authenticated user
user := auth.GetUser[User](ctx)

// Get verified token
token := auth.GetToken(ctx)

// Check if authorized
if auth.UserAuthorized(ctx) {
    // user is authenticated
}
```

### Request Scope (multi-tenant)

`auth.Scope` derives one value off the verified user (a tenant, a customer, an organisation) once, after `VerifyHandler`, and stores it for every handler and query after it. A handler that forgets to filter by tenant is a data leak and a handler that derives it again is a second chance to get it wrong, so the derivation happens in one place.

```go
func Scope[User any, ID comparable](derive func(User) ID) context.Handler
func ScopeID[ID comparable](ctx *context.Context) (ID, bool)
```

```go
app.Use(a.VerifyHandler(), auth.Scope(func(u Customer) string {
    return u.TenantID
}))

// Downstream, in a handler or an mvc.BeforeWrite hook:
tenantID, ok := auth.ScopeID[string](ctx)
```

- A missing user answers 401 with `auth.ErrMissingUser`, the same failure `VerifyHandler` answers with. So does a user of **another type**: reading it untyped and deriving from a zero value would answer 200 with an empty scope, which is a cross-tenant read.
- A derived **zero** id is stored as it is: a global administrator may legitimately have no scope. Guard those requests by checking the id, not by expecting `Scope` to refuse them.
- `ScopeID` reports false when `Scope` did not run or stored an id of another type, so an untrustworthy zero cannot pass for a real one.
- Cost: one `Values` read, the `derive` call and one `Values` set per request; `ScopeID` is one map lookup and one type assertion. No reflection.
- The stamping half (writing the scope onto every decoded body) is `mvc.BeforeWrite`, see [builtin-controllers.md](./builtin-controllers.md).

## Identity Server

Complete user management system with PostgreSQL backend.

### Setup

```go
import "github.com/kataras/iris/v14/auth/identity"

identityServer := identity.Load("identity.yml")
app.Router("/auth", identityServer)
```

### User Model

```go
type User struct {
    ID               string
    CreatedAt        time.Time
    UpdatedAt        time.Time
    Username         string            // Unique (email or username)
    Password         string            // Bcrypt hash
    OTP              string            // TOTP key URL
    OTPRecoveryCodes []RecoveryCode    // Recovery codes
    Attrs            json.RawMessage   // Custom JSONB attributes
    DeletedAt        time.Time         // Soft delete
}
```

### User API Endpoints

User routes are registered under the `/u` prefix of the identity server (the SDK calls `/u/signup`, `/u/list`, and so on).

**Authentication:**
- `POST /signup` - Create new user
- `POST /signin` - Authenticate user, returns JWT tokens
- `POST /refresh` - Get new access token
- `POST /enrich` - Add extra claims to an access token, returns a new enriched token (restricted column names are rejected; the result is stamped with `enriched` and `enriched_at` claims)
- `POST /signout` - Invalidate token
- `POST /signout-all` - Invalidate all user tokens

**User Management:**
- `GET /schema` - Render the user schema (attribute names, types, params) as JSON
- `GET /{identifier}` - Get user by ID or username
- `PUT /` - Update user profile
- `PATCH /attrs` - Update user attributes only (partial updates)
- `DELETE /` - Soft delete user
- `POST /restore` - Restore a single soft-deleted user by ID or username

**Admin/Bulk Operations:**
- `POST /bulk` - Insert multiple users
- `DELETE /bulk` - Delete multiple users
- `POST /bulk/restore` - Restore deleted users
- `POST /list` and `QUERY /list` - List users with pagination and filtering (see below)
- `GET /count` - User counts: `{"active_count": N, "inactive_count": N, "total_count": N}`

**Password Management:**
- `POST /request-password-reset` - Send reset token
- `POST /confirm-password-reset` - Reset with token
- `POST /reset-password` - Force password reset

### List Filtering

`/list` answers both QUERY (RFC 10008) and POST, for clients built before QUERY, and accepts `entity.UserFilterOptions` as the request body:

```go
type UserFilterOptions struct {
    Sort           string      // sort expression (default "created_at")
    SortDescending bool        // descending order when true
    Terms          FilterTerms // []*FilterTerm
    IncludeDeleted bool        // include soft-deleted users
    IncludeIDs     []string    // only return users with these IDs
}

type FilterTerm struct {
    Field    string // "username", "created_at", or a custom attrs field
    Operator string // =, IS, IS NOT, ILIKE, @>, #>>, <, >, <=, >= (default "=")
    Value    any
    Logic    string // AND (default) or OR
}
```

Custom (non-core) field names are rewritten to JSONB expressions on the `attrs` column automatically: `phone` becomes `(attrs::jsonb)->>'phone'`, a dotted path like `profile.name` becomes a nested lookup, and the `@>` operator matches JSONB containment against the whole `attrs` object.

### User Schema Validation

Define custom attribute validation declaratively:

```yaml
Schema:
  - Name: "firstname"
    Type: "text"
    Params: ["3", "50"]       # min/max length
    Required: true
    Indexed: true
  - Name: "age"
    Type: "number"
    Params: ["0", "120"]      # min/max value
  - Name: "email"
    Type: "email"
  - Name: "birthdate"
    Type: "date"
    Params: ["1900", "0"]     # min year, 0=no max
  - Name: "phone"
    Type: "phone"
    Params: ["international"]
  - Name: "plan"
    Type: "text"
    Access: "privileged"      # who may write it; "self" is the default
```

`Access` says who may write an attribute. `self` (`identity.AttrAccessSelf`, the default) is a profile field a client with `users:update` may change. `privileged` (`identity.AttrAccessPrivileged`) is written by an admin client, or by a server client whose scopes include `users:attr:<name>` (`identity.AttrScope(name)` returns that scope); the account holder does not write it.

Supported types: `text`, `number`, `email`, `date`, `phone`, `bool`, `uuid`, `slice`, `map`

### Application Types

**User Application** (frontend client):
- Authorization Code Flow
- Device Code Flow
- Read user profile scope only

**Server Application** (backend service):
- Password Grant Flow (service-to-service)
- Full API access (CRUD users)
- Shared encryption key

```go
// Create server application
serverApp, _ := identityServer.AddServerApplication(ctx,
    "My Backend", "https://api.example.com", "https://example.com/logo.png")

// Create user application. It has no grant types or scopes to choose:
// a user application always gets authorization_code, device_code and
// refresh_token, and the scopes of UserAppPolicies.
userApp, _ := identityServer.AddUserApplication(ctx, identity.UserApplicationRegistration{
    Name:            "Web Frontend",                       // required
    Website:         "https://example.com",                // required
    LogoURL:         "https://example.com/logo.png",       // required
    RedirectURI:     "https://example.com/callback",       // required
    VerificationURI: "https://example.com/device-verify",  // required
    CodeMaxAge:      3600,   // seconds; zero means 3600
    AccessMaxAge:    3600,   // seconds; zero means no expiry
    RefreshMaxAge:   604800, // seconds; at least AccessMaxAge+60 when both are set
})

// Create (or upgrade) the bootstrap admin client: a server application that
// also holds the clients:* scopes. An existing one keeps its secret and only
// gains the missing scopes; a user application cannot be upgraded.
adminApp, _ := identityServer.AddAdminApplication(ctx,
    "Admin", "https://example.com", "https://example.com/logo.png")
```

### Client Management API

An admin client's token reaches these routes under `/clients`, each guarded by its own policy (`identity.CreateClientsPolicy`, `ReadClientsPolicy`, `UpdateClientsPolicy`, `DeleteClientsPolicy`; together `identity.ManageClientsPolicies`, scopes `ManageClientsScopes`, the `clients:*` scopes). `identity.AdminAppPolicies` is the server policies plus these.

| Route | Policy | SDK method |
|-------|--------|------------|
| `POST /clients` | create | `AdminCreateClient(ctx, identity.CreateClientRequest) (*identity.CreateClientResponse, error)` |
| `POST` or `QUERY /clients/list` | read | `AdminListClients(ctx, page, *oauth2.ClientFilter) (*iris.Page[oauth2.Client], error)` (the SDK sends POST) |
| `GET /clients/count` | read | `AdminCountClients(ctx) (int64, error)` |
| `GET /clients/{id:uuid}` | read | `AdminGetClient(ctx, id) (*oauth2.Client, error)` |
| `DELETE /clients/{id:uuid}` | delete | `AdminDeleteClient(ctx, id) (bool, error)` |
| `POST /clients/{id:uuid}/token` | update | `AdminIssueClientToken(ctx, id) (*identity.ClientCredentials, error)` |
| `POST /clients/{id:uuid}/revoke-tokens` | update | `AdminRevokeClientTokens(ctx, id) (bool, error)` |

- `identity.CreateClientRequest` has `Name`, `Website`, `LogoURL`, `Kind` (`identity.ClientKindServer`, `ClientKindUser` or `ClientKindAdmin`), `RedirectURI` (required for the user kind), `VerificationURI` (adds the device code grant to a user client) and the three max ages in seconds (zero means the default of the kind).
- `identity.ClientCredentials` is what a consumer needs: `Audience` (the client id; consumers put it in `Audiences`), `Kind`, `Token`, and `EncryptionKey` for server and admin clients. `CreateClientResponse` embeds it next to the `*oauth2.Client`, secret included.
- `oauth2.ClientFilter` filters by `Name` (contains, case-insensitive) and `Scope` (exact, e.g. `clients:create`), and orders by `Sort` (`name`, `created_at`, `updated_at`) with `SortDescending`.
- A client cannot delete itself, and the last admin client that can create clients cannot be deleted (both answer `FAILED_PRECONDITION`).
- The SDK methods that take an id return `sdk.ErrClientIDRequired` for an empty one before sending anything.
- `identity.KindOfClient(*oauth2.Client) ClientKind` reports the kind of a stored client.

## OAuth2 Server

Full OAuth2 2.0 implementation.

### Supported Grant Flows

1. **Authorization Code** - Standard web app flow with consent page
2. **Device Code** - For IoT/TV devices without browser
3. **Password Grant** - Service-to-service authentication
4. **Refresh Token** - Token rotation and renewal

### Standard Endpoints

- `GET /.well-known/jwks.json` - Public keys
- `POST /oauth2/token` - Token endpoint
- `GET /oauth2/authorize` - Authorization endpoint
- `POST /oauth2/device/code` - Device code endpoint

### Resource-Action Policies

```go
// User app policies (read-only)
ReadUserPolicy = oauth2.ResourcePolicy{Resource: "user", Action: "read"}

// Server app policies (full access)
CreateUsersPolicy = oauth2.ResourcePolicy{Resource: "users", Action: "create"}
ReadUsersPolicy   = oauth2.ResourcePolicy{Resource: "users", Action: "read"}
UpdateUsersPolicy = oauth2.ResourcePolicy{Resource: "users", Action: "update"}
DeleteUsersPolicy = oauth2.ResourcePolicy{Resource: "users", Action: "delete"}
```

### Open Policy Agent (OPA) Integration

Define access control policies in Rego:

```rego
package authz

default allow = false

allow {
    input.client_id == "example-client"
    input.resource == "users"
    input.action == "read"
}
```

## Security Features

### Password Security
- Bcrypt hashing (automatically salted, cost 10)
- AES-GCM encryption for transmission
- Random password generation

### Token Security
- EdDSA signatures (Ed25519)
- Encrypted refresh tokens (AES-GCM)
- Token invalidation tracking
- Per-user and global token revocation

### 2FA/OTP
- TOTP (Time-based One-Time Password)
- 4 recovery codes (12 chars each, one-time use)
- QR code generation for authenticator apps

### Cookie Security
- AES-GCM encryption
- HMAC authentication
- Secure, HttpOnly, SameSite attributes

## Client SDK

Multi-language client library for consuming the identity server API. The Go implementation lives in `auth/identity/sdk` and builds on the top-level `client/` package (see ./http-client.md): `SDK[T].Client` is a `*client.Client`.

```yaml
BaseURL: "https://identity-server.example.com"
Token: "client-token"
EncryptionKey: "hex-encoded-key"
PasswordStrengthLevel: "strong"   # very weak (default), weak, medium, strong, very strong
```

```go
import "github.com/kataras/iris/v14/auth/identity/sdk"

// T is your public user shape. It must NOT contain a json "password"
// field: sdk.New returns an error if it does. Passwords are passed
// separately and encrypted with the shared EncryptionKey.
type Profile struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    // custom attrs fields...
}

s, err := sdk.New[Profile](sdk.Options{
    BaseURL:       "https://identity-server.example.com",
    Token:         "client-token", // sent via the X-Token header on admin calls
    EncryptionKey: "hex-encoded-key",
})
```

`New` fetches `/.well-known/jwks.json`, verifies the client token, and prepares an `OAuth2Config` (authorize, device code, and token endpoints).

### SDK Methods

| Method | Endpoint | Notes |
|--------|----------|-------|
| `AdminGetUserSchema(ctx)` | `GET /u/schema` | Returns `[]*identity.UserAttribute` |
| `AdminUserSignup(ctx, user, password)` | `POST /u/signup` | Validates strength and encrypts the plain password |
| `AdminBulkInsertUsers(ctx, users, passwords)` | `POST /u/bulk` | Batch insert. On username conflict: updates password/attrs, keeps the existing ID, and restores soft-deleted users. Returns `identity.BulkInsertUsersResponse{TotalProvided, TotalInserted, Message}` |
| `UserSignin(ctx, username, password)` | `POST /oauth2/token` | OAuth2 password grant, returns `*oauth2.Token` |
| `AdminUserSignin(ctx, req)` | `POST /u/signin` | Sign in as any user by ID or username via the admin token, no password |
| `AdminDeleteUser(ctx, req)` | `DELETE /u` | `Soft: true` marks deleted instead of removing |
| `AdminBulkDeleteUsers(ctx, req)` | `DELETE /u/bulk` | By IDs, optional `Soft`, returns affected count |
| `AdminRestoreUser(ctx, req)` | `POST /u/restore` | Restore one soft-deleted user by ID or username |
| `AdminResetUserPassword(ctx, req)` | request + confirm reset | Two-step: requests a reset token, then confirms with the encrypted new password |
| `AdminListUsers(ctx, page, filter)` | `POST /u/list` (the server also answers QUERY; the SDK sends POST because some proxies reject QUERY) | `iris.PageOptions` + `*entity.UserFilterOptions`, returns `*iris.Page[T]` |
| `AdminUpdateUsers(ctx, users, onlyColumns...)` | `PUT /u` | Full updates; optional column restriction via `?columns=` |
| `AdminUpdateUsersPartial(ctx, updates)` | `PATCH /u/attrs` | Partial updates, see below |
| `TokenIntrospect[Claims](ctx, accessToken)` | `POST /oauth2/token/introspect` | Server-side token introspection, returns the decoded Claims |
| `Verify[Claims](token, validators...)` | local | Verifies a token against the fetched JWKS, returns the decoded Claims |

Generic method on the SDK instance (Go 1.27):

```go
// Decode token claims locally, no verification or other checks.
user, err := sdk.Introspect[Profile](tok.AccessToken)
```

Partial updates target core fields (username, password) and nested JSONB attrs paths in one call. A `"password"` value in `Set` is strength-validated and encrypted automatically:

```go
updates := []entity.PartialUpdateSpec{{
    ID: "user-123", // or Username for identification
    Set: map[string]any{
        "username":     "new@example.com",
        "password":     "newPlainPassword",  // encrypted automatically
        "profile.name": "John",              // nested JSONB path
    },
    Remove: []string{"profile.avatar"},
}}
resp, err := s.AdminUpdateUsersPartial(ctx, updates)
```

For protecting your own routes with identity-server tokens, the SDK also provides `s.VerifyHandler(authConfig, ...)` (an `iris.Handler`) and `sdk.NewAuth[T](s, authConfig)` which returns a ready `*auth.Auth[T]`.

## Source Files

- `auth/auth.go` - Core JWT signer/verifier
- `auth/configuration.go` - Configuration loading and validation
- `auth/provider.go` - Provider, Transformer, ErrorHandler interfaces
- `auth/user.go` - Context helpers for user retrieval
- `auth/scope.go` - `Scope`, `ScopeID`, `ErrMissingUser`
- `auth/identity/` - Identity server implementation
- `auth/identity/entity/` - User repository, filter options, partial update specs
- `auth/oauth2/` - OAuth2 server implementation
- `auth/identity/sdk/` - Client SDK
