package identity

import (
	stdContext "context"
	"fmt"
	"os"
	"strings"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/auth/oauth2"
	"github.com/kataras/iris/v14/rest/errors"
	// Identity is Postgres-backed: restore the pg error mapping.
	_ "github.com/kataras/iris/v14/rest/errors/pgerrors"

	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/middleware/cache/httpcache"
	"github.com/kataras/iris/v14/middleware/httpcost"

	"github.com/kataras/jwt"
	"github.com/kataras/pg"
	"gopkg.in/yaml.v3"
)

// Header is the header key used to send the server application client's token.
const Header = "X-Token"

// Options is the configuration for the identity server.
// It's used to create a new identity server.
// The Options can loaded from a YAML file.
type Options struct {
	// The postgres database connection string.
	// Example connection string:
	// host=localhost port=5432 user=makis password=_Pass_ search_path=public dbname=_MyDB_ sslmode=require pool_max_conns=12 pool_min_conns=1 pool_max_conn_lifetime=1h pool_max_conn_idle_time=45m pool_health_check_period=15m
	ConnectionString string             `yaml:"ConnectionString"`
	Auth             auth.Configuration `yaml:"Auth"`
	// EncryptionKey is the required key used to encrypt (AES-GCM) any sensitive data
	// sent to the server by its clients.
	// It's sent to the client once and it should be kept secret on both sides,
	// it's not communicated back to the server.
	// The EncryptionKey is used to encrypt & decrypt the client's secret and the user's password.
	// If it's empty, it's resolved automatically from the Auth's access token key's configured
	// encryption key or, if that one is empty too, from the refresh token key's one.
	// `New` panics if neither of them is configured.
	EncryptionKey string `yaml:"EncryptionKey"`
	// For authorization & device code flows.
	LoginPageURL   string `yaml:"LoginPageURL"`
	ConsentPageURL string `yaml:"ConsentPageURL"`
	// AllowLocalCORS allows loopback origins. Leave it false in production.
	AllowLocalCORS bool `yaml:"AllowLocalCORS"`
	// When true, the username field is treated as an email address
	// and used for password reset and other email-based features.
	UsernameIsEmail bool `yaml:"UsernameIsEmail"`
	// When true, the user can sign up with the same password as the one used to restore the account.
	RestoreOnSamePasswordSignup bool `yaml:"RestoreOnSamePasswordSignup"`

	// Schema is a list of user attributes that are allowed to be set by the user.
	// It is used to validate the user attributes based on the provided JSON schema.
	//
	// Supported field types:
	//   - "text": String field with optional min/max length validation
	//   - "number": Numeric field with optional min/max value validation
	//   - "email": Email field with built-in email format validation
	//   - "date": Date field with optional min/max year validation (YYYY-MM-DD format)
	//   - "phone": Phone number field with optional format validation
	//   - "bool": Boolean field accepting true/false values or string representations
	//   - "slice": Slice field accepting a list of values with optional type and length validation
	//   - "map": Map field accepting map[string]any with optional required key validation
	//   - "uuid": UUID field with optional database indexing
	// See `parseUserSchema` for more information.
	//
	// Examples:
	//   - Name: "firstname"
	//     Type: "text"
	//     Params: ["3", "15"]         # Text field with min 3, max 15 characters
	//   - Name: "lastname"
	//     Type: "text"
	//     Params: ["3", "15"]         # Text field with min 3, max 15 characters
	//   - Name: "age"
	//     Type: "number"
	//     Params: ["18", "105"]       # Numeric field with min 18, max 105
	//   - Name: "email"
	//     Type: "email"
	//     Indexed: true               # Email field with database index
	//     Required: true              # Required field, must be provided
	//   - Name: "company_id"
	//     Type: "uuid"
	//     Indexed: true               # UUID field with database index
	//   - Name: "birthdate"
	//     Type: "date"
	//     Params: ["1900", "now"]     # Date field with min year 1900, max current year
	//   - Name: "phone"
	//     Type: "phone"
	//     Params: ["international"]   # Phone field with international format
	//   - Name: "is_active"
	//     Type: "bool"                # Boolean field accepting true/false or "true"/"false"
	//   - Name: "tags"
	//     Type: "slice"
	//     Params: ["text", "1", "10"]  # Slice of text values between 1 and 10 items
	//   - Name: "tags"
	//     Type: "slice"
	//     Params: ["text"]             # Slice of text values without restrictions
	//   - Name: "tags"
	//     Type: "slice"
	//     Params: ["text", "1"]        # Slice of text values with minimum 1 item
	//   - Name: "preferences"
	//     Type: "map"
	//     Params: ["theme", "lang"]   # Map field requiring "theme" and "lang" keys
	//   - Name: "metadata"
	//     Type: "map"                 # Map field with no required keys
	Schema []*UserAttribute `yaml:"Schema"`
	// Cache options, optional.
	// Example:
	// 	Cache:
	//   Disable: false
	//   Addr:     "127.0.0.1:6379"
	//   Password: "admin!123"
	//   TTL: 10m
	//   CommandTimeout: 15s
	Cache httpcache.Options `yaml:"Cache"`
	// Cache is the user cache options.
	// It is used to cache user data and paginated user lists.
	// If empty, the cache is disabled and all operations are no-op.
	// Cache UserCacheOptions `yaml:"Cache"`
	// Billing & costing options, optional.
	Cost httpcost.Options `yaml:"Cost"`
}

// openDatabase opens a new postgres database connection.
func (opts *Options) openDatabase() (*pg.DB, error) {
	schema := pg.NewSchema()
	schema.Strict = false

	schema.MustRegister("origins", entity.Origin{})
	schema.MustRegister("oauth2_clients", oauth2.Client{})
	schema.MustRegister("oauth2_client_webhooks", entity.ClientWebhook{})
	schema.MustRegister("metrics", entity.Metric{})
	schema.MustRegister("users", entity.User{})
	schema.MustRegister("user_groups", entity.UserGroup{})
	schema.MustRegister("billings", entity.Billing{}, pg.Presenter)

	db, err := pg.Open(stdContext.Background(), schema, opts.ConnectionString)
	if err != nil {
		return nil, err
	}

	err = db.CreateSchema(stdContext.Background())
	if err == nil {
		err = db.CheckSchema(stdContext.Background())
	}
	return db, err
}

func ensureUserAttrsIndexes(db *pg.DB, userJSONBIndexes []string) error {
	for _, field := range userJSONBIndexes {
		indexName := fmt.Sprintf("idx_users_attrs_%s", field)
		sql := fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %s ON users USING GIN ((attrs->'%s'));`,
			indexName, field,
		)
		if _, err := db.Exec(stdContext.Background(), sql); err != nil {
			return fmt.Errorf("failed to create index for field %q: %w", field, err)
		}
	}

	// NOTE:
	// PG library already sets a gin index to the "attrs" column.
	// But it's actually bad to have both a gin index for column and specific fields?
	// No, it's not bad.
	// We can have both a general GIN index on the whole column and additional indexes
	// on specific fields (e.g., attrs->'firstname').
	// The general GIN index is useful for ad-hoc, flexible queries on any attribute.
	// Specific field indexes (also GIN) are useful if you frequently query/filter/sort
	// on a particular attribute, and want to optimize those queries further.
	return nil
}

// Load loads the identity server from a YAML configuration file.
// It panics on failure.
func Load(optionsFilename string) *Server {
	contents, err := os.ReadFile(optionsFilename)
	if err != nil {
		panic(err)
	}

	var opts Options
	if err = yaml.Unmarshal(contents, &opts); err != nil {
		panic(err)
	}

	return New(opts)
}

// Server is the identity server.
// The server is responsible to manage the user-side and server-side client OAuth2 applications.
type Server struct {
	options *Options

	db *pg.DB

	auth          *auth.Auth[entity.FlatUser]
	clientAuth    *auth.Auth[oauth2.ClientTokenClaims]
	signer        *ClientTokenSigner
	oauth2Server  *oauth2.Server[entity.FlatUser]
	EncryptionKey string
	Provider      *UserProvider

	// schema holds the new user attributes validator and indexes.
	// It returns a pointer to the UserSchema instance which can be used to set custom validators for specific fields.
	// The validator validates user attributes based on the provided JSON schema.
	// It allows setting custom validators for specific fields, such as text, number, email, date, phone, bool, and map.
	// Usage:
	//
	//	identityServer.schema.SetText("firstname", 3, 15)                    // Set a text validator with min 3, max 15 characters
	//	identityServer.schema.SetNumber("age", 0, 120)                       // Set a number validator with min 0, max 120
	//	identityServer.schema.SetEmail("email")                              // Set an email validator
	//	identityServer.schema.SetDate("birthdate", 1900, 0)                  // Set a date validator with min year 1900, no max
	//	identityServer.schema.SetPhone("phone", "international")             // Set a phone validator with international format
	//	identityServer.schema.SetBool("is_active")                           // Set a boolean validator accepting true/false or string representations
	//	identityServer.schema.SetMap("preferences", []string{"theme", "lang"}) // Set a map validator requiring "theme" and "lang" keys
	//	identityServer.schema.SetMap("metadata", nil)                        // Set a map validator with no required keys
	//  identityServer.schema.SetSlice("tags", "text", 1, 10)                // Set a slice validator for tags with text type between 1 and 10 items.
	// Note that if a user validator exists for an attribute but it's missing, it's considered a required one.
	// Cannot be modified after parsing,
	// because we create the database indexes based on it.
	schema *UserSchema

	// Cache is the HTTP cache used to cache user data and paginated user lists.
	cache *httpcache.Cache
}

// New creates and returns a new identity server.
// It panics on failure.
func New(opts Options) *Server {
	userSchema, err := parseUserSchema(opts.Schema)
	if err != nil {
		panic(fmt.Errorf("failed to parse user attributes validator: %w", err))
	}

	db, err := opts.openDatabase()
	if err != nil {
		panic(fmt.Errorf("failed to connect to the database: %w", err))
	}
	iris.RegisterOnInterrupt(db.Close)

	if err = ensureUserAttrsIndexes(db, userSchema.GetDatabaseIndexes()); err != nil {
		panic(fmt.Errorf("failed to ensure user attributes indexes: %w", err))
	}

	// EncryptionKey is a shared secret between the client and the server.
	// It's sent once on server app client creation.
	// It's never sent back to the server, it's used to encrypt the client's user's password.
	encryptionKey := opts.EncryptionKey
	if encryptionKey == "" { // try to auto-resolve the encryption key from access token's configuration.
		if keyConfig, _ := opts.Auth.Keys.Get(auth.KIDAccess); keyConfig.EncryptionKey != "" {
			encryptionKey = keyConfig.EncryptionKey
		} else if keyConfig, _ = opts.Auth.Keys.Get(auth.KIDRefresh); keyConfig.EncryptionKey != "" { // or refresh token's configuration.
			encryptionKey = keyConfig.EncryptionKey
		} else {
			panic(fmt.Errorf("encryption key is required, please set the EncryptionKey field in the configuration, e.g. %s", jwt.MustGenerateRandomString(16)))
		}

		opts.EncryptionKey = encryptionKey
	}

	// Clone and modify the user authentication configuration to be used as an app client authenticator as well.
	clientAuthConfiguration := opts.Auth.Clone()
	clientAuthConfiguration.Cookie = auth.CookieConfiguration{} // disable any cookie setting.
	clientAuthConfiguration.Headers = []string{Header}          // set the header to a different one without bearer prefix.
	clientAuthConfiguration.HeadersNoBearer = true
	// force the refresh token to be encrypted with the same key as the identity server,
	// jwt package uses the same algorithm (GCM-AES) to encrypt the token as well.
	// We need it to encrypt the server app client token.
	for i, keyConfig := range clientAuthConfiguration.Keys {
		clientAuthConfiguration.Keys[i].MaxAge = 0 // disable the default max age for the client app tokens.

		if keyConfig.ID == auth.KIDRefresh {
			clientAuthConfiguration.Keys[i].EncryptionKey = encryptionKey
		} else {
			clientAuthConfiguration.Keys[i].EncryptionKey = "" // remove any encryption for the access token of the app clients.
		}
	}
	clientAuthController, err := auth.New[oauth2.ClientTokenClaims](clientAuthConfiguration)
	if err != nil {
		panic(fmt.Errorf("failed to create client auth controller: %w", err))
	}

	authController, err := auth.New[entity.FlatUser](opts.Auth)
	if err != nil {
		panic(fmt.Errorf("failed to create auth controller: %w", err))
	}

	oauth2Server := oauth2.New(authController,
		oauth2.WithClientStore(oauth2.NewPostgresClientStore("oauth2_clients", db)),
		oauth2.WithLoginPageURL(opts.LoginPageURL),
		oauth2.WithConsentPageURL(opts.ConsentPageURL),
		oauth2.WithDeviceInterval(5),
	)

	cache, err := httpcache.New(opts.Cache)
	if err != nil {
		panic(fmt.Errorf("failed to connect to the cache: %w", err))
	}
	iris.RegisterOnInterrupt(cache.Close)

	return &Server{
		options: &opts,
		db:      db,

		auth:          authController,
		clientAuth:    clientAuthController,
		signer:        NewClientTokenSigner(clientAuthController, encryptionKey),
		oauth2Server:  oauth2Server,
		EncryptionKey: encryptionKey,
		Provider: NewUserProvider(
			entity.NewRegistry(db),
			authController,
			encryptionKey,
			opts.UsernameIsEmail,
			opts.RestoreOnSamePasswordSignup,
		),
		schema: userSchema,
		cache:  cache,
	}
}

// UserApp is the client application that is used by the user.
// It has no access for the OAuth2 password grant flow neither any of the APIs available outside of the /oauth2 ones.
type UserApp struct {
	Token string `yaml:"Token"`

	Client *oauth2.Client `yaml:"-"` // The client application used by the user.
}

// UserApplicationRegistration holds the values AddUserApplication registers a user application with.
// It has no grant types or scopes: a user application always gets the authorization_code,
// device code and refresh_token grant types, and the scopes of UserAppPolicies.
//
// Name, Website and LogoURL are required, the same way as in oauth2.ClientRegistration.
type UserApplicationRegistration struct {
	// Name is the unique name of the application. Required.
	Name string
	// Website is the website URL of the application. Required.
	Website string
	// LogoURL is the URL of the application's logo. Required.
	LogoURL string
	// RedirectURI is the callback URL of the authorization code flow.
	// Required, because a user application always has the authorization code grant type.
	RedirectURI string
	// VerificationURI is the URL where the user enters the device code.
	// Required, because a user application always has the device code grant type.
	VerificationURI string
	// CodeMaxAge is the max age of the authorization and device codes, in seconds.
	// Zero or negative means 3600 (one hour).
	CodeMaxAge int
	// AccessMaxAge is the max age of the access tokens, in seconds.
	// Zero or negative means the token does not expire.
	AccessMaxAge int
	// RefreshMaxAge is the max age of the refresh tokens, in seconds.
	// Zero or negative means the token does not expire. When both max ages are above zero,
	// RefreshMaxAge must be at least AccessMaxAge plus 60.
	RefreshMaxAge int
}

// AddUserApplication registers a new user application.
// If the application already exists, it generates a token based on the existing one.
// It returns the user application structure which contains the token.
//
// A user application always gets the oauth2.GrantTypeAuthorizationCode,
// oauth2.GrantTypeDeviceCode and oauth2.GrantTypeRefreshToken grant types
// and the scopes of UserAppPolicies.
func (s *Server) AddUserApplication(ctx stdContext.Context, reg UserApplicationRegistration) (*UserApp, error) {
	client, err := s.oauth2Server.RegisterClient(ctx, oauth2.ClientRegistration{
		Name:            reg.Name,
		Website:         reg.Website,
		LogoURL:         reg.LogoURL,
		RedirectURI:     reg.RedirectURI,
		VerificationURI: reg.VerificationURI,
		GrantTypes:      []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeDeviceCode, oauth2.GrantTypeRefreshToken},
		Scopes:          oauth2.PolicyScopes(UserAppPolicies...),
		CodeMaxAge:      reg.CodeMaxAge,
		AccessMaxAge:    reg.AccessMaxAge,
		RefreshMaxAge:   reg.RefreshMaxAge,
	})
	if err != nil {
		return nil, err
	}

	token, err := s.signClientToken(client)
	if err != nil {
		return nil, err
	}

	app := &UserApp{
		Token:  token.AccessToken,
		Client: client,
	}
	return app, nil
}

// ServerApp is the client application that is used by the server.
// It has access to all the APIs, including the OAuth2 password grant flow.
// It's used to create, read, update and delete users.
type ServerApp struct {
	Token         string `yaml:"Token"`
	EncryptionKey string `yaml:"EncryptionKey"`

	Client *oauth2.Client `yaml:"-"`
}

// AddServerApplication registers a new server application.
// It returns the server application structure which contains the token and the shared encryption key.
// If the application already exists, it generates a token based on the existing one.
// Usage:
//
//	serverApp, err := identityServer.AddServerApplication(
//	    context.Background(),
//	    "Example-Server-Name",
//	    "https://www.iris-go.com",
//	    "https://www.iris-go.com/images/logo.png",
//	)
func (s *Server) AddServerApplication(ctx stdContext.Context, name, website, logo string) (*ServerApp, error) {
	client, err := s.oauth2Server.RegisterClient(ctx, oauth2.ClientRegistration{
		Name:          name,
		Website:       website,
		LogoURL:       logo,
		GrantTypes:    []string{oauth2.GrantTypePassword, oauth2.GrantTypeRefreshToken},
		Scopes:        oauth2.PolicyScopes(ServerAppPolicies...),
		CodeMaxAge:    3600 * 2, // code max age is 2 hours, but it's not used for this type of apps.
		AccessMaxAge:  3600 * 2, // access max age is 2 hours.
		RefreshMaxAge: 604800,   // refresh max age is 7 days.
	})
	if err != nil {
		return nil, err
	}

	token, err := s.signClientToken(client)
	if err != nil {
		return nil, err
	}

	app := &ServerApp{
		Token:         token.RefreshToken, // use the encrypted one.
		EncryptionKey: s.EncryptionKey,
		Client:        client,
	}
	return app, nil
}

// AddAdminApplication registers (or upgrades) a server application that may also manage clients.
// An existing application keeps its secret and version, it only gains the missing "clients:" scopes.
// It returns an error if the existing application is a user application, which has no password grant.
func (s *Server) AddAdminApplication(ctx stdContext.Context, name, website, logo string) (*ServerApp, error) {
	spec, err := specForKind(ClientKindAdmin, "")
	if err != nil {
		return nil, err
	}

	client, err := s.oauth2Server.RegisterClient(ctx, oauth2.ClientRegistration{
		Name:          name,
		Website:       website,
		LogoURL:       logo,
		GrantTypes:    spec.GrantTypes,
		Scopes:        spec.Scopes,
		CodeMaxAge:    spec.CodeMaxAge,
		AccessMaxAge:  spec.AccessMaxAge,
		RefreshMaxAge: spec.RefreshMaxAge,
	})
	if err != nil {
		return nil, err
	}

	if !client.MatchGrantType(oauth2.GrantTypePassword) {
		return nil, fmt.Errorf("%w: application %q is a user application and cannot become an admin one", entity.ErrInvalidArgument, name)
	}

	if _, err = s.oauth2Server.EnsureClientScopes(ctx, client, spec.Scopes); err != nil {
		return nil, err
	}

	token, err := s.signClientToken(client)
	if err != nil {
		return nil, err
	}

	app := &ServerApp{
		Token:         token.RefreshToken, // use the encrypted one.
		EncryptionKey: s.EncryptionKey,
		Client:        client,
	}
	return app, nil
}

// ErrSignToken is returned when the server fails to sign a token.
var ErrSignToken = errors.New("failed to sign token")

func (s *Server) signClientToken(client *oauth2.Client) (*oauth2.Token, error) {
	return s.signer.Sign(client)
}

// Routing registers the identity server's handlers to the provided router.
// It implements the router.Controller/Routing interface.
//
// Usage:
//
//	app.Router("/auth", identityServer)
func (s *Server) Routing(r router.Router) {
	corsService := NewCORSService(entity.NewRegistry(s.db), s.options.AllowLocalCORS)
	r.UseRouter(corsService.Handler())

	r.Use(NewCostHandler(s.options.Cost, s.db))

	// Register the public keys JWKS endpoint.
	// So the client apps can verify the JWT tokens.
	r.Get("/.well-known/jwks.json", s.auth.JWKSHandler())

	// Register OpenID Connect discovery and UserInfo endpoints
	// s.RegisterOIDCRoutes(r)
	// Register the oauth2 server's handlers on the root of "r".
	// s.registerOAuth2WithOIDC(r)
	// Register the oauth2 server's handlers on the root of "r".
	r.Router("/oauth2", s.oauth2Server)

	r.Use(RegisterCodeErrors().Handler())
	// This verifies and sets the auth.GetUser[*Client]. Also because we use a key set,
	// it recognise the correct (e.g. refresh token which is encrypted) and calls the correct validation process.
	// Any valid client may call /clients self routes. Server routes below add their own gate:
	// AllowPolicies is any-of, and ServerAppPolicies includes user:read, so that list cannot be the gate.
	r.Use(s.clientAuth.VerifyHandler(s.oauth2Server.ClientVerifier()))

	// Register cross-API dependencies.
	r.RegisterDependency(
		s.db,
		entity.NewRegistry,
		s.cache,
	)
	// App Clients controller. Self routes stay open to any authenticated client.
	// Admin routes carry their own clients:* policies.
	r.Router("/clients", iris.Dependencies(s.oauth2Server, s.signer), NewClientController)

	serverGate := oauth2.AllowPolicies(ServerOnlyPolicies...)
	// The Billing & Usage controller.
	r.Router("/billing", serverGate, NewBillingController)
	// Control the cache, cleanup and view information about cached entries.
	r.Router("/cache", serverGate, NewCacheController)
	r.Router("/cors", serverGate, iris.Dependencies(corsService), NewCORSController)
	// Register the signup and other server-app user-related handlers (e.g. signup, delete, list)
	// under the "/u" and "/u/groups" (see inside NewUserController) paths.
	r.Router("/u", serverGate, iris.Dependencies(s.oauth2Server, s.schema, s.Provider, s.auth.WithProvider, NewUserGroupService),
		NewUserController,
	)
}

// Close closes the database and cache connections.
func (s *Server) Close() {
	s.db.Close()
	if s.cache != nil {
		s.cache.Close()
	}
}

/*
// registerOAuth2WithOIDC registers OAuth2 routes with OIDC enhancements
func (s *Server) registerOAuth2WithOIDC(r router.Router) {
	oauth2Router := r.Router("/oauth2")

	// Register standard OAuth2 endpoints
	oauth2Router.Get("/authorize", s.oauth2Server.AuthorizeHandler())
	oauth2Router.Post("/device/code", s.oauth2Server.DeviceCodeHandler())
	oauth2Router.Post("/token", s.oauth2Server.TokenHandler())
	oauth2Router.Post("/token/introspect", s.oauth2Server.TokenIntrospectHandler())

	// Enhanced consent page for OIDC
	if s.options.ConsentPageURL != "" {
		// If the consent page url matches the internal, default consent url then register the enhanced OIDC consent page
		if s.options.ConsentPageURL == "default" || strings.HasSuffix(s.options.ConsentPageURL, "/oauth2/consent") {
			oauth2Router.Get("/consent", s.OIDCConsentPageHandler())
		}

		// This remains as it is as the custom consent pages should hit this one in order to pass the confirmation
		// and authorize a client.
		oauth2Router.Post("/consent", s.oauth2Server.ConfirmConsentHandler())

		// A simple endpoint helper to forget the consent decision.
		oauth2Router.Get("/consent/revoke", s.oauth2Server.RevokeConsentHandler())
	}
}
*/

// RegisterCodeErrors registers the error codes used by the identity server.
// It returns a new errors.Options instance with the error codes mapped to their corresponding HTTP status codes.
func RegisterCodeErrors() *errors.Options {
	return errors.NewOptions().
		// Order of mapping matters inside functions,
		// order of error code name mapping is based on its status code (low to high).
		MapErrors(errors.NotFound, entity.ErrNotFound, oauth2.ErrClientNotFound).
		MapErrors(errors.AlreadyExists, entity.ErrAlreadyExists, oauth2.ErrClientAlreadyExists).
		MapErrors(errors.InvalidArgument,
			pg.ErrIntentionalRollback,
			entity.ErrInvalidArgument,
			entity.ErrInvalidSortOption,
			oauth2.ErrInvalidClientFilter,
		).MapErrors(errors.Internal, ErrSignToken).
		MapErrorFunc(errors.InvalidArgument, func(err error) error {
			field, ok := pg.IsErrInputSyntax(err)
			if ok {
				return fmt.Errorf("invalid input syntax for field %q: %w", field, err)
			}

			errText := err.Error()
			if strings.Contains(errText, "encoding/hex: invalid byte:") {
				return fmt.Errorf("invalid secret format: %w", err)
			}

			if strings.Contains(errText, "invalid input syntax for type") {
				// invalid input syntax for type uuid: "kataras2006@hotmail.com" (SQLSTATE 22P02)
				// remove the SQLSTATE part.
				parts := strings.Split(errText, " (SQLSTATE")
				if len(parts) > 0 {
					errText = parts[0]
				}
				// parts = strings.Split(errText, ":")
				// if len(parts) >= 2 {
				// 	typeParts := strings.Split(parts[0], " for type ")
				// 	if len(typeParts) == 2 {
				// 		valueParts := strings.Split(typeParts[1], ": ")
				// 		if len(valueParts) == 2 {
				// 			return fmt.Errorf("invalid input syntax for type %q, value %q: %w", strings.Trim(valueParts[0], `"' `), strings.Trim(valueParts[1], `"' `), err)
				// 		}
				// 		return fmt.Errorf("invalid input syntax for type %q: %w", strings.Trim(typeParts[1], `"' `), err)
				// 	}
				// }
				return errors.New(errText)
			}

			return nil
		})
}
