package identity

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/auth/identity/password"
	"github.com/kataras/iris/v14/auth/oauth2"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/middleware/cache/httpcache"
	"github.com/kataras/iris/v14/middleware/rate"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/rest/errors/validation"

	"github.com/kataras/jwt"
)

// credentialRateLimit allows 10 requests/second, burst 20, on sign-in and password reset.
var credentialRateLimit = rate.Limit(10, 20, rate.Options{})

// UserController is the API controller for the User entity.
type UserController struct {
	auth         *auth.Auth[entity.FlatUser]
	userProvider *UserProvider
	userSchema   *UserSchema

	cache *httpcache.Cache
}

// NewUserController creates and returns a new UserController API.
func NewUserController(auth *auth.Auth[entity.FlatUser], userProvider *UserProvider, userSchema *UserSchema, cache *httpcache.Cache) *UserController {
	return &UserController{
		auth:         auth,
		userProvider: userProvider,
		userSchema:   userSchema,

		cache: cache,
	}
}

// Routing registers the API routes.
// It implements the router.Controller interface.
func (c *UserController) Routing(r router.Router) {
	r.Router("/groups", NewUserGroupController)

	// caching := c.cache.HandlerWithRouteKey()
	// This will invalidates the cached-only handlers,
	// in this case, these prefixes:
	// httpcache:/u/signin:POST
	// httpcache:/u/{identifier:string}:GET
	// httpcache:/u/list:POST
	// httpcache:/u/list:QUERY
	// invalidating := c.cache.InvalidateRoutes(r)
	// A more generic which will invalidate all sub-routers too:
	// c.cache.InvalidateHandler("httpcache:"+r.GetRelPath())

	caching, invalidating := c.cache.Pair(r)

	r.Get("/schema", c.renderSchema)
	r.Post("/signup", invalidating, c.signup)
	r.Post("/bulk", invalidating, c.bulkInsertUsers)
	r.Post("/signin", credentialRateLimit, c.signin)
	r.Post("/refresh", c.refresh)
	r.Post("/enrich", c.enrichUserToken)
	r.Get("/{identifier:string}", caching, c.getUser)
	r.Delete("/", invalidating, c.deleteUser)
	r.Delete("/bulk", invalidating, c.deleteUsers)
	r.Post("/restore", invalidating, c.restoreDeletedUser)
	r.Post("/bulk/restore", invalidating, c.restoreDeletedUsers)
	r.Post("/request-password-reset", credentialRateLimit, c.requestPasswordReset) // request-password-reset is used to request a password reset token, which is sent to the user's identifier. Can be sent as a URL-encoded email.
	r.Post("/confirm-password-reset", credentialRateLimit, c.confirmPasswordReset) // confirm-password-reset is used to reset the password using a token sent to the user's identifier.
	r.Post("/reset-password", credentialRateLimit, c.resetPassword)                // force-reset.
	r.Put("/", invalidating, c.updateUsers)
	r.Patch("/attrs", invalidating, c.updateUsersPartial) // PATCH for partial updates of attrs.
	// QUERY (RFC 10008), and POST for SDKs built before it.
	r.Handle(router.MethodPost+" "+router.MethodQuery, "/list", caching, c.listUsers)
	r.Get("/count", c.countUsers)
}

func (c *UserController) renderSchema(ctx *context.Context) {
	// Render the user schema as JSON.
	// This is useful for clients to know what attributes are available and their types.
	ctx.OK(c.userSchema.Attrs, nil)
}

/* replaced with raw json (entity.FlatUser)
// UserSignupRequest is the request body for the signup endpoint.
type UserSignupRequest struct {
	Username string          `json:"username"`
	Password string          `json:"password"` // client-encrypted password.
	Attrs    json.RawMessage `json:"attrs"`
}
*/

func (c *UserController) signup(ctx *context.Context) {
	req, ok := ctx.BindJSON[entity.FlatUser]()
	if !ok {
		return
	}

	newUser, err := entity.ToUser(req)
	if err != nil {
		errors.InvalidArgument.Err(ctx, fmt.Errorf("invalid user data: %w", err))
		return
	}

	if errors.HandleRequestError(ctx, validation.Join(c.usernameValidator(newUser.Username), c.passwordValidator(newUser.Password))) {
		return
	}

	if err := c.userSchema.Validate(newUser.Attrs); err != nil {
		errors.InvalidArgument.Err(ctx, err)
		return
	}

	client, ok := oauth2.GetClient(ctx)
	if !ok {
		errors.Unauthenticated.Message(ctx, "client not found")
		return
	}

	if c.denyPrivilegedAttrs(ctx, client, newUser.Attrs) {
		return
	}

	tok, _, err := c.userProvider.Signup(ctx, client, newUser)
	ctx.Created(tok, err)
	// if ctx.Created(tok, err) {
	// 	c.userCache.CacheUser(id, newUser.Username, req) // cache the user in a go routine.
	// }
}

// BulkInsertUsersResponse represents the response for bulk user insertion
type BulkInsertUsersResponse struct {
	TotalProvided int64  `json:"total_provided"`
	TotalInserted int64  `json:"total_inserted"`
	Message       string `json:"message"`
}

func (c *UserController) bulkInsertUsers(ctx *context.Context) {
	flatUsers, ok := ctx.BindJSON[[]entity.FlatUser]()
	if !ok {
		return
	}

	if len(flatUsers) == 0 {
		errors.InvalidArgument.Message(ctx, "users array cannot be empty")
		return
	}

	client, ok := oauth2.GetClient(ctx)
	if !ok {
		errors.Unauthenticated.Message(ctx, "client not found")
		return
	}

	// Convert FlatUsers to Users and validate
	users := make([]entity.User, 0, len(flatUsers))
	for i, flatUser := range flatUsers {
		user, err := entity.ToUser(flatUser)
		if err != nil {
			errors.InvalidArgument.Detail(ctx, fmt.Sprintf("invalid user data at index %d", i), err.Error())
			return
		}

		// Validate username and password (if provided) - use same approach as signup
		if user.Password != "" {
			// Validate both username and password if password is provided
			if errors.HandleRequestError(ctx, validation.Join(c.usernameValidator(user.Username), c.passwordValidator(user.Password))) {
				return
			}
		} else {
			// Only validate username if password is empty (will be auto-generated)
			if errors.HandleRequestError(ctx, validation.Join(c.usernameValidator(user.Username))) {
				return
			}
		}

		// Validate user attributes against schema
		if err := c.userSchema.Validate(user.Attrs); err != nil {
			errors.InvalidArgument.Detail(ctx, fmt.Sprintf("invalid user attributes at index %d", i), err.Error())
			return
		}
		if c.denyPrivilegedAttrs(ctx, client, user.Attrs) {
			return
		}

		users = append(users, user)
	}

	// Perform bulk insert
	totalInserted, err := c.userProvider.BulkInsertUsers(ctx, users)
	if err != nil {
		errors.Internal.Err(ctx, fmt.Errorf("bulk insert failed: %w", err))
		return
	}

	response := BulkInsertUsersResponse{
		TotalProvided: int64(len(flatUsers)),
		TotalInserted: totalInserted,
		Message:       fmt.Sprintf("Successfully inserted %d out of %d users", totalInserted, len(flatUsers)),
	}

	ctx.Created(response, nil)
}

func (c *UserController) usernameValidator(username string) *validation.StringError {
	// min 3 chars.
	usernameValidation := validation.String("username", username).Length(3, 0)
	if c.userProvider.usernameIsEmail {
		usernameValidation.Email()
	}

	return usernameValidation
}

func (c *UserController) passwordValidator(encrypted string) *validation.StringError {
	// The body carries the AES-GCM ciphertext. The length rule applies to the plaintext.
	plain, err := password.Decrypt(encrypted, c.userProvider.encryptionKey)
	if err != nil {
		plain = ""
	}
	return validation.String("password", plain).Length(8, 0)
}

// IdentifierRequest is a request body that contains an identifier for user operations.
type IdentifierRequest struct {
	Identifier string `json:"identifier"` // id or username (email).

	// Compatibility with ID & Username.
	// If one is set it is used as identifier if that is missing.
	ID       string `json:"id,omitempty"`       // ID of the user.
	Username string `json:"username,omitempty"` // Username of the user, used only if ID is not set.
}

type BulkIdentifierRequest struct {
	IDs []string `json:"ids"` // IDs of the users.
}

func (req *BulkIdentifierRequest) GetIdentifiers() []string {
	return req.IDs
}

// GetIdentifier returns the identifier to be used for signing in.
func (req *IdentifierRequest) GetIdentifier() string {
	if req.Identifier != "" {
		return req.Identifier
	}
	if req.ID != "" {
		return req.ID
	}
	return req.Username
}

// UserSigninRequest is the request body for the signin endpoint.
type UserSigninRequest = IdentifierRequest

func (c *UserController) signin(ctx *context.Context) {
	req, ok := ctx.BindJSON[UserSigninRequest]()
	if !ok {
		return
	}

	client, ok := oauth2.GetClient(ctx)
	if !ok {
		errors.Unauthenticated.Message(ctx, "client not found")
		return
	}

	tok, err := c.userProvider.SigninByIdentifier(ctx, client, req.GetIdentifier())
	ctx.OK(tok, err)
}

func (c *UserController) refresh(ctx *context.Context) {
	req, ok := ctx.BindJSON[auth.RefreshRequest]()
	if !ok {
		return
	}

	client, ok := oauth2.GetClient(ctx)
	if !ok {
		errors.Unauthenticated.Message(ctx, "client not found")
		return
	}

	flatUser, _, err := c.auth.Verify(ctx, []byte(req.RefreshToken))
	if err != nil {
		errors.Unauthenticated.Detail(ctx, "invalid refresh token", err.Error())
		return
	}

	userID, ok, err := entity.GetIDFromFlatUser(flatUser)
	if err != nil {
		errors.Unauthenticated.Detail(ctx, "failed to get user ID from claims", err.Error())
		return
	}
	if !ok {
		errors.Unauthenticated.Message(ctx, "user id is missing from claims")
		return
	}

	tok, err := c.userProvider.SigninByIdentifier(ctx, client, userID)
	ctx.OK(tok, err)
}

type (
	UserTokenEnrichmentRequest struct {
		AccessToken string         `json:"access_token"`
		ExtraClaims map[string]any `json:"extra_claims"` // additional claims to be added to the token.
		// No, let's do it freely as we don't actually know what the client will send (yet).
		// Optional: specify which claims are allowed to be added.
		// AllowedClaims []string `json:"allowed_claims,omitempty"` // if empty, all claims are allowed.
	}

	TokenEnrichmentResponse struct {
		AccessToken string `json:"access_token"`
	}
)

func (c *UserController) enrichUserToken(ctx *context.Context) {
	req, ok := ctx.BindJSON[UserTokenEnrichmentRequest]()
	if !ok {
		return
	}

	if len(req.ExtraClaims) == 0 {
		errors.InvalidArgument.Message(ctx, "extra claims must be a valid JSON object")
		return
	}

	claimNames := make([]string, 0, len(req.ExtraClaims))
	for name := range req.ExtraClaims {
		claimNames = append(claimNames, name)
	}
	if c.denyPrivilegedClaims(ctx, claimNames) {
		return
	}

	restrictedColumns := append(RestrictedUserUpdateColumns, "username")
	for k := range req.ExtraClaims {
		if slices.Contains(restrictedColumns, k) {
			errors.InvalidArgument.Detail(ctx, "extra claims cannot contain restricted columns", fmt.Sprintf("column '%s' is restricted", k))
			return
		}
	}

	req.ExtraClaims["enriched"] = true                  // mark the token as enriched.
	req.ExtraClaims["enriched_at"] = jwt.Clock().Unix() // timestamp of when the token was enriched.

	newAccessToken, err := c.userProvider.enrichUserToken(req.AccessToken, req.ExtraClaims)
	ctx.OK(TokenEnrichmentResponse{AccessToken: string(newAccessToken)}, err)
}

// getUser retrieves a user by their identifier (ID or username) and
// returns their flattened user data from the database.
// Use oauth2/token/introspect to get the user data from the token.
func (c *UserController) getUser(ctx *context.Context) {
	identifier := ctx.Params().GetString("identifier")

	user, err := c.userProvider.GetUserFlattened(ctx, identifier, false, true)
	ctx.OK(user, err)
}

// UserDeleteRequest is the request body for the delete endpoint.
type UserDeleteRequest struct {
	IdentifierRequest

	// When true the user is not actually deleted, is still there but marked as deleted and cannot be signed-in.
	Soft bool `json:"soft,omitzero"`
}

func (c *UserController) deleteUser(ctx *context.Context) {
	req, ok := ctx.BindJSON[UserDeleteRequest]()
	if !ok {
		return
	}

	deleted, err := c.userProvider.DeleteUser(ctx, req.GetIdentifier(), req.Soft)
	ctx.NoContent(deleted, err)
}

type BulkUserDeleteRequest struct {
	BulkIdentifierRequest
	Soft bool `json:"soft,omitzero"` // when true the users are not actually deleted, they are still there but marked as deleted and cannot be signed-in.
}

func (c *UserController) deleteUsers(ctx *context.Context) {
	req, ok := ctx.BindJSON[BulkUserDeleteRequest]()
	if !ok {
		return
	}

	count, err := c.userProvider.BulkDeleteUsers(ctx, req.GetIdentifiers(), req.Soft)
	ctx.Count(count, err)
}

// UserRestoreRequest is the request body for the restore endpoint.
type UserRestoreRequest = IdentifierRequest

func (c *UserController) restoreDeletedUser(ctx *context.Context) {
	req, ok := ctx.BindJSON[UserRestoreRequest]()
	if !ok {
		return
	}
	restored, err := c.userProvider.RestoreUser(ctx, req.GetIdentifier())
	ctx.NoContent(restored, err)
}

func (c *UserController) restoreDeletedUsers(ctx *context.Context) {
	req, ok := ctx.BindJSON[BulkIdentifierRequest]()
	if !ok {
		return
	}
	allRestored, err := c.userProvider.RestoreUsers(ctx, req.GetIdentifiers())
	ctx.NoContent(allRestored, err)
}

// NOTE: maybe change the structure
// to a list of []Object{"identifier": "id_or_username", "attrs":{...}}
// So the "identifier" can be used to update a user by ID or Username[email].
// Otherwise we leave it as it's so the schema is the same as the get requests.
func (c *UserController) updateUsers(ctx *context.Context) {
	req, err := ctx.Body()
	if err != nil {
		errors.InvalidArgument.Detail(ctx, "unable to parse request body", err.Error())
		return
	}

	var flatUsers []entity.FlatUser
	if err = json.Unmarshal(req, &flatUsers); err != nil {
		if strings.Contains(err.Error(), "cannot unmarshal array") { // it's a single object.
			var flatUser entity.FlatUser
			if err = json.Unmarshal(req, &flatUser); err != nil {
				errors.InvalidArgument.Detail(ctx, "unable to parse request body", err.Error())
				return
			}
			flatUsers = []entity.FlatUser{flatUser} // convert to a slice.
		}
	}

	client, ok := oauth2.GetClient(ctx)
	if !ok {
		errors.Unauthenticated.Message(ctx, "client not found")
		return
	}

	usersToUpdate := make([]entity.UpdateableUser, 0, len(flatUsers))
	for _, flatUser := range flatUsers {
		fullUser, err := entity.ToUser(flatUser)
		if err != nil {
			errors.InvalidArgument.Detail(ctx, "unable to parse user data", err.Error())
			return
		}

		user := entity.UpdateableUser{
			ID:       fullUser.ID,
			Username: fullUser.Username,
			Attrs:    fullUser.Attrs,
		}

		if user.ID == "" || user.Username == "" {
			errors.InvalidArgument.Message(ctx, "user ID and username must be provided")
			return
		}

		if err := c.userSchema.Validate(user.Attrs); err != nil {
			errors.InvalidArgument.Err(ctx, fmt.Errorf("%s: %w", user.Username, err))
			return
		}
		if c.denyPrivilegedAttrs(ctx, client, user.Attrs) {
			return
		}

		usersToUpdate = append(usersToUpdate, user)
	}

	// columns := ctx.Query().GetSlice("columns")
	columns := []string{"username", "attrs"} // always update these columns, no id, no created_at, no updated_at, no deleted_at, no otp codes.
	rowsAffected, err := c.userProvider.UpdateUsers(ctx, usersToUpdate, columns...)
	ctx.Count(rowsAffected, err)
}

// updateUsersPartial handles partial updates to the core fields and attrs JSONB column.
func (c *UserController) updateUsersPartial(ctx *context.Context) {
	updates, ok := ctx.BindJSON[[]entity.PartialUpdateSpec]()
	if !ok {
		return
	}

	if len(updates) == 0 {
		errors.InvalidArgument.Message(ctx, "updates array cannot be empty")
		return
	}

	client, ok := oauth2.GetClient(ctx)
	if !ok {
		errors.Unauthenticated.Message(ctx, "client not found")
		return
	}

	// Validate each update specification.
	for i, update := range updates {
		// Ensure either ID or Username is provided.
		if update.ID == "" && update.Username == "" {
			errors.InvalidArgument.Detail(ctx, fmt.Sprintf("update at index %d must provide either ID or Username", i), "missing user identifier")
			return
		}

		// Ensure at least one operation is specified.
		if len(update.Set) == 0 && len(update.Remove) == 0 {
			errors.InvalidArgument.Detail(ctx, fmt.Sprintf("update at index %d has no operations", i), "must specify Set or Remove")
			return
		}

		names := make([]string, 0, len(update.Set)+len(update.Remove))
		for path := range update.Set {
			names = append(names, path)
		}
		names = append(names, update.Remove...)
		if c.denyPrivilegedWrite(ctx, client, names) {
			return
		}

		/* We can't validate against the schema because it may contain only partial attributes.
		// Validate PartialAttrs against schema if provided.
		if len(update.PartialAttrs) > 0 {
			partialAttrsJSON, err := json.Marshal(update.PartialAttrs)
			if err != nil {
				errors.InvalidArgument.Detail(ctx, fmt.Sprintf("invalid PartialAttrs at index %d", i), err.Error())
				return
			}

			// Validate partial attributes against the user schema.
			if err := c.userSchema.Validate(partialAttrsJSON); err != nil {
				errors.InvalidArgument.Detail(ctx, fmt.Sprintf("schema validation failed for PartialAttrs at index %d", i), err.Error())
				return
			}
		}

		// Validate Set values against schema if they represent complete attribute objects.
		for path, value := range update.Set {
			// For top-level attribute changes, validate against schema
			if !strings.Contains(path, ".") {
				valueJSON, err := json.Marshal(map[string]any{path: value})
				if err != nil {
					errors.InvalidArgument.Detail(ctx, fmt.Sprintf("invalid Set value for '%s' at index %d", path, i), err.Error())
					return
				}

				if err := c.userSchema.Validate(valueJSON); err != nil {
					errors.InvalidArgument.Detail(ctx, fmt.Sprintf("schema validation failed for Set '%s' at index %d", path, i), err.Error())
					return
				}
			}
		}
		*/
	}

	rowsAffected, err := c.userProvider.UpdateUsersPartial(ctx, updates)
	ctx.Count(rowsAffected, err)
}

type (
	// UserRequestPasswordResetRequest is the request body for the password reset request endpoint.
	UserRequestPasswordResetRequest struct {
		IdentifierRequest
		MaxAgeSeconds int64 `json:"max_age_seconds,omitzero"` // optional, max age of the token in seconds, defaults to 1 hour (3600).
	}
	// UserConfirmPasswordResetRequest is the request body for the password reset confirmation endpoint.
	UserConfirmPasswordResetRequest struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"` // client-encrypted password.
	}

	UserPasswordResetRequest struct {
		IdentifierRequest
		NewPassword string `json:"new_password"` // client-encrypted password.
	}
)

// TODO: here idea: add a webhook on password reset, so the server-side can send a customized email or SMS or anything
// to the user with the token. which fires a request to the reset-password endpoint.
func (c *UserController) requestPasswordReset(ctx *context.Context) {
	req, ok := ctx.BindJSON[UserRequestPasswordResetRequest]()
	if !ok {
		return
	}

	client, ok := oauth2.GetClient(ctx)
	if !ok {
		errors.Unauthenticated.Message(ctx, "client not found")
		return
	}

	maxAgeSeconds := req.MaxAgeSeconds
	if maxAgeSeconds <= 0 {
		maxAgeSeconds = 3600 // default to 1 hour.
	}

	token, err := c.userProvider.GeneratePasswordResetToken(ctx, client, req.GetIdentifier(), maxAgeSeconds)
	if errors.HandleError(ctx, err) {
		return
	}

	// resp := oauth2.Token{
	// 	AccessToken: token,
	// 	TokenType:   "bearer",
	// 	Expiry:      jwt.Clock().Add(time.Duration(maxAgeSeconds) * time.Second),
	// 	ExpiresIn:   maxAgeSeconds,
	// }
	// Send the token to the user's email.
	// For simplicity, we just return it in the response here.
	ctx.OK(map[string]string{"token": token}, nil)
}

func (c *UserController) confirmPasswordReset(ctx *context.Context) {
	req, ok := ctx.BindJSON[UserConfirmPasswordResetRequest]()
	if !ok {
		return
	}

	user, err := c.userProvider.VerifyPasswordResetToken(ctx, req.Token)
	if err != nil {
		errors.Unauthenticated.Message(ctx, "invalid or expired token")
		return
	}

	userID, ok, _ := entity.GetIDFromFlatUser(user)
	if !ok {
		errors.InvalidArgument.Detail(ctx, "invalid user data", "user ID not found in the token")
		return
	}

	updated, err := c.userProvider.ResetPassword(ctx, userID, req.NewPassword)
	ctx.NoContent(updated, err)
}

func (c *UserController) resetPassword(ctx *context.Context) {
	req, ok := ctx.BindJSON[UserPasswordResetRequest]()
	if !ok {
		return
	}

	updated, err := c.userProvider.ResetPassword(ctx, req.GetIdentifier(), req.NewPassword)
	ctx.NoContent(updated, err)
}

func (c *UserController) listUsers(ctx *context.Context) {
	ctx.Paginated(c.userProvider.ListUsersPaginatedFlattened)
}

type CountResponse struct {
	ActiveCount   int64 `json:"active_count"`
	InactiveCount int64 `json:"inactive_count"`
	TotalCount    int64 `json:"total_count"`
}

func (c *UserController) countUsers(ctx *context.Context) {
	active, inactive, err := c.userProvider.CountTotalUsers(ctx)
	ctx.OK(CountResponse{ActiveCount: active, InactiveCount: inactive, TotalCount: active + inactive}, err)
}
