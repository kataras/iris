package router

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris/v12/context"

	"github.com/kataras/golog"
)

// randStringBytesMaskImprSrc helps us to generate random paths for the test,
// the below piece of code is external, as an answer to a stackoverflow question.
//
// START.
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func randStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return strings.ToLower(string(b))
}

// END.

func genPaths(routesLength, minCharLength, maxCharLength int) []string {
	// b := new(strings.Builder)
	b := new(bytes.Buffer)
	paths := make([]string, routesLength)
	pathStart := '/'
	for i := 0; i < routesLength; i++ {
		pathSegmentCharsLength := rand.Intn(maxCharLength-minCharLength) + minCharLength

		b.WriteRune(pathStart)
		b.WriteString(randStringBytesMaskImprSrc(pathSegmentCharsLength))
		b.WriteString("/{name:string}/") // sugar.
		b.WriteString(randStringBytesMaskImprSrc(pathSegmentCharsLength))
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
// GOCACHE=off && go test -run=XXX -bench=BenchmarkAPIBuilder$ -benchtime=10s
func BenchmarkAPIBuilder(b *testing.B) {
	rand.New(rand.NewSource(time.Now().Unix()))

	noOpHandler := func(ctx *context.Context) {}
	handlersPerRoute := make(context.Handlers, 12)
	for i := 0; i < cap(handlersPerRoute); i++ {
		handlersPerRoute[i] = noOpHandler
	}

	routesLength := 144
	// i.e /gzhyweumidvelqewrvoyqmzopvuxli/{name:string}/bibrkratnrrhvsjwsxygfwmqwhcstc/{age:int}/end
	paths := genPaths(routesLength, 15, 42)

	api := NewAPIBuilder(golog.Default)
	requestHandler := NewDefaultHandler(nil, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < routesLength; i++ {
		api.Any(paths[i], handlersPerRoute...)
	}

	if err := requestHandler.Build(api); err != nil {
		b.Fatal(err)
	}

	b.StopTimer()

	b.Logf("%d routes have just builded\n", len(api.GetRoutes()))
}
