package iris

import (
	"strings"
	"testing"
)

type testAPIUsersHandler struct {
	Annotated `get:"/api/users/:userId" template:"user.html"`
}

func (t *testAPIUsersHandler) Handle(ctx *Context, renderer *Renderer) {}

type testStructedRoute struct {
	handler                  Annotated
	expectedMethods          []string
	expectedPathPrefix       string
	expectedTemplateFilename string
}

var structedTests = [...]testStructedRoute{{
	handler:                  new(testAPIUsersHandler),
	expectedMethods:          []string{"GET"},
	expectedPathPrefix:       "/api/users/",
	expectedTemplateFilename: "/user.html",
}}

func TestRouterHandleAnnotated(t *testing.T) {
	iris := New()
	for _, sr := range structedTests {
		route, err := iris.router.handleAnnotated(sr.handler)
		//var err error
		//route := iris.Handle(sr.handler)
		if err != nil {
			t.Fatal("Error on TestRouterHandleAnnotated: " + err.Error())
		} else {
			if !slicesAreEqual(sr.expectedMethods, route.methods) {
				t.Fatal("Error on compare Methods: " + strings.Join(sr.expectedMethods, ",") + " != " + strings.Join(route.methods, ","))
			}

			if sr.expectedPathPrefix != route.pathPrefix {
				t.Fatal("Error on compare pathPrefix: " + sr.expectedPathPrefix + " != " + route.pathPrefix)
			}

			if templatesDirectory+sr.expectedTemplateFilename != route.templates.filesTemp[0] {
				t.Fatal("Error on compare Template filename: " + templatesDirectory + sr.expectedTemplateFilename + " != " + route.templates.filesTemp[0])
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
