package iris

import (
	_ "strings"
	"testing"
)

type testAPIUsersHandler struct {
	Handler `get:"/api/users/:userId"`
}

func (t *testAPIUsersHandler) Serve(ctx *Context) {}

type testStructedRoute struct {
	handler                  Handler
	expectedMethod           string
	expectedPathPrefix       string
	expectedTemplateFilename string
}

var structedTests = [...]testStructedRoute{{
	handler:                  new(testAPIUsersHandler),
	expectedMethod:           HTTPMethods.GET,
	expectedPathPrefix:       "/api/users/",
	expectedTemplateFilename: "/user.html",
}}

func TestRouterHandleAnnotated(t *testing.T) {
	iris := New()
	for _, sr := range structedTests {
		route, err := iris.HandleAnnotated(sr.handler)
		if err != nil {
			t.Fatal("Error on TestRouterHandleAnnotated: " + err.Error())
		} else {

			if sr.expectedPathPrefix != route.PathPrefix {
				t.Fatal("Error on compare pathPrefix: " + sr.expectedPathPrefix + " != " + route.PathPrefix)
			}

		}
	}

}

func slicesAreEqual(s1, s2 []string) bool {

	if s1 == nil && s2 == nil {
		return true
	}

	if s1 == nil || s2 == nil {
		return false
	}

	if len(s1) != len(s2) {
		return false
	}

	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}

	return true
}
