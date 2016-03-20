// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
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

			if sr.expectedPathPrefix != route.GetPathPrefix() {
				t.Fatal("Error on compare pathPrefix: " + sr.expectedPathPrefix + " != " + route.GetPathPrefix())
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
