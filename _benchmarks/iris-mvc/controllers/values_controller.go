package controllers

import "github.com/kataras/iris/mvc"

// ValuesController is the equivalent
// `ValuesController` of the .net core 2.0 mvc application.
type ValuesController struct {
	mvc.Controller
}

// Get handles "GET" requests to "api/values/{id}".
func (vc *ValuesController) Get() {
	// id,_ := vc.Params.GetInt("id")
	vc.Ctx.WriteString("value")
}

// Put handles "PUT" requests to "api/values/{id}".
func (vc *ValuesController) Put() {}

// Delete handles "DELETE" requests to "api/values/{id}".
func (vc *ValuesController) Delete() {}
