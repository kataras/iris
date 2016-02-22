package iris

import (
	"net/http"
	"testing"
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

// go test -bench BenchmarkTheRouter -run XXX
// go test -run=XXX -bench=.
// working: go test -bench BenchmarkTheRouter
func BenchmarkRouter(b *testing.B) {
	api := New()
	for _, route := range inlineRoutes {
		api.Handle(route.Path, func(res http.ResponseWriter, req *http.Request) {

		}).Methods(route.Methods...)
	}
	go http.ListenAndServe(":8080", api)
	server.URL = "http://localhost:8080"
	b.ReportAllocs()
	b.ResetTimer()
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
PASS
BenchmarkRouter-8          50000             35582 ns/op            9961 B/op        121 allocs/op
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
