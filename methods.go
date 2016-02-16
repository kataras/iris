package iris

/* as ENUM, maybe at the Future
type HttpMethodType struct {
	GET    string
	POST   string
	PUT    string
	DELETE string
}

func (c *HttpMethodType) GetName(i int) string {
	return HttpMethodReflectType.Field(i).Name
}

var HttpMethods = HttpMethodType{"GET", "POST", "PUT", "DELETE"}
var HttpMethodReflectType = reflect.TypeOf(HttpMethods)
*/

var HTTPMethods = struct {
	GET, POST, PUT, DELETE, CONNECT, HEAD, PATCH, OPTIONS, TRACE string
	ALL, ANY                                                     []string //ALL and ANY are exctactly the same I keep both keys, no problem no big array :P
}{"GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE",
	[]string{"GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE"},
	[]string{"GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE"}}
