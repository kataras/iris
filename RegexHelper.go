package iris

import (
	"regexp"
	"strings"
)

const (
	// ParameterStart the character which the named path is starting
	ParameterStart = ":"
	// ParameterPatternStart the character which the regex custom pattern of a named parameter starts
	ParameterPatternStart = "("
	// ParameterPatternEnd the character which the regex custom pattern of a named parameter ends
	ParameterPatternEnd = ")"
	// RegexParenthesisAndContent a str regex which returns all []contents inside parenthesis
	RegexParenthesisAndContent = "\\([^)]*\\)" //"\\([^\\)]*\\)"
	// RegexBracketsContent a str regex which returns all []contents inside brackets
	RegexBracketsContent = "{(.*?)}" //not used any more, for now. //{(.*?)} -> is /g (global) on pattern.findAllString
	// RegexRouteNamedParameter finds the whole word of the regex str pattern inside a named parameter
	RegexRouteNamedParameter = "(" + ParameterStart + "\\w+)" //finds words starts with : . It used as /g (global) on pattern.findAllString
	// RegexParenthesisContent finds whatever inside parenthesis
	RegexParenthesisContent = ParameterPatternStart + "(.*?)" + ParameterPatternEnd
	// MatchEverything is used for routes, is the character which match everything to a specific handler
	MatchEverything = "*"
)

var (
	// supportedRegexTypes the available types of string which can be converted to a regex str pattern
	supportedRegexTypes = [2]string{"string", "int"}
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
	for _, supportedType := range supportedRegexTypes {
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
	if registedPath != MatchEverything {
		regexpRoute := registedPath

		routeWithoutParenthesis := regexp.MustCompile(RegexParenthesisAndContent).ReplaceAllString(registedPath, "")

		pattern := regexp.MustCompile(RegexRouteNamedParameter)

		//find only :keys without parenthesis if any
		keys := pattern.FindAllString(routeWithoutParenthesis, -1)

		for keyIndex, key := range keys {
			backupKey := key // the full :name we will need it for the replace.
			key = key[1:len(key)]
			keys[keyIndex] = key // :name is name now.

			a1 := strings.Index(registedPath, key) + len(key)

			if len(registedPath) > a1 && string(registedPath[a1]) == ParameterPatternStart {
				//check if first character, of the ending of the key from the original full registedPath, is parenthesis

				keyPattern1 := registedPath[a1:]                                //here we take all string after a1, which maybe be follow up with other paths, we will substring it in the next line
				lastParIndex := strings.Index(keyPattern1, ParameterPatternEnd) //find last parenthesis index of the keyPattern1
				keyPattern1 = keyPattern1[0 : lastParIndex+1]                   // find contents and it's parenthesis, we will need it for replace
				keyPatternReg := keyPattern1[1 : len(keyPattern1)-1]            //find the contents between parenthesis
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
