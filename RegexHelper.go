package iris

// THIS FILE HAS NOT ANY USE ANYMORE, I DROP THE SUPPORT FOR REGEX PATTERNS FOR ROUTE PATH
// ONLY USAGE IS FOR CONSTS, AND MAYBE SOME FUTURE WORK ON PRE_DEFINED TYPES OF PATH PARAMETERS.
import ()

const (
	// ParameterStart the character which the named path is starting
	ParameterStart     = ":"
	ParameterStartByte = byte(':')
	SlashByte          = byte('/')
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
	MatchEverything     = "*"
	MatchEverythingByte = byte('*')
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
		thePattern = "[a-zA-Z]+" //"\\w+" //xmmm or  [a-zA-Z0-9]+

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
