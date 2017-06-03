package raymond

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aymerick/raymond/ast"
)

var (
	// @note borrowed from https://github.com/golang/go/tree/master/src/text/template/exec.go
	errorType       = reflect.TypeOf((*error)(nil)).Elem()
	fmtStringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()

	zero reflect.Value
)

// evalVisitor evaluates a handlebars template with context
type evalVisitor struct {
	tpl *Template

	// contexts stack
	ctx []reflect.Value

	// current data frame (chained with parent)
	dataFrame *DataFrame

	// block parameters stack
	blockParams []map[string]interface{}

	// block statements stack
	blocks []*ast.BlockStatement

	// expressions stack
	exprs []*ast.Expression

	// memoize expressions that were function calls
	exprFunc map[*ast.Expression]bool

	// used for info on panic
	curNode ast.Node
}

// NewEvalVisitor instanciate a new evaluation visitor with given context and initial private data frame
//
// If privData is nil, then a default data frame is created
func newEvalVisitor(tpl *Template, ctx interface{}, privData *DataFrame) *evalVisitor {
	frame := privData
	if frame == nil {
		frame = NewDataFrame()
	}

	return &evalVisitor{
		tpl:       tpl,
		ctx:       []reflect.Value{reflect.ValueOf(ctx)},
		dataFrame: frame,
		exprFunc:  make(map[*ast.Expression]bool),
	}
}

// at sets current node
func (v *evalVisitor) at(node ast.Node) {
	v.curNode = node
}

//
// Contexts stack
//

// pushCtx pushes new context to the stack
func (v *evalVisitor) pushCtx(ctx reflect.Value) {
	v.ctx = append(v.ctx, ctx)
}

// popCtx pops last context from stack
func (v *evalVisitor) popCtx() reflect.Value {
	if len(v.ctx) == 0 {
		return zero
	}

	var result reflect.Value
	result, v.ctx = v.ctx[len(v.ctx)-1], v.ctx[:len(v.ctx)-1]

	return result
}

// rootCtx returns root context
func (v *evalVisitor) rootCtx() reflect.Value {
	return v.ctx[0]
}

// curCtx returns current context
func (v *evalVisitor) curCtx() reflect.Value {
	return v.ancestorCtx(0)
}

// ancestorCtx returns ancestor context
func (v *evalVisitor) ancestorCtx(depth int) reflect.Value {
	index := len(v.ctx) - 1 - depth
	if index < 0 {
		return zero
	}

	return v.ctx[index]
}

//
// Private data frame
//

// setDataFrame sets new data frame
func (v *evalVisitor) setDataFrame(frame *DataFrame) {
	v.dataFrame = frame
}

// popDataFrame sets back parent data frame
func (v *evalVisitor) popDataFrame() {
	v.dataFrame = v.dataFrame.parent
}

//
// Block Parameters stack
//

// pushBlockParams pushes new block params to the stack
func (v *evalVisitor) pushBlockParams(params map[string]interface{}) {
	v.blockParams = append(v.blockParams, params)
}

// popBlockParams pops last block params from stack
func (v *evalVisitor) popBlockParams() map[string]interface{} {
	var result map[string]interface{}

	if len(v.blockParams) == 0 {
		return result
	}

	result, v.blockParams = v.blockParams[len(v.blockParams)-1], v.blockParams[:len(v.blockParams)-1]
	return result
}

// blockParam iterates on stack to find given block parameter, and returns its value or nil if not founc
func (v *evalVisitor) blockParam(name string) interface{} {
	for i := len(v.blockParams) - 1; i >= 0; i-- {
		for k, v := range v.blockParams[i] {
			if name == k {
				return v
			}
		}
	}

	return nil
}

//
// Blocks stack
//

// pushBlock pushes new block statement to stack
func (v *evalVisitor) pushBlock(block *ast.BlockStatement) {
	v.blocks = append(v.blocks, block)
}

// popBlock pops last block statement from stack
func (v *evalVisitor) popBlock() *ast.BlockStatement {
	if len(v.blocks) == 0 {
		return nil
	}

	var result *ast.BlockStatement
	result, v.blocks = v.blocks[len(v.blocks)-1], v.blocks[:len(v.blocks)-1]

	return result
}

// curBlock returns current block statement
func (v *evalVisitor) curBlock() *ast.BlockStatement {
	if len(v.blocks) == 0 {
		return nil
	}

	return v.blocks[len(v.blocks)-1]
}

//
// Expressions stack
//

// pushExpr pushes new expression to stack
func (v *evalVisitor) pushExpr(expression *ast.Expression) {
	v.exprs = append(v.exprs, expression)
}

// popExpr pops last expression from stack
func (v *evalVisitor) popExpr() *ast.Expression {
	if len(v.exprs) == 0 {
		return nil
	}

	var result *ast.Expression
	result, v.exprs = v.exprs[len(v.exprs)-1], v.exprs[:len(v.exprs)-1]

	return result
}

// curExpr returns current expression
func (v *evalVisitor) curExpr() *ast.Expression {
	if len(v.exprs) == 0 {
		return nil
	}

	return v.exprs[len(v.exprs)-1]
}

//
// Error functions
//

// errPanic panics
func (v *evalVisitor) errPanic(err error) {
	panic(fmt.Errorf("Evaluation error: %s\nCurrent node:\n\t%s", err, v.curNode))
}

// errorf panics with a custom message
func (v *evalVisitor) errorf(format string, args ...interface{}) {
	v.errPanic(fmt.Errorf(format, args...))
}

//
// Evaluation
//

// evalProgram eEvaluates program with given context and returns string result
func (v *evalVisitor) evalProgram(program *ast.Program, ctx interface{}, data *DataFrame, key interface{}) string {
	blockParams := make(map[string]interface{})

	// compute block params
	if len(program.BlockParams) > 0 {
		blockParams[program.BlockParams[0]] = ctx
	}

	if (len(program.BlockParams) > 1) && (key != nil) {
		blockParams[program.BlockParams[1]] = key
	}

	// push contexts
	if len(blockParams) > 0 {
		v.pushBlockParams(blockParams)
	}

	ctxVal := reflect.ValueOf(ctx)
	if ctxVal.IsValid() {
		v.pushCtx(ctxVal)
	}

	if data != nil {
		v.setDataFrame(data)
	}

	// evaluate program
	result, _ := program.Accept(v).(string)

	// pop contexts
	if data != nil {
		v.popDataFrame()
	}

	if ctxVal.IsValid() {
		v.popCtx()
	}

	if len(blockParams) > 0 {
		v.popBlockParams()
	}

	return result
}

// evalPath evaluates all path parts with given context
func (v *evalVisitor) evalPath(ctx reflect.Value, parts []string, exprRoot bool) (reflect.Value, bool) {
	partResolved := false

	for i := 0; i < len(parts); i++ {
		part := parts[i]

		// "[foo bar]"" => "foo bar"
		if (len(part) >= 2) && (part[0] == '[') && (part[len(part)-1] == ']') {
			part = part[1 : len(part)-1]
		}

		ctx = v.evalField(ctx, part, exprRoot)
		if !ctx.IsValid() {
			break
		}

		// we resolved at least one part of path
		partResolved = true
	}

	return ctx, partResolved
}

// evalField evaluates field with given context
func (v *evalVisitor) evalField(ctx reflect.Value, fieldName string, exprRoot bool) reflect.Value {
	result := zero

	ctx, _ = indirect(ctx)
	if !ctx.IsValid() {
		return result
	}

	// check if this is a method call
	result, isMeth := v.evalMethod(ctx, fieldName, exprRoot)
	if !isMeth {
		switch ctx.Kind() {
		case reflect.Struct:
			// example: firstName => FirstName
			expFieldName := strings.Title(fieldName)

			// check if struct have this field and that it is exported
			if tField, ok := ctx.Type().FieldByName(expFieldName); ok && (tField.PkgPath == "") {
				// struct field
				result = ctx.FieldByIndex(tField.Index)
				break
			}

			// attempts to find template variable name as a struct tag
			result = v.evalStructTag(ctx, fieldName)
		case reflect.Map:
			nameVal := reflect.ValueOf(fieldName)
			if nameVal.Type().AssignableTo(ctx.Type().Key()) {
				// map key
				result = ctx.MapIndex(nameVal)
			}
		case reflect.Array, reflect.Slice:
			if i, err := strconv.Atoi(fieldName); (err == nil) && (i < ctx.Len()) {
				result = ctx.Index(i)
			}
		}
	}

	// check if result is a function
	result, _ = indirect(result)
	if result.Kind() == reflect.Func {
		result = v.evalFieldFunc(fieldName, result, exprRoot)
	}

	return result
}

// evalFieldFunc tries to evaluate given method name, and a boolean to indicate if this was a method call
func (v *evalVisitor) evalMethod(ctx reflect.Value, name string, exprRoot bool) (reflect.Value, bool) {
	if ctx.Kind() != reflect.Interface && ctx.CanAddr() {
		ctx = ctx.Addr()
	}

	method := ctx.MethodByName(name)
	if !method.IsValid() {
		// example: subject() => Subject()
		method = ctx.MethodByName(strings.Title(name))
	}

	if !method.IsValid() {
		return zero, false
	}

	return v.evalFieldFunc(name, method, exprRoot), true
}

// evalFieldFunc evaluates given function
func (v *evalVisitor) evalFieldFunc(name string, funcVal reflect.Value, exprRoot bool) reflect.Value {
	ensureValidHelper(name, funcVal)

	var options *Options
	if exprRoot {
		// create function arg with all params/hash
		expr := v.curExpr()
		options = v.helperOptions(expr)

		// ok, that expression was a function call
		v.exprFunc[expr] = true
	} else {
		// we are not at root of expression, so we are a parameter... and we don't like
		// infinite loops caused by trying to parse ourself forever
		options = newEmptyOptions(v)
	}

	return v.callFunc(name, funcVal, options)
}

// evalStructTag checks for the existence of a struct tag containing the
// name of the variable in the template. This allows for a template variable to
// be separated from the field in the struct.
func (v *evalVisitor) evalStructTag(ctx reflect.Value, name string) reflect.Value {
	val := reflect.ValueOf(ctx.Interface())

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		tag := field.Tag.Get("handlebars")
		if tag == name {
			return val.Field(i)
		}
	}

	return zero
}

// findBlockParam returns node's block parameter
func (v *evalVisitor) findBlockParam(node *ast.PathExpression) (string, interface{}) {
	if len(node.Parts) > 0 {
		name := node.Parts[0]
		if value := v.blockParam(name); value != nil {
			return name, value
		}
	}

	return "", nil
}

// evalPathExpression evaluates a path expression
func (v *evalVisitor) evalPathExpression(node *ast.PathExpression, exprRoot bool) interface{} {
	var result interface{}

	if name, value := v.findBlockParam(node); value != nil {
		// block parameter value

		// We push a new context so we can evaluate the path expression (note: this may be a bad idea).
		//
		// Example:
		//   {{#foo as |bar|}}
		//     {{bar.baz}}
		//   {{/foo}}
		//
		// With data:
		//   {"foo": {"baz": "bat"}}
		newCtx := map[string]interface{}{name: value}

		v.pushCtx(reflect.ValueOf(newCtx))
		result = v.evalCtxPathExpression(node, exprRoot)
		v.popCtx()
	} else {
		ctxTried := false

		if node.IsDataRoot() {
			// context path
			result = v.evalCtxPathExpression(node, exprRoot)

			ctxTried = true
		}

		if (result == nil) && node.Data {
			// if it is @root, then we tried to evaluate with root context but nothing was found
			// so let's try with private data

			// private data
			result = v.evalDataPathExpression(node, exprRoot)
		}

		if (result == nil) && !ctxTried {
			// context path
			result = v.evalCtxPathExpression(node, exprRoot)
		}
	}

	return result
}

// evalDataPathExpression evaluates a private data path expression
func (v *evalVisitor) evalDataPathExpression(node *ast.PathExpression, exprRoot bool) interface{} {
	// find data frame
	frame := v.dataFrame
	for i := node.Depth; i > 0; i-- {
		if frame.parent == nil {
			return nil
		}
		frame = frame.parent
	}

	// resolve data
	// @note Can be changed to v.evalCtx() as context can't be an array
	result, _ := v.evalCtxPath(reflect.ValueOf(frame.data), node.Parts, exprRoot)
	return result
}

// evalCtxPathExpression evaluates a context path expression
func (v *evalVisitor) evalCtxPathExpression(node *ast.PathExpression, exprRoot bool) interface{} {
	v.at(node)

	if node.IsDataRoot() {
		// `@root` - remove the first part
		parts := node.Parts[1:len(node.Parts)]

		result, _ := v.evalCtxPath(v.rootCtx(), parts, exprRoot)
		return result
	}

	return v.evalDepthPath(node.Depth, node.Parts, exprRoot)
}

// evalDepthPath iterates on contexts, starting at given depth, until there is one that resolve given path parts
func (v *evalVisitor) evalDepthPath(depth int, parts []string, exprRoot bool) interface{} {
	var result interface{}
	partResolved := false

	ctx := v.ancestorCtx(depth)

	for (result == nil) && ctx.IsValid() && (depth <= len(v.ctx) && !partResolved) {
		// try with context
		result, partResolved = v.evalCtxPath(ctx, parts, exprRoot)

		// As soon as we find the first part of a path, we must not try to resolve with parent context if result is finally `nil`
		// Reference: "Dotted Names - Context Precedence" mustache test
		if !partResolved && (result == nil) {
			// try with previous context
			depth++
			ctx = v.ancestorCtx(depth)
		}
	}

	return result
}

// evalCtxPath evaluates path with given context
func (v *evalVisitor) evalCtxPath(ctx reflect.Value, parts []string, exprRoot bool) (interface{}, bool) {
	var result interface{}
	partResolved := false

	switch ctx.Kind() {
	case reflect.Array, reflect.Slice:
		// Array context
		var results []interface{}

		for i := 0; i < ctx.Len(); i++ {
			value, _ := v.evalPath(ctx.Index(i), parts, exprRoot)
			if value.IsValid() {
				results = append(results, value.Interface())
			}
		}

		result = results
	default:
		// NOT array context
		var value reflect.Value

		value, partResolved = v.evalPath(ctx, parts, exprRoot)
		if value.IsValid() {
			result = value.Interface()
		}
	}

	return result, partResolved
}

//
// Helpers
//

// isHelperCall returns true if given expression is a helper call
func (v *evalVisitor) isHelperCall(node *ast.Expression) bool {
	if helperName := node.HelperName(); helperName != "" {
		return v.findHelper(helperName) != zero
	}
	return false
}

// findHelper finds given helper
func (v *evalVisitor) findHelper(name string) reflect.Value {
	// check template helpers
	if h := v.tpl.findHelper(name); h != zero {
		return h
	}

	// check global helpers
	return findHelper(name)
}

// callFunc calls function with given options
func (v *evalVisitor) callFunc(name string, funcVal reflect.Value, options *Options) reflect.Value {
	params := options.Params()

	funcType := funcVal.Type()

	// @todo Is there a better way to do that ?
	strType := reflect.TypeOf("")
	boolType := reflect.TypeOf(true)

	// check parameters number
	addOptions := false
	numIn := funcType.NumIn()

	if numIn == len(params)+1 {
		lastArgType := funcType.In(numIn - 1)
		if reflect.TypeOf(options).AssignableTo(lastArgType) {
			addOptions = true
		}
	}

	if !addOptions && (len(params) != numIn) {
		v.errorf("Helper '%s' called with wrong number of arguments, needed %d but got %d", name, numIn, len(params))
	}

	// check and collect arguments
	args := make([]reflect.Value, numIn)
	for i, param := range params {
		arg := reflect.ValueOf(param)
		argType := funcType.In(i)

		if !arg.IsValid() {
			if canBeNil(argType) {
				arg = reflect.Zero(argType)
			} else if argType.Kind() == reflect.String {
				arg = reflect.ValueOf("")
			} else {
				// @todo Maybe we can panic on that
				return reflect.Zero(strType)
			}
		}

		if !arg.Type().AssignableTo(argType) {
			if strType.AssignableTo(argType) {
				// convert parameter to string
				arg = reflect.ValueOf(strValue(arg))
			} else if boolType.AssignableTo(argType) {
				// convert parameter to bool
				val, _ := isTrueValue(arg)
				arg = reflect.ValueOf(val)
			} else {
				v.errorf("Helper %s called with argument %d with type %s but it should be %s", name, i, arg.Type(), argType)
			}
		}

		args[i] = arg
	}

	if addOptions {
		args[numIn-1] = reflect.ValueOf(options)
	}

	result := funcVal.Call(args)

	return result[0]
}

// callHelper invoqs helper function for given expression node
func (v *evalVisitor) callHelper(name string, helper reflect.Value, node *ast.Expression) interface{} {
	result := v.callFunc(name, helper, v.helperOptions(node))
	if !result.IsValid() {
		return nil
	}

	// @todo We maybe want to ensure here that helper returned a string or a SafeString
	return result.Interface()
}

// helperOptions computes helper options argument from an expression
func (v *evalVisitor) helperOptions(node *ast.Expression) *Options {
	var params []interface{}
	var hash map[string]interface{}

	for _, paramNode := range node.Params {
		param := paramNode.Accept(v)
		params = append(params, param)
	}

	if node.Hash != nil {
		hash, _ = node.Hash.Accept(v).(map[string]interface{})
	}

	return newOptions(v, params, hash)
}

//
// Partials
//

// findPartial finds given partial
func (v *evalVisitor) findPartial(name string) *partial {
	// check template partials
	if p := v.tpl.findPartial(name); p != nil {
		return p
	}

	// check global partials
	return findPartial(name)
}

// partialContext computes partial context
func (v *evalVisitor) partialContext(node *ast.PartialStatement) reflect.Value {
	if nb := len(node.Params); nb > 1 {
		v.errorf("Unsupported number of partial arguments: %d", nb)
	}

	if (len(node.Params) > 0) && (node.Hash != nil) {
		v.errorf("Passing both context and named parameters to a partial is not allowed")
	}

	if len(node.Params) == 1 {
		return reflect.ValueOf(node.Params[0].Accept(v))
	}

	if node.Hash != nil {
		hash, _ := node.Hash.Accept(v).(map[string]interface{})
		return reflect.ValueOf(hash)
	}

	return zero
}

// evalPartial evaluates a partial
func (v *evalVisitor) evalPartial(p *partial, node *ast.PartialStatement) string {
	// get partial template
	partialTpl, err := p.template()
	if err != nil {
		v.errPanic(err)
	}

	// push partial context
	ctx := v.partialContext(node)
	if ctx.IsValid() {
		v.pushCtx(ctx)
	}

	// evaluate partial template
	result, _ := partialTpl.program.Accept(v).(string)

	// ident partial
	result = indentLines(result, node.Indent)

	if ctx.IsValid() {
		v.popCtx()
	}

	return result
}

// indentLines indents all lines of given string
func indentLines(str string, indent string) string {
	if indent == "" {
		return str
	}

	var indented []string

	lines := strings.Split(str, "\n")
	for i, line := range lines {
		if (i == (len(lines) - 1)) && (line == "") {
			// input string ends with a new line
			indented = append(indented, line)
		} else {
			indented = append(indented, indent+line)
		}
	}

	return strings.Join(indented, "\n")
}

//
// Functions
//

// wasFuncCall returns true if given expression was a function call
func (v *evalVisitor) wasFuncCall(node *ast.Expression) bool {
	// check if expression was tagged as a function call
	return v.exprFunc[node]
}

//
// Visitor interface
//

// Statements

// VisitProgram implements corresponding Visitor interface method
func (v *evalVisitor) VisitProgram(node *ast.Program) interface{} {
	v.at(node)

	buf := new(bytes.Buffer)

	for _, n := range node.Body {
		if str := Str(n.Accept(v)); str != "" {
			if _, err := buf.Write([]byte(str)); err != nil {
				v.errPanic(err)
			}
		}
	}

	return buf.String()
}

// VisitMustache implements corresponding Visitor interface method
func (v *evalVisitor) VisitMustache(node *ast.MustacheStatement) interface{} {
	v.at(node)

	// evaluate expression
	expr := node.Expression.Accept(v)

	// check if this is a safe string
	isSafe := isSafeString(expr)

	// get string value
	str := Str(expr)
	if !isSafe && !node.Unescaped {
		// escape html
		str = Escape(str)
	}

	return str
}

// VisitBlock implements corresponding Visitor interface method
func (v *evalVisitor) VisitBlock(node *ast.BlockStatement) interface{} {
	v.at(node)

	v.pushBlock(node)

	var result interface{}

	// evaluate expression
	expr := node.Expression.Accept(v)

	if v.isHelperCall(node.Expression) || v.wasFuncCall(node.Expression) {
		// it is the responsability of the helper/function to evaluate block
		result = expr
	} else {
		val := reflect.ValueOf(expr)

		truth, _ := isTrueValue(val)
		if truth {
			if node.Program != nil {
				switch val.Kind() {
				case reflect.Array, reflect.Slice:
					concat := ""

					// Array context
					for i := 0; i < val.Len(); i++ {
						// Computes new private data frame
						frame := v.dataFrame.newIterDataFrame(val.Len(), i, nil)

						// Evaluate program
						concat += v.evalProgram(node.Program, val.Index(i).Interface(), frame, i)
					}

					result = concat
				default:
					// NOT array
					result = v.evalProgram(node.Program, expr, nil, nil)
				}
			}
		} else if node.Inverse != nil {
			result, _ = node.Inverse.Accept(v).(string)
		}
	}

	v.popBlock()

	return result
}

// VisitPartial implements corresponding Visitor interface method
func (v *evalVisitor) VisitPartial(node *ast.PartialStatement) interface{} {
	v.at(node)

	// partialName: helperName | sexpr
	name, ok := ast.HelperNameStr(node.Name)
	if !ok {
		if subExpr, ok := node.Name.(*ast.SubExpression); ok {
			name, _ = subExpr.Accept(v).(string)
		}
	}

	if name == "" {
		v.errorf("Unexpected partial name: %q", node.Name)
	}

	partial := v.findPartial(name)
	if partial == nil {
		v.errorf("Partial not found: %s", name)
	}

	return v.evalPartial(partial, node)
}

// VisitContent implements corresponding Visitor interface method
func (v *evalVisitor) VisitContent(node *ast.ContentStatement) interface{} {
	v.at(node)

	// write content as is
	return node.Value
}

// VisitComment implements corresponding Visitor interface method
func (v *evalVisitor) VisitComment(node *ast.CommentStatement) interface{} {
	v.at(node)

	// ignore comments
	return ""
}

// Expressions

// VisitExpression implements corresponding Visitor interface method
func (v *evalVisitor) VisitExpression(node *ast.Expression) interface{} {
	v.at(node)

	var result interface{}
	done := false

	v.pushExpr(node)

	// helper call
	if helperName := node.HelperName(); helperName != "" {
		if helper := v.findHelper(helperName); helper != zero {
			result = v.callHelper(helperName, helper, node)
			done = true
		}
	}

	if !done {
		// literal
		if literal, ok := node.LiteralStr(); ok {
			if val := v.evalField(v.curCtx(), literal, true); val.IsValid() {
				result = val.Interface()
				done = true
			}
		}
	}

	if !done {
		// field path
		if path := node.FieldPath(); path != nil {
			// @todo Find a cleaner way ! Don't break the pattern !
			// this is an exception to visitor pattern, because we need to pass the info
			// that this path is at root of current expression
			if val := v.evalPathExpression(path, true); val != nil {
				result = val
			}
		}
	}

	v.popExpr()

	return result
}

// VisitSubExpression implements corresponding Visitor interface method
func (v *evalVisitor) VisitSubExpression(node *ast.SubExpression) interface{} {
	v.at(node)

	return node.Expression.Accept(v)
}

// VisitPath implements corresponding Visitor interface method
func (v *evalVisitor) VisitPath(node *ast.PathExpression) interface{} {
	return v.evalPathExpression(node, false)
}

// Literals

// VisitString implements corresponding Visitor interface method
func (v *evalVisitor) VisitString(node *ast.StringLiteral) interface{} {
	v.at(node)

	return node.Value
}

// VisitBoolean implements corresponding Visitor interface method
func (v *evalVisitor) VisitBoolean(node *ast.BooleanLiteral) interface{} {
	v.at(node)

	return node.Value
}

// VisitNumber implements corresponding Visitor interface method
func (v *evalVisitor) VisitNumber(node *ast.NumberLiteral) interface{} {
	v.at(node)

	return node.Number()
}

// Miscellaneous

// VisitHash implements corresponding Visitor interface method
func (v *evalVisitor) VisitHash(node *ast.Hash) interface{} {
	v.at(node)

	result := make(map[string]interface{})

	for _, pair := range node.Pairs {
		if value := pair.Accept(v); value != nil {
			result[pair.Key] = value
		}
	}

	return result
}

// VisitHashPair implements corresponding Visitor interface method
func (v *evalVisitor) VisitHashPair(node *ast.HashPair) interface{} {
	v.at(node)

	return node.Val.Accept(v)
}
