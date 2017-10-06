package activator

import (
	"reflect"
)

// CallOnActivate simply calls the "controller"'s `OnActivate(*ActivatePayload)` function,
// if any.
//
// Look `activator.go#Register` and `ActivateListener` for more.
func CallOnActivate(controller interface{},
	bindValues *[]interface{}, registerFunc RegisterFunc) {

	if ac, ok := controller.(ActivateListener); ok {
		p := &ActivatePayload{
			BindValues: bindValues,
			Handle:     registerFunc,
		}
		ac.OnActivate(p)
	}
}

// ActivateListener is an interface which should be declared
// on a Controller which needs to register or change the bind values
// that the caller-"user" has been passed to; via the `app.Controller`.
// If that interface is completed by a controller
// then the `OnActivate` function will be called ONCE, NOT in every request
// but ONCE at the application's lifecycle.
type ActivateListener interface {
	// OnActivate accepts a pointer to the `ActivatePayload`.
	//
	// The `Controller` can make use of the `OnActivate` function
	// to register custom routes
	// or modify the provided values that will be binded to the
	// controller later on.
	//
	// Look `ActivatePayload` for more.
	OnActivate(*ActivatePayload)
}

// ActivatePayload contains the necessary information and the ability
// to alt a controller's registration options, i.e the binder.
//
// With `ActivatePayload` the `Controller` can register custom routes
// or modify the provided values that will be binded to the
// controller later on.
type ActivatePayload struct {
	BindValues *[]interface{}
	Handle     RegisterFunc
}

// EnsureBindValue will make sure that this "bindValue"
// will be registered to the controller's binder
// if its type is not already passed by the caller..
//
// For example, on `SessionController` it looks if *sessions.Sessions
// has been binded from the caller and if not then the "bindValue"
// will be binded and used as a default sessions manager instead.
//
// At general, if the caller has already provided a value with the same Type
// then the "bindValue" will be ignored and not be added to the controller's bind values.
//
// Returns true if the caller has NOT already provided a value with the same Type
// and "bindValue" is NOT ignored therefore is appended to the controller's bind values.
func (i *ActivatePayload) EnsureBindValue(bindValue interface{}) bool {
	valueTyp := reflect.TypeOf(bindValue)
	localBindValues := *i.BindValues

	for _, bindedValue := range localBindValues {
		// type already exists, remember: binding here is per-type.
		if reflect.TypeOf(bindedValue) == valueTyp {
			return false
		}
	}

	*i.BindValues = append(localBindValues, bindValue)
	return true
}
