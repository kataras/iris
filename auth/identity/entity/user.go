package entity

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/kataras/iris/v14/auth"

	"github.com/kataras/iris/v14/rest/types"
)

type (
	// RecoveryCode is the model that describes a recovery code.
	RecoveryCode struct {
		// Code is the recovery code.
		Code string `json:"code"`
		// Used is the flag that indicates if the recovery code was used.
		Used bool `json:"used"`
	}

	// Attrs an alias of the JSONB(json.RawMessage) type, it can be used to store any custom data.
	Attrs = json.RawMessage // map[string]any

	// User is the model that describes a user.
	User struct {
		// ID is the primary key of the user.
		ID string `pg:"type=uuid,primary" json:"id,omitempty"`
		// CreatedAt is the timestamp when the user was created.
		CreatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"created_at,omitempty"`
		// UpdatedAt is the timestamp when the user was last updated.
		// The updated_at is updated on each row update (see the trigger_set_timestamp sql function).
		UpdatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"updated_at,omitempty"`

		// Username is the unique username (or email) of the user.
		Username string `pg:"type=varchar(255),unique" json:"username"` // email or username.
		// Password is the encrypted password of the user.
		Password string `pg:"type=varchar(255)" json:"password,omitempty"` // encrypted password, omit for JSON & JWT responses.
		// OTP is the two-factor authentication key of the user.
		OTP string `pg:"type=varchar(255),default=''" json:"-"` //  otp key url (we can generate a new key based on this, secret lives here too).
		// OTPRecoveryCodes are the recovery codes of the user.
		OTPRecoveryCodes []RecoveryCode `pg:"type=jsonb,default='[]'::jsonb" json:"-"` // 4 recovery codes of 12 chars length that can be used only once (see identity.RecoveryCodesLength and identity.RecoveryCodeLength).
		// Attrs are the custom attributes of the user.
		// The custom attributes are stored as a JSONB column in the database.
		// With a default of {}, every row has an indexed JSONB value (even if empty),
		// so WHERE attrs ? 'foo' or WHERE attrs @> '{"foo":"bar"}' can use the GIN index consistently.
		Attrs Attrs `pg:"type=jsonb,default='{}',index=gin" json:"attrs,omitempty"` // custom attributes.

		DeletedAt types.ISO8601 `pg:"type=timestamp,nullable" json:"-"` // soft delete flag.
	}

	// FlatUser represents a flattened user structure for JSON output.
	// Custom profile images can be stored in the `photo_url` or `avatar_url` fields.
	// photo_url (snake_case)
	// photoUrl (camelCase)
	// avatar_url (alternative naming)
	// avatarUrl (alternative camelCase)
	FlatUser = json.RawMessage // it needs to be a shortcut and not a different type.
)

type AttrsBuilder struct {
	kv map[string]any
}

// NewAttrsBuilder creates a new AttrsBuilder.
func NewAttrsBuilder() *AttrsBuilder {
	return &AttrsBuilder{
		kv: make(map[string]any),
	}
}

func (b *AttrsBuilder) Set(key string, value any) *AttrsBuilder {
	b.kv[key] = value
	return b
}

// ToFlatUser converts the user to a FlatUser.
func (u User) ToFlatUser() (FlatUser, error) {
	// Remove sensitive fields on flat user, always, as this is a response for end-server and clients.
	u.Password = ""
	u.OTP = ""
	u.OTPRecoveryCodes = nil

	attrsCopy := make(Attrs, len(u.Attrs))
	copy(attrsCopy, u.Attrs)

	u.Attrs = nil // to remove the attrs field from the object.
	reservedUserBytes, err := json.Marshal(u)
	if err != nil {
		return nil, err
	}

	if len(attrsCopy) == 0 {
		// If the attrs are empty, we can return the reserved user bytes only.
		return reservedUserBytes, nil
	}

	// We use MergeJSON instead of having to unmarshal the user and attrs separated and build a map[string]any.
	// This is faster.
	return types.MergeJSON(reservedUserBytes, attrsCopy)
}

// GetIDFromFlatUser extracts the user ID from a FlatUser.
func GetIDFromFlatUser(fu FlatUser) (string, bool, error) {
	var value struct {
		ID string `json:"id"`
	}
	err := json.Unmarshal(fu, &value)
	return value.ID, value.ID != "", err
}

// GetUsernameFromFlatUser extracts the username from a FlatUser.
func GetUsernameFromFlatUser(fu FlatUser) (string, bool, error) {
	var value struct {
		Username string `json:"username"`
	}
	err := json.Unmarshal(fu, &value)
	return value.Username, value.Username != "", err
}

func GetIDAndUsernameFromFlatUser(fu FlatUser) (string, string, bool, error) {
	var value struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	err := json.Unmarshal(fu, &value)
	if err != nil {
		return "", "", false, err
	}
	return value.ID, value.Username, value.ID != "" && value.Username != "", nil
}

// GetPasswordFromFlatUser extracts the user password from a FlatUser.
func GetPasswordFromFlatUser(fu FlatUser) (string, bool, error) {
	var value struct {
		Password string `json:"password"`
	}
	err := json.Unmarshal(fu, &value)
	return value.Password, value.Password != "", err
}

// ToUser converts the flat user to a user.
// It is a slow operation and it's only used on some operations.
func ToUser(fu FlatUser) (User, error) { // slow-op.
	// We need a map because we can't have fields like id, created_at, updated_at, username, password, otp, otp_recovery_codes
	// on the attrs field, so we need to unmarshal the flat user to a map[string] first.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(fu, &m); err != nil {
		return User{}, fmt.Errorf("failed to unmarshal flat user to map: %v", err)
	}

	var (
		u            User
		userAttrsMap map[string]json.RawMessage
	)

	if emailValue, ok := m["email"]; ok {
		var email string

		if usernameValue, hasUsername := m["username"]; hasUsername {
			if err := json.Unmarshal(usernameValue, &u.Username); err != nil {
				return User{}, fmt.Errorf("failed to unmarshal username: %v", err)
			}
		}

		// Special case for email, we treat it as username if username is empty and we remove it from attrs.
		if err := json.Unmarshal(emailValue, &email); err != nil {
			return User{}, fmt.Errorf("failed to unmarshal email to username: %v", err)
		}

		if u.Username == "" {
			u.Username = email
		}

		// Other case is both username and email are present and they are equal, they both the same email,
		// then we remove the "email" from attrs.
		if u.Username == email {
			// remove email from the map if Username was empty or Username and email are the same,
			//  we don't want it in the attrs field.
			delete(m, "email")
		}
	}

	for k, v := range m {
		switch k {
		case "id":
			if err := json.Unmarshal(v, &u.ID); err != nil {
				return User{}, fmt.Errorf("failed to unmarshal field %s: %v", k, err)
			}
		case "created_at":
			err := json.Unmarshal(v, &u.CreatedAt) // we know that v is a string, so we can safely convert it.
			if err != nil {
				return User{}, fmt.Errorf("failed to unmarshal field %s: %v", k, err)
			}
		case "updated_at":
			err := json.Unmarshal(v, &u.UpdatedAt) // we know that v is a string, so we can safely convert it.
			if err != nil {
				return User{}, fmt.Errorf("failed to unmarshal field %s: %v", k, err)
			}
		case "deleted_at":
			if err := json.Unmarshal(v, &u.DeletedAt); err != nil {
				return User{}, fmt.Errorf("failed to unmarshal field %s: %v", k, err)
			}
		case "username":
			if u.Username == "" {
				if err := json.Unmarshal(v, &u.Username); err != nil {
					return User{}, fmt.Errorf("failed to unmarshal field %s: %v", k, err)
				}
			}
		case "password":
			if err := json.Unmarshal(v, &u.Password); err != nil {
				return User{}, fmt.Errorf("failed to unmarshal field %s: %v", k, err)
			}
		case "otp":
			if err := json.Unmarshal(v, &u.OTP); err != nil {
				return User{}, fmt.Errorf("failed to unmarshal field %s: %v", k, err)
			}
		case "otp_recovery_codes":
			var codes []RecoveryCode
			if err := json.Unmarshal(v, &codes); err != nil {
				return User{}, fmt.Errorf("failed to unmarshal field %s: %v", k, err)
			}
			u.OTPRecoveryCodes = codes
		default:
			// If the key is not one of the reserved fields, we assume it's a custom attribute.
			// We can safely assign it to the Attrs field.
			if userAttrsMap == nil {
				userAttrsMap = make(map[string]json.RawMessage)
			}

			// We need to unmarshal the value to json.RawMessage to keep the original type.
			// This is because the value can be of any type, and we need to keep it as is.
			if _, ok := userAttrsMap[k]; !ok {
				// If the key does not exist in the userAttrsMap, we add it.
				userAttrsMap[k] = v // we can safely assign it, as v is a json.RawMessage.
			}
		}
	}

	if len(userAttrsMap) > 0 {
		// If we have custom attributes, we need to marshal them to json.RawMessage.
		attrsBytes, err := json.Marshal(userAttrsMap)
		if err != nil {
			return User{}, fmt.Errorf("failed to marshal user attributes: %v", err)
		}
		u.Attrs = attrsBytes // assign the marshaled attributes to the user.
	}

	return u, nil
}

// UpdateableUser is the model that describes an updateable user.
type UpdateableUser struct {
	ID       string          `json:"id"`
	Username string          `json:"username"`
	Attrs    json.RawMessage `json:"attrs"`
}

// ToUser converts the updateable user to a user.
func (u UpdateableUser) ToUser() User {
	return User{
		ID:       u.ID,
		Username: u.Username,
		Attrs:    u.Attrs,
	}
}

// ClearSensitiveFields clears the sensitive fields of the user.
func (u *User) ClearSensitiveFields() {
	u.Password = ""
	u.OTP = ""
	u.OTPRecoveryCodes = nil
}

// IsDeleted returns true if the user is deleted.
func (u *User) IsDeleted() bool {
	return !u.DeletedAt.IsZero()
}

// FlatTransformer returns a transformer for the given type T.
// It flattens the entity.User to the custom type T.
// It's used when the custom type T is not the entity.User itself.
// It returns nil if "T" is already an entity.User.
// See `sdk.NewAuth` for more.
func FlatTransformer[T any]() auth.Transformer[T] {
	var t T
	if reflect.TypeOf(t) == reflect.TypeFor[User]() {
		return nil
	}

	return auth.TransformerFunc[T](func(ctx context.Context, tok *auth.VerifiedToken) (T, error) {
		var (
			u  User
			to T
		)
		if err := tok.Claims(&u); err != nil {
			return to, fmt.Errorf("failed to unmarshal base user entity: %v", err)
		}

		if err := tok.Claims(&to); err != nil {
			return to, fmt.Errorf("failed to unmarshal user: %v", err)
		}

		if len(u.Attrs) > 0 {
			if err := json.Unmarshal(u.Attrs, &to); err != nil {
				return to, fmt.Errorf("failed to unmarshal user attributes: %v", err)
			}
		}

		return to, nil
	})
}
