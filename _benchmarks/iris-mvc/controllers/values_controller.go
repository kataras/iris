package controllers

// ValuesController is the equivalent
// `ValuesController` of the .net core 2.0 mvc application.
type ValuesController struct{}

// Get handles "GET" requests to "api/values/{id}".
func (vc *ValuesController) Get() string {
	return "value"
}

// Put handles "PUT" requests to "api/values/{id}".
func (vc *ValuesController) Put() {}

// Delete handles "DELETE" requests to "api/values/{id}".
func (vc *ValuesController) Delete() {}
