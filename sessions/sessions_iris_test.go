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
//
// This source code file is based on the Gorilla's sessions package.
//
package sessions

import (
	"github.com/kataras/iris"
	"net/http"
	"testing"
)

type fakeResponseWriter struct{}

func (f *fakeResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (f *fakeResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (f *fakeResponseWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (f *fakeResponseWriter) WriteHeader(int) {}

type routeTest struct {
	method string
	path   string
}

type TestSession struct {
	Path                 string
	Values               map[string]string
	ExpectedBodyResponse string
}

var testValues = map[string]string{"username": "kataras", "project": "iris", "year": "2016"}

func TestSessionsWithIris(t *testing.T) {
	secret := []byte("myIrisSecretKey")
	store := NewCookieStore(secret)
	wrapper := New("test_sessions", store)

	iris.Get("/test_set", func(c *iris.Context) {
		//get the session for this context
		session, err := wrapper.Get(c)

		if err != nil {
			t.Fatal("Sessions error: " + err.Error())
			c.SendStatus(500, err.Error())
			return
		}
		//set session values
		for k, v := range testValues {
			session.Set(k, v)
		}

		//save them
		session.Save(c)

		//write anthing
		c.SendStatus(200, "ok")
	})

	iris.Get("/test_get", func(c *iris.Context) {
		//again get the session for this context
		session, err := wrapper.Get(c)

		if err != nil {
			t.Fatal("Sessions error: " + err.Error())
			c.SendStatus(500, err.Error())
			return
		}
		//get the session value
		for k, v := range testValues {

			if p := session.GetString(k); p != v {
				t.Log("Sessions Error: on test_get(1). Session values: ")
				for sesK, sesV := range session.Values {
					t.Log("\n", sesK, " = ", sesV)
				}
				t.Fatal("Sessions error: on test_get(2). Values are not matched key(" + k + ") value: " + v + " != " + p)
			}
		}

		c.SendStatus(200, "ok")
	})

	iris.Get("/test_clear", func(c *iris.Context) {
		session, err := wrapper.Get(c)
		if err != nil {
			t.Fatal("Sessions error: " + err.Error())
			c.SendStatus(500, err.Error())
			return
		}

		for k := range testValues {
			session.Delete(k)
		}
		for k := range testValues {
			if p := session.GetString(k); p != "" {
				t.Fatal("Sessions error: on test_clear, values are not deleted, this should be nil " + k + " == " + p + "?")
			}
		}
		c.SendStatus(200, "ok")

	})

	res := new(fakeResponseWriter)
	req, _ := http.NewRequest("GET", "/", nil)
	iris.ServeHTTP(res, mockReq(req, "GET", "/test_set"))
	iris.ServeHTTP(res, mockReq(req, "GET", "/test_get"))
	iris.ServeHTTP(res, mockReq(req, "GET", "/test_clear"))

}

func mockReq(req *http.Request, method string, path string) *http.Request {

	u := req.URL
	req.Method = method
	req.RequestURI = path
	u.Path = path

	return req
}
