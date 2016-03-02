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

	ExpectedParameters map[string]string //used on server, inside handler
}

type TestRoute struct {
	Methods []string
	Path    string

	Requests []TestRequestRoute
}

var (
	api        *Server
	testServer *httptest.Server
	server     = struct {
		URL, IP string
		PORT    int
	}{URL: "http://localhost", PORT: 80, IP: "127.0.0.1"}
	/*
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
	   			Methods: HTTPMethods.ANY, Path: "/api/users/:userId(int)",
	   			Requests: []TestRequestRoute{{
	   				Method: "GET", Path: "/api/users/1",
	   				Body:               []byte("body for the api/users/:userId(int)"),
	   				ExpectedStatusCode: 200,
	   				ExpectedParameters: map[string]string{"userId": "1"},
	   			}, {
	   				Method: "GET", Path: "/api/users/thisisastringnotanumb3r",
	   				Body:               []byte("body for the api/users/:userId(int)"),
	   				ExpectedStatusCode: 404,
	   				ExpectedParameters: nil,
	   			}},
	   		},
	   		{
	   			Methods: HTTPMethods.ANY, Path: "/profile/:username/friends/:friendId(int)",
	   			Requests: []TestRequestRoute{{
	   				Method: "GET", Path: "/profile/kataras/friends/2",
	   				Body:               []byte("body for the /profile/:username/friends/:friendId"),
	   				ExpectedStatusCode: 200,
	   				ExpectedParameters: map[string]string{"username": "kataras", "friendId": "2"},
	   			}, {
	   				Method: "GET", Path: "/profile/kataras/friends/stringerrorisnotanumberman",
	   				Body:               []byte("body for the  /profile/:username/friends/:friendId(int)"),
	   				ExpectedStatusCode: 404,
	   				ExpectedParameters: nil,
	   			}},
	   		},
	   		{
	   			Methods: HTTPMethods.ANY, Path: "/profile/:username/friends/somethinghere/:friendId(int)/something/here",
	   			Requests: []TestRequestRoute{{
	   				Method: "GET", Path: "/profile/kataras/friends/somethinghere/2/something/here",
	   				Body:               []byte("body for the /profile/:username/friends/somethinghere/:friendId(int)/something/here"),
	   				ExpectedStatusCode: 200,
	   				ExpectedParameters: map[string]string{"username": "kataras", "friendId": "2"},
	   			}, {
	   				Method: "GET", Path: "/profile/kataras/friends/somethinghere/stringerrorisnotanumberman/something/here",
	   				Body:               []byte("body for the /profile/:username/friends/somethinghere/:friendId(int)/something/here"),
	   				ExpectedStatusCode: 404,
	   				ExpectedParameters: nil,
	   			}},
	   		},
	   		{
	   			Methods: HTTPMethods.ANY, Path: "/profile/:username/friends/:friendId(int)/something/here/:thirdParam",
	   			Requests: []TestRequestRoute{{
	   				Method: "GET", Path: "/profile/kataras/friends/2/something/here/thethirdparameter",
	   				Body:               []byte("body for the /profile/:username/friends/:friendId(int)/something/here/:thirdParam"),
	   				ExpectedStatusCode: 200,
	   				ExpectedParameters: map[string]string{"username": "kataras", "friendId": "2", "thirdParam": "thethirdparameter"},
	   			}, {
	   				Method: "GET", Path: "/profile/kataras/friends/2/something/here",
	   				Body:               []byte("body for the /profile/:username/friends/:friendId(int)/something/here/:thirdParam"),
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
	   			Methods: HTTPMethods.ANY, Path: "/home/:username([a-zA-Z]+)",
	   			Requests: []TestRequestRoute{{
	   				Method: "GET", Path: "/home/Kataras",
	   				Body:               []byte("body for the /home/:username([a-zA-Z]+)"),
	   				ExpectedStatusCode: 200,
	   				ExpectedParameters: map[string]string{"username": "Kataras"},
	   			}, {
	   				Method: "GET", Path: "/home/shouldN0tF0und",
	   				Body:               []byte("body for the /home/:username([a-zA-Z]+)"),
	   				ExpectedStatusCode: 404,
	   				ExpectedParameters: nil,
	   			}},
	   		},
	   	}
	   )
	*/

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
			Methods: HTTPMethods.ANY, Path: "/api/users/:userId",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/api/users/1",
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
			Methods: HTTPMethods.ANY, Path: "/profile/:username/friends/somethinghere/:friendId/something/here",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/profile/kataras/friends/somethinghere/2/something/here",
				Body:               []byte("body for the /profile/:username/friends/somethinghere/:friendId/something/here"),
				ExpectedStatusCode: 200,
				ExpectedParameters: map[string]string{"username": "kataras", "friendId": "2"},
			}, {
				Method: "GET", Path: "/profile/kataras/friends/somethinghere/stringerrorisnotanumberman/something",
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
			Methods: HTTPMethods.ANY, Path: "/wildcard/:username/any/*",
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
			Methods: HTTPMethods.ANY, Path: "/wildcard2/*",
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
	//println("iris_test.go TestMain started")
	setup()

	result := m.Run()
	teardown()
	os.Exit(result)
}

func setup() {
	api = New()
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

func checkParams(c Context, expected map[string]string) error {
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

func checkBody(c Context, expectedBody []byte) error {
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

//tests are not working here, I tried with recorder on request and also sily passing the testing.T here but doesnt work too so I will use the normal 'log' package for errors
func handleRoute(route TestRoute) func(c Context) {
	return func(c Context) {
		defer c.Close()

		c.Write("Response from server to the client for route: " + route.Path + " client req url: " + c.Request.URL.Path)

		reqURL := c.Request.URL.Path
		requestRoute := getRequestRoute(route, reqURL)

		if requestRoute == nil {
			log.Fatal("No test-registed request url found for route ", route.Path)
			return
		}

		if err := checkParams(c, requestRoute.ExpectedParameters); err != nil {
			log.Fatal(err.Error())
		}

		if err := checkBody(c, requestRoute.Body); err != nil {
			log.Fatal(err.Error())
		}

	}
}

func TestRoutesServerSide(t *testing.T) {
	for _, route := range inlineRoutes {
		api.Handle(route.Path, handleRoute(route)).Methods(route.Methods...)
	}

	// Set custom error messages
	api.Errors.On(http.StatusNotFound, func(res http.ResponseWriter, req *http.Request) {
		http.Error(res, CustomNotFoundErrorMessage, http.StatusNotFound)
	})

	testServer = httptest.NewUnstartedServer(api)
	/*recorder := httptest.NewRecorder()
	testServer = httptest.NewUnstartedServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		route, errCode := api.server.router.find(req)
		if route == nil {
			res.WriteHeader(errCode)
			res.Write([]byte("from the handler"))
			req.Body.Close()
		} else {
			route.ServeHTTP(recorder, req)
		}
	}))*/
	testServer.Start()
	server.URL = testServer.URL
}

func TestRoutesClientSide(t *testing.T) {
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
				//	res, err := client.Do(req)
				res, err := http.DefaultClient.Do(req)
				res.Close = true

				if err != nil {
					t.Fatal("Error on do client request to the server for Route: ", route.Path+" ERR: ", err.Error())
				} else {

					defer res.Body.Close()

					if res.StatusCode != requestRoute.ExpectedStatusCode {
						t.Fatal("Expecting StatusCode: ", requestRoute.ExpectedStatusCode, " but we got: ", res.StatusCode, " for Route: "+route.Path)
					} else {
						customErrHandler := api.Errors.getByCode(res.StatusCode)
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
