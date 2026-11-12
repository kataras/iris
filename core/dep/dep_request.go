package dep

import (
	"net/http"
	"reflect"
	"slices"

	"github.com/kataras/iris/v14/core/pkg/reflex"
)

// Some are commented types because they are registered through their respective sub-packages
// in order to make it independent from the rest of the packages.
var (
	// contextType     = reflect.TypeOf((*context.Context)(nil))
	httpRequestType = reflect.TypeFor[*http.Request]()
	stdContextType  = reflex.StdContextType
	// routingType =
	// controllerType =
	// sessionType     = reflect.TypeOf((*sessions.Session)(nil))
	timeType = reflex.TimeType

	requestDependencyTypes = []reflect.Type{
		// contextType,
		httpRequestType,
		// routingType,
		// controllerType,
		stdContextType,
		// sessionType,
		timeType,
	}
)

// RegisterRequestDependencyType adds a new type to the request dependency types.
// It's useful when you want to add a custom type that can be resolved by the request's context.
// Call it before any dependency registration. It's not thread-safe.
func RegisterRequestDependencyType(valueOrType any) {
	if valueOrType == nil {
		return
	}

	typ := reflex.TypeOf(valueOrType)
	if slices.Contains(requestDependencyTypes, typ) {
		return
	}

	requestDependencyTypes = append(requestDependencyTypes, typ)
}

func isRequestInputDependency(in reflect.Type) bool {
	return slices.Contains(requestDependencyTypes, in)
}
