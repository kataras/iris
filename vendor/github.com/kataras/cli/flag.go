package cli

import (
	goflags "flag"
	"github.com/kataras/go-errors"
	"reflect"
	"strings"
)

// Flags the flags passed to the command's Action
type Flags []*Flag

// Flag is the command's action's flag, use it by flags.Get("myflag").Alias()/.Name/.Usage/.Raw/.Value
type Flag struct {
	Name    string
	Default interface{}
	Usage   string
	Value   interface{}
	Raw     *goflags.FlagSet
}

// Alias returns the alias of the flag's name (the first letter)
func (f Flag) Alias() string {
	if len(f.Name) > 1 {
		return f.Name[0:1]
	}
	return f.Name
}

// Get returns a flag by it's name, if flag not found returns nil
func (c Flags) Get(name string) *Flag {
	for idx, v := range c {
		if v.Name == name {
			return c[idx]
		}
	}

	return nil
}

// String returns the flag's value as string by it's name, if not found returns empty string ""
// panics on !string
func (c Flags) String(name string) string {
	f := c.Get(name)
	if f == nil {
		return ""
	}
	return *f.Value.(*string) //*f.Value if string
}

// Bool returns the flag's value as bool by it's name, if not found returns false
// panics on !bool
func (c Flags) Bool(name string) bool {
	f := c.Get(name)
	if f != nil {
		return *f.Value.(*bool)
	}
	return false
}

// Int returns the flag's value as int by it's name, if can't parse int then returns -1
// panics on !int
func (c Flags) Int(name string) int {
	f := c.Get(name)
	if f == nil {
		return -1
	}
	return *f.Value.(*int)
}

// IsValid returns true if flags are valid, otherwise false
func (c Flags) IsValid() bool {
	if c.Validate() != nil {
		return false
	}
	return true
}

var errFlagMissing = errors.New("Required flag [-%s] is missing.")

// Validate returns nil if this flags are valid, otherwise returns an error message
func (c Flags) Validate() error {
	var notFilled []string
	for _, v := range c {
		// if no value given (nil) for required flag then it is not valid
		isRequired := v.Default == nil
		val := reflect.ValueOf(v.Value).Elem().String()
		if isRequired && val == "" {
			notFilled = append(notFilled, v.Name)
		}
	}

	if len(notFilled) > 0 {
		if len(notFilled) == 1 {
			return errFlagMissing.Format(notFilled[0])
		}
		return errFlagMissing.Format(strings.Join(notFilled, ","))

	}
	return nil

}

// ToString returns all flags in form of string and comma seperated
func (c Flags) ToString() (summary string) {
	for idx, v := range c {
		summary += "-" + v.Alias()
		if idx < len(c)-1 {
			summary += ", "
		}
	}

	if len(summary) > 0 {
		summary = "[" + summary + "]"
	}

	return
}

func requestFlagValue(flagset *goflags.FlagSet, name string, defaultValue interface{}, usage string) interface{} {
	if defaultValue == nil { // if it's nil then set it to a string because we will get err: interface is nil, not string if we pass a required flag
		defaultValue = ""
	}
	switch defaultValue.(type) {
	case int:
		{
			valPointer := flagset.Int(name, defaultValue.(int), usage)

			// it's not h (-h) for example but it's host, then assign it's alias also
			if len(name) > 1 {
				alias := name[0:1]
				flagset.IntVar(valPointer, alias, defaultValue.(int), usage)
			}
			return valPointer
		}
	case bool:
		{
			valPointer := flagset.Bool(name, defaultValue.(bool), usage)

			// it's not h (-h) for example but it's host, then assign it's alias also
			if len(name) > 1 {
				alias := name[0:1]
				flagset.BoolVar(valPointer, alias, defaultValue.(bool), usage)
			}
			return valPointer
		}
	default:
		valPointer := flagset.String(name, defaultValue.(string), usage)

		// it's not h (-h) for example but it's host, then assign it's alias also
		if len(name) > 1 {
			alias := name[0:1]
			flagset.StringVar(valPointer, alias, defaultValue.(string), usage)
		}

		return valPointer

	}
}
