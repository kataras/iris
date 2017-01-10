// Black-box Testing
package iris_test

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gavv/httpexpect"
	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
)

// White-box testing *
func TestContextDoNextStop(t *testing.T) {
	var context iris.Context
	ok := false
	afterStop := false
	context.Middleware = iris.Middleware{iris.HandlerFunc(func(*iris.Context) {
		ok = true
	}), iris.HandlerFunc(func(*iris.Context) {
		ok = true
	}), iris.HandlerFunc(func(*iris.Context) {
		// this will never execute
		afterStop = true
	})}
	context.Do()
	if context.Pos != 0 {
		t.Fatalf("Expecting position 0 for context's middleware but we got: %d", context.Pos)
	}
	if !ok {
		t.Fatalf("Unexpected behavior, first context's middleware didn't executed")
	}
	ok = false

	context.Next()

	if int(context.Pos) != 1 {
		t.Fatalf("Expecting to have position %d but we got: %d", 1, context.Pos)
	}
	if !ok {
		t.Fatalf("Next context's middleware didn't executed")
	}

	context.StopExecution()
	if context.Pos != 255 {
		t.Fatalf("Context's StopExecution didn't worked, we expected to have position %d but we got %d", 255, context.Pos)
	}

	if !context.IsStopped() {
		t.Fatalf("Should be stopped")
	}

	context.Next()

	if afterStop {
		t.Fatalf("We stopped the execution but the next handler was executed")
	}
}

type pathParameter struct {
	Key   string
	Value string
}
type pathParameters []pathParameter

// White-box testing *
func TestContextParams(t *testing.T) {
	context := &iris.Context{}
	params := pathParameters{
		pathParameter{Key: "testkey", Value: "testvalue"},
		pathParameter{Key: "testkey2", Value: "testvalue2"},
		pathParameter{Key: "id", Value: "3"},
		pathParameter{Key: "bigint", Value: "548921854390354"},
	}
	for _, p := range params {
		context.Set(p.Key, p.Value)
	}

	if v := context.Param(params[0].Key); v != params[0].Value {
		t.Fatalf("Expecting parameter value to be %s but we got %s", params[0].Value, context.Param("testkey"))
	}
	if v := context.Param(params[1].Key); v != params[1].Value {
		t.Fatalf("Expecting parameter value to be %s but we got %s", params[1].Value, context.Param("testkey2"))
	}

	if context.ParamsLen() != len(params) {
		t.Fatalf("Expecting to have %d parameters but we got %d", len(params), context.ParamsLen())
	}

	if vi, err := context.ParamInt(params[2].Key); err != nil {
		t.Fatalf("Unexpecting error on context's ParamInt while trying to get the integer of the %s", params[2].Value)
	} else if vi != 3 {
		t.Fatalf("Expecting to receive %d but we got %d", 3, vi)
	}

	if vi, err := context.ParamInt64(params[3].Key); err != nil {
		t.Fatalf("Unexpecting error on context's ParamInt while trying to get the integer of the %s", params[2].Value)
	} else if vi != 548921854390354 {
		t.Fatalf("Expecting to receive %d but we got %d", 548921854390354, vi)
	}

	// end-to-end test now, note that we will not test the whole mux here, this happens on http_test.go

	iris.ResetDefault()
	expectedParamsStr := "param1=myparam1,param2=myparam2,param3=myparam3afterstatic,anything=/andhere/anything/you/like"
	iris.Get("/path/:param1/:param2/staticpath/:param3/*anything", func(ctx *iris.Context) {
		paramsStr := ctx.ParamsSentence()
		ctx.WriteString(paramsStr)
	})

	httptest.New(iris.Default, t).GET("/path/myparam1/myparam2/staticpath/myparam3afterstatic/andhere/anything/you/like").Expect().Status(iris.StatusOK).Body().Equal(expectedParamsStr)

}

func TestContextURLParams(t *testing.T) {
	iris.ResetDefault()
	passedParams := map[string]string{"param1": "value1", "param2": "value2"}
	iris.Get("/", func(ctx *iris.Context) {
		params := ctx.URLParams()
		ctx.JSON(iris.StatusOK, params)
	})
	e := httptest.New(iris.Default, t, httptest.Debug(true))

	e.GET("/").WithQueryObject(passedParams).Expect().Status(iris.StatusOK).JSON().Equal(passedParams)
}

// hoststring returns the full host, will return the HOST:IP
func TestContextHostString(t *testing.T) {
	iris.ResetDefault()
	iris.Default.Config.VHost = "0.0.0.0:8080"
	iris.Get("/", func(ctx *iris.Context) {
		ctx.WriteString(ctx.Host())
	})

	iris.Get("/wrong", func(ctx *iris.Context) {
		ctx.WriteString(ctx.Host() + "w")
	})

	e := httptest.New(iris.Default, t)
	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(iris.Default.Config.VHost)
	e.GET("/wrong").Expect().Body().NotEqual(iris.Default.Config.VHost)
}

// VirtualHostname returns the hostname only,
// if the host starts with 127.0.0.1 or localhost it gives the registered hostname part of the listening addr
func TestContextVirtualHostName(t *testing.T) {
	iris.ResetDefault()
	vhost := "mycustomvirtualname.com"
	iris.Default.Config.VHost = vhost + ":8080"
	iris.Get("/", func(ctx *iris.Context) {
		ctx.WriteString(ctx.VirtualHostname())
	})

	iris.Get("/wrong", func(ctx *iris.Context) {
		ctx.WriteString(ctx.VirtualHostname() + "w")
	})

	e := httptest.New(iris.Default, t)
	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(vhost)
	e.GET("/wrong").Expect().Body().NotEqual(vhost)
}

func TestContextFormValueString(t *testing.T) {
	iris.ResetDefault()
	var k, v string
	k = "postkey"
	v = "postvalue"
	iris.Post("/", func(ctx *iris.Context) {
		ctx.WriteString(k + "=" + ctx.FormValue(k))
	})
	e := httptest.New(iris.Default, t)

	e.POST("/").WithFormField(k, v).Expect().Status(iris.StatusOK).Body().Equal(k + "=" + v)
}

func TestContextSubdomain(t *testing.T) {
	iris.ResetDefault()
	iris.Default.Config.VHost = "mydomain.com:9999"
	//Default.Config.Tester.ListeningAddr = "mydomain.com:9999"
	// Default.Config.Tester.ExplicitURL = true
	iris.Party("mysubdomain.").Get("/mypath", func(ctx *iris.Context) {
		ctx.WriteString(ctx.Subdomain())
	})

	e := httptest.New(iris.Default, t)

	e.GET("/").WithURL("http://mysubdomain.mydomain.com:9999").Expect().Status(iris.StatusNotFound)
	e.GET("/mypath").WithURL("http://mysubdomain.mydomain.com:9999").Expect().Status(iris.StatusOK).Body().Equal("mysubdomain")

	//e.GET("http://mysubdomain.mydomain.com:9999").Expect().Status(StatusNotFound)
	//e.GET("http://mysubdomain.mydomain.com:9999/mypath").Expect().Status(iris.StatusOK).Body().Equal("mysubdomain")
}

type testBinderData struct {
	Username string
	Mail     string
	Data     []string `form:"mydata" json:"mydata"`
}

type testBinderXMLData struct {
	XMLName    xml.Name `xml:"info"`
	FirstAttr  string   `xml:"first,attr"`
	SecondAttr string   `xml:"second,attr"`
	Name       string   `xml:"name",json:"name"`
	Birth      string   `xml:"birth",json:"birth"`
	Stars      int      `xml:"stars",json:"stars"`
}

type testBinder struct {
	//pointer of  testBinderDataJSON or testBinderXMLData
	vp          interface{}
	m           iris.Unmarshaler
	shouldError bool
}

func (tj *testBinder) Decode(data []byte) error {
	if tj.shouldError {
		return fmt.Errorf("Should error")
	}
	return tj.m.Unmarshal(data, tj.vp)
}

func testUnmarshaler(t *testing.T, tb *testBinder,
	write func(ctx *iris.Context)) *httpexpect.Request {

	// a very dirty and awful way but here we must test in deep
	// the custom object's decoder error with the custom
	// unmarshaler result whenever the testUnmarshaler called.
	if tb.shouldError == false {
		tb.shouldError = true
		testUnmarshaler(t, tb, nil)
		tb.shouldError = false
	}

	iris.ResetDefault()
	h := func(ctx *iris.Context) {
		err := ctx.UnmarshalBody(tb.vp, tb.m)
		if tb.shouldError && err == nil {
			t.Fatalf("Should prompted for error 'Should error' but not error returned from the custom decoder!")
		} else if err != nil {
			t.Fatalf("Error when parsing the body: %s", err.Error())
		}
		if write != nil {
			write(ctx)
		}
	}

	iris.Post("/bind_req_body", h)

	e := httptest.New(iris.Default, t)
	return e.POST("/bind_req_body")
}

// same as DecodeBody
// JSON, XML by DecodeBody passing the default unmarshalers
func TestContextBinders(t *testing.T) {

	passed := map[string]interface{}{"Username": "myusername",
		"Mail":   "mymail@iris-go.com",
		"mydata": []string{"mydata1", "mydata2"}}
	expectedObject := testBinderData{Username: "myusername",
		Mail: "mymail@iris-go.com",
		Data: []string{"mydata1", "mydata2"}}

	// JSON
	vJSON := &testBinder{&testBinderData{},
		iris.UnmarshalerFunc(json.Unmarshal), false}

	testUnmarshaler(
		t,
		vJSON,
		func(ctx *iris.Context) {
			ctx.JSON(iris.StatusOK, vJSON.vp)
		}).
		WithJSON(passed).
		Expect().
		Status(iris.StatusOK).
		JSON().Object().Equal(expectedObject)

	// XML
	expectedObj := testBinderXMLData{
		XMLName:    xml.Name{Local: "info", Space: "info"},
		FirstAttr:  "this is the first attr",
		SecondAttr: "this is the second attr",
		Name:       "Iris web framework",
		Birth:      "13 March 2016",
		Stars:      5758,
	}
	expectedAndPassedObjText := `<` + expectedObj.XMLName.Local + ` first="` +
		expectedObj.FirstAttr + `" second="` +
		expectedObj.SecondAttr + `"><name>` +
		expectedObj.Name + `</name><birth>` +
		expectedObj.Birth + `</birth><stars>` +
		strconv.Itoa(expectedObj.Stars) + `</stars></info>`

	// JSON
	vXML := &testBinder{&testBinderXMLData{},
		iris.UnmarshalerFunc(xml.Unmarshal), false}
	testUnmarshaler(
		t,
		vXML,
		func(ctx *iris.Context) {
			ctx.XML(iris.StatusOK, vXML.vp)
		}).
		WithText(expectedAndPassedObjText).
		Expect().
		Status(iris.StatusOK).
		Body().Equal(expectedAndPassedObjText)

}

func TestContextReadForm(t *testing.T) {
	iris.ResetDefault()

	iris.Post("/form", func(ctx *iris.Context) {
		obj := testBinderData{}
		err := ctx.ReadForm(&obj)
		if err != nil {
			t.Fatalf("Error when parsing the FORM: %s", err.Error())
		}
		ctx.JSON(iris.StatusOK, obj)
	})

	e := httptest.New(iris.Default, t)

	passed := map[string]interface{}{"Username": "myusername", "Mail": "mymail@iris-go.com", "mydata": url.Values{"[0]": []string{"mydata1"},
		"[1]": []string{"mydata2"}}}

	expectedObject := testBinderData{Username: "myusername", Mail: "mymail@iris-go.com", Data: []string{"mydata1", "mydata2"}}

	e.POST("/form").WithForm(passed).Expect().Status(iris.StatusOK).JSON().Object().Equal(expectedObject)
}

// TestContextRedirectTo tests the named route redirect action
func TestContextRedirectTo(t *testing.T) {
	iris.ResetDefault()
	h := func(ctx *iris.Context) { ctx.WriteString(ctx.Path()) }
	iris.Get("/mypath", h)("my-path")
	iris.Get("/mypostpath", h)("my-post-path")
	iris.Get("mypath/with/params/:param1/:param2", func(ctx *iris.Context) {
		if l := ctx.ParamsLen(); l != 2 {
			t.Fatalf("Strange error, expecting parameters to be two but we got: %d", l)
		}
		ctx.WriteString(ctx.Path())
	})("my-path-with-params")

	iris.Get("/redirect/to/:routeName/*anyparams", func(ctx *iris.Context) {
		routeName := ctx.Param("routeName")
		var args []interface{}
		anyparams := ctx.Param("anyparams")
		if anyparams != "" && anyparams != "/" {
			params := strings.Split(anyparams[1:], "/") // firstparam/secondparam
			for _, s := range params {
				args = append(args, s)
			}
		}
		ctx.RedirectTo(routeName, args...)
	})

	e := httptest.New(iris.Default, t)

	e.GET("/redirect/to/my-path/").Expect().Status(iris.StatusOK).Body().Equal("/mypath")
	e.GET("/redirect/to/my-post-path/").Expect().Status(iris.StatusOK).Body().Equal("/mypostpath")
	e.GET("/redirect/to/my-path-with-params/firstparam/secondparam").Expect().Status(iris.StatusOK).Body().Equal("/mypath/with/params/firstparam/secondparam")
}

func TestContextUserValues(t *testing.T) {
	iris.ResetDefault()
	testCustomObjUserValue := struct{ Name string }{Name: "a name"}
	values := map[string]interface{}{"key1": "value1", "key2": "value2", "key3": 3, "key4": testCustomObjUserValue, "key5": map[string]string{"key": "value"}}

	iris.Get("/test", func(ctx *iris.Context) {

		for k, v := range values {
			ctx.Set(k, v)
		}

	}, func(ctx *iris.Context) {
		for k, v := range values {
			userValue := ctx.Get(k)
			if userValue != v {
				t.Fatalf("Expecting user value: %s to be equal with: %#v but got: %#v", k, v, userValue)
			}

			if m, isMap := userValue.(map[string]string); isMap {
				if m["key"] != v.(map[string]string)["key"] {
					t.Fatalf("Expecting user value: %s to be equal with: %#v but got: %#v", k, v.(map[string]string)["key"], m["key"])
				}
			} else {
				if userValue != v {
					t.Fatalf("Expecting user value: %s to be equal with: %#v but got: %#v", k, v, userValue)
				}
			}

		}
	})

	e := httptest.New(iris.Default, t)

	e.GET("/test").Expect().Status(iris.StatusOK)

}

func TestContextCookieSetGetRemove(t *testing.T) {
	iris.ResetDefault()
	key := "mykey"
	value := "myvalue"
	iris.Get("/set", func(ctx *iris.Context) {
		ctx.SetCookieKV(key, value) // should return non empty cookies
	})

	iris.Get("/set_advanced", func(ctx *iris.Context) {
		c := &http.Cookie{}
		c.Name = key
		c.Value = value
		c.HttpOnly = true
		c.Expires = time.Now().Add(time.Duration((60 * 60 * 24 * 7 * 4)) * time.Second)
		ctx.SetCookie(c)
	})

	iris.Get("/get", func(ctx *iris.Context) {
		ctx.WriteString(ctx.GetCookie(key)) // should return my value
	})

	iris.Get("/remove", func(ctx *iris.Context) {
		ctx.RemoveCookie(key)
		cookieFound := false
		ctx.VisitAllCookies(func(k, v string) {
			cookieFound = true
		})
		if cookieFound {
			t.Fatalf("Cookie has been found, when it shouldn't!")
		}
		ctx.WriteString(ctx.GetCookie(key)) // should return ""
	})

	e := httptest.New(iris.Default, t)
	e.GET("/set").Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/get").Expect().Status(iris.StatusOK).Body().Equal(value)
	e.GET("/remove").Expect().Status(iris.StatusOK).Body().Equal("")
	// test again with advanced set
	e.GET("/set_advanced").Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/get").Expect().Status(iris.StatusOK).Body().Equal(value)
	e.GET("/remove").Expect().Status(iris.StatusOK).Body().Equal("")
}

func TestContextSessions(t *testing.T) {
	t.Parallel()
	values := map[string]interface{}{
		"Name":   "iris",
		"Months": "4",
		"Secret": "dsads£2132215£%%Ssdsa",
	}

	iris.ResetDefault()
	iris.Default.Config.Sessions.Cookie = "mycustomsessionid"

	writeValues := func(ctx *iris.Context) {
		sessValues := ctx.Session().GetAll()
		ctx.JSON(iris.StatusOK, sessValues)
	}

	if testEnableSubdomain {
		iris.Party(testSubdomain+".").Get("/get", func(ctx *iris.Context) {
			writeValues(ctx)
		})
	}

	iris.Post("set", func(ctx *iris.Context) {
		vals := make(map[string]interface{}, 0)
		if err := ctx.ReadJSON(&vals); err != nil {
			t.Fatalf("Cannot readjson. Trace %s", err.Error())
		}
		for k, v := range vals {
			ctx.Session().Set(k, v)
		}
	})

	iris.Get("/get", func(ctx *iris.Context) {
		writeValues(ctx)
	})

	iris.Get("/clear", func(ctx *iris.Context) {
		ctx.Session().Clear()
		writeValues(ctx)
	})

	iris.Get("/destroy", func(ctx *iris.Context) {
		ctx.SessionDestroy()
		writeValues(ctx)
		// the cookie and all values should be empty
	})

	// request cookie should be empty
	iris.Get("/after_destroy", func(ctx *iris.Context) {
	})
	iris.Default.Config.VHost = "mydomain.com"
	e := httptest.New(iris.Default, t)

	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	if testEnableSubdomain {
		es := subdomainTester(e)
		es.Request("GET", "/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	}

	// test destroy which also clears first
	d := e.GET("/destroy").Expect().Status(iris.StatusOK)
	d.JSON().Null()
	// 	This removed: d.Cookies().Empty(). Reason:
	// httpexpect counts the cookies setted or deleted at the response time, but cookie is not removed, to be really removed needs to SetExpire(now-1second) so,
	// test if the cookies removed on the next request, like the browser's behavior.
	e.GET("/after_destroy").Expect().Status(iris.StatusOK).Cookies().Empty()
	// set and clear again
	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/clear").Expect().Status(iris.StatusOK).JSON().Object().Empty()
}

type renderTestInformationType struct {
	XMLName    xml.Name `xml:"info"`
	FirstAttr  string   `xml:"first,attr"`
	SecondAttr string   `xml:"second,attr"`
	Name       string   `xml:"name",json:"name"`
	Birth      string   `xml:"birth",json:"birth"`
	Stars      int      `xml:"stars",json:"stars"`
}

func TestContextRenderRest(t *testing.T) {
	iris.ResetDefault()

	dataContents := []byte("Some binary data here.")
	textContents := "Plain text here"
	JSONPContents := map[string]string{"hello": "jsonp"}
	JSONPCallback := "callbackName"
	JSONXMLContents := renderTestInformationType{
		XMLName:    xml.Name{Local: "info", Space: "info"}, // only need to verify that later
		FirstAttr:  "this is the first attr",
		SecondAttr: "this is the second attr",
		Name:       "Iris web framework",
		Birth:      "13 March 2016",
		Stars:      4064,
	}
	markdownContents := "# Hello dynamic markdown from Iris"

	iris.Get("/data", func(ctx *iris.Context) {
		ctx.Data(iris.StatusOK, dataContents)
	})

	iris.Get("/text", func(ctx *iris.Context) {
		ctx.Text(iris.StatusOK, textContents)
	})

	iris.Get("/jsonp", func(ctx *iris.Context) {
		ctx.JSONP(iris.StatusOK, JSONPCallback, JSONPContents)
	})

	iris.Get("/json", func(ctx *iris.Context) {
		ctx.JSON(iris.StatusOK, JSONXMLContents)
	})
	iris.Get("/xml", func(ctx *iris.Context) {
		ctx.XML(iris.StatusOK, JSONXMLContents)
	})

	iris.Get("/markdown", func(ctx *iris.Context) {
		ctx.Markdown(iris.StatusOK, markdownContents)
	})

	e := httptest.New(iris.Default, t)
	dataT := e.GET("/data").Expect().Status(iris.StatusOK)
	dataT.Header("Content-Type").Equal("application/octet-stream")
	dataT.Body().Equal(string(dataContents))

	textT := e.GET("/text").Expect().Status(iris.StatusOK)
	textT.Header("Content-Type").Equal("text/plain; charset=UTF-8")
	textT.Body().Equal(textContents)

	JSONPT := e.GET("/jsonp").Expect().Status(iris.StatusOK)
	JSONPT.Header("Content-Type").Equal("application/javascript; charset=UTF-8")
	JSONPT.Body().Equal(JSONPCallback + `({"hello":"jsonp"});`)

	JSONT := e.GET("/json").Expect().Status(iris.StatusOK)
	JSONT.Header("Content-Type").Equal("application/json; charset=UTF-8")
	JSONT.JSON().Object().Equal(JSONXMLContents)

	XMLT := e.GET("/xml").Expect().Status(iris.StatusOK)
	XMLT.Header("Content-Type").Equal("text/xml; charset=UTF-8")
	XMLT.Body().Equal(`<` + JSONXMLContents.XMLName.Local + ` first="` + JSONXMLContents.FirstAttr + `" second="` + JSONXMLContents.SecondAttr + `"><name>` + JSONXMLContents.Name + `</name><birth>` + JSONXMLContents.Birth + `</birth><stars>` + strconv.Itoa(JSONXMLContents.Stars) + `</stars></info>`)

	markdownT := e.GET("/markdown").Expect().Status(iris.StatusOK)
	markdownT.Header("Content-Type").Equal("text/html; charset=UTF-8")
	markdownT.Body().Equal("<h1>" + markdownContents[2:] + "</h1>\n")
}

func TestContextPreRender(t *testing.T) {
	iris.ResetDefault()

	preRender := func(errMsg string, shouldContinue bool) iris.PreRender {
		return func(ctx *iris.Context,
			src string,
			binding interface{},
			options ...map[string]interface{}) bool {
			// put the 'Error' binding here, for the shake of the test
			if b, isMap := binding.(map[string]interface{}); isMap {
				msg := ""
				if prevMsg := b["Error"]; prevMsg != nil {
					// we have a previous message
					msg += prevMsg.(string)
				}
				msg += errMsg
				b["Error"] = msg
			}
			return shouldContinue
		}
	}
	errMsg1 := "thereIsAnError"
	errMsg2 := "thereIsASecondError"
	errMsg3 := "thereisAThirdError"
	// only errMsg1 and errMsg2 should be rendered because
	// on errMsg2 we stop the execution
	iris.UsePreRender(preRender(errMsg1, true))
	iris.UsePreRender(preRender(errMsg2, false))
	iris.UsePreRender(preRender(errMsg3, false)) // false doesn't matters here

	iris.Get("/", func(ctx *iris.Context) {
		ctx.RenderTemplateSource(iris.StatusOK, "<h1>HI {{.Username}}. Error: {{.Error}}</h1>", map[string]interface{}{"Username": "kataras"})
	})

	e := httptest.New(iris.Default, t)
	expected := "<h1>HI kataras. Error: " + errMsg1 + errMsg2 + "</h1>"
	e.GET("/").Expect().Status(iris.StatusOK).Body().Contains(expected)
}

func TestTemplatesDisabled(t *testing.T) {
	iris.ResetDefault()
	defer iris.Close()

	iris.Default.Config.DisableTemplateEngines = true

	file := "index.html"
	errTmpl := "<h2>Template: %s</h2><b>%s</b>"
	expctedErrMsg := fmt.Sprintf(errTmpl, file, "Error: Unable to execute a template. Trace: Templates are disabled '.Config.DisableTemplatesEngines = true' please turn that to false, as defaulted.\n")

	iris.Get("/renderErr", func(ctx *iris.Context) {
		ctx.MustRender(file, nil)
	})

	e := httptest.New(iris.Default, t)
	e.GET("/renderErr").Expect().Status(iris.StatusServiceUnavailable).Body().Equal(expctedErrMsg)
}

func TestTransactions(t *testing.T) {
	iris.ResetDefault()
	firstTransactionFailureMessage := "Error: Virtual failure!!!"
	secondTransactionSuccessHTMLMessage := "<h1>This will sent at all cases because it lives on different transaction and it doesn't fails</h1>"
	persistMessage := "<h1>I persist show this message to the client!</h1>"

	maybeFailureTransaction := func(shouldFail bool, isRequestScoped bool) func(t *iris.Transaction) {
		return func(t *iris.Transaction) {
			// OPTIONAl, the next transactions and the flow will not be skipped if this transaction fails
			if isRequestScoped {
				t.SetScope(iris.RequestTransactionScope)
			}

			// OPTIONAL STEP:
			// create a new custom type of error here to keep track of the status code and reason message
			err := iris.NewTransactionErrResult()

			t.Context.Text(iris.StatusOK, "Blablabla this should not be sent to the client because we will fill the err with a message and status")

			fail := shouldFail

			if fail {
				err.StatusCode = iris.StatusInternalServerError
				err.Reason = firstTransactionFailureMessage
			}

			// OPTIONAl STEP:
			// but useful if we want to post back an error message to the client if the transaction failed.
			// if the reason is empty then the transaction completed successfully,
			// otherwise we rollback the whole response body and cookies and everything lives inside the transaction.Request.
			t.Complete(err)
		}
	}

	successTransaction := func(scope *iris.Transaction) {
		if scope.Context.Request.RequestURI == "/failAllBecauseOfRequestScopeAndFailure" {
			t.Fatalf("We are inside successTransaction but the previous REQUEST SCOPED TRANSACTION HAS FAILED SO THiS SHOULD NOT BE RAN AT ALL")
		}
		scope.Context.HTML(iris.StatusOK,
			secondTransactionSuccessHTMLMessage)
		// * if we don't have any 'throw error' logic then no need of scope.Complete()
	}

	persistMessageHandler := func(ctx *iris.Context) {
		// OPTIONAL, depends on the usage:
		// at any case, what ever happens inside the context's transactions send this to the client
		ctx.HTML(iris.StatusOK, persistMessage)
	}

	iris.Get("/failFirsTransactionButSuccessSecondWithPersistMessage", func(ctx *iris.Context) {
		ctx.BeginTransaction(maybeFailureTransaction(true, false))
		ctx.BeginTransaction(successTransaction)
		persistMessageHandler(ctx)
	})

	iris.Get("/failFirsTransactionButSuccessSecond", func(ctx *iris.Context) {
		ctx.BeginTransaction(maybeFailureTransaction(true, false))
		ctx.BeginTransaction(successTransaction)
	})

	iris.Get("/failAllBecauseOfRequestScopeAndFailure", func(ctx *iris.Context) {
		ctx.BeginTransaction(maybeFailureTransaction(true, true))
		ctx.BeginTransaction(successTransaction)
	})

	customErrorTemplateText := "<h1>custom error</h1>"
	iris.OnError(iris.StatusInternalServerError, func(ctx *iris.Context) {
		ctx.Text(iris.StatusInternalServerError, customErrorTemplateText)
	})

	failureWithRegisteredErrorHandler := func(ctx *iris.Context) {
		ctx.BeginTransaction(func(transaction *iris.Transaction) {
			transaction.SetScope(iris.RequestTransactionScope)
			err := iris.NewTransactionErrResult()
			err.StatusCode = iris.StatusInternalServerError // set only the status code in order to execute the registered template
			transaction.Complete(err)
		})

		ctx.Text(iris.StatusOK, "this will not be sent to the client because first is requested scope and it's failed")
	}

	iris.Get("/failAllBecauseFirstTransactionFailedWithRegisteredErrorTemplate", failureWithRegisteredErrorHandler)

	e := httptest.New(iris.Default, t)

	e.GET("/failFirsTransactionButSuccessSecondWithPersistMessage").
		Expect().
		Status(iris.StatusOK).
		ContentType("text/html", iris.Config.Charset).
		Body().
		Equal(secondTransactionSuccessHTMLMessage + persistMessage)

	e.GET("/failFirsTransactionButSuccessSecond").
		Expect().
		Status(iris.StatusOK).
		ContentType("text/html", iris.Config.Charset).
		Body().
		Equal(secondTransactionSuccessHTMLMessage)

	e.GET("/failAllBecauseOfRequestScopeAndFailure").
		Expect().
		Status(iris.StatusInternalServerError).
		Body().
		Equal(firstTransactionFailureMessage)

	e.GET("/failAllBecauseFirstTransactionFailedWithRegisteredErrorTemplate").
		Expect().
		Status(iris.StatusInternalServerError).
		Body().
		Equal(customErrorTemplateText)
}

func TestLimitRequestBodySize(t *testing.T) {
	const maxBodySize = 1 << 20

	api := iris.New()

	// or inside handler: ctx.SetMaxRequestBodySize(int64(maxBodySize))
	api.Use(iris.LimitRequestBodySize(maxBodySize))

	api.Post("/", func(ctx *iris.Context) {
		b, err := ioutil.ReadAll(ctx.Request.Body)
		if len(b) > maxBodySize {
			// this is a fatal error it should never happened.
			t.Fatalf("body is larger (%d) than maxBodySize (%d) even if we add the LimitRequestBodySize middleware", len(b), maxBodySize)
		}
		// if is larger then send a bad request status
		if err != nil {
			ctx.WriteHeader(iris.StatusBadRequest)
			ctx.Writef(err.Error())
			return
		}

		ctx.Write(b)
	})

	// UseGlobal should be called at the end used to prepend handlers
	// api.UseGlobal(iris.LimitRequestBodySize(int64(maxBodySize)))

	e := httptest.New(api, t)

	// test with small body
	e.POST("/").WithBytes([]byte("ok")).Expect().Status(iris.StatusOK).Body().Equal("ok")
	// test with equal to max body size limit
	bsent := make([]byte, maxBodySize, maxBodySize)
	e.POST("/").WithBytes(bsent).Expect().Status(iris.StatusOK).Body().Length().Equal(len(bsent))
	// test with larger body sent and wait for the custom response
	largerBSent := make([]byte, maxBodySize+1, maxBodySize+1)
	e.POST("/").WithBytes(largerBSent).Expect().Status(iris.StatusBadRequest).Body().Equal("http: request body too large")

}
