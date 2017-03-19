package router

import (
	"fmt"
)

type PathTmpl struct {
	Params         []PathParamTmpl
	SegmentsLength int
}

type PathParamTmpl struct {
	SegmentIndex int
	Param        ParamTmpl
}

const (
	PathSeparator = '/'
	// Out means that it doesn't being included in param.
	ParamStartOut = '{'
	ParamEndOut   = '}'
)

// /users/{id:int range(1,5)}/profile
// parses only the contents inside {}
// but it gives back the position so it will be '1'
func ParsePath(source string) (*PathTmpl, error) {
	t := new(PathTmpl)
	cursor := 0
	segmentIndex := -1

	// error if path is empty
	if len(source) < 1 {
		return nil, fmt.Errorf("source cannot be empty ")
	}
	// error if not starts with '/'
	if source[0] != PathSeparator {
		return nil, fmt.Errorf("source '%s' should start with a path separator(%q)", source, PathSeparator)
	}
	// if path ends with '/' remove the last '/'
	if source[len(source)-1] == PathSeparator {
		source = source[0 : len(source)-1]
	}

	for i := range source {
		if source[i] == PathSeparator {
			segmentIndex++
			t.SegmentsLength++
			continue
		}

		if source[i] == ParamStartOut {
			cursor = i + 1
			continue
		}

		if source[i] == ParamEndOut {
			// take the left part id:int range(1,5)
			paramSource := source[cursor:i]
			paramTmpl, err := ParseParam(paramSource)
			if err != nil {
				return nil, err
			}

			t.Params = append(t.Params, PathParamTmpl{SegmentIndex: segmentIndex, Param: *paramTmpl})

			cursor = i + 1
			continue
		}
	}

	return t, nil
}
