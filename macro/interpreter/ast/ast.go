package ast

type (
	// ParamType holds the necessary information about a parameter type for the parser to lookup for.
	ParamType interface {
		// The name of the parameter type.
		// Indent should contain the characters for the parser.
		Indent() string
	}

	// MasterParamType if implemented and its `Master()` returns true then empty type param will be translated to this param type.
	// Also its functions will be available to the rest of the macro param type's funcs.
	//
	// Only one Master is allowed.
	MasterParamType interface {
		ParamType
		Master() bool
	}

	// TrailingParamType if implemented and its `Trailing()` returns true
	// then it should be declared at the end of a route path and can accept any trailing path segment as one parameter.
	TrailingParamType interface {
		ParamType
		Trailing() bool
	}

	// AliasParamType if implemeneted nad its `Alias()` returns a non-empty string
	// then the param type can be written with that string literal too.
	AliasParamType interface {
		ParamType
		Alias() string
	}
)

// IsMaster returns true if the "pt" param type is a master one.
func IsMaster(pt ParamType) bool {
	p, ok := pt.(MasterParamType)
	return ok && p.Master()
}

// IsTrailing returns true if the "pt" param type is a marked as trailing,
// which should accept more than one path segment when in the end.
func IsTrailing(pt ParamType) bool {
	p, ok := pt.(TrailingParamType)
	return ok && p.Trailing()
}

// HasAlias returns any alias of the "pt" param type.
// If alias is empty or not found then it returns false as its second output argument.
func HasAlias(pt ParamType) (string, bool) {
	if p, ok := pt.(AliasParamType); ok {
		alias := p.Alias()
		return alias, len(alias) > 0
	}

	return "", false
}

// GetMasterParamType accepts a list of ParamType and returns its master.
// If no `Master` specified:
// and len(paramTypes) > 0 then it will return the first one,
// otherwise it returns nil.
func GetMasterParamType(paramTypes ...ParamType) ParamType {
	for _, pt := range paramTypes {
		if IsMaster(pt) {
			return pt
		}
	}

	if len(paramTypes) > 0 {
		return paramTypes[0]
	}

	return nil
}

// LookupParamType accepts the string
// representation of a parameter type.
// Example:
// "string"
// "number" or "int"
// "long" or "int64"
// "uint8"
// "uint64"
// "boolean" or "bool"
// "alphabetical"
// "file"
// "path"
func LookupParamType(indentOrAlias string, paramTypes ...ParamType) (ParamType, bool) {
	for _, pt := range paramTypes {
		if pt.Indent() == indentOrAlias {
			return pt, true
		}

		if alias, has := HasAlias(pt); has {
			if alias == indentOrAlias {
				return pt, true
			}
		}
	}

	return nil, false
}

// ParamStatement is a struct
// which holds all the necessary information about a macro parameter.
// It holds its type (string, int, alphabetical, file, path),
// its source ({param:type}),
// its name ("param"),
// its attached functions by the user (min, max...)
// and the http error code if that parameter
// failed to be evaluated.
type ParamStatement struct {
	Src       string      // the original unparsed source, i.e: {id:int range(1,5) else 404}
	Name      string      // id
	Type      ParamType   // int
	Funcs     []ParamFunc // range
	ErrorCode int         // 404
}

// ParamFunc holds the name of a parameter's function
// and its arguments (values)
// A param func is declared with:
// {param:int range(1,5)},
// the range is the
// param function name
// the 1 and 5 are the two param function arguments
// range(1,5)
type ParamFunc struct {
	Name string   // range
	Args []string // ["1","5"]
}
