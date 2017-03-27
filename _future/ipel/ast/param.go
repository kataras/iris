package ast

type ParamType uint8

const (
	ParamTypeUnExpected ParamType = iota
	// /myparam1
	ParamTypeString
	// /42
	ParamTypeInt
	// /myparam
	ParamTypeAlphabetical
	// /main.css
	ParamTypeFile
	// /myparam1/myparam2
	ParamTypePath
)

var paramTypes = map[string]ParamType{
	"string":       ParamTypeString,
	"int":          ParamTypeInt,
	"alphabetical": ParamTypeAlphabetical,
	"file":         ParamTypeFile,
	"path":         ParamTypePath,
	// could be named also:
	// "tail":
	// "wild"
	// "wildcard"

}

func LookupParamType(ident string) ParamType {
	if typ, ok := paramTypes[ident]; ok {
		return typ
	}
	return ParamTypeUnExpected
}

type ParamStatement struct {
	Name      string      // id
	Type      ParamType   // int
	Funcs     []ParamFunc // range
	ErrorCode int         // 404
}
