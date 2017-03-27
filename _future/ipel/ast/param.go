package ast

type ParamType uint8

const (
	ParamTypeUnExpected ParamType = iota
	// /42
	ParamTypeInt
	// /myparam1
	ParamTypeString
	// /myparam
	ParamTypeAlphabetical
	// /myparam1/myparam2
	ParamPath
)

var paramTypes = map[string]ParamType{
	"int":          ParamTypeInt,
	"string":       ParamTypeString,
	"alphabetical": ParamTypeAlphabetical,
	"path":         ParamPath,
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
