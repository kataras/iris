package iris

import (
	"io"
	"net/http"
	_ "runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/lars"
)

var (
	githubGin  http.Handler
	githubLARS http.Handler
)

//this goes to the benchmark_github.test.go which have the init() func.
func init() {
	if loadOtherBenchmarks {
		calcMem("Gin", func() {
			githubGin = loadGin(githubAPI)
		})

		println()

		calcMem("lars", func() {
			githubLARS = loadLARS(githubAPI)
		})

		println()
	}

}

func larsHandleTest(c lars.Context) {
	io.WriteString(c.Response(), c.Request().RequestURI)
}

func larsHandleTestTypical(res http.ResponseWriter, req *http.Request) {
	io.WriteString(res, req.RequestURI)
}

func loadLARS(routes []routeTest) http.Handler {
	h := larsHandleTest

	l := lars.New()

	for _, r := range routes {
		switch r.method {
		case lars.GET:
			l.Get(r.path, h)
		case lars.POST:
			l.Post(r.path, h)
		case lars.PUT:
			l.Put(r.path, h)
		case lars.PATCH:
			l.Patch(r.path, h)
		case lars.DELETE:
			l.Delete(r.path, h)
		default:
			panic("Unknow HTTP method: " + r.method)
		}
	}
	return l.Serve()
}

func BenchmarkLARS_GithubAll(b *testing.B) {
	benchRoutes(b, githubLARS, githubAPI)
}

//Gin doesn't provide typical handle
func ginHandleTest(c *gin.Context) {
	io.WriteString(c.Writer, c.Request.RequestURI)
}

func loadGin(routes []routeTest) http.Handler {
	h := ginHandleTest
	gin.SetMode("release") // turn off console debug messages
	api := gin.New()

	for _, route := range routes {
		api.Handle(route.method, route.path, h)
	}
	return api

}

//results: 30000	54103 ns/op		0 B/op 		0 allocs/op
func BenchmarkGin_GithubAll(b *testing.B) {
	benchRoutes(b, githubGin, githubAPI)
}
