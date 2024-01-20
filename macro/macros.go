package macro

import (
	"errors"
	"fmt"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/v12/macro/interpreter/ast"

	"github.com/google/uuid"
)

var (
	// String type
	// Allows anything (single path segment, as everything except the `Path`).
	// Its functions can be used by the rest of the macros and param types whenever not available function by name is used.
	// Because of its "master" boolean value to true (third parameter).
	String = NewMacro("string", "", "", true, false, nil).
		RegisterFunc("regexp", MustRegexp).
		// checks if param value starts with the 'prefix' arg
		RegisterFunc("prefix", func(prefix string) func(string) bool {
			return func(paramValue string) bool {
				return strings.HasPrefix(paramValue, prefix)
			}
		}).
		// checks if param value ends with the 'suffix' arg
		RegisterFunc("suffix", func(suffix string) func(string) bool {
			return func(paramValue string) bool {
				return strings.HasSuffix(paramValue, suffix)
			}
		}).
		// checks if param value contains the 's' arg
		RegisterFunc("contains", func(s string) func(string) bool {
			return func(paramValue string) bool {
				return strings.Contains(paramValue, s)
			}
		}).
		// checks if param value's length is at least 'min'
		RegisterFunc("min", func(min int) func(string) bool {
			return func(paramValue string) bool {
				return len(paramValue) >= min
			}
		}).
		// checks if param value's length is not bigger than 'max'
		RegisterFunc("max", func(max int) func(string) bool {
			return func(paramValue string) bool {
				return max >= len(paramValue)
			}
		}).
		// checks if param value's matches the given input
		RegisterFunc("eq", func(s string) func(string) bool {
			return func(paramValue string) bool {
				return paramValue == s
			}
		}).
		// checks if param value's matches at least one of the inputs
		RegisterFunc("eqor", func(texts []string) func(string) bool {
			if len(texts) == 1 {
				text := texts[0]
				return func(paramValue string) bool {
					return paramValue == text
				}
			}

			return func(paramValue string) bool {
				for _, s := range texts {
					if paramValue == s {
						return true
					}
				}

				return false
			}
		})

	// Int or number type
	// both positive and negative numbers, actual value can be min-max int64 or min-max int32 depends on the arch.
	// If x64: -9223372036854775808 to 9223372036854775807.
	// If x32: -2147483648 to 2147483647 and etc..
	Int = NewMacro("int", "number", 0, false, false, func(paramValue string) (interface{}, bool) {
		v, err := strconv.Atoi(paramValue)
		if err != nil {
			return err, false
		}

		return v, true
	}).
		// checks if the param value's int representation is
		// bigger or equal than 'min'
		RegisterFunc("min", func(min int) func(int) bool {
			return func(paramValue int) bool {
				return paramValue >= min
			}
		}).
		// checks if the param value's int representation is
		// smaller or equal than 'max'.
		RegisterFunc("max", func(max int) func(int) bool {
			return func(paramValue int) bool {
				return paramValue <= max
			}
		}).
		// checks if the param value's int representation is
		// between min and max, including 'min' and 'max'.
		RegisterFunc("range", func(min, max int) func(int) bool {
			return func(paramValue int) bool {
				return !(paramValue < min || paramValue > max)
			}
		})

	// Int8 type
	// -128 to 127.
	Int8 = NewMacro("int8", "", int8(0), false, false, func(paramValue string) (interface{}, bool) {
		v, err := strconv.ParseInt(paramValue, 10, 8)
		if err != nil {
			return err, false
		}
		return int8(v), true
	}).
		RegisterFunc("min", func(min int8) func(int8) bool {
			return func(paramValue int8) bool {
				return paramValue >= min
			}
		}).
		RegisterFunc("max", func(max int8) func(int8) bool {
			return func(paramValue int8) bool {
				return paramValue <= max
			}
		}).
		RegisterFunc("range", func(min, max int8) func(int8) bool {
			return func(paramValue int8) bool {
				return !(paramValue < min || paramValue > max)
			}
		})

	// Int16 type
	// -32768 to 32767.
	Int16 = NewMacro("int16", "", int16(0), false, false, func(paramValue string) (interface{}, bool) {
		v, err := strconv.ParseInt(paramValue, 10, 16)
		if err != nil {
			return err, false
		}
		return int16(v), true
	}).
		RegisterFunc("min", func(min int16) func(int16) bool {
			return func(paramValue int16) bool {
				return paramValue >= min
			}
		}).
		RegisterFunc("max", func(max int16) func(int16) bool {
			return func(paramValue int16) bool {
				return paramValue <= max
			}
		}).
		RegisterFunc("range", func(min, max int16) func(int16) bool {
			return func(paramValue int16) bool {
				return !(paramValue < min || paramValue > max)
			}
		})

	// Int32 type
	// -2147483648 to 2147483647.
	Int32 = NewMacro("int32", "", int32(0), false, false, func(paramValue string) (interface{}, bool) {
		v, err := strconv.ParseInt(paramValue, 10, 32)
		if err != nil {
			return err, false
		}
		return int32(v), true
	}).
		RegisterFunc("min", func(min int32) func(int32) bool {
			return func(paramValue int32) bool {
				return paramValue >= min
			}
		}).
		RegisterFunc("max", func(max int32) func(int32) bool {
			return func(paramValue int32) bool {
				return paramValue <= max
			}
		}).
		RegisterFunc("range", func(min, max int32) func(int32) bool {
			return func(paramValue int32) bool {
				return !(paramValue < min || paramValue > max)
			}
		})

	// Int64 as int64 type
	// -9223372036854775808 to 9223372036854775807.
	Int64 = NewMacro("int64", "long", int64(0), false, false, func(paramValue string) (interface{}, bool) {
		v, err := strconv.ParseInt(paramValue, 10, 64)
		if err != nil { // if err == strconv.ErrRange...
			return err, false
		}
		return v, true
	}).
		// checks if the param value's int64 representation is
		// bigger or equal than 'min'.
		RegisterFunc("min", func(min int64) func(int64) bool {
			return func(paramValue int64) bool {
				return paramValue >= min
			}
		}).
		// checks if the param value's int64 representation is
		// smaller or equal than 'max'.
		RegisterFunc("max", func(max int64) func(int64) bool {
			return func(paramValue int64) bool {
				return paramValue <= max
			}
		}).
		// checks if the param value's int64 representation is
		// between min and max, including 'min' and 'max'.
		RegisterFunc("range", func(min, max int64) func(int64) bool {
			return func(paramValue int64) bool {
				return !(paramValue < min || paramValue > max)
			}
		})

	// Uint as uint type
	// actual value can be min-max uint64 or min-max uint32 depends on the arch.
	// If x64: 0 to 18446744073709551615.
	// If x32: 0 to 4294967295 and etc.
	Uint = NewMacro("uint", "", uint(0), false, false, func(paramValue string) (interface{}, bool) {
		v, err := strconv.ParseUint(paramValue, 10, strconv.IntSize) // 32,64...
		if err != nil {
			return err, false
		}
		return uint(v), true
	}).
		// checks if the param value's int representation is
		// bigger or equal than 'min'
		RegisterFunc("min", func(min uint) func(uint) bool {
			return func(paramValue uint) bool {
				return paramValue >= min
			}
		}).
		// checks if the param value's int representation is
		// smaller or equal than 'max'.
		RegisterFunc("max", func(max uint) func(uint) bool {
			return func(paramValue uint) bool {
				return paramValue <= max
			}
		}).
		// checks if the param value's int representation is
		// between min and max, including 'min' and 'max'.
		RegisterFunc("range", func(min, max uint) func(uint) bool {
			return func(paramValue uint) bool {
				return !(paramValue < min || paramValue > max)
			}
		})

	// Uint8 as uint8 type
	// 0 to 255.
	Uint8 = NewMacro("uint8", "", uint8(0), false, false, func(paramValue string) (interface{}, bool) {
		v, err := strconv.ParseUint(paramValue, 10, 8)
		if err != nil {
			return err, false
		}
		return uint8(v), true
	}).
		// checks if the param value's uint8 representation is
		// bigger or equal than 'min'.
		RegisterFunc("min", func(min uint8) func(uint8) bool {
			return func(paramValue uint8) bool {
				return paramValue >= min
			}
		}).
		// checks if the param value's uint8 representation is
		// smaller or equal than 'max'.
		RegisterFunc("max", func(max uint8) func(uint8) bool {
			return func(paramValue uint8) bool {
				return paramValue <= max
			}
		}).
		// checks if the param value's uint8 representation is
		// between min and max, including 'min' and 'max'.
		RegisterFunc("range", func(min, max uint8) func(uint8) bool {
			return func(paramValue uint8) bool {
				return !(paramValue < min || paramValue > max)
			}
		})

	// Uint16 as uint16 type
	// 0 to 65535.
	Uint16 = NewMacro("uint16", "", uint16(0), false, false, func(paramValue string) (interface{}, bool) {
		v, err := strconv.ParseUint(paramValue, 10, 16)
		if err != nil {
			return err, false
		}
		return uint16(v), true
	}).
		RegisterFunc("min", func(min uint16) func(uint16) bool {
			return func(paramValue uint16) bool {
				return paramValue >= min
			}
		}).
		RegisterFunc("max", func(max uint16) func(uint16) bool {
			return func(paramValue uint16) bool {
				return paramValue <= max
			}
		}).
		RegisterFunc("range", func(min, max uint16) func(uint16) bool {
			return func(paramValue uint16) bool {
				return !(paramValue < min || paramValue > max)
			}
		})

	// Uint32 as uint32 type
	// 0 to 4294967295.
	Uint32 = NewMacro("uint32", "", uint32(0), false, false, func(paramValue string) (interface{}, bool) {
		v, err := strconv.ParseUint(paramValue, 10, 32)
		if err != nil {
			return err, false
		}
		return uint32(v), true
	}).
		RegisterFunc("min", func(min uint32) func(uint32) bool {
			return func(paramValue uint32) bool {
				return paramValue >= min
			}
		}).
		RegisterFunc("max", func(max uint32) func(uint32) bool {
			return func(paramValue uint32) bool {
				return paramValue <= max
			}
		}).
		RegisterFunc("range", func(min, max uint32) func(uint32) bool {
			return func(paramValue uint32) bool {
				return !(paramValue < min || paramValue > max)
			}
		})

	// Uint64 as uint64 type
	// 0 to 18446744073709551615.
	Uint64 = NewMacro("uint64", "", uint64(0), false, false, func(paramValue string) (interface{}, bool) {
		v, err := strconv.ParseUint(paramValue, 10, 64)
		if err != nil {
			return err, false
		}
		return v, true
	}).
		// checks if the param value's uint64 representation is
		// bigger or equal than 'min'.
		RegisterFunc("min", func(min uint64) func(uint64) bool {
			return func(paramValue uint64) bool {
				return paramValue >= min
			}
		}).
		// checks if the param value's uint64 representation is
		// smaller or equal than 'max'.
		RegisterFunc("max", func(max uint64) func(uint64) bool {
			return func(paramValue uint64) bool {
				return paramValue <= max
			}
		}).
		// checks if the param value's uint64 representation is
		// between min and max, including 'min' and 'max'.
		RegisterFunc("range", func(min, max uint64) func(uint64) bool {
			return func(paramValue uint64) bool {
				return !(paramValue < min || paramValue > max)
			}
		})

	// Bool or boolean as bool type
	// a string which is "1" or "t" or "T" or "TRUE" or "true" or "True"
	// or "0" or "f" or "F" or "FALSE" or "false" or "False".
	Bool = NewMacro("bool", "boolean", false, false, false, func(paramValue string) (interface{}, bool) {
		// a simple if statement is faster than regex ^(true|false|True|False|t|0|f|FALSE|TRUE)$
		// in this case.
		v, err := strconv.ParseBool(paramValue)
		if err != nil {
			return err, false
		}
		return v, true
	})

	// ErrParamNotAlphabetical is fired when the parameter value is not an alphabetical text.
	ErrParamNotAlphabetical = errors.New("parameter is not alphabetical")
	alphabeticalEval        = MustRegexp("^[a-zA-Z ]+$")
	// Alphabetical letter type
	// letters only (upper or lowercase)
	Alphabetical = NewMacro("alphabetical", "", "", false, false, func(paramValue string) (interface{}, bool) {
		if !alphabeticalEval(paramValue) {
			return fmt.Errorf("%s: %w", paramValue, ErrParamNotAlphabetical), false
		}
		return paramValue, true
	})

	// ErrParamNotFile is fired when the parameter value is not a form of a file.
	ErrParamNotFile = errors.New("parameter is not a file")
	fileEval        = MustRegexp("^[a-zA-Z0-9_.-]*$")
	// File type
	// letters (upper or lowercase)
	// numbers (0-9)
	// underscore (_)
	// dash (-)
	// point (.)
	// no spaces! or other character
	File = NewMacro("file", "", "", false, false, func(paramValue string) (interface{}, bool) {
		if !fileEval(paramValue) {
			return fmt.Errorf("%s: %w", paramValue, ErrParamNotFile), false
		}
		return paramValue, true
	})
	// Path type
	// anything, should be the last part
	//
	// It allows everything, we have String and Path as different
	// types because I want to give the opportunity to the user
	// to organise the macro functions based on wildcard or single dynamic named path parameter.
	// Should be living in the latest path segment of a route path.
	Path = NewMacro("path", "", "", false, true, nil)

	// UUID string type for validating a uuidv4 (and v1) path parameter.
	// Read more at: https://tools.ietf.org/html/rfc4122.
	UUID = NewMacro("uuid", "uuidv4", "", false, false, func(paramValue string) (interface{}, bool) {
		_, err := uuid.Parse(paramValue) // this is x10+ times faster than regexp.
		if err != nil {
			return err, false
		}

		return paramValue, true
	})

	// Email string type for validating an e-mail path parameter. It returns the address as string, instead of an *mail.Address.
	// Read more at go std mail.ParseAddress method. See the ':email' path parameter for a more strictly version of validation.
	Mail = NewMacro("mail", "", "", false, false, func(paramValue string) (interface{}, bool) {
		_, err := mail.ParseAddress(paramValue)
		if err != nil {
			return fmt.Errorf("%s: %w", paramValue, err), false
		}

		return paramValue, true
	})

	// Email string type for validating an e-mail path parameter. It returns the address as string, instead of an *mail.Address.
	// It is a combined validation using mail.ParseAddress and net.LookupMX so only valid domains can be passed.
	// It's a more strictly version of the ':mail' path parameter.
	Email = NewMacro("email", "", "", false, false, func(paramValue string) (interface{}, bool) {
		_, err := mail.ParseAddress(paramValue)
		if err != nil {
			return fmt.Errorf("%s: %w", paramValue, err), false
		}

		domainPart := strings.Split(paramValue, "@")[1]

		mx, err := net.LookupMX(domainPart)
		if err != nil {
			return fmt.Errorf("%s: %w", paramValue, err), false
		}

		if len(mx) == 0 {
			return fmt.Errorf("%s: mx is empty", paramValue), false
		}

		return paramValue, true
	})

	simpleDateLayout = "2006/01/02"

	// Date type.
	Date = NewMacro("date", "", time.Time{}, false, true, func(paramValue string) (interface{}, bool) {
		tt, err := time.Parse(simpleDateLayout, paramValue)
		if err != nil {
			return fmt.Errorf("%s: %w", paramValue, err), false
		}

		return tt, true
	})

	// ErrParamNotWeekday is fired when the parameter value is not a form of a time.Weekday.
	ErrParamNotWeekday = errors.New("parameter is not a valid weekday")
	longDayNames       = map[string]time.Weekday{
		"Sunday":    time.Sunday,
		"Monday":    time.Monday,
		"Tuesday":   time.Tuesday,
		"Wednesday": time.Wednesday,
		"Thursday":  time.Thursday,
		"Friday":    time.Friday,
		"Saturday":  time.Saturday,
		// lowercase.
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
	}

	// Weekday type, returns a type of time.Weekday.
	// Valid values:
	// 0 to 7 (leading zeros don't matter)  or "Sunday" to "Monday" or "sunday" to "monday".
	Weekday = NewMacro("weekday", "", time.Weekday(0), false, false, func(paramValue string) (interface{}, bool) {
		d, ok := longDayNames[paramValue]
		if !ok {
			// try parse from integer.
			n, err := strconv.Atoi(paramValue)
			if err != nil {
				return fmt.Errorf("%s: %w", paramValue, err), false
			}

			if n < 0 || n > 6 {
				return fmt.Errorf("%s: %w", paramValue, ErrParamNotWeekday), false
			}

			return time.Weekday(n), true
		}

		return d, true
	})

	// Defaults contains the defaults macro and parameters types for the router.
	//
	// Read https://github.com/kataras/iris/tree/main/_examples/routing/macros for more details.
	Defaults = &Macros{
		String,
		Int,
		Int8,
		Int16,
		Int32,
		Int64,
		Uint,
		Uint8,
		Uint16,
		Uint32,
		Uint64,
		Bool,
		Alphabetical,
		File,
		Path,
		UUID,
		Mail,
		Email,
		Date,
		Weekday,
	}
)

// Macros is just a type of a slice of *Macro
// which is responsible to register and search for macros based on the indent(parameter type).
type Macros []*Macro

// Register registers a custom Macro.
// The "indent" should not be empty and should be unique, it is the parameter type's name, i.e "string".
// The "alias" is optionally and it should be unique, it is the alias of the parameter type.
// The "valueType" should be the zero value of the parameter type, i.e "" for string, 0 for int and etc.
// "isMaster" and "isTrailing" is for default parameter type and wildcard respectfully.
// The "evaluator" is the function that is converted to an Iris handler which is executed every time
// before the main chain of a route's handlers that contains this macro of the specific parameter type.
//
// Read https://github.com/kataras/iris/tree/main/_examples/routing/macros for more details.
func (ms *Macros) Register(indent, alias string, valueType any, isMaster, isTrailing bool, evaluator ParamEvaluator) *Macro {
	macro := NewMacro(indent, alias, valueType, isMaster, isTrailing, evaluator)
	if ms.register(macro) {
		return macro
	}
	return nil
}

func (ms *Macros) register(macro *Macro) bool {
	if macro.Indent() == "" {
		return false
	}

	cp := *ms

	for _, m := range cp {
		// can't add more than one with the same ast characteristics.
		if macro.Indent() == m.Indent() {
			return false
		}

		if alias := macro.Alias(); alias != "" {
			if alias == m.Alias() || alias == m.Indent() {
				return false
			}
		}

		if macro.Master() && m.Master() {
			return false
		}
	}

	cp = append(cp, macro)

	*ms = cp
	return true
}

// Unregister removes a macro and its parameter type from the list.
func (ms *Macros) Unregister(indent string) bool {
	cp := *ms

	for i, m := range cp {
		if m.Indent() == indent {
			copy(cp[i:], cp[i+1:])
			cp[len(cp)-1] = nil
			cp = cp[:len(cp)-1]

			*ms = cp
			return true
		}
	}

	return false
}

// Lookup returns the responsible macro for a parameter type, it can return nil.
func (ms *Macros) Lookup(pt ast.ParamType) *Macro {
	if m := ms.Get(pt.Indent()); m != nil {
		return m
	}

	if alias, has := ast.HasAlias(pt); has {
		if m := ms.Get(alias); m != nil {
			return m
		}
	}

	return nil
}

// Get returns the responsible macro for a parameter type, it can return nil.
func (ms *Macros) Get(indentOrAlias string) *Macro {
	if indentOrAlias == "" {
		return nil
	}

	for _, m := range *ms {
		if m.Indent() == indentOrAlias {
			return m
		}

		if m.Alias() == indentOrAlias {
			return m
		}
	}

	return nil
}

// GetMaster returns the default macro and its parameter type,
// by default it will return the `String` macro which is responsible for the "string" parameter type.
func (ms *Macros) GetMaster() *Macro {
	for _, m := range *ms {
		if m.Master() {
			return m
		}
	}

	return nil
}

// GetTrailings returns the macros that have support for wildcards parameter types.
// By default it will return the `Path` macro which is responsible for the "path" parameter type.
func (ms *Macros) GetTrailings() (macros []*Macro) {
	for _, m := range *ms {
		if m.Trailing() {
			macros = append(macros, m)
		}
	}

	return
}

// SetErrorHandler registers a common type path parameter error handler.
// The "fnHandler" MUST be a type of handler.ParamErrorHandler:
// func(ctx iris.Context, paramIndex int, err error). It calls
// the Macro.HandleError method for each of the "ms" entries.
func (ms *Macros) SetErrorHandler(fnHandler interface{}) {
	for _, m := range *ms {
		if m == nil {
			continue
		}

		m.HandleError(fnHandler)
	}
}
