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

		for k, _ := range testValues {
			session.Delete(k)
		}
		for k, _ := range testValues {
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
