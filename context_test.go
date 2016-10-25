// Black-box Testing
package iris_test

/*
The most part of  the context covered,
the other part contains serving static methods,
find remote ip, GetInt and the view engine rendering(templates)
I am not waiting unexpected behaviors from the rest of the funcs,
so that's all with context's tests.

CONTRIBUTE & DISCUSSION ABOUT TESTS TO: https://github.com/iris-contrib/tests
*/

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"github.com/valyala/fasthttp"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
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
	context := &iris.Context{RequestCtx: &fasthttp.RequestCtx{}}
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
		ctx.Write(paramsStr)
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
	e := httptest.New(iris.Default, t)

	e.GET("/").WithQueryObject(passedParams).Expect().Status(iris.StatusOK).JSON().Equal(passedParams)
}

// hoststring returns the full host, will return the HOST:IP
func TestContextHostString(t *testing.T) {
	iris.ResetDefault()
	iris.Default.Config.VHost = "0.0.0.0:8080"
	iris.Get("/", func(ctx *iris.Context) {
		ctx.Write(ctx.HostString())
	})

	iris.Get("/wrong", func(ctx *iris.Context) {
		ctx.Write(ctx.HostString() + "w")
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
		ctx.Write(ctx.VirtualHostname())
	})

	iris.Get("/wrong", func(ctx *iris.Context) {
		ctx.Write(ctx.VirtualHostname() + "w")
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
		ctx.Write(k + "=" + ctx.FormValueString(k))
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
		ctx.Write(ctx.Subdomain())
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

func TestContextReadJSON(t *testing.T) {
	iris.ResetDefault()
	iris.Post("/json", func(ctx *iris.Context) {
		obj := testBinderData{}
		err := ctx.ReadJSON(&obj)
		if err != nil {
			t.Fatalf("Error when parsing the JSON body: %s", err.Error())
		}
		ctx.JSON(iris.StatusOK, obj)
	})

	iris.Post("/json_pointer", func(ctx *iris.Context) {
		obj := &testBinderData{}
		err := ctx.ReadJSON(obj)
		if err != nil {
			t.Fatalf("Error when parsing the JSON body: %s", err.Error())
		}
		ctx.JSON(iris.StatusOK, obj)
	})

	e := httptest.New(iris.Default, t)
	passed := map[string]interface{}{"Username": "myusername", "Mail": "mymail@iris-go.com", "mydata": []string{"mydata1", "mydata2"}}
	expectedObject := testBinderData{Username: "myusername", Mail: "mymail@iris-go.com", Data: []string{"mydata1", "mydata2"}}

	e.POST("/json").WithJSON(passed).Expect().Status(iris.StatusOK).JSON().Object().Equal(expectedObject)
	e.POST("/json_pointer").WithJSON(passed).Expect().Status(iris.StatusOK).JSON().Object().Equal(expectedObject)
}

type testJSONBinderDataWithDecoder struct {
	Username    string
	Mail        string
	Data        []string `json:"mydata"`
	shouldError bool
}

func (tj *testJSONBinderDataWithDecoder) Decode(data []byte) error {
	if tj.shouldError {
		return fmt.Errorf("Should error")
	}
	return json.Unmarshal(data, tj)
}

func TestContextReadJSONWithDecoder(t *testing.T) {
	iris.ResetDefault()
	iris.Post("/json_should_error", func(ctx *iris.Context) {
		obj := testJSONBinderDataWithDecoder{shouldError: true}
		err := ctx.ReadJSON(&obj)
		if err == nil {
			t.Fatalf("Should prompted for error 'Should error' but not error returned from the custom decoder!")
		}
		ctx.Write(err.Error())
		ctx.SetStatusCode(iris.StatusOK)
	})

	iris.Post("/json", func(ctx *iris.Context) {
		obj := testJSONBinderDataWithDecoder{}
		err := ctx.ReadJSON(&obj)
		if err != nil {
			t.Fatalf("Error when parsing the JSON body: %s", err.Error())
		}
		ctx.JSON(iris.StatusOK, obj)
	})

	iris.Post("/json_pointer", func(ctx *iris.Context) {
		obj := &testJSONBinderDataWithDecoder{}
		err := ctx.ReadJSON(obj)
		if err != nil {
			t.Fatalf("Error when parsing the JSON body: %s", err.Error())
		}
		ctx.JSON(iris.StatusOK, obj)
	})

	e := httptest.New(iris.Default, t)
	passed := map[string]interface{}{"Username": "kataras", "Mail": "mymail@iris-go.com", "mydata": []string{"mydata1", "mydata2"}}
	expectedObject := testJSONBinderDataWithDecoder{Username: "kataras", Mail: "mymail@iris-go.com", Data: []string{"mydata1", "mydata2"}}

	e.POST("/json_should_error").WithJSON(passed).Expect().Status(iris.StatusOK).Body().Equal("Should error")
	e.POST("/json").WithJSON(passed).Expect().Status(iris.StatusOK).JSON().Object().Equal(expectedObject)
	e.POST("/json_pointer").WithJSON(passed).Expect().Status(iris.StatusOK).JSON().Object().Equal(expectedObject)
} // no need for xml, it's exact the same.

func TestContextReadXML(t *testing.T) {
	iris.ResetDefault()

	iris.Post("/xml", func(ctx *iris.Context) {
		obj := testBinderXMLData{}
		err := ctx.ReadXML(&obj)
		if err != nil {
			t.Fatalf("Error when parsing the XML body: %s", err.Error())
		}
		ctx.XML(iris.StatusOK, obj)
	})

	e := httptest.New(iris.Default, t)
	expectedObj := testBinderXMLData{
		XMLName:    xml.Name{Local: "info", Space: "info"},
		FirstAttr:  "this is the first attr",
		SecondAttr: "this is the second attr",
		Name:       "Iris web framework",
		Birth:      "13 March 2016",
		Stars:      4064,
	}
	// so far no WithXML or .XML like WithJSON and .JSON on httpexpect I added a feature request as post issue and we're waiting
	expectedBody := `<` + expectedObj.XMLName.Local + ` first="` + expectedObj.FirstAttr + `" second="` + expectedObj.SecondAttr + `"><name>` + expectedObj.Name + `</name><birth>` + expectedObj.Birth + `</birth><stars>` + strconv.Itoa(expectedObj.Stars) + `</stars></info>`
	e.POST("/xml").WithText(expectedBody).Expect().Status(iris.StatusOK).Body().Equal(expectedBody)
}

// TestContextRedirectTo tests the named route redirect action
func TestContextRedirectTo(t *testing.T) {
	iris.ResetDefault()
	h := func(ctx *iris.Context) { ctx.Write(ctx.PathString()) }
	iris.Get("/mypath", h)("my-path")
	iris.Get("/mypostpath", h)("my-post-path")
	iris.Get("mypath/with/params/:param1/:param2", func(ctx *iris.Context) {
		if l := ctx.ParamsLen(); l != 2 {
			t.Fatalf("Strange error, expecting parameters to be two but we got: %d", l)
		}
		ctx.Write(ctx.PathString())
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
		//println("Redirecting to: " + routeName + " with path: " + Path(routeName, args...))
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
		c := fasthttp.AcquireCookie()
		c.SetKey(key)
		c.SetValue(value)
		c.SetHTTPOnly(true)
		c.SetExpire(time.Now().Add(time.Duration((60 * 60 * 24 * 7 * 4)) * time.Second))
		ctx.SetCookie(c)
		fasthttp.ReleaseCookie(c)
	})

	iris.Get("/get", func(ctx *iris.Context) {
		ctx.Write(ctx.GetCookie(key)) // should return my value
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
		ctx.Write(ctx.GetCookie(key)) // should return ""
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

func TestContextFlashMessages(t *testing.T) {
	iris.ResetDefault()
	firstKey := "name"
	lastKey := "package"

	values := pathParameters{pathParameter{Key: firstKey, Value: "kataras"}, pathParameter{Key: lastKey, Value: "iris"}}
	jsonExpected := map[string]string{firstKey: "kataras", lastKey: "iris"}
	// set the flashes, the cookies are filled
	iris.Put("/set", func(ctx *iris.Context) {
		for _, v := range values {
			ctx.SetFlash(v.Key, v.Value)
		}
	})

	// get the first flash, the next should be available to the next requess
	iris.Get("/get_first_flash", func(ctx *iris.Context) {
		for _, v := range values {
			val, err := ctx.GetFlash(v.Key)
			if err == nil {
				ctx.JSON(iris.StatusOK, map[string]string{v.Key: val})
			} else {
				ctx.JSON(iris.StatusOK, nil) // return nil
			}

			break
		}

	})

	// just an empty handler to test if the flashes should remeain to the next if GetFlash/GetFlashes used
	iris.Get("/get_no_getflash", func(ctx *iris.Context) {
	})

	// get the last flash, the next should be available to the next requess
	iris.Get("/get_last_flash", func(ctx *iris.Context) {
		for i, v := range values {
			if i == len(values)-1 {
				val, err := ctx.GetFlash(v.Key)
				if err == nil {
					ctx.JSON(iris.StatusOK, map[string]string{v.Key: val})
				} else {
					ctx.JSON(iris.StatusOK, nil) // return nil
				}

			}
		}
	})

	iris.Get("/get_zero_flashes", func(ctx *iris.Context) {
		ctx.JSON(iris.StatusOK, ctx.GetFlashes()) // should return nil
	})

	// we use the GetFlash to get the flash messages, the messages and the cookies should be empty after that
	iris.Get("/get_flash", func(ctx *iris.Context) {
		kv := make(map[string]string)
		for _, v := range values {
			val, err := ctx.GetFlash(v.Key)
			if err == nil {
				kv[v.Key] = val
			}
		}
		ctx.JSON(iris.StatusOK, kv)
	}, func(ctx *iris.Context) {
		// at the same request, flashes should be available
		if len(ctx.GetFlashes()) == 0 {
			t.Fatalf("Flashes should be remeain to the whole request lifetime")
		}
	})

	iris.Get("/get_flashes", func(ctx *iris.Context) {
		// one time one handler, using GetFlashes
		kv := make(map[string]string)
		flashes := ctx.GetFlashes()
		//second time on the same handler, using the GetFlash
		for k := range flashes {
			kv[k], _ = ctx.GetFlash(k)
		}
		if len(flashes) != len(kv) {
			ctx.SetStatusCode(iris.StatusNoContent)
			return
		}
		ctx.Next()

	}, func(ctx *iris.Context) {
		// third time on a next handler
		// test the if next handler has access to them(must) because flash are request lifetime now.
		// print them to the client for test the response also
		ctx.JSON(iris.StatusOK, ctx.GetFlashes())
	})

	e := httptest.New(iris.Default, t)
	e.PUT("/set").Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/get_first_flash").Expect().Status(iris.StatusOK).JSON().Object().ContainsKey(firstKey).NotContainsKey(lastKey)
	// just a request which does not use the flash message, so flash messages should be available on the next request
	e.GET("/get_no_getflash").Expect().Status(iris.StatusOK)
	e.GET("/get_last_flash").Expect().Status(iris.StatusOK).JSON().Object().ContainsKey(lastKey).NotContainsKey(firstKey)
	g := e.GET("/get_zero_flashes").Expect().Status(iris.StatusOK)
	g.JSON().Null()
	g.Cookies().Empty()
	// set the magain
	e.PUT("/set").Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	// get them again using GetFlash
	e.GET("/get_flash").Expect().Status(iris.StatusOK).JSON().Object().Equal(jsonExpected)
	// this should be empty again
	g = e.GET("/get_zero_flashes").Expect().Status(iris.StatusOK)
	g.JSON().Null()
	g.Cookies().Empty()
	//set them again
	e.PUT("/set").Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	// get them again using GetFlashes
	e.GET("/get_flashes").Expect().Status(iris.StatusOK).JSON().Object().Equal(jsonExpected)
	// this should be empty again
	g = e.GET("/get_zero_flashes").Expect().Status(iris.StatusOK)
	g.JSON().Null()
	g.Cookies().Empty()

	// test Get, and get again should return nothing
	e.PUT("/set").Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/get_first_flash").Expect().Status(iris.StatusOK).JSON().Object().ContainsKey(firstKey).NotContainsKey(lastKey)
	g = e.GET("/get_first_flash").Expect().Status(iris.StatusOK)
	g.JSON().Null()
	g.Cookies().Empty()
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
	errMsg1 := "thereIsAnError"
	iris.UsePreRender(func(ctx *iris.Context, src string, binding interface{}, options ...map[string]interface{}) bool {
		// put the 'Error' binding here, for the shake of the test
		if b, isMap := binding.(map[string]interface{}); isMap {
			b["Error"] = errMsg1
		}
		// continue to the next prerender
		return true
	})
	errMsg2 := "thereIsASecondError"
	iris.UsePreRender(func(ctx *iris.Context, src string, binding interface{}, options ...map[string]interface{}) bool {
		// put the 'Error' binding here, for the shake of the test
		if b, isMap := binding.(map[string]interface{}); isMap {
			prev := b["Error"].(string)
			msg := prev + errMsg2
			b["Error"] = msg
		}
		// DO NOT CONTINUE to the next prerender
		return false
	})

	errMsg3 := "thereisAThirdError"
	iris.UsePreRender(func(ctx *iris.Context, src string, binding interface{}, options ...map[string]interface{}) bool {
		// put the 'Error' binding here, for the shake of the test
		if b, isMap := binding.(map[string]interface{}); isMap {
			prev := b["Error"].(string)
			msg := prev + errMsg3
			b["Error"] = msg
		}
		// doesn't matters the return statement, we don't have other prerender
		return true
	})

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
	ip := "0.0.0.0"
	errTmpl := "<h2>Template: %s\nIP: %s</h2><b>%s</b>"
	expctedErrMsg := fmt.Sprintf(errTmpl, file, ip, "Error: Unable to execute a template. Trace: Templates are disabled '.Config.DisableTemplatesEngines = true' please turn that to false, as defaulted.\n")

	iris.Get("/renderErr", func(ctx *iris.Context) {
		ctx.MustRender(file, nil)
	})

	e := httptest.New(iris.Default, t)
	e.GET("/renderErr").Expect().Status(iris.StatusServiceUnavailable).Body().Equal(expctedErrMsg)
}
