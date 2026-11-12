package sdk

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"github.com/kataras/iris/v14/auth"
	// NOTE: On other language SDKs these imports should be replaced
	// with actual types of the language.
	"github.com/kataras/iris/v14/auth/identity"
	"github.com/kataras/iris/v14/auth/identity/entity"

	//
	"github.com/kataras/iris/v14/rest/errors/validation"
	"github.com/kataras/iris/v14/rest/types"

	"github.com/kataras/iris/v14/client"
	irisContext "github.com/kataras/iris/v14/context"

	"github.com/kataras/jwt"
	"golang.org/x/oauth2"
)

// Options is the configuration for the identity SDK.
// See the `New` function for more.
type Options struct {
	BaseURL       string `yaml:"BaseURL"`       // The base URL of the identity server.
	Token         string `yaml:"Token"`         // The client token to authenticate with the identity server.
	EncryptionKey string `yaml:"EncryptionKey"` // The encryption key to encrypt the user password.

	// PasswordStrengthLevel is used to determine the minimum length and complexity of the password.
	// It is used to validate the password strength during signup and password reset.
	// Possible values:
	//
	// 	- "very weak" (default)
	// 	- "weak"
	// 	- "medium"
	// 	- "strong"
	// 	- "very strong"
	PasswordStrengthLevel validation.PasswordStrengthLevel `yaml:"PasswordStrengthLevel"`
}

// LoadKeys loads the public keys from the identity server.
func (opts *Options) LoadKeys() (jwt.Keys, error) {
	jwksURL := opts.BaseURL + "/.well-known/jwks.json"
	keys, err := jwt.FetchPublicKeys(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch public keys: %v", err)
	}

	// If it's a server-side client application. Means that it contains an encryption key,
	// that we must use to transmit sensitive user data such as the password
	// and the token of the server app client (`Header`).
	if opts.EncryptionKey != "" {
		encryptionKey, err := hex.DecodeString(opts.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decode encryption key: %v", err)
		}

		// Set the encryption and decryption methods for the refresh token,
		// as the server app needs it.
		encrypt, decrypt, err := jwt.GCM(encryptionKey, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create encryption and decryption methods for the encryption token: %v", err)
		}
		for i, key := range keys {
			if key.ID == auth.KIDRefresh {
				key.Encrypt = encrypt
				key.Decrypt = decrypt
				keys[i] = key
			}
		}
	}

	return keys, nil
}

// SDK is the identity SDK.
// It's a client for the identity client's and server's API.
type SDK[T any] struct { // TODO: generic T of any User, the only thing is that we must document the Username and Password fields are required for signup.
	opts *Options

	keys jwt.Keys

	Claims *ClientTokenClaims
	Client *client.Client

	adminOption client.RequestOption

	// Usage:
	// 	encryptedPassword, err := sdk.EncryptPassword("my-password", opts.EncryptionKey) // the same hex key given to `New` through `Options.EncryptionKey`.
	//  mySDK.OAuth2Config.PasswordCredentialsToken(mySDK.ClientContext(context.Background()), "my-username", encryptedPassword)
	OAuth2Config *oauth2.Config
}

// Header is the header key to send the client token.
// Without the Bearer prefix.
const Header = "X-Token"

// New creates a new Identity SDK with the provided options.
// It automatically loads the public keys from the identity server and verifies the client token.
func New[T any](opts Options) (*SDK[T], error) {
	keys, err := opts.LoadKeys()
	if err != nil {
		return nil, err
	}

	if opts.PasswordStrengthLevel == "" {
		opts.PasswordStrengthLevel = validation.PasswordVeryWeak
	}

	var clientClaims ClientTokenClaims
	if err = keys.VerifyToken([]byte(opts.Token), &clientClaims); err != nil {
		return nil, fmt.Errorf("failed to verify client token: %v", err)
	}

	// Check if T type contains a json tagged "password" field, and if so,
	// error it so that the user knows that the SDK is not meant to be used with such type.
	if hasJSONTagField[T]("password") {
		return nil, fmt.Errorf("the provided type %T contains a 'password' field, "+
			"the SDK is not meant to be used with such type, please use a custom type that does not contain a 'password' field", *new(T))
	}

	sdk := &SDK[T]{
		opts:        &opts,
		keys:        keys,
		Claims:      &clientClaims,
		Client:      client.New(client.BaseURL(opts.BaseURL)),
		adminOption: client.RequestHeader(true, Header, opts.Token),

		OAuth2Config: &oauth2.Config{
			ClientID:     clientClaims.ClientID,
			ClientSecret: clientClaims.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:       opts.BaseURL + "/oauth2/authorize",
				DeviceAuthURL: opts.BaseURL + "/oauth2/device/code",
				TokenURL:      opts.BaseURL + "/oauth2/token",
			},
			Scopes: clientClaims.Scopes,
			// RedirectURL is handled by the server, not the client.
		},
	}
	return sdk, nil
}

// HasJSONTagField checks if the type T has a field with the given JSON tag.
func hasJSONTagField[T any](tag string) bool {
	var t T
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return false
	}
	for sf := range typ.Fields() {
		jsonTag := sf.Tag.Get("json")
		if jsonTag == tag || (len(jsonTag) > len(tag) && jsonTag[:len(tag)+1] == tag+",") {
			return true
		}
	}
	return false
}

// ClientContext returns a new context with the client's HTTP client.
// Useful for custom oauth2-config based clients.
func (s *SDK[T]) ClientContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, s.Client)
}

// Verify decodes and verifies the token and returns its claims as Claims.
// The claims type is independent of the SDK's user type T, which is why it
// is a method type parameter.
func (s *SDK[T]) Verify[Claims any](token []byte, validators ...jwt.TokenValidator) (Claims, error) {
	var claims Claims
	err := s.keys.VerifyToken(token, &claims, validators...)
	return claims, err
}

// AdminGetUserSchema retrieves the user schema from the identity server.
func (s *SDK[T]) AdminGetUserSchema(ctx context.Context) ([]*identity.UserAttribute, error) {
	attrs, err := s.Client.BindJSON[[]*identity.UserAttribute](ctx, http.MethodGet, "/u/schema", nil, s.adminOption)
	if err != nil {
		return nil, fmt.Errorf("failed to get user schema: %w", err)
	}

	return attrs, nil
}

// AdminUserSignup signs up a user with the provided username, plain password, and any custom attributes.
// The User type of T must not contain a "password" field (initialization step).
func (s *SDK[T]) AdminUserSignup(ctx context.Context, user T, password string) (*oauth2.Token, error) {
	if err := validation.String("password", password).Password(s.opts.PasswordStrengthLevel).Get(); err != nil {
		return nil, err
	}

	encryptedPassword, err := EncryptPassword(password, s.opts.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}

	userBytes, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user: %v", err)
	}

	req, err := types.MergeJSON(userBytes, []byte(`{"password": "`+encryptedPassword+`"}`))
	if err != nil {
		return nil, fmt.Errorf("failed to merge user and password: %v", err)
	}

	resp, err := s.Client.BindJSON[oauth2.Token](ctx, http.MethodPost, "/u/signup", req, s.adminOption)
	return &resp, err
}

// AdminBulkInsertUsers inserts multiple users with their plain passwords.
// The User type of T must not contain a "password" field (initialization step).
func (s *SDK[T]) AdminBulkInsertUsers(ctx context.Context, users []T, usersPlainpasswords []string) (identity.BulkInsertUsersResponse, error) {
	if len(users) != len(usersPlainpasswords) {
		return identity.BulkInsertUsersResponse{}, fmt.Errorf("the number of users and passwords must be the same")
	}

	var req []map[string]any
	for i, user := range users {
		if err := validation.String("password", usersPlainpasswords[i]).Password(s.opts.PasswordStrengthLevel).Get(); err != nil {
			return identity.BulkInsertUsersResponse{}, fmt.Errorf("password validation failed for user at index %d: %w", i, err)
		}

		encryptedPassword, err := EncryptPassword(usersPlainpasswords[i], s.opts.EncryptionKey)
		if err != nil {
			return identity.BulkInsertUsersResponse{}, fmt.Errorf("failed to encrypt password for user at index %d: %w", i, err)
		}

		userBytes, err := json.Marshal(user)
		if err != nil {
			return identity.BulkInsertUsersResponse{}, fmt.Errorf("failed to marshal user at index %d: %v", i, err)
		}

		userMap := make(map[string]any, len(users))
		if err := json.Unmarshal(userBytes, &userMap); err != nil {
			return identity.BulkInsertUsersResponse{}, fmt.Errorf("failed to unmarshal user bytes to map at index %d: %v", i, err)
		}

		userMap["password"] = encryptedPassword
		req = append(req, userMap)
	}

	return s.Client.BindJSON[identity.BulkInsertUsersResponse](ctx, http.MethodPost, "/u/bulk", req, s.adminOption)
}

// UserSignin signs in a user by username and plain password.
func (s *SDK[T]) UserSignin(ctx context.Context, username, plainPassword string) (*oauth2.Token, error) {
	encryptedPassword, err := EncryptPassword(plainPassword, s.opts.EncryptionKey)
	if err != nil {
		return nil, err
	}

	tok, err := s.OAuth2Config.PasswordCredentialsToken(s.ClientContext(ctx), username, encryptedPassword)
	if err != nil {
		return nil, err
	}

	return tok, err
	// return jwt.NewTokenPair([]byte(tok.AccessToken), []byte(tok.RefreshToken)), nil
}

// AdminUserSigninRequest is the request body for the admin signin endpoint.
type AdminUserSigninRequest struct {
	Identifier string `json:"identifier"` // ID or Username.
}

// AdminUserSignin signs in an admin user by id or username.
func (s *SDK[T]) AdminUserSignin(ctx context.Context, req AdminUserSigninRequest) (*oauth2.Token, error) {
	resp, err := s.Client.BindJSON[oauth2.Token](ctx, http.MethodPost, "/u/signin", req, s.adminOption)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Introspect decodes the token locally (no verification, no network) and
// returns its claims as Claims.
// Usage:
//
//	user, err := sdk.Introspect[MyUser](tok.AccessToken) // tok is the *oauth2.Token result of SDK.UserSignin.
//	if err != nil {
//		log.Fatalf("failed to introspect token: %v", err)
//	}
//	log.Printf("introspect token: %#+v", user)
func (s *SDK[T]) Introspect[Claims any](token string) (Claims, error) {
	var claims Claims

	tok, err := jwt.Decode([]byte(token)) // no verification or any other checks here, we just want to extract its data.
	if err != nil {
		return claims, err
	}

	err = tok.Claims(&claims)
	return claims, err
}

// TokenIntrospect fires a POST request to the identity server to introspect a
// token and returns the token's data decoded into Claims.
func (s *SDK[T]) TokenIntrospect[Claims any](ctx context.Context, accessToken string) (Claims, error) {
	var tokenIntrospectRequest = struct {
		AccessToken string `json:"access_token"`
	}{
		AccessToken: accessToken,
	}

	return s.Client.BindJSON[Claims](ctx, http.MethodPost, "/oauth2/token/introspect", tokenIntrospectRequest)
}

// AdminDeleteUserRequest is the request body for the admin delete user endpoint.
type AdminDeleteUserRequest struct {
	Identifier string `json:"identifier"` // ID or Username.
	// Soft marks the user as deleted but keeps on the database, otherwise it will be a hard delete.
	Soft bool `json:"soft,omitzero"`
}

// AdminDeleteUser fires a DELETE request to the identity server to delete a user by ID or Username.
func (s *SDK[T]) AdminDeleteUser(ctx context.Context, req AdminDeleteUserRequest) (bool, error) {
	resp, err := s.Client.Do(ctx, http.MethodDelete, "/u", req, s.adminOption)
	if err != nil {
		return false, err
	}
	defer s.Client.DrainResponseBody(resp)

	if resp.StatusCode == http.StatusNoContent {
		return true, nil
	} // else not modified.

	return false, nil
}

type BulkIdentifierRequest struct {
	IDs []string `json:"ids"` // IDs of the users.
}

type BulkUserDeleteRequest struct {
	BulkIdentifierRequest
	Soft bool `json:"soft,omitzero"` // when true the users are not actually deleted, they are still there but marked as deleted and cannot be signed-in.
}

// AdminBulkDeleteUsers fires a DELETE request to the identity server to delete multiple users by ids.
func (s *SDK[T]) AdminBulkDeleteUsers(ctx context.Context, req BulkUserDeleteRequest) (int64, error) {
	resp, err := s.Client.BindJSON[irisContext.CountResponse[int64]](ctx, http.MethodDelete, "/u/bulk", req, s.adminOption)
	if err != nil {
		return 0, err
	}

	return resp.Count, nil
}

type AdminRestoreUserRequest struct {
	Identifier string `json:"identifier"` // ID or Username.
}

// AdminRestoreUser fires a POST request to the identity server to restore a deleted user by ID or Username.
func (s *SDK[T]) AdminRestoreUser(ctx context.Context, req AdminRestoreUserRequest) (bool, error) {
	resp, err := s.Client.Do(ctx, http.MethodPost, "/u/restore", req, s.adminOption)
	if err != nil {
		return false, err
	}
	defer s.Client.DrainResponseBody(resp)

	if resp.StatusCode == http.StatusNoContent {
		return true, nil
	} // else not modified.

	return false, nil
}

type AdminResetUserPasswordRequest struct {
	Identifier string `json:"identifier"` // ID or Username.
	Password   string `json:"password"`   // new plain password.
}

// AdminResetUserPassword resets the password of a user by ID or Username.
func (s *SDK[T]) AdminResetUserPassword(ctx context.Context, req AdminResetUserPasswordRequest) (bool, error) {
	if err := validation.String("password", req.Password).Password(s.opts.PasswordStrengthLevel).Get(); err != nil {
		return false, err
	}

	encryptedPassword, err := EncryptPassword(req.Password, s.opts.EncryptionKey)
	if err != nil {
		return false, err
	}

	var requestPasswordResetRequest = struct {
		Identifier string `json:"identifier"` // ID or Username.
	}{Identifier: req.Identifier}

	type tokenResponse struct {
		Token string `json:"token"`
	}
	tok, err := s.Client.BindJSON[tokenResponse](ctx, http.MethodPost, "/u/request-password-reset", requestPasswordResetRequest, s.adminOption)
	if err != nil {
		return false, err
	}

	var passwordResetRequest = struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"` // client-encrypted password.
	}{
		Token:       tok.Token,
		NewPassword: encryptedPassword,
	}

	resp, err := s.Client.Do(ctx, http.MethodPost, "/u/confirm-password-reset", passwordResetRequest, s.adminOption)
	if err != nil {
		return false, err
	}
	defer s.Client.DrainResponseBody(resp)

	if resp.StatusCode == http.StatusNoContent {
		return true, nil
	} // else not modified.

	return false, nil
}

// AdminListUsers fires a POST request to the identity server to list all users with pagination.
// The server also accepts QUERY (RFC 10008), but proxies such as the Azure Container Apps ingress reject it, so the SDK sends POST.
// It accepts a `page` and a `filter` options.
func (s *SDK[T]) AdminListUsers(ctx context.Context, page irisContext.PageOptions, filter *entity.UserFilterOptions) (*irisContext.Page[T], error) {
	resp, err := s.Client.BindJSON[irisContext.Page[T]](ctx, http.MethodPost, "/u/list", filter, s.adminOption, page.Bind)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// AdminUpdateUsers fires a PUT request to the identity server to update multiple users.
func (s *SDK[T]) AdminUpdateUsers(ctx context.Context, users []T, onlyColumns ...string) (irisContext.CountResponse[int64], error) {
	return s.Client.BindJSON[irisContext.CountResponse[int64]](ctx, http.MethodPut, "/u", users, s.adminOption, client.RequestQuery(url.Values{"columns": onlyColumns}))
}

// AdminUpdateUsersPartial fires a PATCH request to the identity server to perform partial updates on users.
// It supports updating both core table fields (username, password) and JSONB attrs fields.
// The password field, if present, will be automatically encrypted using the SDK's encryption key.
//
// Example Code:
//
//	updates := []entity.PartialUpdateSpec{
//	    {
//	        ID: "user-123",
//	        Set: map[string]any{
//	            "username": "new_username@example.com",
//	            "password": "newPlainPassword123", // Will be encrypted automatically
//	        },
//	    },
//	}
//
//	resp, err := identitySDK.AdminUpdateUsersPartial(context.Background(), updates)
//
//	if err != nil {
//	    log.Fatal(err)
//	}
func (s *SDK[T]) AdminUpdateUsersPartial(ctx context.Context, updates []entity.PartialUpdateSpec) (irisContext.CountResponse[int64], error) {
	// Process updates to encrypt any password fields
	processedUpdates := make([]entity.PartialUpdateSpec, len(updates))
	copy(processedUpdates, updates)

	for i, update := range processedUpdates {
		if passwordValue, hasPassword := update.Set["password"]; hasPassword {
			plainPassword, ok := passwordValue.(string)
			if !ok {
				return irisContext.CountResponse[int64]{}, fmt.Errorf("password at update index %d must be a string", i)
			}

			// Validate password strength
			if err := validation.String("password", plainPassword).Password(s.opts.PasswordStrengthLevel).Get(); err != nil {
				return irisContext.CountResponse[int64]{}, fmt.Errorf("password validation failed at update index %d: %w", i, err)
			}

			// Encrypt the password
			encryptedPassword, err := EncryptPassword(plainPassword, s.opts.EncryptionKey)
			if err != nil {
				return irisContext.CountResponse[int64]{}, fmt.Errorf("failed to encrypt password at update index %d: %w", i, err)
			}

			// Replace the plain password with the encrypted password
			processedUpdates[i].Set["password"] = encryptedPassword
		}
	}

	return s.Client.BindJSON[irisContext.CountResponse[int64]](ctx, http.MethodPatch, "/u/attrs", processedUpdates, s.adminOption)
}
