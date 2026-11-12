package identity

import (
	stdContext "context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/auth/identity/password"
	"github.com/kataras/iris/v14/auth/oauth2"

	"github.com/kataras/iris/v14/middleware/jwt"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/rest/types"
)

// UserProvider is the user authentication and service provider.
type UserProvider struct {
	r    *entity.Registry
	auth *auth.Auth[entity.FlatUser]
	// cache                       *UserCache
	encryptionKey               string
	usernameIsEmail             bool
	restoreOnSamePasswordSignup bool
}

var (
	_ auth.Provider[entity.FlatUser] = (*UserProvider)(nil)
)

// NewUserProvider returns a new UserProvider.
func NewUserProvider(r *entity.Registry, authServer *auth.Auth[entity.FlatUser], encryptionKey string, usernameIsEmail, restoreOnSamePasswordSignup bool) *UserProvider {
	return &UserProvider{
		r:                           r,
		auth:                        authServer,
		encryptionKey:               encryptionKey,
		usernameIsEmail:             usernameIsEmail,
		restoreOnSamePasswordSignup: restoreOnSamePasswordSignup,
	}
}

func (s *UserProvider) InTransaction(ctx stdContext.Context, fn func(*UserProvider) error) error {
	if s.r.IsInTransaction() { // do not initialize a new service instance if not necessary.
		return fn(s)
	}

	return s.r.InTransaction(ctx, func(r *entity.Registry) error {
		newService := NewUserProvider(r, s.auth, s.encryptionKey, s.usernameIsEmail, s.restoreOnSamePasswordSignup)
		return fn(newService)
	})
}

// GetUser retrieves a user by ID or username/email.
func (s *UserProvider) GetUser(ctx stdContext.Context, identifier string, includePassword, includeDeleted bool) (entity.User, error) {
	opts := entity.FetchOptions{
		IncludePassword: includePassword,
		IncludeDeleted:  includeDeleted,
	}
	// flatUser, err := s.cache.GetUser(ctx, identifier, opts)
	// if err == nil {
	// 	// If the user is found in the cache, convert it to User and return.
	// 	user, err := entity.ToUser(flatUser)
	// 	if err != nil {
	// 		return entity.User{}, fmt.Errorf("failed to convert cached flat user to user: %w", err)
	// 	}
	// 	return user, nil
	// }

	user, err := FetchUser(ctx,
		opts,
		identifier,
		s.usernameIsEmail,
		s.r.Users().GetUser,
	)
	// if err == nil {
	// 	if flatUser, err := user.ToFlatUser(); err == nil {
	// 		// Cache the user if found.
	// 		s.cache.CacheUser(opts, user.ID, user.Username, flatUser)
	// 	}
	// }

	return user, err
}

// GetUserFlattened retrieves a user by ID or username/email in a flattened JSON format.
func (s *UserProvider) GetUserFlattened(ctx stdContext.Context, identifier string, includePassword, includeDeleted bool) (entity.FlatUser, error) {
	opts := entity.FetchOptions{
		IncludePassword: includePassword,
		IncludeDeleted:  includeDeleted,
	}

	flatUser, err := FetchUser(ctx,
		opts,
		identifier,
		s.usernameIsEmail,
		s.r.Users().GetUserFlattened,
	)

	return flatUser, err
}

// DeleteUser deletes a user by identifier (ID or username).
func (s *UserProvider) DeleteUser(ctx stdContext.Context, identifier string, soft bool) (bool, error) {
	// idJob := toProcessJobithValue(func(ctx stdContext.Context, id string, soft bool) (bool, error) {
	// 	ok, err := s.r.Users().DeleteUserByID(ctx, id, soft)
	// 	if ok {
	// 		s.cache.InvalidateUser(id, "", true) // delete from cache if exists.
	// 	}

	// 	return ok, err
	// }, soft)

	return ProcessUser(ctx,
		identifier,
		s.usernameIsEmail,
		toProcessJobithValue(s.r.Users().DeleteUserByID, soft),
		toProcessJobithValue(s.r.Users().DeleteUserByUsername, soft),
	)
}

func (s *UserProvider) BulkDeleteUsers(ctx stdContext.Context, ids []string, soft bool) (int64, error) {
	/*
		mutated := true

		err := s.InTransaction(ctx, func(p *UserProvider) error {
			for _, identifier := range identifiers {
				ok, err := ProcessUser(ctx,
					identifier,
					s.usernameIsEmail,
					toProcessJobithValue(s.r.Users().DeleteUserByID, soft),
					toProcessJobithValue(s.r.Users().DeleteUserByUsername, soft),
				)
				if err != nil {
					return fmt.Errorf("failed to delete user %s: %w", identifier, err)
				}
				if !ok {
					mutated = false
				}
			}

			return nil
		})
		if err != nil {
			return false, err
		}

		return mutated, err
	*/
	return s.r.Users().BulkDeleteUsers(ctx, ids, soft)
}

// RestoreUser restores a user by identifier (ID or username).
func (s *UserProvider) RestoreUser(ctx stdContext.Context, identifier string) (bool, error) {
	// idJob := func(ctx stdContext.Context, id string) (bool, error) {
	// 	ok, err := s.r.Users().RestoreUserByID(ctx, id)
	// 	if ok {
	// 		s.cache.InvalidateUser(id, "", true) // delete from cache if exists.
	// 	}
	// 	return ok, err
	// }

	return ProcessUser(ctx,
		identifier,
		s.usernameIsEmail,
		s.r.Users().RestoreUserByID,
		s.r.Users().RestoreUserByUsername,
	)
}

func (s *UserProvider) RestoreUsers(ctx stdContext.Context, identifiers []string) (bool, error) {
	mutated := true

	err := s.InTransaction(ctx, func(p *UserProvider) error {
		for _, identifier := range identifiers {
			ok, err := ProcessUser(ctx,
				identifier,
				p.usernameIsEmail,
				p.r.Users().RestoreUserByID,
				p.r.Users().RestoreUserByUsername,
			)
			if err != nil {
				return fmt.Errorf("failed to restore user %s: %w", identifier, err)
			}
			if !ok {
				mutated = false
			}
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	return mutated, err
}

// ListUsersPaginated returns a list of users based on the given page options and filter options.
func (s *UserProvider) ListUsersPaginated(ctx stdContext.Context, l entity.PageOptions, f *entity.UserFilterOptions) ([]entity.User, int64, error) {
	return s.r.Users().ListUsersPaginated(ctx, l, f)
}

// ListUsersPaginatedFlattened returns users as a slice of json.RawMessage.
func (s *UserProvider) ListUsersPaginatedFlattened(ctx stdContext.Context, l entity.PageOptions, f *entity.UserFilterOptions) ([]entity.FlatUser, int64, error) {
	flatUsers, totalCount, err := s.r.Users().ListUsersPaginatedFlattened(ctx, l, f)
	// if len(flatUsers) > 0 && err == nil {
	// 	s.cache.CacheUserList(f, &l, flatUsers, totalCount)
	// }
	return flatUsers, totalCount, err
}

// CountTotalUsers returns the total numbers of enabled and disabled users.
func (s *UserProvider) CountTotalUsers(ctx stdContext.Context) (int64, int64, error) {
	return s.r.Users().CountTotalUsers(ctx)
}

// GeneratePasswordResetToken generates a password reset token for the given id or email (username).
func (s *UserProvider) GeneratePasswordResetToken(ctx stdContext.Context, appClient *oauth2.Client, identifier string, maxAgeSeconds int64) (string, error) {
	user, err := s.GetUser(ctx, identifier, false, false)
	if err != nil {
		return "", fmt.Errorf("failed to generate password reset token: get user: %s: %w", identifier, err)
	}

	if maxAgeSeconds <= 0 {
		maxAgeSeconds = 3600 // default to 1 hour
	}

	// Create a token with a given expiration time.
	token, err := s.signUserToken(appClient, user, jwt.MaxAge(time.Duration(maxAgeSeconds)*time.Second))
	if err != nil {
		return "", fmt.Errorf("failed to generate password reset token: %w", err)
	}
	// Send the token to the user's email.
	// For simplicity, we just return it here.
	return token.AccessToken, nil
}

// ResetPassword resets the password for the given user.
func (s *UserProvider) ResetPassword(ctx stdContext.Context, identifier, encryptedNewPassword string) (bool, error) {
	plainNewPassword, err := password.Decrypt(encryptedNewPassword, s.encryptionKey)
	if err != nil {
		return false, err
	}

	hashedPassword, err := password.HashPassword(plainNewPassword)
	if err != nil {
		return false, err
	}

	/*
		ident, err := ParseUserIdentifier(identifier, s.usernameIsEmail)
		if err != nil {
			return false, fmt.Errorf("%w: %w", entity.ErrInvalidArgument, err)
		}

		if ident.IsUUID() {
			return s.r.Users().UpdatePasswordByID(ctx, identifier, hashedPassword)
		}

		return s.r.Users().UpdatePasswordByUsername(ctx, identifier, hashedPassword)
	*/

	return ProcessUser(ctx, identifier, s.usernameIsEmail,
		toProcessJobithValue(s.r.Users().UpdatePasswordByID, hashedPassword),
		toProcessJobithValue(s.r.Users().UpdatePasswordByUsername, hashedPassword),
	)
}

// VerifyPasswordResetToken verifies the password reset token and returns the associated user.
func (s *UserProvider) VerifyPasswordResetToken(ctx stdContext.Context, token string) (entity.FlatUser, error) {
	user, _, err := s.auth.Verify(ctx, []byte(token))
	return user, err
}

// ReservedUserFields contains field names that cannot be set in custom attributes
// because they are already part of the User entity structure.
var ReservedUserFields = []string{
	"id", "username", "password", "created_at", "updated_at", "password", "otp", "otp_recovery_codes",
}

// RestrictedUserUpdateColumns are the columns that are not allowed to be updated by the UpdateUsers method.
var RestrictedUserUpdateColumns = []string{
	"id", "created_at", "updated_at", "password", "otp", "otp_recovery_codes",
}

// UpdatableCoreFields are core table fields that can be updated through the set_paths API
var UpdatableCoreFields = map[string]bool{
	"username": true,
	"password": true,
}

// ProtectedFields are system-managed fields that cannot be updated by users
var ProtectedFields = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
	// "password":   true,
}

// checkMapContainsInvalidFields validates that map keys don't contain protected system fields
// but allows updatable core fields and custom attrs fields
func checkMapContainsInvalidFields(m map[string]any) error {
	for k := range m {
		if ProtectedFields[k] {
			return fmt.Errorf("%w: field '%s' is protected and cannot be updated", entity.ErrInvalidArgument, k)
		}
		// Note: We no longer prevent updatable core fields like username, password
		// They will be handled separately in the repository layer
	}
	return nil
}

// checkSliceContainsInvalidFields validates that remove paths don't target protected fields
func checkSliceContainsInvalidFields(paths []string) error {
	for _, p := range paths {
		// Check if path starts with a protected field
		for field := range ProtectedFields {
			if strings.HasPrefix(p, field) {
				return fmt.Errorf("%w: cannot remove protected field '%s' (path: '%s')", entity.ErrInvalidArgument, field, p)
			}
		}
		// Check if trying to remove updatable core fields (they should be updated, not removed from attrs)
		for field := range UpdatableCoreFields {
			if strings.HasPrefix(p, field) {
				return fmt.Errorf("%w: cannot remove core field '%s' from attrs (path: '%s') - use direct field update instead", entity.ErrInvalidArgument, field, p)
			}
		}
	}
	return nil
}

func (s *UserProvider) UpdateUsers(ctx stdContext.Context, updateableUsers []entity.UpdateableUser, columns ...string) (int64, error) {
	columns = slices.DeleteFunc(columns, func(s string) bool {
		// these are not allowed to be updated by this "UpdateUsers" function,
		// there are separate functions for some of them (e.g. reset password, enableOTP).
		return s == "" || slices.Contains(RestrictedUserUpdateColumns, s) // Remove restricted columns.
	})

	// Validate username is email if required
	if s.usernameIsEmail {
		for _, u := range updateableUsers {
			if u.Username != "" && !isValidEmail(u.Username) {
				return 0, fmt.Errorf("%w: invalid email format for username: %s", entity.ErrInvalidArgument, u.Username)
			}
		}
	}

	users := make([]entity.User, 0, len(updateableUsers))
	for _, u := range updateableUsers {
		users = append(users, u.ToUser())
		// s.cache.InvalidateUser(u.ID, u.Username, false) // TODO: it may contain both or only one and that's causing problems.Maybe think the identifier as a return the uuid if it's a username, instead of having both to manage in cache too.
	}

	affectedRows, err := s.r.Users().UpdateUsers(ctx, users, columns...)
	// if err == nil {
	// 	s.cache.InvalidateUserListCache()
	// }

	return affectedRows, err
}

// UpdateUsersPartial performs partial updates on the core feilds and 'attrs' JSONB column for multiple users.
// It supports merging new attributes, setting specific paths, and removing paths.
func (s *UserProvider) UpdateUsersPartial(ctx stdContext.Context, updates []entity.PartialUpdateSpec) (int64, error) {
	// Validate and process updates
	processedUpdates := make([]entity.PartialUpdateSpec, len(updates))
	copy(processedUpdates, updates)

	for i, update := range processedUpdates {
		if len(update.Set) == 0 && len(update.Remove) == 0 {
			return 0, fmt.Errorf("update specification at index %d has no updates", i)
		}

		// Ensure user identification is provided.
		if update.ID == "" && update.Username == "" {
			return 0, fmt.Errorf("update specification at index %d must provide either ID or Username", i)
		}

		// Validate paths in Set don't reference restricted fields.
		if err := checkMapContainsInvalidFields(update.Set); err != nil {
			return 0, fmt.Errorf("update specification at index %d: set: %w", i, err)
		}

		// Validate paths in Remove don't reference restricted fields.
		if err := checkSliceContainsInvalidFields(update.Remove); err != nil {
			return 0, fmt.Errorf("update specification at index %d: remove: %w", i, err)
		}

		// Process password field if present in Set - decrypt and hash it
		if passwordValue, hasPassword := update.Set["password"]; hasPassword {
			encryptedPassword, ok := passwordValue.(string)
			if !ok {
				return 0, fmt.Errorf("update specification at index %d: password must be a string", i)
			}

			// Decrypt the password using the encryption key
			plainPassword, err := password.Decrypt(encryptedPassword, s.encryptionKey)
			if err != nil {
				return 0, fmt.Errorf("update specification at index %d: failed to decrypt password: %w", i, err)
			}

			// Hash the decrypted password
			hashedPassword, err := password.HashPassword(plainPassword)
			if err != nil {
				return 0, fmt.Errorf("update specification at index %d: failed to hash password: %w", i, err)
			}

			// Replace the encrypted password with the hashed password
			processedUpdates[i].Set["password"] = hashedPassword
		}
	}

	affectedRows, err := s.r.Users().UpdateUsersPartial(ctx, processedUpdates)
	// if err == nil {
	// 	s.cache.InvalidateUserListCache()
	// 	// Optionally invalidate specific user caches if we track user IDs
	// 	for _, update := range updates {
	// 		s.cache.InvalidateUser(update.UserID, update.Username, false)
	// 	}
	// }

	return affectedRows, err
}

// BulkInsertUsers inserts multiple users in batches with automatic password generation and AES decryption support.
// If a user's password is empty, generates a strong random password and hashes it.
// If a user's password is AES-encrypted (hex format), attempts to decrypt it using the provider's shared key.
// Users are inserted in batches of 500 within separate transactions for optimal performance.
// Returns the total number of users successfully inserted and any error encountered.
func (s *UserProvider) BulkInsertUsers(ctx stdContext.Context, users []entity.User) (int64, error) {
	return s.r.Users().BulkInsertUsers(ctx, s.encryptionKey, users)
}

// signUserToken signs a user token with the given appClient and user.
func (s *UserProvider) signUserToken(appClient *oauth2.Client, user any, opts ...jwt.SignOption) (*oauth2.Token, error) {
	// user.ClearSensitiveFields()
	rawClaims, err := types.MergeJSON(appClient.Claims(UserAppScopes), user)
	if err != nil {
		return nil, fmt.Errorf("%w: merge JSON: %w", ErrSignToken, err)
	}

	accessTokenOption := appClient.AccessTokenOptions()
	refreshTokenOption := appClient.RefreshTokenOptions()

	opts = append([]jwt.SignOption{accessTokenOption, refreshTokenOption}, opts...)
	tok, err := s.auth.SignToken(rawClaims, opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSignToken, err)
	}

	return tok, nil
}

func (s *UserProvider) enrichUserToken(accessToken string, extraClaims any) ([]byte, error) {
	return s.auth.Keys().EnrichToken([]byte(accessToken), extraClaims)
}

// Signup accepts a username, an encrypted password and optional attributes and
// should return a valid OAuth2 Token which holds the User value or an error describing
func (s *UserProvider) Signup(ctx stdContext.Context, appClient *oauth2.Client, newUser entity.User) (*oauth2.Token, string, error) {
	plainPassword, err := password.Decrypt(newUser.Password, s.encryptionKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	hashedPassword, err := password.HashPassword(plainPassword)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}
	newUser.Password = hashedPassword

	/*
		// Maybe some time in the near future:
		s.r.InTransaction(ctx, func(r *entity.Registry) error {
			// generate OTP.
			key, img, err := GenerateOTP(newUser.ID, newUser.Email)
			if err != nil {
				return fmt.Errorf("generate user otp: %w", err)
			}

			recoveryCodes, err := r.SetUserOTP(ctx, newUser.ID, key.String())
			if err != nil {
				return fmt.Errorf("store user otp: %w", err)
			}
		})
	*/

	// Check if a deleted user with this username exists.
	existingUser, err := s.GetUser(ctx, newUser.Username, true, true)
	if err != nil && !errors.Is(err, entity.ErrNotFound) {
		return nil, "", err
	}

	userExists := existingUser.ID != ""

	var (
		tok    *oauth2.Token
		userID string
	)

	if userExists {
		if !password.ComparePassword(existingUser.Password, plainPassword) {
			// If the password doesn't match, we don't want to leak that the user is marked for deletion.
			// So we return an error indicating that the user already exists.
			if s.restoreOnSamePasswordSignup && existingUser.IsDeleted() {
				return nil, "", fmt.Errorf("%w (probably user marked as deleted but passwords are not match)", entity.ErrAlreadyExists)
			}

			return nil, existingUser.ID, entity.ErrAlreadyExists
		}

		if s.restoreOnSamePasswordSignup && existingUser.IsDeleted() {
			err = s.r.Users().InTransaction(ctx, func(users *entity.UserRepository) error {
				restored, restoreErr := users.RestoreUserAndAttrsByUsername(ctx, newUser.Username, newUser.Attrs)
				if restoreErr != nil {
					return fmt.Errorf("unable to restore a deleted user on signup: %w", restoreErr)
				}

				if !restored {
					// Means that for some reason we failed to restore the user.
					// This should never happen, but just in case.
					return entity.ErrAlreadyExists
				}

				existingUser.Attrs = newUser.Attrs
				existingUser.DeletedAt = types.ISO8601(time.Time{}) // zero-time.

				flatUser, err := existingUser.ToFlatUser()
				if err != nil {
					return fmt.Errorf("failed to convert existing user to flat: %w", err)
				}
				tok, err = s.signUserToken(appClient, flatUser)
				return err
			})
			if err != nil {
				return nil, "", err
			}
		} else {
			// User already exists and we are not allowed to try to restore it, return an error.
			return nil, existingUser.ID, entity.ErrAlreadyExists
		}
	} else {
		// User doesn't exist, we can create a new one and generate the token at the same transaction,
		// so if the gen token fails then no new user is created. Note that in restore we do exactly the same.

		// var newUser entity.User
		// newUser, err = entity.ToUser(attrs)
		// if err != nil {
		// 	return nil, "", fmt.Errorf("failed to parse user attributes: %w", err)
		// }
		// newUser.Username = username
		// newUser.Password = hashedPassword

		// existingUserID, _, _ := entity.GetIDFromFlatUser(attrs)
		// newUser := entity.User{
		// 	ID:       existingUserID,
		// 	Username: username,
		// 	Password: hashedPassword,
		// 	Attrs:    attrs,
		// }

		err = s.r.Users().InTransaction(ctx, func(users *entity.UserRepository) error {
			err := users.InsertSingle(ctx, newUser, &newUser.ID)
			if err != nil {
				return err
			}
			userID = newUser.ID

			now := types.ISO8601(time.Now())
			newUser.CreatedAt = now
			newUser.UpdatedAt = now

			flatUser, err := newUser.ToFlatUser()
			if err != nil {
				return fmt.Errorf("failed to convert existing user to flat: %w", err)
			}
			tok, err = s.signUserToken(appClient, flatUser)
			return err
		})
	}

	return tok, userID, err
}

// SigninByIdentifier accepts a username or ID and should return a valid oauth2 Token which holds the T value or an error describing
// the user authentication or verification reason of failure.
func (s *UserProvider) SigninByIdentifier(ctx stdContext.Context, appClient *oauth2.Client, identifier string) (*oauth2.Token, error) {
	user, err := s.GetUserFlattened(ctx, identifier, false, false)
	if err != nil {
		return nil, err
	}

	return s.signUserToken(appClient, user)
}

// Implements the auth.Provider[T].

var ErrInvalidUserCredentials = errors.New("invalid user credentials")

// Signin accepts a username (or email) and a password and should
// return a valid T value or an error describing
// the user authentication or verification reason of failure.
//
// The first input argument standard context can be
// casted to iris.Context if executed through Auth.SigninHandler.
//
// It's called on Auth.SigninHandler.
func (p *UserProvider) Signin(ctx stdContext.Context, username, encryptedPassword string) (entity.FlatUser, error) {
	plainPassword, err := password.Decrypt(encryptedPassword, p.encryptionKey)
	if err != nil {
		return nil, err
	}

	user, err := p.GetUser(ctx, username, true, false)
	if err != nil {
		return nil, err
	}

	if !password.ComparePassword(user.Password, plainPassword) {
		return nil, ErrInvalidUserCredentials
	}

	flatUser, err := user.ToFlatUser()
	if err != nil {
		return nil, fmt.Errorf("failed to convert user to flat: %w", err)
	}

	return flatUser, nil
}

// Refresh is mostly used to refresh the user information from a persistence storage.
// For example, if the user has changed their email or other attributes like roles.
func (p *UserProvider) Refresh(ctx stdContext.Context, t entity.FlatUser) (entity.FlatUser, error) {
	userID, ok, err := entity.GetIDFromFlatUser(t)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID from claims: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("user id is missing from claims")
	}

	fetchOpts := entity.FetchOptions{IncludePassword: false, IncludeDeleted: false, Column: "id"}
	return p.r.Users().GetUserFlattened(ctx, fetchOpts, userID)
}

// func (p *UserProvider) EnrichUserToken(ctx stdContext.Context, accessToken string, extraClaims map[string]any)

// ValidateToken accepts the standard JWT claims and the T value obtained
// by the Signin method and should return a nil error on validation success
// or a non-nil error for validation failure.
// It is mostly used to perform checks of the T value's struct fields or
// the standard claim's (e.g. origin jwt token id).
// It can be an empty method too which returns a nil error.
//
// The first input argument standard context can be
// casted to iris.Context if executed through Auth.VerifyHandler.
//
// It's caleld on Auth.VerifyHandler.
func (p *UserProvider) ValidateToken(ctx stdContext.Context, standardClaims auth.StandardClaims, t entity.FlatUser) error {
	return nil
}

// InvalidateToken is optional and can be used to allow tokens to be invalidated
// from server-side. Commonly, implement when a token and user pair is saved
// on a persistence storage and server can decide which token is valid or invalid.
// It can be an empty method too which returns a nil error.
//
// The first input argument standard context can be
// casted to iris.Context if executed through Auth.SignoutHandler.
//
// It's called on Auth.SignoutHandler.
func (p *UserProvider) InvalidateToken(ctx stdContext.Context, standardClaims auth.StandardClaims, t entity.FlatUser) error {
	return nil
}

// InvalidateTokens is like InvalidateToken but it should invalidate
// all tokens generated for a specific T value.
// It can be an empty method too which returns a nil error.
//
// The first input argument standard context can be
// casted to iris.Context if executed through Auth.SignoutAllHandler.
//
// It's called on Auth.SignoutAllHandler.
func (p *UserProvider) InvalidateTokens(ctx stdContext.Context, t entity.FlatUser) error {
	return nil
}
