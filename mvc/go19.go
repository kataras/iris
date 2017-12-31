// +build go1.9

package mvc

import "github.com/kataras/iris/hero"

type (

	// Result is a type alias for the `hero#Result`, useful for output controller's methods.
	Result = hero.Result
	// Response is a type alias for the `hero#Response`, useful for output controller's methods.
	Response = hero.Response
	// View is a type alias for the `hero#View`, useful for output controller's methods.
	View = hero.View
)

var (
	// Try is a type alias for the `hero#Try`,
	// useful to return a result based on two cases: failure(including panics) and a succeess.
	Try = hero.Try
)
