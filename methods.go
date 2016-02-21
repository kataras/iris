package iris

// HTTPMethods is just a representation of the available http methods, use to make API.
// I know they are already exist in the http package, ex: http.MethodConnect, maybe at the future I will remove them from here and keep only the ANY.
var HTTPMethods = struct {
	GET, POST, PUT, DELETE, CONNECT, HEAD, PATCH, OPTIONS, TRACE string
	ALL, ANY                                                     []string //ALL and ANY are exctactly the same I keep both keys, no problem no big array :P
}{"GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE",
	[]string{"GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE"},
	[]string{"GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE"}}
