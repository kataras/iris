package gapi

import (
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

	ExpectedParameters map[string]interface{} //used on server, inside handler
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

	tests = [...]TestRoute{{
		Methods: HTTPMethods.ANY, Path: "/profile/:username",
		Requests: []TestRequestRoute{{
			Method: "GET", Path: "/profile/kataras",
			Body:               []byte("body for the profile/:username"),
			ExpectedStatusCode: 200,
			ExpectedParameters: map[string]interface{}{"username": "kataras"},
		}},
	},
		{
			Methods: HTTPMethods.ANY, Path: "/api/users/:userId(int)",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/api/users/1",
				Body:               []byte("body for the api/users/:userId(int)"),
				ExpectedStatusCode: 200,
				ExpectedParameters: map[string]interface{}{"userId": 1},
			}},
		},
		{
			Methods: HTTPMethods.ANY, Path: "/profile/:username/friends/:friendId(int)",
			Requests: []TestRequestRoute{{
				Method: "GET", Path: "/profile/kataras/friends/2",
				Body:               []byte("body for the /profile/:username/friends/:friendId"),
				ExpectedStatusCode: 200,
				ExpectedParameters: map[string]interface{}{"username": "kataras", "friendId": 2},
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

func TestRoutesServerSide(t *testing.T) {
	for _, route := range tests {
		
		api.Handle(route.Path, func(c *Context) {
			c.Write("Response from server to the client for route: " + route.Path + " client req url: " + c.Request.URL.Path)

		}).Methods(route.Methods...)

	}

	testServer = httptest.NewUnstartedServer(api)
	testServer.Start()
	server.URL = testServer.URL
}

func TestRoutesClientSide(t *testing.T) {
	for _, route := range tests {
		for _, requestRoute := range route.Requests {
			client := &http.Client{}
			req, err := http.NewRequest(requestRoute.Method, server.URL+requestRoute.Path, nil)
			if err != nil {
				t.Fatal("Error creating the NewRequest for Route: ", route.Path+" Error with url: ", err.Error())
			} else {
				res, err := client.Do(req)
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
