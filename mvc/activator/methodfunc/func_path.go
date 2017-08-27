package methodfunc

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

const (
	by        = "By"
	wildcard  = "Wildcard"
	paramName = "param"
)

type pathInfo struct {
	GoParamType string
	ParamType   string
	RelPath     string
}

const (
	paramTypeInt    = "int"
	paramTypeLong   = "long"
	paramTypeString = "string"
	paramTypePath   = "path"
)

var macroTypes = map[string]string{
	"int":    paramTypeInt,
	"int64":  paramTypeLong,
	"string": paramTypeString,
	// there is "path" param type but it's being captured "on-air"
	// "file" param type is not supported by the current implementation, yet
	// but if someone ask for it I'll implement it, it's easy.
}

func resolveRelativePath(info FuncInfo) (p pathInfo, ok bool) {
	if info.Trailing == "" {
		// it's valid
		// it's just don't have a relative path,
		// therefore p.RelPath will be empty, as we want.
		return p, true
	}

	var (
		typ     = info.Type
		tr      = info.Trailing
		relPath = resolvePathFromFunc(tr)

		goType, paramType string
	)

	byKeywordIdx := strings.LastIndex(tr, by)
	if byKeywordIdx != -1 && typ.NumIn() == 2 { // first is the struct receiver.
		funcPath := tr[0:byKeywordIdx] // remove the "By"
		goType = typ.In(1).Name()
		afterBy := byKeywordIdx + len(by)
		if len(tr) > afterBy {
			if tr[afterBy:] == wildcard {
				paramType = paramTypePath
			} else {
				// invalid syntax
				return p, false
			}
		} else {
			// it's not wildcard, so check base on our available macro types.
			if paramType, ok = macroTypes[goType]; !ok {
				// ivalid type
				return p, false
			}
		}

		// int and string are supported.
		// as there is no way to get the parameter name
		// we will use the "param" everywhere.
		suffix := fmt.Sprintf("/{%s:%s}", paramName, paramType)
		relPath = resolvePathFromFunc(funcPath) + suffix
	}

	// if GetSomething/PostSomething/PutSomething...
	// we will not check for "Something" because we could
	// occur unexpected behaviors to the existing users
	// who using exported functions for controller's internal
	// functionalities and not for serving a request path.

	return pathInfo{
		GoParamType: goType,
		ParamType:   paramType,
		RelPath:     relPath,
	}, true
}

func resolvePathFromFunc(funcName string) string {
	end := len(funcName)
	start := -1
	buf := &bytes.Buffer{}

	for i, n := 0, end; i < n; i++ {
		c := rune(funcName[i])
		if unicode.IsUpper(c) {
			// it doesn't count the last uppercase
			if start != -1 {
				end = i
				s := "/" + strings.ToLower(funcName[start:end])
				buf.WriteString(s)
			}
			start = i
			continue
		}
		end = i + 1
	}

	if end > 0 && len(funcName) >= end {
		buf.WriteString("/" + strings.ToLower(funcName[start:end]))
	}

	return buf.String()
}
