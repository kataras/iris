// black-box
package iris_test

import (
	"testing"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/httptest"
)

const testCustomHandlerParamName = "myparam"

type myTestHandlerData struct {
	Sysname      string // this will be the same for all requests
	Version      int    // this will be the same for all requests
	URLParameter string // this will be different for each request
}

type myTestCustomHandler struct {
	data myTestHandlerData
}

func (m *myTestCustomHandler) Serve(ctx *iris.Context) {
	data := &m.data
	data.URLParameter = ctx.URLParam(testCustomHandlerParamName)
	ctx.JSON(iris.StatusOK, data)
}

func TestCustomHandler(t *testing.T) {
	app := iris.New()
	app.Adapt(newTestNativeRouter())

	myData := myTestHandlerData{
		Sysname: "Redhat",
		Version: 1,
	}
	app.Handle("GET", "/custom_handler_1", &myTestCustomHandler{myData})
	app.Handle("GET", "/custom_handler_2", &myTestCustomHandler{myData})

	e := httptest.New(app, t)
	// two times per testRoute
	param1 := "thisimyparam1"
	expectedData1 := myData
	expectedData1.URLParameter = param1
	e.GET("/custom_handler_1/").WithQuery(testCustomHandlerParamName, param1).Expect().Status(iris.StatusOK).JSON().Equal(expectedData1)

	param2 := "thisimyparam2"
	expectedData2 := myData
	expectedData2.URLParameter = param2
	e.GET("/custom_handler_1/").WithQuery(testCustomHandlerParamName, param2).Expect().Status(iris.StatusOK).JSON().Equal(expectedData2)

	param3 := "thisimyparam3"
	expectedData3 := myData
	expectedData3.URLParameter = param3
	e.GET("/custom_handler_2/").WithQuery(testCustomHandlerParamName, param3).Expect().Status(iris.StatusOK).JSON().Equal(expectedData3)

	param4 := "thisimyparam4"
	expectedData4 := myData
	expectedData4.URLParameter = param4
	e.GET("/custom_handler_2/").WithQuery(testCustomHandlerParamName, param4).Expect().Status(iris.StatusOK).JSON().Equal(expectedData4)
}
