package iris

import (
	"net/http"
	"net/url"
	"strings"
)

// Parameters is just a type of pair (map[string]string) which contains the request's path parameters only
type Parameters map[string]string

// Get gets a value from a key inside this Parameters map
func (params Parameters) Get(key string) string {
	return params[key]
}

// Params returns all path named parameters (if any) from a request
func Params(req *http.Request) Parameters {
	_cookie, _err := req.Cookie(CookieName)
	if _err != nil {
		return nil
	}
	value := _cookie.Value

	params := make(Parameters)

	paramsStr := strings.Split(value, ",")

	for _, _fullVarStr := range paramsStr {
		vars := strings.Split(_fullVarStr, "=")
		if len(vars) != 2 { //check if key=val=somethingelse here ,is wrong, only key=value allowed, then just ignore this
			continue
		}
		params[vars[0]] = vars[1]
	}

	return params
}

// Param receives a request and a key and returns the value of the path named parameter inside request
func Param(req *http.Request, key string) string {
	params := Params(req)
	param := ""
	if params != nil {
		param = params[key]
	}
	return param
}

// URLParams the URL.Query() is a complete function which returns the url get parameters from the url query, We don't have to do anything else here.
func URLParams(req *http.Request) url.Values {
	return req.URL.Query()
}

// URLParam returns the get parameter from a request , if any
func URLParam(req *http.Request, key string) string {
	return req.URL.Query().Get(key)
}
