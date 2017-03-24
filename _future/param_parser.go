package router

import (
	"fmt"
	"strconv"
	"strings"
)

type ParamTmpl struct {
	// id
	Name string
	// int range(1,5)!fail # fail fails on int parse or on range, it will work reclusive
	Expression string
	// fail
	FailStatusCode int

	Macro MacroTmpl
}

type MacroTmpl struct {
	// int
	Name string
	// Macro will allow more than one funcs.
	// []*MacroFuncs{ {Name: range, Params: []string{1,5}}}
	Funcs []MacroFuncTmpl
}

type MacroFuncTmpl struct {
	// range
	Name string
	// 1,5
	Params []string
}

const (
	ParamNameSeperator = ':'
	FuncSeperator      = ' '
	FuncStart          = '('
	FuncEnd            = ')'
	FuncParamSeperator = ','
	FailSeparator      = '!'
)

const DefaultFailStatusCode = 404

type CursorState int

func (cs *CursorState) Is(s CursorState) bool {
	c := *cs
	if c == s {
		switch s {
		case CursorStateNone:
			*cs = CursorStateStarted
		case CursorStateStarted:
			*cs = CursorStatePending
		case CursorStatePending:
			*cs = CursorStateRecording
		case CursorStateRecording:
			*cs = CursorStateStarted
		}
		return true
	}
	return false
}

// we use that to set a value of each "state".
// because macro's function's arguments can accepts space and regex with any character
const (
	// no : is found yet
	CursorStateNone CursorState = iota
	// macro name and first expression guess parsed
	// and we' ready to walk forward
	CursorStateStarted
	// after space, waiting for a macro func begin or !+fail_status_code
	CursorStatePending
	// inside the macro func, after the opening parenthesis
	CursorStateRecording
)

// Parse i.e:
// id:int range(1,5) otherFunc(3) !404
//
// id        = param name | can front-end end here but the parser should add :any
// int       = marco | can end here
//
// range     = marco's funcs(range)
// 1,5       = range's  func.params | can end here
//           +
// otherFunc = marco's funcs(otherFunc)
// 3         = otherFunc's func.params  | can end here
//
// 404      = fail http custom error status code -> handler , will fail rescuslive
func ParseParam(source string) (*ParamTmpl, error) {
	// first, do a simple check if that's 'valid'
	sepIndex := strings.IndexRune(source, ParamNameSeperator)

	if sepIndex <= 0 {
		// if not found or
		// if starts with :
		return nil, fmt.Errorf("invalid source '%s', separator should be after the parameter name", source)
	}

	t := new(ParamTmpl)
	// id:int range(1,5)
	// id:int min(1) max(5)
	// id:int range(1,5)!404 or !404, space doesn't matters on fail error code.
	// cursor := 0
	state := CursorStateNone
	cursor := 0
	for i := 0; i < len(source); i++ {
		// TODO: find a better way instead of introducing variables like waitForFunc, insideFunc,
		// one way is to move the functions with the reverse order but this can fix the problem for now
		// later it will introduce new bugs, we can find a better static way to check these things, tomorrow.
		// :int ...
		if source[i] == ParamNameSeperator && state.Is(CursorStateNone) {
			if i+1 >= len(source) {
				return nil, fmt.Errorf("missing marco or raw expression after seperator, on source '%s'", source)
			}

			// id: , take the left, skip the : and continue
			t.Name = source[0:i]
			// set the expression, after the i, i.e:
			// int range(1,5)
			t.Expression = source[i+1:]
			// set the macro's name to the full expression
			// because we don't know if the user has put functions
			// and we follow the < left 'pattern'
			//  (I don't know if that's valid but that is what
			//  I think to do and is working).
			t.Macro = MacroTmpl{Name: t.Expression}

			cursor = i + 1
			continue
		}

		if source[i] == FuncSeperator && state.Is(CursorStateStarted) {
			// take the left part: int if it's the first
			// space after the param name
			if t.Macro.Name == t.Expression {
				t.Macro.Name = source[cursor:i]
			} // else we have one or more functions, skip.
			cursor = i + 1
			continue
		}

		// if not inside a func body
		//         the cursor is a point which can receive a func
		//         starts with (
		if source[i] == FuncStart && state.Is(CursorStatePending) {
			// take the left part: range
			funcName := source[cursor:i]
			t.Macro.Funcs = append(t.Macro.Funcs, MacroFuncTmpl{Name: funcName})
			cursor = i + 1
			continue
		}

		// 1,5)
		// we are inside func and )
		if source[i] == FuncEnd && state.Is(CursorStateRecording) {
			// check if we have end parenthesis but not start
			if len(t.Macro.Funcs) == 0 {
				return nil, fmt.Errorf("missing start macro's '%s' function, on source '%s'", t.Macro.Name, source)
			}

			// take the left part, between Start and End: 1,5
			funcParamsStr := source[cursor:i]

			funcParams := strings.SplitN(funcParamsStr, string(FuncParamSeperator), -1)
			t.Macro.Funcs[len(t.Macro.Funcs)-1].Params = funcParams
			cursor = i + 1
			continue
		}

		if source[i] == FailSeparator && (state.Is(CursorStateStarted) || state.Is(CursorStatePending)) {
			// it should be the last element
			// so no problem if we set the cursor here and work with that
			// we will not need that later.
			cursor += 1
			if i+1 >= len(source) {
				return nil, fmt.Errorf("missing fail status code after '%q', on source '%s'", FailSeparator, source)
			}

			failCodeStr := source[cursor:] // should be the last
			failCode, err := strconv.Atoi(failCodeStr)
			if err != nil {
				return nil, fmt.Errorf("fail status code should be integer but got '%s', on source '%s'", failCodeStr, source)
			}

			t.FailStatusCode = failCode

			break
		}

	}

	if t.FailStatusCode == 0 {
		t.FailStatusCode = DefaultFailStatusCode
	}

	return t, nil
}
