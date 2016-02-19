package iris

// HTTPMethods is just a representation of the available http methods, use to make API.
var HTTPMethods = struct {
	GET, POST, PUT, DELETE, CONNECT, HEAD, PATCH, OPTIONS, TRACE string
	ALL, ANY                                                     []string //ALL and ANY are exctactly the same I keep both keys, no problem no big array :P
}{"GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE",
	[]string{"GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE"},
	[]string{"GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE"}}
