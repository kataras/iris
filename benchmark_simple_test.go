package iris

import (
	"github.com/pkg/profile"
	"net/http"
	"testing"
)

const (
	BenchmarkProfiler = false
)

// used ONLY in benchmark test
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

// BenchmarkRouter tests the router using contexted handler and a middleware
// go test -bench BenchmarkRouter -run XXX
// go test -run=XXX -bench=.
// working: go test -bench BenchmarkRouter
// add -benchtime 5s for example to see how much perfomance on 5 seconds instead of the 1s (default)
func BenchmarkRouter(b *testing.B) {

	api := New()
	for _, route := range inlineRoutes {
		r := api.Handle(route.Path, func(c *Context) {
			c.Close()
		})
		r.UseFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {

		})
		r.Methods(route.Methods...)
	}
	go http.ListenAndServe(":8080", api)
	server.URL = "http://localhost:8080"

	b.ReportAllocs()
	b.ResetTimer()
	if BenchmarkProfiler {
		defer profile.Start().Stop()
	}

	for i := 0; i < b.N; i++ {
		for _, route := range inlineRoutes {
			for _, requestRoute := range route.Requests {
				res := new(fakeResponseWriter)
				req, err := http.NewRequest(requestRoute.Method, server.URL+requestRoute.Path, nil)

				if err != nil {
					b.Fatal("Error creating the NewRequest for Route: ", route.Path+" Error with url: ", err.Error())
				} else {
					api.ServeHTTP(res, req)
					if err != nil {
						b.Fatal("Error on do client request to the server for Route: ", route.Path+" ERR: ", err.Error())
					}
				}
			}

		}

	}
}

/* Results :
BenchmarkRouter            50000             32881 ns/op            9828 B/op        117 allocs/op
BenchmarkRouter-8          50000             33521 ns/op            9834 B/op        117 allocs/op
//Almost no difference between them, I have to look it.
*/

// Other backup
// With the testing server and all checks, not a good idea but keep it.
// go test -bench BenchmarkTheRouter -run XXX
// go test -run=XXX -bench=.
/*func BenchmarkRouter(b *testing.B) {
	setup()
	TestRoutesServerSide(nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, route := range inlineRoutes {
			for _, requestRoute := range route.Requests {
				buffer := new(bytes.Buffer)
				_, err := buffer.Write(requestRoute.Body)
				if err != nil {
					b.Fatal("Error creating the buffer for Route's body : ", route.Path+" Error: ", err.Error())
				}
				req, err := http.NewRequest(requestRoute.Method, server.URL+requestRoute.Path, buffer)

				if err != nil {
					b.Fatal("Error creating the NewRequest for Route: ", route.Path+" Error with url: ", err.Error())
				} else {
					//	res, err := client.Do(req)
					res, err := http.DefaultClient.Do(req)
					res.Close = true

					if err != nil {
						b.Fatal("Error on do client request to the server for Route: ", route.Path+" ERR: ", err.Error())
					} else {
					}
				}
			}

		}

	}
	teardown()
}*/
