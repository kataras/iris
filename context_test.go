package iris_test

import (
	"io/ioutil"
	"testing"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/httptest"
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

	app := iris.New()
	app.Adapt(httprouter.New())
	expectedParamsStr := "param1=myparam1,param2=myparam2,param3=myparam3afterstatic,anything=/andhere/anything/you/like"
	app.Get("/path/:param1/:param2/staticpath/:param3/*anything", func(ctx *iris.Context) {
		paramsStr := ctx.ParamsSentence()
		ctx.WriteString(paramsStr)
	})

	httptest.New(app, t).GET("/path/myparam1/myparam2/staticpath/myparam3afterstatic/andhere/anything/you/like").Expect().Status(iris.StatusOK).Body().Equal(expectedParamsStr)

}

func TestContextURLParams(t *testing.T) {
	app := iris.New()
	app.Adapt(newTestNativeRouter())
	passedParams := map[string]string{"param1": "value1", "param2": "value2"}
	app.Get("/", func(ctx *iris.Context) {
		params := ctx.URLParams()
		ctx.JSON(iris.StatusOK, params)
	})
	e := httptest.New(app, t)

	e.GET("/").WithQueryObject(passedParams).Expect().Status(iris.StatusOK).JSON().Equal(passedParams)
}

// hoststring returns the full host, will return the HOST:IP
func TestContextHostString(t *testing.T) {
	app := iris.New(iris.Configuration{VHost: "0.0.0.0:8080"})
	app.Adapt(newTestNativeRouter())

	app.Get("/", func(ctx *iris.Context) {
		ctx.WriteString(ctx.Host())
	})

	app.Get("/wrong", func(ctx *iris.Context) {
		ctx.WriteString(ctx.Host() + "w")
	})

	e := httptest.New(app, t)
	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(app.Config.VHost)
	e.GET("/wrong").Expect().Body().NotEqual(app.Config.VHost)
}

// VirtualHostname returns the hostname only,
// if the host starts with 127.0.0.1 or localhost it gives the registered hostname part of the listening addr
func TestContextVirtualHostName(t *testing.T) {
	vhost := "mycustomvirtualname.com"
	app := iris.New(iris.Configuration{VHost: vhost + ":8080"})
	app.Adapt(newTestNativeRouter())

	app.Get("/", func(ctx *iris.Context) {
		ctx.WriteString(ctx.VirtualHostname())
	})

	app.Get("/wrong", func(ctx *iris.Context) {
		ctx.WriteString(ctx.VirtualHostname() + "w")
	})

	e := httptest.New(app, t)
	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(vhost)
	e.GET("/wrong").Expect().Body().NotEqual(vhost)
}

func TestContextFormValueString(t *testing.T) {
	app := iris.New()
	app.Adapt(httprouter.New())
	var k, v string
	k = "postkey"
	v = "postvalue"
	app.Post("/", func(ctx *iris.Context) {
		ctx.WriteString(k + "=" + ctx.FormValue(k))
	})
	e := httptest.New(app, t)

	e.POST("/").WithFormField(k, v).Expect().Status(iris.StatusOK).Body().Equal(k + "=" + v)
}

func TestContextSubdomain(t *testing.T) {
	app := iris.New(iris.Configuration{VHost: "mydomain.com:9999"})
	app.Adapt(httprouter.New())

	//Default.Config.Tester.ListeningAddr = "mydomain.com:9999"
	// Default.Config.Tester.ExplicitURL = true
	app.Party("mysubdomain.").Get("/mypath", func(ctx *iris.Context) {
		ctx.WriteString(ctx.Subdomain())
	})

	e := httptest.New(app, t)

	e.GET("/").WithURL("http://mysubdomain.mydomain.com:9999").Expect().Status(iris.StatusNotFound)
	e.GET("/mypath").WithURL("http://mysubdomain.mydomain.com:9999").Expect().Status(iris.StatusOK).Body().Equal("mysubdomain")

	// e.GET("http://mysubdomain.mydomain.com:9999").Expect().Status(iris.StatusNotFound)
	// e.GET("http://mysubdomain.mydomain.com:9999/mypath").Expect().Status(iris.StatusOK).Body().Equal("mysubdomain")
}

func TestLimitRequestBodySizeMiddleware(t *testing.T) {
	const maxBodySize = 1 << 20

	app := iris.New()
	app.Adapt(newTestNativeRouter())
	// or inside handler: ctx.SetMaxRequestBodySize(int64(maxBodySize))
	app.Use(iris.LimitRequestBodySize(maxBodySize))

	app.Post("/", func(ctx *iris.Context) {
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
	// app.UseGlobal(iris.LimitRequestBodySize(int64(maxBodySize)))

	e := httptest.New(app, t)

	// test with small body
	e.POST("/").WithBytes([]byte("ok")).Expect().Status(iris.StatusOK).Body().Equal("ok")
	// test with equal to max body size limit
	bsent := make([]byte, maxBodySize, maxBodySize)
	e.POST("/").WithBytes(bsent).Expect().Status(iris.StatusOK).Body().Length().Equal(len(bsent))
	// test with larger body sent and wait for the custom response
	largerBSent := make([]byte, maxBodySize+1, maxBodySize+1)
	e.POST("/").WithBytes(largerBSent).Expect().Status(iris.StatusBadRequest).Body().Equal("http: request body too large")

}
