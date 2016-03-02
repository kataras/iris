package iris

import (
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

// URLParams the URL.Query() is a complete function which returns the url get parameters from the url query, We don't have to do anything else here.
func URLParams(req *http.Request) url.Values {
	return req.URL.Query()
}

// URLParam returns the get parameter from a request , if any
func URLParam(req *http.Request, key string) string {
	return req.URL.Query().Get(key)
}

func Params(r *Route, urlPath string) (params PathParameters) {
	//params := make(PathParameters, 0, partsLen)
	for i := 0; i < len(r.parts); i++ {

		if r.parts[i][0] == ParameterStartByte { //strings.IndexByte(r.parts[i], ParameterStartByte) == 0 { //r.parts[i][0] == ParameterStartByte { //strings.Index(r.parts[i], ParameterStart) == 0 { //r.parts[i][0:1] == ParameterStart { //takes the first character and check if it's parameter part
			//paramKey := r.parts[i][1:]
			//paramValue := reqParts[i]
			indexOfVal := -1
			if i == 0 {
				indexOfVal = strings.IndexByte(r.fullpath, r.parts[i][0]) //or just IndexByte(...,r.parts[i][0]
			} else {
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
				params = append(params, p)
				//println("new params len: ", len(params))
			} else {
				//println(len(urlPath), " is slower than indexOfVal: ", indexOfVal)
			}

			//user/profile/:username/test
			//user/profile/kataras/test
			// :username -> kataras theloume to kataras
			// /test -> /test
			//params = append(params, PathParameter{r.parts[i][1:], reqParts[i]})
			//params[i] = PathParameter{r.parts[i][1:], reqParts[i]} //no perfomance difference so keep it with append
		} else {
			//println(r.parts[i] + " doesnt start with : ")
		}
	}
	return
}
