package iris

import (
	"math/rand"
	"reflect"
	"strings"
	"time"
	"unsafe"
)

//THESE ARE FROM Go Authors
var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	// "&#34;" is shorter than "&quot;".
	`"`, "&#34;",
	// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
	"'", "&#39;",
)

func htmlEscape(s string) string {
	return htmlReplacer.Replace(s)
}

func findLower(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// check if middleware passsed to a route has cors
// this is a poor implementation which only works with the iris/middleware/cors middleware
// it's bad and anti-pattern to check if a kind of  middleware has passed but I don't have any other options right now
// because I don't want to check in the router if method == req.Method || method == "OPTIONS" this will be low at least 900-2000 nanoseconds
// I made a func CorsMethodMatch, which is setted to the router.methodMatch if and only if the user passed the middleware cors on any of the routes
// only then the  second check of || method == "OPTIONS" will be evalutated. This is the way iris is working and have the best performance, maybe poor code I don't like to do but I Have to do.
// look at .Plant here, and on station.forceOptimusPrime
func hasCors(route IRoute) bool {
	for _, h := range route.GetMiddleware() {
		if _, ok := h.(interface {
			// Capitalize fix of isMethodAllowed by @thesyncim
			IsMethodAllowed(method string) bool
		}); ok {
			return true
		}
	}

	return false
}

// these are experimentals, will be used inside plugins to extend their power.

// BytesToString accepts bytes and returns their string presentation
// instead of string() this method doesn't generate memory allocations,
// BUT it is not safe to use anywhere because it points
// this helps on 0 memory allocations
func BytesToString(b []byte) string {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := reflect.StringHeader{bh.Data, bh.Len}
	return *(*string)(unsafe.Pointer(&sh))
}

// StringToBytes accepts string and returns their []byte presentation
// instead of byte() this method doesn't generate memory allocations,
// BUT it is not safe to use anywhere because it points
// this helps on 0 memory allocations
func StringToBytes(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{sh.Data, sh.Len, 0}
	return *(*[]byte)(unsafe.Pointer(&bh))
}

//
const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

// RandomString accepts a number(10 for example) and returns a random string using simple but fairly safe random algorithm
func RandomString(n int) string {
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

	return string(b)
}
