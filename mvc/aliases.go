package mvc

import (
	"github.com/kataras/iris/v12/hero"
	"github.com/kataras/iris/v12/versioning"
)

type (
	// Result is a type alias for the `hero#Result`, useful for output controller's methods.
	Result = hero.Result
	// Response is a type alias for the `hero#Response`, useful for output controller's methods.
	Response = hero.Response
	// View is a type alias for the `hero#View`, useful for output controller's methods.
	View = hero.View
	// Code is a type alias for the `hero#Code`, useful for
	// http error handling in controllers.
	// This can be one of the input parameters of the `Controller.HandleHTTPError`.
	Code = hero.Code
	// Err is a type alias for the `hero#Err`.
	// It is a special type for error stored in mvc responses or context.
	// It's used for a builtin dependency to map the error given by a previous
	// method or middleware.
	Err = hero.Err
	// DeprecationOptions describes the deprecation headers key-values.
	// Is a type alias for the `versioning#DeprecationOptions`.
	//
	// See `Deprecated` package-level option.
	DeprecationOptions = versioning.DeprecationOptions
)

// Try is a type alias for the `hero#Try`,
// useful to return a result based on two cases: failure(including panics) and a succeess.
var Try = hero.Try
