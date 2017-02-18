// Package options implements the Functional Options pattern which provides clean APIs in Go. Options implemented as a function set the state of that option.
// Inspired by https://github.com/tmrts/go-patterns/blob/master/idiom/functional-options.md
package options

import (
	"github.com/fatih/structs"
	"github.com/kataras/go-errors"
)

const (
	// Version current version number
	Version = "0.0.1"
)

var (
	errWrongKind = errors.New("Wrong kind of value %s")
)

// OptionSetter contains the Set function
// Is the receiver to set any options
type OptionSetter interface {
	// Set receives an Options, which itself implements the OptionSetter, and sets any key-based pair values
	Set(Options)
}

// Options is just a map[string]interface{} which implements the OptionSetter too
// this map can be converted to Struct if call 'options.Struct'
type Options map[string]interface{}

// Set implements the OptionSetter
// makes the Options an OptionSetter also
func (temp Options) Set(main Options) {
	for k, v := range temp {
		main[k] = v
	}
}

// Option sets a single option's value, implements the OptionSetter, so it can be passed to the .New,.Default & .Struct also
// useful when you 'like' to pass .Default(Option("key","value")) instead of .Default(Options{Key:value}), both do the same thing
func Option(key string, val interface{}) OptionSetter {
	return Options{key: val}
}

// New receives default options, which can be empty and any option setters
// returns two values
// returns the new, filled from the setters, Options
// and a function which is optionally called by the caller, which receives an interface and returns this interface(pointer to struct) filled by the Setters(same as .Struct)
func New(defOptions Options, opt ...OptionSetter) (Options, func(interface{}) error) {
	for _, o := range opt {
		o.Set(defOptions)
	}

	return defOptions,
		// the return here is optional
		func(a interface{}) error {
			s := structs.New(a)
			for k, v := range defOptions {
				if f, ok := s.FieldOk(k); ok {
					f.Set(v)
				} else {
					return errWrongKind.Format(v)
				}
			}
			return nil
		}
}

// Default accepts option setters and returns the new filled Options, you can pass multi Options as OptionSetter too
func Default(opt ...OptionSetter) Options {
	optionsMap, _ := New(Options{}, opt...)
	return optionsMap
}

// Struct receives default options (pointer to struct) and any option setters
// fills the static pointer to struct (defaultOptions)
// returns an error if somethinf bad happen, for example if wrong type kind of value is setted
func Struct(defaultOptions interface{}, opt ...OptionSetter) error {
	_, theStaticFunc := New(structs.Map(defaultOptions), opt...)
	return theStaticFunc(defaultOptions)
}
