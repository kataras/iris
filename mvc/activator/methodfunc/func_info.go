package methodfunc

import (
	"reflect"
	"strings"
	"unicode"
)

var availableMethods = [...]string{
	"ANY",  // will be registered using the `core/router#APIBuilder#Any`
	"ALL",  // same as ANY
	"NONE", // offline route
	// valid http methods
	"GET",
	"POST",
	"PUT",
	"DELETE",
	"CONNECT",
	"HEAD",
	"PATCH",
	"OPTIONS",
	"TRACE",
}

// FuncInfo is part of the `TController`,
// it contains the index for a specific http method,
// taken from user's controller struct.
type FuncInfo struct {
	// Name is the map function name.
	Name string
	// Trailing is not empty when the Name contains
	// characters after the titled method, i.e
	// if Name = Get -> empty
	// if Name = GetLogin -> Login
	// if Name = GetUserPost -> UserPost
	Trailing string

	// The Type of the method, includes the receivers.
	Type reflect.Type

	// Index is the index of this function inside the controller type.
	Index int
	// HTTPMethod is the original http method that this
	// function should be registered to and serve.
	// i.e "GET","POST","PUT"...
	HTTPMethod string
}

// or resolve methods
func fetchInfos(typ reflect.Type) (methods []FuncInfo) {
	// search the entire controller
	// for any compatible method function
	// and add that.
	for i, n := 0, typ.NumMethod(); i < n; i++ {
		m := typ.Method(i)
		name := m.Name

		for _, method := range availableMethods {
			possibleMethodFuncName := methodTitle(method)

			if strings.Index(name, possibleMethodFuncName) == 0 {
				trailing := ""
				// if has chars after the method itself
				if lname, lmethod := len(name), len(possibleMethodFuncName); lname > lmethod {
					ch := rune(name[lmethod])
					// if the next char is upper, otherise just skip the whole func info.
					if unicode.IsUpper(ch) {
						trailing = name[lmethod:]
					} else {
						continue
					}
				}

				methodInfo := FuncInfo{
					Name:       name,
					Trailing:   trailing,
					Type:       m.Type,
					HTTPMethod: method,
					Index:      m.Index,
				}
				methods = append(methods, methodInfo)
			}
		}
	}
	return
}

func methodTitle(httpMethod string) string {
	httpMethodFuncName := strings.Title(strings.ToLower(httpMethod))
	return httpMethodFuncName
}
