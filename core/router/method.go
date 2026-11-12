package router

const (
	// MethodNone is a Virtual method
	// to store the "offline" routes.
	MethodNone = "NONE"

	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodPatch   = "PATCH" // RFC 5789
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
	// MethodQuery is the RFC 10008 QUERY method: safe, idempotent,
	// carries a request body and its response is cacheable.
	// It becomes http.MethodQuery once go.mod requires Go 1.28.
	MethodQuery = "QUERY"
)

// AllMethods contains the valid HTTP Methods:
// "GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD",
// "PATCH", "OPTIONS", "TRACE", "QUERY".
var AllMethods = []string{
	MethodGet,
	MethodHead,
	MethodPatch,
	MethodPut,
	MethodPost,
	MethodDelete,
	MethodOptions,
	MethodConnect,
	MethodTrace,
	MethodQuery,
}

// RegisterMethods adds custom http methods to the "AllMethods" list.
// Use it on initialization of your program.
func RegisterMethods(newCustomHTTPVerbs ...string) {
	newMethods := append(AllMethods, newCustomHTTPVerbs...)
	AllMethods = RemoveDuplicates(newMethods)
}

// RemoveDuplicates removes the duplicated elements from a slice of T.
func RemoveDuplicates[T comparable](elements []T) []T {
	seen := make(map[T]struct{}, len(elements))
	result := make([]T, 0, len(elements))
	for _, val := range elements {
		if _, ok := seen[val]; ok {
			continue
		}
		seen[val] = struct{}{}
		result = append(result, val)
	}
	return result
}
