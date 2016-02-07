package gapi

import (

)
var (
	SUPPORTED_TYPES = [2]string{"string","int"}
)
//for now, supported: string,int
func toPattern(theType string) string {
	var thePattern string
	
	switch theType {
		case "string":
			thePattern = "\\w+"
		
		case "int":
			thePattern = "[0-9]+"
		default :
		
	}
	
	return thePattern
}

func isSupportedType(theType string) bool {
	for _,supportedType:= range SUPPORTED_TYPES {
		if theType == supportedType {
			return true
		}
	}
	return false
}