package macro

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type (
	// ParamEvaluator is the signature for param type evaluator.
	// It accepts the param's value as string and returns
	// the <T> value (which its type is used for the input argument of the parameter functions, if any)
	// and a true value for passed, otherwise nil and false should be returned.
	ParamEvaluator func(paramValue string) (interface{}, bool)
)

var goodEvaluatorFuncs = []reflect.Type{
	reflect.TypeOf(func(string) (interface{}, bool) { return nil, false }),
	reflect.TypeOf(ParamEvaluator(func(string) (interface{}, bool) { return nil, false })),
}

func goodParamFunc(typ reflect.Type) bool {
	if typ.Kind() == reflect.Func { // it should be a func which returns a func (see below check).
		if typ.NumOut() == 1 {
			typOut := typ.Out(0)
			if typOut.Kind() != reflect.Func {
				return false
			}

			if typOut.NumOut() == 2 { // if it's a type of EvaluatorFunc, used for param evaluator.
				for _, fType := range goodEvaluatorFuncs {
					if typOut == fType {
						return true
					}
				}
				return false
			}

			if typOut.NumIn() == 1 && typOut.NumOut() == 1 { // if it's a type of func(paramValue [int,string...]) bool, used for param funcs.
				return typOut.Out(0).Kind() == reflect.Bool
			}
		}
	}

	return false
}

// Regexp accepts a regexp "expr" expression
// and returns its MatchString.
// The regexp is compiled before return.
//
// Returns a not-nil error on regexp compile failure.
func Regexp(expr string) (func(string) bool, error) {
	if expr == "" {
		return nil, fmt.Errorf("empty regex expression")
	}

	// add the last $ if missing (and not wildcard(?))
	if i := expr[len(expr)-1]; i != '$' && i != '*' {
		expr += "$"
	}

	r, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}

	return r.MatchString, nil
}

// MustRegexp same as Regexp
// but it panics on the "expr" parse failure.
func MustRegexp(expr string) func(string) bool {
	r, err := Regexp(expr)
	if err != nil {
		panic(err)
	}
	return r
}

// goodParamFuncName reports whether the function name is a valid identifier.
func goodParamFuncName(name string) bool {
	if name == "" {
		return false
	}
	// valid names are only letters and _
	for _, r := range name {
		switch {
		case r == '_':
		case !unicode.IsLetter(r):
			return false
		}
	}
	return true
}

// the convertBuilderFunc return value is generating at boot time.
// convertFunc converts an interface to a valid full param function.
func convertBuilderFunc(fn interface{}) ParamFuncBuilder {

	typFn := reflect.TypeOf(fn)
	if !goodParamFunc(typFn) {
		return nil
	}

	numFields := typFn.NumIn()

	return func(args []string) reflect.Value {
		if len(args) != numFields {
			// no variadics support, for now.
			panic("args should be the same len as numFields")
		}
		var argValues []reflect.Value
		for i := 0; i < numFields; i++ {
			field := typFn.In(i)
			arg := args[i]

			// try to convert the string literal as we get it from the parser.
			var (
				val interface{}

				panicIfErr = func(err error) {
					if err != nil {
						panic(fmt.Sprintf("on field index: %d: %v", i, err))
					}
				}
			)

			// try to get the value based on the expected type.
			switch field.Kind() {
			case reflect.Int:
				v, err := strconv.Atoi(arg)
				panicIfErr(err)
				val = v
			case reflect.Int8:
				v, err := strconv.ParseInt(arg, 10, 8)
				panicIfErr(err)
				val = int8(v)
			case reflect.Int16:
				v, err := strconv.ParseInt(arg, 10, 16)
				panicIfErr(err)
				val = int16(v)
			case reflect.Int32:
				v, err := strconv.ParseInt(arg, 10, 32)
				panicIfErr(err)
				val = int32(v)
			case reflect.Int64:
				v, err := strconv.ParseInt(arg, 10, 64)
				panicIfErr(err)
				val = v
			case reflect.Uint:
				v, err := strconv.ParseUint(arg, 10, strconv.IntSize)
				panicIfErr(err)
				val = uint(v)
			case reflect.Uint8:
				v, err := strconv.ParseUint(arg, 10, 8)
				panicIfErr(err)
				val = uint8(v)
			case reflect.Uint16:
				v, err := strconv.ParseUint(arg, 10, 16)
				panicIfErr(err)
				val = uint16(v)
			case reflect.Uint32:
				v, err := strconv.ParseUint(arg, 10, 32)
				panicIfErr(err)
				val = uint32(v)
			case reflect.Uint64:
				v, err := strconv.ParseUint(arg, 10, 64)
				panicIfErr(err)
				val = v
			case reflect.Float32:
				v, err := strconv.ParseFloat(arg, 32)
				panicIfErr(err)
				val = float32(v)
			case reflect.Float64:
				v, err := strconv.ParseFloat(arg, 64)
				panicIfErr(err)
				val = v
			case reflect.Bool:
				v, err := strconv.ParseBool(arg)
				panicIfErr(err)
				val = v
			case reflect.Slice:
				if len(arg) > 1 {
					if arg[0] == '[' && arg[len(arg)-1] == ']' {
						// it is a single argument but as slice.
						val = strings.Split(arg[1:len(arg)-1], ",") // only string slices.
					}
				}

			default:
				val = arg
			}

			argValue := reflect.ValueOf(val)
			if expected, got := field.Kind(), argValue.Kind(); expected != got {
				panic(fmt.Sprintf("func's input arguments should have the same type: [%d] expected %s but got %s", i, expected, got))
			}

			argValues = append(argValues, argValue)
		}

		evalFn := reflect.ValueOf(fn).Call(argValues)[0]

		// var evaluator EvaluatorFunc
		// // check for typed and not typed
		// if _v, ok := evalFn.(EvaluatorFunc); ok {
		// 	evaluator = _v
		// } else if _v, ok = evalFn.(func(string) bool); ok {
		// 	evaluator = _v
		// }
		// return func(paramValue interface{}) bool {
		// 	return evaluator(paramValue)
		// }
		return evalFn
	}
}

type (
	// Macro represents the parsed macro,
	// which holds
	// the evaluator (param type's evaluator + param functions evaluators)
	// and its param functions.
	//
	// Any type contains its own macro
	// instance, so an String type
	// contains its type evaluator
	// which is the "Evaluator" field
	// and it can register param functions
	// to that macro which maps to a parameter type.
	Macro struct {
		indent   string
		alias    string
		master   bool
		trailing bool

		Evaluator ParamEvaluator
		funcs     []ParamFunc
	}

	// ParamFuncBuilder is a func
	// which accepts a param function's arguments (values)
	// and returns a function as value, its job
	// is to make the macros to be registered
	// by user at the most generic possible way.
	ParamFuncBuilder func([]string) reflect.Value // the func(<T>) bool

	// ParamFunc represents the parsed
	// parameter function, it holds
	// the parameter's name
	// and the function which will build
	// the evaluator func.
	ParamFunc struct {
		Name string
		Func ParamFuncBuilder
	}
)

// NewMacro creates and returns a Macro that can be used as a registry for
// a new customized parameter type and its functions.
func NewMacro(indent, alias string, master, trailing bool, evaluator ParamEvaluator) *Macro {
	return &Macro{
		indent:   indent,
		alias:    alias,
		master:   master,
		trailing: trailing,

		Evaluator: evaluator,
	}
}

// Indent returns the name of the parameter type.
func (m *Macro) Indent() string {
	return m.indent
}

// Alias returns the alias of the parameter type, if any.
func (m *Macro) Alias() string {
	return m.alias
}

// Master returns true if that macro's parameter type is the
// default one if not :type is followed by a parameter type inside the route path.
func (m *Macro) Master() bool {
	return m.master
}

// Trailing returns true if that macro's parameter type
// is wildcard and can accept one or more path segments as one parameter value.
// A wildcard should be registered in the last path segment only.
func (m *Macro) Trailing() bool {
	return m.trailing
}

// func (m *Macro) SetParamResolver(fn func(memstore.Entry) interface{}) *Macro {
// 	m.ParamResolver = fn
// 	return m
// }

// RegisterFunc registers a parameter function
// to that macro.
// Accepts the func name ("range")
// and the function body, which should return an EvaluatorFunc
// a bool (it will be converted to EvaluatorFunc later on),
// i.e RegisterFunc("min", func(minValue int) func(paramValue string) bool){})
func (m *Macro) RegisterFunc(funcName string, fn interface{}) *Macro {
	fullFn := convertBuilderFunc(fn)
	m.registerFunc(funcName, fullFn)

	return m
}

func (m *Macro) registerFunc(funcName string, fullFn ParamFuncBuilder) {
	if !goodParamFuncName(funcName) {
		return
	}

	for _, fn := range m.funcs {
		if fn.Name == funcName {
			fn.Func = fullFn
			return
		}
	}

	m.funcs = append(m.funcs, ParamFunc{
		Name: funcName,
		Func: fullFn,
	})
}

func (m *Macro) getFunc(funcName string) ParamFuncBuilder {
	for _, fn := range m.funcs {
		if fn.Name == funcName {
			if fn.Func == nil {
				continue
			}
			return fn.Func
		}
	}
	return nil
}
