package raymond

import (
	"fmt"
	"log"
	"reflect"
	"sync"
)

// Options represents the options argument provided to helpers and context functions.
type Options struct {
	// evaluation visitor
	eval *evalVisitor

	// params
	params []interface{}
	hash   map[string]interface{}
}

// helpers stores all globally registered helpers
var helpers = make(map[string]reflect.Value)

// protects global helpers
var helpersMutex sync.RWMutex

func init() {
	// register builtin helpers
	RegisterHelper("if", ifHelper)
	RegisterHelper("unless", unlessHelper)
	RegisterHelper("with", withHelper)
	RegisterHelper("each", eachHelper)
	RegisterHelper("log", logHelper)
	RegisterHelper("lookup", lookupHelper)
}

// RegisterHelper registers a global helper. That helper will be available to all templates.
func RegisterHelper(name string, helper interface{}) {
	helpersMutex.Lock()
	defer helpersMutex.Unlock()

	if helpers[name] != zero {
		panic(fmt.Errorf("Helper already registered: %s", name))
	}

	val := reflect.ValueOf(helper)
	ensureValidHelper(name, val)

	helpers[name] = val
}

// RegisterHelpers registers several global helpers. Those helpers will be available to all templates.
func RegisterHelpers(helpers map[string]interface{}) {
	for name, helper := range helpers {
		RegisterHelper(name, helper)
	}
}

// ensureValidHelper panics if given helper is not valid
func ensureValidHelper(name string, funcValue reflect.Value) {
	if funcValue.Kind() != reflect.Func {
		panic(fmt.Errorf("Helper must be a function: %s", name))
	}

	funcType := funcValue.Type()

	if funcType.NumOut() != 1 {
		panic(fmt.Errorf("Helper function must return a string or a SafeString: %s", name))
	}

	// @todo Check if first returned value is a string, SafeString or interface{} ?
}

// findHelper finds a globally registered helper
func findHelper(name string) reflect.Value {
	helpersMutex.RLock()
	defer helpersMutex.RUnlock()

	return helpers[name]
}

// newOptions instanciates a new Options
func newOptions(eval *evalVisitor, params []interface{}, hash map[string]interface{}) *Options {
	return &Options{
		eval:   eval,
		params: params,
		hash:   hash,
	}
}

// newEmptyOptions instanciates a new empty Options
func newEmptyOptions(eval *evalVisitor) *Options {
	return &Options{
		eval: eval,
		hash: make(map[string]interface{}),
	}
}

//
// Context Values
//

// Value returns field value from current context.
func (options *Options) Value(name string) interface{} {
	value := options.eval.evalField(options.eval.curCtx(), name, false)
	if !value.IsValid() {
		return nil
	}

	return value.Interface()
}

// ValueStr returns string representation of field value from current context.
func (options *Options) ValueStr(name string) string {
	return Str(options.Value(name))
}

// Ctx returns current evaluation context.
func (options *Options) Ctx() interface{} {
	return options.eval.curCtx().Interface()
}

//
// Hash Arguments
//

// HashProp returns hash property.
func (options *Options) HashProp(name string) interface{} {
	return options.hash[name]
}

// HashStr returns string representation of hash property.
func (options *Options) HashStr(name string) string {
	return Str(options.hash[name])
}

// Hash returns entire hash.
func (options *Options) Hash() map[string]interface{} {
	return options.hash
}

//
// Parameters
//

// Param returns parameter at given position.
func (options *Options) Param(pos int) interface{} {
	if len(options.params) > pos {
		return options.params[pos]
	}

	return nil
}

// ParamStr returns string representation of parameter at given position.
func (options *Options) ParamStr(pos int) string {
	return Str(options.Param(pos))
}

// Params returns all parameters.
func (options *Options) Params() []interface{} {
	return options.params
}

//
// Private data
//

// Data returns private data value.
func (options *Options) Data(name string) interface{} {
	return options.eval.dataFrame.Get(name)
}

// DataStr returns string representation of private data value.
func (options *Options) DataStr(name string) string {
	return Str(options.eval.dataFrame.Get(name))
}

// DataFrame returns current private data frame.
func (options *Options) DataFrame() *DataFrame {
	return options.eval.dataFrame
}

// NewDataFrame instanciates a new data frame that is a copy of current evaluation data frame.
//
// Parent of returned data frame is set to current evaluation data frame.
func (options *Options) NewDataFrame() *DataFrame {
	return options.eval.dataFrame.Copy()
}

// newIterDataFrame instanciates a new data frame and set iteration specific vars
func (options *Options) newIterDataFrame(length int, i int, key interface{}) *DataFrame {
	return options.eval.dataFrame.newIterDataFrame(length, i, key)
}

//
// Evaluation
//

// evalBlock evaluates block with given context, private data and iteration key
func (options *Options) evalBlock(ctx interface{}, data *DataFrame, key interface{}) string {
	result := ""

	if block := options.eval.curBlock(); (block != nil) && (block.Program != nil) {
		result = options.eval.evalProgram(block.Program, ctx, data, key)
	}

	return result
}

// Fn evaluates block with current evaluation context.
func (options *Options) Fn() string {
	return options.evalBlock(nil, nil, nil)
}

// FnCtxData evaluates block with given context and private data frame.
func (options *Options) FnCtxData(ctx interface{}, data *DataFrame) string {
	return options.evalBlock(ctx, data, nil)
}

// FnWith evaluates block with given context.
func (options *Options) FnWith(ctx interface{}) string {
	return options.evalBlock(ctx, nil, nil)
}

// FnData evaluates block with given private data frame.
func (options *Options) FnData(data *DataFrame) string {
	return options.evalBlock(nil, data, nil)
}

// Inverse evaluates "else block".
func (options *Options) Inverse() string {
	result := ""
	if block := options.eval.curBlock(); (block != nil) && (block.Inverse != nil) {
		result, _ = block.Inverse.Accept(options.eval).(string)
	}

	return result
}

// Eval evaluates field for given context.
func (options *Options) Eval(ctx interface{}, field string) interface{} {
	if ctx == nil {
		return nil
	}

	if field == "" {
		return nil
	}

	val := options.eval.evalField(reflect.ValueOf(ctx), field, false)
	if !val.IsValid() {
		return nil
	}

	return val.Interface()
}

//
// Misc
//

// isIncludableZero returns true if 'includeZero' option is set and first param is the number 0
func (options *Options) isIncludableZero() bool {
	b, ok := options.HashProp("includeZero").(bool)
	if ok && b {
		nb, ok := options.Param(0).(int)
		if ok && nb == 0 {
			return true
		}
	}

	return false
}

//
// Builtin helpers
//

// #if block helper
func ifHelper(conditional interface{}, options *Options) interface{} {
	if options.isIncludableZero() || IsTrue(conditional) {
		return options.Fn()
	}

	return options.Inverse()
}

// #unless block helper
func unlessHelper(conditional interface{}, options *Options) interface{} {
	if options.isIncludableZero() || IsTrue(conditional) {
		return options.Inverse()
	}

	return options.Fn()
}

// #with block helper
func withHelper(context interface{}, options *Options) interface{} {
	if IsTrue(context) {
		return options.FnWith(context)
	}

	return options.Inverse()
}

// #each block helper
func eachHelper(context interface{}, options *Options) interface{} {
	if !IsTrue(context) {
		return options.Inverse()
	}

	result := ""

	val := reflect.ValueOf(context)
	switch val.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			// computes private data
			data := options.newIterDataFrame(val.Len(), i, nil)

			// evaluates block
			result += options.evalBlock(val.Index(i).Interface(), data, i)
		}
	case reflect.Map:
		// note: a go hash is not ordered, so result may vary, this behaviour differs from the JS implementation
		keys := val.MapKeys()
		for i := 0; i < len(keys); i++ {
			key := keys[i].Interface()
			ctx := val.MapIndex(keys[i]).Interface()

			// computes private data
			data := options.newIterDataFrame(len(keys), i, key)

			// evaluates block
			result += options.evalBlock(ctx, data, key)
		}
	case reflect.Struct:
		var exportedFields []int

		// collect exported fields only
		for i := 0; i < val.NumField(); i++ {
			if tField := val.Type().Field(i); tField.PkgPath == "" {
				exportedFields = append(exportedFields, i)
			}
		}

		for i, fieldIndex := range exportedFields {
			key := val.Type().Field(fieldIndex).Name
			ctx := val.Field(fieldIndex).Interface()

			// computes private data
			data := options.newIterDataFrame(len(exportedFields), i, key)

			// evaluates block
			result += options.evalBlock(ctx, data, key)
		}
	}

	return result
}

// #log helper
func logHelper(message string) interface{} {
	log.Print(message)
	return ""
}

// #lookup helper
func lookupHelper(obj interface{}, field string, options *Options) interface{} {
	return Str(options.Eval(obj, field))
}
