package controllers

// import "github.com/kataras/iris/mvc2"

// ValuesController is the equivalent
// `ValuesController` of the .net core 2.0 mvc application.
type ValuesController struct{} //{ mvc2.C }

/* on windows tests(older) the Get was:
func (vc *ValuesController) Get() {
	// id,_ := vc.Params.GetInt("id")
	// vc.Ctx.WriteString("value")
}
but as Iris is always going better, now supports return values as well*/

// Get handles "GET" requests to "api/values/{id}".
func (vc *ValuesController) Get() string {
	return "value"
}

// Put handles "PUT" requests to "api/values/{id}".
func (vc *ValuesController) Put() {}

// Delete handles "DELETE" requests to "api/values/{id}".
func (vc *ValuesController) Delete() {}
