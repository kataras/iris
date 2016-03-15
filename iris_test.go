package iris

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const (
	CustomNotFoundErrorMessage = "custom error 404"
)

type TestRequestRoute struct {
	Method             string
	Path               string
	Body               []byte
	ExpectedStatusCode int

	ExpectedParameters map[string]string
}

type TestRoute struct {
	Methods []string
	Path    string

	Requests []TestRequestRoute
}

var (
	api        *Station
	testServer *httptest.Server
	server     = struct {
		URL, IP string
		PORT    int
	}{URL: "http://localhost", PORT: 80, IP: "127.0.0.1"}

	inlineRoutes = [...]TestRoute{
		{
			Methods: HTTPMethods.ANY, Path: "/simple",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/simple",
				Body:               []byte("body for the /simple"),
				ExpectedStatusCode: 200,
				ExpectedParameters: nil,
			}, {
				Method: "GET", Path: "/simple/wrongpath",
				Body:               []byte("body for the /simple but wrongpath"),
				ExpectedStatusCode: 404,
				ExpectedParameters: nil,
			}},
		},
		{
			Methods: HTTPMethods.ANY, Path: "/test/others/:something/work/path/muchmore/biggest",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/test/others/something/work/path/muchmore/biggest",
				Body:               []byte("body for the /test/others/:something/work/path/muchmore/biggest"),
				ExpectedStatusCode: 200,
				ExpectedParameters: nil,
			}},
		},
		{
			Methods: HTTPMethods.ANY, Path: "/test/others/:something/work/path",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/test/others/something/work/path",
				Body:               []byte("body for the /test/others/:something/work/path"),
				ExpectedStatusCode: 200,
				ExpectedParameters: nil,
			}, {
				Method: "GET", Path: "/test/others/anything/doesnt/work",
				Body:               []byte("body for the /test/others/anything/doesnt/work"),
				ExpectedStatusCode: 404,
				ExpectedParameters: nil,
			}},
		},
		{
			Methods: HTTPMethods.ANY, Path: "/api/users/:userId",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/api/users/1", // an to kanw /dsadsa adi gia /1 leitourgei ara kati pezei me ta index kai ta :
				Body:               []byte("body for the api/users/:userId"),
				ExpectedStatusCode: 200,
				ExpectedParameters: map[string]string{"userId": "1"},
			}, {
				Method: "GET", Path: "/api/users",
				Body:               []byte("body for the api/users/:userId"),
				ExpectedStatusCode: 404,
				ExpectedParameters: nil,
			}},
		},
		{
			Methods: HTTPMethods.ANY, Path: "/profile/:username/friends/:friendId",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/profile/kataras/friends/2",
				Body:               []byte("body for the /profile/:username/friends/:friendId"),
				ExpectedStatusCode: 200,
				ExpectedParameters: map[string]string{"username": "kataras", "friendId": "2"},
			}, {
				Method: "GET", Path: "/profile/kataras/friends/dsadsad/sadsa",
				Body:               []byte("body for the  /profile/:username/friends/:friendId"),
				ExpectedStatusCode: 404,
				ExpectedParameters: nil,
			}},
		},
		{
			Methods: HTTPMethods.ANY, Path: "/profile/:username/friends/:friendId/something/here",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/profile/kataras/friends/2/something/here",
				Body:               []byte("body for the /profile/:username/friends/somethinghere/:friendId/something/here"),
				ExpectedStatusCode: 200,
				ExpectedParameters: map[string]string{"username": "kataras", "friendId": "2"},
			}, {
				Method: "GET", Path: "/profile/kataras/friends/stringerrorisnotanumberman/something",
				Body:               []byte("body for the /profile/:username/friends/somethinghere/:friendId/something/here"),
				ExpectedStatusCode: 404,
				ExpectedParameters: nil,
			}},
		},
		{
			Methods: HTTPMethods.ANY, Path: "/profile/:username",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/profile/kataras",
				Body:               []byte("body for the profile/:username"),
				ExpectedStatusCode: 200,
				ExpectedParameters: map[string]string{"username": "kataras"},
			}, {
				Method: "GET", Path: "/profile/kataras/somethingelsehere",
				Body:               []byte("body for the profile/:username"),
				ExpectedStatusCode: 404,
				ExpectedParameters: nil,
			}},
		},
		{
			Methods: HTTPMethods.ANY, Path: "/wildcard/:username/any/*anyusername",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/wildcard/kataras/any/blablabla/bleleblelbe",
				Body:               []byte("body for the /wildcard/any/*"),
				ExpectedStatusCode: 200,
				ExpectedParameters: map[string]string{"username": "kataras"},
			}, {
				Method: "GET", Path: "/wildcard/kataras/any",
				Body:               []byte("body for the /wildcard/kataras/any"),
				ExpectedStatusCode: 404,
				ExpectedParameters: nil,
			}},
		},
		{
			Methods: HTTPMethods.ANY, Path: "/wildcard2/*anything",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/wildcard2/kataras/dsadsadsa/dsasasa",
				Body:               []byte("body for the /wildcard2/*"),
				ExpectedStatusCode: 200,
				ExpectedParameters: nil,
			}, {
				Method: "GET", Path: "/wildcard2",
				Body:               []byte("body for the /wildcard"),
				ExpectedStatusCode: 404,
				ExpectedParameters: nil,
			}},
		},
	}
)

func TestMain(m *testing.M) {
	setup()
	result := m.Run()
	teardown()
	os.Exit(result)
}

func setup() {
	//api = New()
	api = Custom(StationOptions{Cache: false})
}

func teardown() {
	if testServer != nil {
		testServer.Close()
	}

}

func getRequestRoute(route TestRoute, reqURL string) *TestRequestRoute {
	for _, reqRoute := range route.Requests {
		if reqRoute.Path == reqURL {
			return &reqRoute
		}
	}
	return nil
}

func checkParams(c *Context, expected map[string]string) error {
	if expected != nil {
		for key, value := range expected {
			contextParamValue := c.Param(key)
			if contextParamValue == "" {
				msg := fmt.Sprintf("Expecting parameter "+key+" which is not registed to the url %s. Context's Parameters: %v", c.Request.URL.Path, c.Params)
				return errors.New(msg)
			}

			if contextParamValue != value {
				msg := fmt.Sprintf(c.Request.URL.Path+" Expected parameter ( "+key+" ) value ( %v ) is not equal to the Context's parameter value: "+contextParamValue, value)
				return errors.New(msg)
			}
		}
	}

	return nil
}

func checkBody(c *Context, expectedBody []byte) error {
	reqBody, err := ioutil.ReadAll(c.Request.Body)

	if err != nil {
		msg := fmt.Sprintf("Error reading body for the request url ( "+c.Request.URL.Path+" ) but it was expected  %v . Error message: ", string(expectedBody), err.Error())
		return errors.New(msg)
	}

	if reqBody == nil && expectedBody != nil {
		msg := fmt.Sprintf("Expecting body ( " + string(expectedBody) + " )but the request has not sent a body ")
		return errors.New(msg)
	}

	if reqBody != nil && expectedBody == nil {
		msg := fmt.Sprintf("The request didn't expect any body but the request has sent a body: ")
		return errors.New(msg)
	}

	if string(reqBody) != string(expectedBody) {
		msg := fmt.Sprintf("The Expecting body IS NOT EQUAL to the requested body %v != %v", string(expectedBody), string(reqBody))
		return errors.New(msg)
	}

	return nil
}

func handleRoute(route TestRoute) func(c *Context) {
	return func(c *Context) {
		defer c.Close()

		c.Write("Response from server to the client for route: " + route.Path + " client req url: " + c.Request.URL.Path)

		reqURL := c.Request.URL.Path
		requestRoute := getRequestRoute(route, reqURL)

		if requestRoute == nil {
			log.Fatal("No test-registed request url found for route ", route.Path)
			return
		}
		go func(cc *Context) {

			if err := checkParams(cc, requestRoute.ExpectedParameters); err != nil {
				log.Fatal(err.Error())
			}

		}(c.Clone())

		if err := checkBody(c, requestRoute.Body); err != nil {
			log.Fatal(err.Error())
		}

	}
}

func TestRoutesServerSide(t *testing.T) {

	for _, route := range inlineRoutes {
		api.HandleFunc(route.Methods[0], route.Path, handleRoute(route))
	}
	// Set custom error messages
	api.Errors().On(http.StatusNotFound, HandlerFunc(func(c *Context) {
		http.Error(c.ResponseWriter, CustomNotFoundErrorMessage, http.StatusNotFound)
	}))

	api.Build()
	testServer = httptest.NewUnstartedServer(api)

	testServer.Start()
	server.URL = testServer.URL
}

func TestRoutesClientSide(t *testing.T) {
	t.Parallel()
	for _, route := range inlineRoutes {
		for _, requestRoute := range route.Requests {
			buffer := new(bytes.Buffer)
			_, err := buffer.Write(requestRoute.Body)
			if err != nil {
				t.Fatal("Error creating the buffer for Route's body : ", route.Path+" Error: ", err.Error())
			}
			req, err := http.NewRequest(requestRoute.Method, server.URL+requestRoute.Path, buffer)

			if err != nil {
				t.Fatal("Error creating the NewRequest for Route: ", route.Path+" Error with url: ", err.Error())
			} else {
				res, err := http.DefaultClient.Do(req)
				res.Close = true

				if err != nil {
					t.Fatal("Error on do client request to the server for Route: ", route.Path+" ERR: ", err.Error())
				} else {

					defer res.Body.Close()

					if res.StatusCode != requestRoute.ExpectedStatusCode {
						t.Fatal("Expecting StatusCode: ", requestRoute.ExpectedStatusCode, " but we got: ", res.StatusCode, " for root Route: "+route.Path, " -> ", requestRoute.Path)
					} else {
						customErrHandler := api.Errors().getByCode(res.StatusCode)
						//if we get the status we want and it was error  read the body to see if the error message is the same as setted as custom error message

						if customErrHandler != nil {
							//we have an error and we have setted a custom error message for this error status code.
							responseBodyBytes, err := ioutil.ReadAll(res.Body)
							responseBody := string(bytes.TrimSpace(responseBodyBytes))
							if err == nil {
								if responseBody != CustomNotFoundErrorMessage {
									//if the body of the error page is different that we have setted, then the test failed
									t.Fatal("Excpecting custom message '", CustomNotFoundErrorMessage, "' for the Status Code ", res.StatusCode, " but we got '", responseBody, "' from response.")
								}
							}
						}

					}

				}

			}

		}

	}
}
