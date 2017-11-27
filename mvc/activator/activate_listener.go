package activator

// CallOnActivate simply calls the "controller"'s `OnActivate(*TController)` function,
// if any.
//
// Look `activator.go#Register` and `ActivateListener` for more.
func CallOnActivate(controller interface{}, tController *TController) {

	if ac, ok := controller.(ActivateListener); ok {
		ac.OnActivate(tController)
	}
}

// ActivateListener is an interface which should be declared
// on a Controller which needs to register or change the bind values
// that the caller-"user" has been passed to; via the `app.Controller`.
// If that interface is completed by a controller
// then the `OnActivate` function will be called ONCE, NOT in every request
// but ONCE at the application's lifecycle.
type ActivateListener interface {
	// OnActivate accepts a pointer to the `TController`.
	//
	// The `Controller` can make use of the `OnActivate` function
	// to register custom routes
	// or modify the provided values that will be binded to the
	// controller later on.
	//
	// Look `TController` for more.
	OnActivate(*TController)
}
