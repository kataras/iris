package router

import (
	"bytes"
	"math/rand/v2"
	"testing"

	"github.com/kataras/iris/v14/context"

	"github.com/kataras/golog"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

// randString helps us to generate random lowercase paths for the test.
func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.IntN(len(letterBytes))]
	}

	return string(b)
}

func genPaths(routesLength, minCharLength, maxCharLength int) []string {
	// b := new(strings.Builder)
	b := new(bytes.Buffer)
	paths := make([]string, routesLength)
	pathStart := '/'
	for i := range routesLength {
		pathSegmentCharsLength := rand.N(maxCharLength-minCharLength) + minCharLength

		b.WriteRune(pathStart)
		b.WriteString(randString(pathSegmentCharsLength))
		b.WriteString("/{name:string}/") // sugar.
		b.WriteString(randString(pathSegmentCharsLength))
		b.WriteString("/{age:int}/end")
		paths[i] = b.String()

		b.Reset()
	}

	return paths
}

// Build 1296(=144*9(the available http methods)) routes
// with up to 2*range(15-42)+ 2 named paths lowercase letters
// and 12 request handlers each.
//
// GOCACHE=off && go test -run=XXX -bench=BenchmarkRouterBuilder$ -benchtime=10s
func BenchmarkRouterBuilder(b *testing.B) {
	noOpHandler := func(ctx *context.Context) {}
	handlersPerRoute := make(context.Handlers, 12)
	for i := 0; i < cap(handlersPerRoute); i++ {
		handlersPerRoute[i] = noOpHandler
	}

	routesLength := 144
	// i.e /gzhyweumidvelqewrvoyqmzopvuxli/{name:string}/bibrkratnrrhvsjwsxygfwmqwhcstc/{age:int}/end
	paths := genPaths(routesLength, 15, 42)

	api := New(nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := range routesLength {
		api.Any(paths[i], handlersPerRoute...)
	}

	if err := api.BuildRouter(RouterOptions{}, golog.Default, nil); err != nil {
		b.Fatal(err)
	}

	b.StopTimer()

	b.Logf("%d routes have just been built\n", len(api.Routes().List()))
}
