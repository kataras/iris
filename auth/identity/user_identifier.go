package identity

import (
	"context"
	"fmt"
	"strings"
	"uuid"

	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/rest/errors/validation"
)

// UserIdentifier represents a user identifier which can be either a UUID or a username/email.
type UserIdentifier struct {
	text string

	isUUID  bool
	isEmail bool // it's a username and it's also a valid email.
}

// String returns the string representation of the identifier.
func (id UserIdentifier) String() string {
	return id.text
}

// IsUUID reports whether the identifier is a UUID.
func (id UserIdentifier) IsUUID() bool {
	return id.isUUID
}

// IsUsername reports whether the identifier is a username or email (not a UUID).
func (id UserIdentifier) IsUsername() bool {
	return !id.isUUID
}

// IsEmail reports whether the identifier is a valid email address.
func (id UserIdentifier) IsEmail() bool {
	return id.isEmail
}

// ParseUserIdentifier parses the given text as an identifier.
// It checks if the text is a valid UUID or a username/email based on the usernameIsEmail flag.
func ParseUserIdentifier(text string, usernameIsEmail bool) (UserIdentifier, error) {
	if text == "" {
		return UserIdentifier{}, fmt.Errorf("identifier is required")
	}

	text = strings.ToLower(text)

	identifier := UserIdentifier{
		text: text,
	}

	_, uuidErr := uuid.Parse(text)
	if uuidErr != nil {
		// Not a UUID, validate as username.
		if usernameIsEmail {
			if !isValidEmail(text) {
				return identifier, fmt.Errorf("invalid identifier: %s: should be a valid uuid(%w) or valid email", text, uuidErr)
			} else {
				identifier.isEmail = true
			}
		} // else it's a valid simple username.
	}

	identifier.isUUID = uuidErr == nil
	return identifier, nil
}

// isValidEmail checks if the given email address is valid.
func isValidEmail(email string) bool {
	return validation.Email(email) == ""
}

type fetchJob[T any] func(context.Context, entity.FetchOptions, any) (T, error)

// FetchUser fetches a user based on the identifier.
func FetchUser[T any](ctx context.Context, opts entity.FetchOptions, identifier string, usernameIsEmail bool, job fetchJob[T]) (T, error) {
	ident, err := ParseUserIdentifier(identifier, usernameIsEmail)
	if err != nil {
		return *new(T), fmt.Errorf("%w: %w", entity.ErrInvalidArgument, err)
	}

	if ident.IsUUID() {
		opts.SetColumn("id")
	} else {
		opts.SetColumn("username")
	}
	return job(ctx, opts, ident.String())
}

type processJob[T any] func(context.Context, string) (T, error)

func toProcessJobithValue[T, V any](job func(context.Context, string, V) (T, error), value V) processJob[T] {
	return func(ctx context.Context, identifier string) (T, error) {
		return job(ctx, identifier, value)
	}
}

// ProcessUser processes the user identifier based on its type (UUID or username).
func ProcessUser[T any](ctx context.Context, identifier string, usernameIsEmail bool, jobWithID, jobWithUsername processJob[T]) (T, error) {
	ident, err := ParseUserIdentifier(identifier, usernameIsEmail)
	if err != nil {
		return *new(T), fmt.Errorf("%w: %w", entity.ErrInvalidArgument, err)
	}

	if ident.IsUUID() {
		return jobWithID(ctx, ident.String())
	}

	return jobWithUsername(ctx, ident.String())
}
