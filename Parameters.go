package iris

import (
	"net/http"
	"strings"
)

//Route's Parameters
type Parameters map[string]string

func (params Parameters) Get(key string) string {
	return params[key]
}

//Global to package
func Params(req *http.Request) Parameters {
	_cookie, _err := req.Cookie(COOKIE_NAME)
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

func Param(req *http.Request, key string) string {
	params := Params(req)
	param := ""
	if params != nil {
		param = params[key]
	}
	return param
}

//
