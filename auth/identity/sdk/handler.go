package sdk

import (
	"fmt"

	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/auth/identity/entity"

	"github.com/kataras/iris/v14/context"
)

// BindAuthConfiguration binds the SDK's public keys to the given auth.Configuration.
func (s *SDK[T]) BindAuthConfiguration(dest *auth.Configuration) error {
	// convert public keys to auth keys configuration.
	newAuthKeysConfig, err := s.keys.Configuration()
	if err != nil {
		return fmt.Errorf("failed to convert public keys to auth keys configuration: %v", err)

	}

	if len(dest.Headers) == 0 {
		dest.Headers = []string{"Authorization", "X-Authorization"}
	}

	for _, newAuthKeyConfig := range newAuthKeysConfig {
		if _, ok := dest.Keys.Get(newAuthKeyConfig.ID); !ok { // add keys if not already exists.
			dest.Keys = append(dest.Keys, newAuthKeyConfig)
			continue
		}
	}

	if dest.Cookie.Name == "" && len(dest.Cookie.Hash) > 0 {
		dest.Cookie.Name = "iris_auth_cookie"
	}

	_, err = dest.Keys.Load()
	return err
}

// VerifyHandler returns a JWT verify midldeware based on the given auth.Configuration and optional verify functions.
// The configuration will be automatically filled up with the public keys retrieved by the identity server.
//
// Usage:
//
//	app.Use(sdk.VerifyHandler(auth.Configuration{...})
func (s *SDK[T]) VerifyHandler(authConfig auth.Configuration, verifyFuncs ...auth.VerifyUserFunc[T]) context.Handler {
	newAuthInstance := NewAuth(s, authConfig)
	// verifyFuncs = append(verifyFuncs, func(ctx stdContext.Context, t T) error {
	// 	if !t.DeletedAt.IsZero() { // just in-case. This is handled by server, as all queries check for that.
	// 		return fmt.Errorf("user is deleted")
	// 	}

	// 	return nil
	// })
	return newAuthInstance.VerifyHandler(verifyFuncs...)
}

// NewAuth creates a new auth instance based on the given auth.Configuration.
// The configuration will be automatically filled up with the public keys retrieved by the identity server.
// By default it tries to flat the entity.User and its Attrs JSON field to the custom T type.
// It panics on failure.
// It's a helper function to reduce the boilerplate code.
//
// Usage:
//
//	var authInstance = NewAuth[entity.User](mySDK, authConfig)
//
// OR
//
//	var authInstance = NewAuth[custom.User](mySDK, authConfig)
//
//	[...]
//	app.Use(authInstance.VerifyHandler())
func NewAuth[T any](s *SDK[T], authConfig auth.Configuration) *auth.Auth[T] {
	if err := s.BindAuthConfiguration(&authConfig); err != nil {
		panic(err)
	}

	newAuthInstance, err := auth.New[T](authConfig)
	if err != nil {
		panic(fmt.Errorf("failed to generate new auth instance: %v", err))
	}

	// Check if needs to use the default transformer to convert entity.User to custom flat T
	// inside the FlatTransformer itself. This default transformer can be overridden
	// by custom call of SetTransformer/SetTransformerFunc after this function call.
	newAuthInstance.SetTransformer(entity.FlatTransformer[T]())
	return newAuthInstance
}
