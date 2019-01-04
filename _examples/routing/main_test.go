package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/kataras/iris/httptest"
)

func calculatePathAndResponse(method, subdomain, path string, paramKeyValue ...string) (string, string) {
	paramsLen := 0

	if l := len(paramKeyValue); l >= 2 {
		paramsLen = len(paramKeyValue) / 2
	}

	paramsInfo := ""
	if paramsLen > 0 {
		for i := 0; i < len(paramKeyValue); i++ {
			paramKey := paramKeyValue[i]
			i++
			if i >= len(paramKeyValue) {
				panic("paramKeyValue should be align with path parameters {} and must be placed in order")
			}

			paramValue := paramKeyValue[i]
			paramsInfo += paramKey + " = " + paramValue + "\n"

			beginParam := strings.IndexByte(path, '{')
			endParam := strings.IndexByte(path, '}')
			if beginParam == -1 || endParam == -1 {
				panic("something wrong with parameters, please define them in order")
			}

			path = path[:beginParam] + paramValue + path[endParam+1:]
		}
	}

	return path, paramsInfo + `Info

Method: ` + method + `
Subdomain: ` + subdomain + `
Path: ` + path + `
Parameters length: ` + strconv.Itoa(paramsLen)
}

type troute struct {
	method, subdomain, path string
	status                  int
	expectedBody            string
	contentType             string
}

func newTroute(method, subdomain, path string, status int, paramKeyValue ...string) troute {
	finalPath, expectedBody := calculatePathAndResponse(method, subdomain, path, paramKeyValue...)
	contentType := "text/plain; charset=UTF-8"

	if status == httptest.StatusNotFound {
		expectedBody = notFoundHTML
		contentType = "text/html; charset=UTF-8"
	}

	return troute{
		contentType:  contentType,
		method:       method,
		subdomain:    subdomain,
		path:         finalPath,
		status:       status,
		expectedBody: expectedBody,
	}
}

func TestRouting(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	var tests = []troute{
		// GET
		newTroute("GET", "", "/healthcheck", httptest.StatusOK),
		newTroute("GET", "", "/games/{gameID}/clans", httptest.StatusOK, "gameID", "42"),
		newTroute("GET", "", "/games/{gameID}/clans/clan/{clanPublicID}", httptest.StatusOK, "gameID", "42", "clanPublicID", "93"),
		newTroute("GET", "", "/games/{gameID}/clans/search", httptest.StatusOK, "gameID", "42"),
		newTroute("GET", "", "/games/challenge", httptest.StatusOK),
		newTroute("GET", "", "/games/challenge/beginner/start", httptest.StatusOK),
		newTroute("GET", "", "/games/challenge/beginner/level/first", httptest.StatusOK),
		newTroute("GET", "", "/games/challenge/intermediate", httptest.StatusOK),
		newTroute("GET", "", "/games/challenge/intermediate/start", httptest.StatusOK),
		newTroute("GET", "mysubdomain", "/", httptest.StatusOK),
		newTroute("GET", "mywildcardsubdomain", "/", httptest.StatusOK),
		newTroute("GET", "mywildcardsubdomain", "/party", httptest.StatusOK),
		// PUT
		newTroute("PUT", "", "/games/{gameID}/players/{clanPublicID}", httptest.StatusOK, "gameID", "42", "clanPublicID", "93"),
		newTroute("PUT", "", "/games/{gameID}/clans/clan/{clanPublicID}", httptest.StatusOK, "gameID", "42", "clanPublicID", "93"),
		// POST
		newTroute("POST", "", "/games/{gameID}/clans", httptest.StatusOK, "gameID", "42"),
		newTroute("POST", "", "/games/{gameID}/players", httptest.StatusOK, "gameID", "42"),
		newTroute("POST", "", "/games/{gameID}/clans/{clanPublicID}/leave", httptest.StatusOK, "gameID", "42", "clanPublicID", "93"),
		newTroute("POST", "", "/games/{gameID}/clans/{clanPublicID}/memberships/application", httptest.StatusOK, "gameID", "42", "clanPublicID", "93"),
		newTroute("POST", "", "/games/{gameID}/clans/{clanPublicID}/memberships/application/{action}", httptest.StatusOK, "gameID", "42", "clanPublicID", "93", "action", "somethinghere"),
		newTroute("POST", "", "/games/{gameID}/clans/{clanPublicID}/memberships/invitation", httptest.StatusOK, "gameID", "42", "clanPublicID", "93"),
		newTroute("POST", "", "/games/{gameID}/clans/{clanPublicID}/memberships/invitation/{action}", httptest.StatusOK, "gameID", "42", "clanPublicID", "93", "action", "somethinghere"),
		newTroute("POST", "", "/games/{gameID}/clans/{clanPublicID}/memberships/delete", httptest.StatusOK, "gameID", "42", "clanPublicID", "93"),
		newTroute("POST", "", "/games/{gameID}/clans/{clanPublicID}/memberships/promote", httptest.StatusOK, "gameID", "42", "clanPublicID", "93"),
		newTroute("POST", "", "/games/{gameID}/clans/{clanPublicID}/memberships/demote", httptest.StatusOK, "gameID", "42", "clanPublicID", "93"),
		// POST: / will be tested alone
		// custom not found
		newTroute("GET", "", "/notfound", httptest.StatusNotFound),
		newTroute("POST", "", "/notfound2", httptest.StatusNotFound),
		newTroute("PUT", "", "/notfound3", httptest.StatusNotFound),
		newTroute("GET", "mysubdomain", "/notfound42", httptest.StatusNotFound),
	}

	for _, tt := range tests {
		et := e.Request(tt.method, tt.path)
		if tt.subdomain != "" {
			et.WithURL("http://" + tt.subdomain + ".localhost:8080")
		}
		et.Expect().Status(tt.status).Body().Equal(tt.expectedBody)
	}

	// test POST "/" limit data and post data return

	// test with small body
	e.POST("/").WithBytes([]byte("ok")).Expect().Status(httptest.StatusOK).Body().Equal("ok")
	// test with equal to max body size limit
	bsent := make([]byte, maxBodySize, maxBodySize)
	e.POST("/").WithBytes(bsent).Expect().Status(httptest.StatusOK).Body().Length().Equal(len(bsent))
	// test with larger body sent and wait for the custom response
	largerBSent := make([]byte, maxBodySize+1, maxBodySize+1)
	e.POST("/").WithBytes(largerBSent).Expect().Status(httptest.StatusBadRequest).Body().Equal("http: request body too large")

	// test the post value (both post and put) and headers.
	e.PUT("/postvalue").WithFormField("name", "test_put").
		WithHeader("headername", "headervalue_put").Expect().
		Status(httptest.StatusOK).Body().Equal("Hello test_put | headervalue_put")

	e.POST("/postvalue").WithFormField("name", "test_post").
		WithHeader("headername", "headervalue_post").Expect().
		Status(httptest.StatusOK).Body().Equal("Hello test_post | headervalue_post")
}
