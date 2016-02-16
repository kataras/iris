package gapi

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
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
	api        *Gapi
	testServer *httptest.Server
	server     = struct {
		URL, IP string
		PORT    int
	}{URL: "http://localhost", PORT: 80, IP: "127.0.0.1"}

	tests = [...]TestRoute{
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

func TestMain(m *testing.M) {
	setup()
	result := m.Run()
	teardown()
	os.Exit(result)
}

func setup() {
	api = New()
}

func teardown() {
	testServer.Close()
}

func getRequestRoute(route TestRoute, reqUrl string) *TestRequestRoute {
	for _, reqRoute := range route.Requests {
		if reqRoute.Path == reqUrl {
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
				msg := fmt.Sprintf("Expected parameter ( "+key+" ) value ( %v ) is not equal to the Context's parameter value: "+contextParamValue, value)
				return errors.New(msg)
			}
		}
	}

	return nil
}

//tests are not working here, I tried with recorder on request and also sily passing the testing.T here but doesnt work too so I will use the normal 'log' package for errors
func handleRoute(route TestRoute) func(c *Context) {
	return func(c *Context) {
		defer c.Close()

		c.Write("Response from server to the client for route: " + route.Path + " client req url: " + c.Request.URL.Path)

		reqUrl := c.Request.URL.Path
		requestRoute := getRequestRoute(route, reqUrl)

		if requestRoute == nil {
			log.Fatal("No test-registed request url found for route ", route.Path)
			return
		}

		if err := checkParams(c, requestRoute.ExpectedParameters); err != nil {
			log.Fatal(err.Error())
		}

	}
}

func TestRoutesServerSide(t *testing.T) {
	for _, route := range tests {
		api.Handle(route.Path, handleRoute(route)).Methods(route.Methods...)
	}

	testServer = httptest.NewUnstartedServer(api)
	/*recorder := httptest.NewRecorder()
	testServer = httptest.NewUnstartedServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		route, errCode := api.server.Router.Find(req)
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
	for _, route := range tests {
		for _, requestRoute := range route.Requests {
			//	client := &http.Client{}
			req, err := http.NewRequest(requestRoute.Method, server.URL+requestRoute.Path, nil)
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
					}

				}

			}

		}

	}
}
