package iris

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
)

type PathParameter struct {
	Key   string
	Value string
}

// PathParameters type for path parameters
// Tt's a slice of PathParameter type, because it's faster than map
type PathParameters []PathParameter

// Get returns a value from a key inside this Parameters
// If no parameter with this key given then it returns an empty string
func (params PathParameters) Get(key string) string {
	for _, p := range params {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

func (params PathParameters) Set(key string, value string) {
	params = append(params, PathParameter{key, value})

}

func (params PathParameters) String() string {
	var buff bytes.Buffer
	for i := 0; i < len(params); i++ {
		buff.WriteString(params[i].Key)
		buff.WriteString("=")
		buff.WriteString(params[i].Value)
		if i < len(params)-1 {
			buff.WriteString(",")
		}

	}
	return buff.String()
}

func ParseParams(str string) PathParameters {
	_paramsstr := strings.Split(str, ",")
	if len(_paramsstr) == 0 {
		return nil
	}

	params := make(PathParameters, 0) // PathParameters{}

	for i := 0; i < len(_paramsstr); i++ {
		idxOfEq := strings.IndexRune(_paramsstr[i], '=')
		if idxOfEq == -1 {
			//error
			return nil
		}

		key := _paramsstr[i][:idxOfEq]
		val := _paramsstr[i][idxOfEq:]
		params = append(params, PathParameter{key, val})

	}
	return params
}

// URLParams the URL.Query() is a complete function which returns the url get parameters from the url query, We don't have to do anything else here.
func URLParams(req *http.Request) url.Values {

	return req.URL.Query()
}

// URLParam returns the get parameter from a request , if any
func URLParam(req *http.Request, key string) string {
	return req.URL.Query().Get(key)
}

//I use these in order to make 0 allocs and 0 bytes use even with params, and it worked :)
var _theParams = make(PathParameters, 0)

func resetParams() {
	_theParams = append(_theParams[:0], _theParams[:0]...)
}
func TryGetParameters(r *Route, urlPath string) PathParameters {
	//check these because the developer pass Context to the handler without nessecary have params to the route
	if r.isStatic {
		return nil
	} else if len(urlPath) < len(r.pathPrefix) {
		return nil
	}
	var pathIndex = r.lastStaticPartIndex
	var part Part
	var endSlash int
	var reqPart string
	//var params PathParameters = _theParams
	//var paramsBuff bytes.Buffer
	var rest string
	reqPath := urlPath[len(r.pathPrefix):] //we start from there to make it faster
	rest = reqPath
	//pIndex := 0
	for pathIndex < r.partsLen {

		endSlash = 1
		for endSlash < len(rest) && rest[endSlash] != '/' {
			endSlash++
		}

		reqPart = rest[0:endSlash]

		part = r.pathParts[pathIndex]
		pathIndex++

		if pathIndex == 0 || pathIndex >= r.partsLen || len(rest) <= endSlash {
			rest = rest[endSlash:]
		} else {
			rest = rest[endSlash+1:]
		}

		if part.isParam {
			//if params == nil {
			//	params = make(PathParameters, r.paramsLength, r.paramsLength)
			//	}
			//save the parameters and continue

			_theParams = append(_theParams, PathParameter{part.Value, reqPart})
			//params[pIndex] = PathParameter{part.Value, reqPart}
			//pIndex++

		}

	}
	return _theParams
}

/*
func params(r *Route, urlPath string) PathParameters {
	//params := make(PathParameters, 0, partsLen)
	//if params == nil {
	//	params = make(PathParameters, 0, len(r.parts))
	//}
	//moved to handler.run on contexted handlers resetParams()
	for i := 0; i < len(r.parts); i++ {

		if r.parts[i][0] == ParameterStartByte { //ParameterStartByte  //strings.IndexByte(r.parts[i], ParameterStartByte) == 0 { //r.parts[i][0] == ParameterStartByte { //strings.Index(r.parts[i], ParameterStart) == 0 { //r.parts[i][0:1] == ParameterStart { //takes the first character and check if it's parameter part
			//paramKey := r.parts[i][1:]
			//paramValue := reqParts[i]
			indexOfVal := -1
			if i == 0 {
				indexOfVal = strings.IndexByte(r.fullpath, r.parts[i][0])
			} else {
				///TODO;
				indexOfVal = strings.Index(r.fullpath, r.parts[i]) - 2 // -slash -:

			}

			if len(urlPath) >= indexOfVal {

				val := urlPath[indexOfVal:]
				//println("val ? " + val)
				valIndexFinish := strings.IndexByte(val, SlashByte)
				if valIndexFinish != -1 {
					val = val[:valIndexFinish]
					//val = val[valIndexFinish+1:]
				}
				//println("2 val ? " + val)
				p := PathParameter{r.parts[i][1:], val}
				//println(i, " path param key: "+p.Key+" Value: "+p.Value)
				//println("parts len:", len(r.parts))
				_theParams = append(_theParams, p)
				//println("new params len: ", len(params))
			}

		}
	}
	return _theParams
}*/
