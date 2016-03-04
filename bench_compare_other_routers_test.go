package iris

import (
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"runtime"
	"testing"
)

var (
	githubGin http.Handler
)

//this goes to the benchmark_github.test.go which have the init() func.
func init() {
	runtime.GOMAXPROCS(1)
	println("#GithubAPI Routes:", len(githubAPI))

	calcMem("Gin", func() {
		githubGin = loadGin(githubAPI)
	})

	println()
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
