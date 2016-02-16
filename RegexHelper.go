package gapi

import (
	"regexp"
	"strings"
)

const (
	PARAMETER_START               = ":"
	PARAMETER_PATTERN_START       = "("
	PARAMETER_PATTERN_END         = ")"
	REGEX_PARENTHESIS_AND_CONTENT = "\\([^)]*\\)"                   //"\\([^\\)]*\\)"
	REGEX_BRACKETS_CONTENT        = "{(.*?)}"                       //not used any more, for now. //{(.*?)} -> is /g (global) on pattern.findAllString
	REGEX_ROUTE_NAMED_PARAMETER   = "(" + PARAMETER_START + "\\w+)" //finds words starts with : . It used as /g (global) on pattern.findAllString
	REGEX_PARENTHESIS_CONTENT     = PARAMETER_PATTERN_START + "(.*?)" + PARAMETER_PATTERN_END
	MATCH_EVERYTHING              = "*"
)

var (
	SUPPORTED_TYPES = [2]string{"string", "int"}
)

//for now, supported: string,int
func toPattern(theType string) string {
	var thePattern string

	switch theType {
	case "string":
		thePattern = "\\w+" //xmmm or  [a-zA-Z0-9]+

	case "int":
		thePattern = "[0-9]+"
	default:

	}

	return thePattern
}

func isSupportedType(theType string) bool {
	for _, supportedType := range SUPPORTED_TYPES {
		if theType == supportedType {
			return true
		}
	}
	return false
}

///TODO: na xrisimopoisw to regex pou vriskei auta p einai stin paren9esi kai ta svinei
//meta to keys := wste ta keys na min exoun mesa ta regex
//giati me to regex p exw twra an o developer valei :name(antregexhere:/w+) 9a bei sta keys[] kai to /w+
//FIXED
//finds and stores pattern for /something/:name(string)
func makePathPattern(Route *Route) {
	registedPath := Route.path
	if registedPath != MATCH_EVERYTHING {
		regexpRoute := registedPath

		routeWithoutParenthesis := regexp.MustCompile(REGEX_PARENTHESIS_AND_CONTENT).ReplaceAllString(registedPath, "")

		pattern := regexp.MustCompile(REGEX_ROUTE_NAMED_PARAMETER)

		//find only :keys without parenthesis if any
		keys := pattern.FindAllString(routeWithoutParenthesis, -1)

		for keyIndex, key := range keys {
			backupKey := key // the full :name we will need it for the replace.
			key = key[1:len(key)]
			keys[keyIndex] = key // :name is name now.

			a1 := strings.Index(registedPath, key) + len(key)

			if len(registedPath) > a1 && string(registedPath[a1]) == PARAMETER_PATTERN_START {
				//check if first character, of the ending of the key from the original full registedPath, is parenthesis

				keyPattern1 := registedPath[a1:]                                  //here we take all string after a1, which maybe be follow up with other paths, we will substring it in the next line
				lastParIndex := strings.Index(keyPattern1, PARAMETER_PATTERN_END) //find last parenthesis index of the keyPattern1
				keyPattern1 = keyPattern1[0 : lastParIndex+1]                     // find contents and it's parenthesis, we will need it for replace
				keyPatternReg := keyPattern1[1 : len(keyPattern1)-1]              //find the contents between parenthesis
				if isSupportedType(keyPatternReg) {
					//if it is (string) or (int) inside contents
					keyPatternReg = toPattern(keyPatternReg)
				}
				//replace the whole :key+(pattern) with just  the converted pattern from int,string or the contents of the parenthesis which is a custom user's regex pattern
				regexpRoute = strings.Replace(regexpRoute, backupKey+keyPattern1, keyPatternReg, -1)

			} else {
				regexpRoute = strings.Replace(regexpRoute, backupKey, "\\w+", -1)
			}

		}
		regexpRoute = strings.Replace(regexpRoute, "/", "\\/", -1) + "$" ///escape / character for regex and finish it with $, if route/:name and req url is route/:name:/somethingelse then it will not be matched
		routePattern := regexp.MustCompile(regexpRoute)
		Route.Pattern = routePattern

		Route.ParamKeys = keys
	}
}

//finds and stores the pattern for /something/{name(string)}
func makePathPatternOld(Route *Route) {
	registedPath := Route.path
	if registedPath != MATCH_EVERYTHING {
		regexpRoute := registedPath
		pattern := regexp.MustCompile(REGEX_BRACKETS_CONTENT) //fint all {key}
		keys := pattern.FindAllString(registedPath, -1)
		println("keys: ", strings.Join(keys, ","))
		for indexKey, key := range keys {
			backupKey := key // the full {name(regex)} we will need it for the replace.
			key = key[1 : len(key)-1]
			println(key)
			keys[indexKey] = key
			startParenthesisIndex := strings.Index(key, "(")
			finishParenthesisIndex := strings.LastIndex(key, ")") // checks only the first (), if more than one (regex) exists for one key then the application will be fail and I dont care :)
			//I did LastIndex because the custom regex maybe has ()parenthesis too.
			if startParenthesisIndex > 0 && finishParenthesisIndex > startParenthesisIndex {
				keyPattern := key[startParenthesisIndex+1 : finishParenthesisIndex]
				key = key[0:startParenthesisIndex] //remove the (regex) from key and  the {, }

				keys[indexKey] = key
				if isSupportedType(keyPattern) {
					//if it is (string) or (int) inside contents
					keyPattern = toPattern(keyPattern)
				}
				regexpRoute = strings.Replace(registedPath, backupKey, keyPattern, -1)
				//println("regex found for "+key)
			} else {

				//if no regex found in this key then add the w+
				regexpRoute = strings.Replace(regexpRoute, backupKey, "\\w+", -1)

			}
		}

		//regexpRoute = pattern.ReplaceAllString(registedPath, "\\w+") + "$" //replace that {key} with /w+ and on the finish $
		regexpRoute = strings.Replace(regexpRoute, "/", "\\/", -1) + "$" ///escape / character for regex and finish it with $, if route/{name} and req url is route/{name}/somethingelse then it will not be matched
		routePattern := regexp.MustCompile(regexpRoute)
		Route.Pattern = routePattern

		Route.ParamKeys = keys
	}
}
