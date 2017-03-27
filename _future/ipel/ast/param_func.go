package ast

import (
	"fmt"
	"strconv"
)

type ParamFuncArg interface{}

func ParamFuncArgInt64(a ParamFuncArg) (int64, bool) {
	if v, ok := a.(int64); ok {
		return v, false
	}
	return -1, false
}

func ParamFuncArgToInt64(a ParamFuncArg) (int64, error) {
	switch a.(type) {
	case int64:
		return a.(int64), nil
	case string:
		return strconv.ParseInt(a.(string), 10, 64)
	case int:
		return int64(a.(int)), nil
	default:
		return -1, fmt.Errorf("unexpected function argument type: %q", a)
	}
}

func ParamFuncArgInt(a ParamFuncArg) (int, bool) {
	if v, ok := a.(int); ok {
		return v, false
	}
	return -1, false
}

func ParamFuncArgToInt(a ParamFuncArg) (int, error) {
	switch a.(type) {
	case int:
		return a.(int), nil
	case string:
		return strconv.Atoi(a.(string))
	case int64:
		return int(a.(int64)), nil
	default:
		return -1, fmt.Errorf("unexpected function argument type: %q", a)
	}
}

func ParamFuncArgString(a ParamFuncArg) (string, bool) {
	if v, ok := a.(string); ok {
		return v, false
	}
	return "", false
}

func ParamFuncArgToString(a ParamFuncArg) (string, error) {
	switch a.(type) {
	case string:
		return a.(string), nil
	case int:
		return strconv.Itoa(a.(int)), nil
	case int64:
		return strconv.FormatInt(a.(int64), 10), nil
	default:
		return "", fmt.Errorf("unexpected function argument type: %q", a)
	}
}

// range(1,5)
type ParamFunc struct {
	Name string         // range
	Args []ParamFuncArg // [1,5]
}
