package mvc2

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/core/router/macro"
	"github.com/kataras/iris/core/router/macro/interpreter/ast"
)

type BaseController interface {
	BeginRequest(context.Context)
	EndRequest(context.Context)
}

// C is the basic BaseController type that can be used as an embedded anonymous field
// to custom end-dev controllers.
//
// func(c *ExampleController) Get() string |
// (string, string) |
// (string, int) |
// int |
// (int, string |
// (string, error) |
// bool |
// (any, bool) |
// error |
// (int, error) |
// (customStruct, error) |
// customStruct |
// (customStruct, int) |
// (customStruct, string) |
// Result or (Result, error)
// where Get is an HTTP Method func.
//
// Look `core/router#APIBuilder#Controller` method too.
//
// It completes the `activator.BaseController` interface.
//
// Example at: https://github.com/kataras/iris/tree/master/_examples/mvc/overview/web/controllers.
// Example usage at: https://github.com/kataras/iris/blob/master/mvc/method_result_test.go#L17.
type C struct {
	// The current context.Context.
	//
	// we have to name it for two reasons:
	// 1: can't ignore these via reflection, it doesn't give an option to
	// see if the functions is derived from another type.
	// 2: end-developer may want to use some method functions
	// or any fields that could be conflict with the context's.
	Ctx context.Context
}

var _ BaseController = &C{}

// BeginRequest starts the request by initializing the `Context` field.
func (c *C) BeginRequest(ctx context.Context) { c.Ctx = ctx }

// EndRequest does nothing, is here to complete the `BaseController` interface.
func (c *C) EndRequest(ctx context.Context) {}

type ControllerActivator struct {
	Engine *Engine
	// the router is used on the `Activate` and can be used by end-dev on the `OnActivate`
	// to register any custom controller's functions as handlers but we will need it here
	// in order to not create a new type like `ActivationPayload` for the `OnActivate`.
	Router router.Party

	initRef BaseController // the BaseController as it's passed from the end-dev.
	Type    reflect.Type   // raw type of the BaseController (initRef).
	// FullName it's the last package path segment + "." + the Name.
	// i.e: if login-example/user/controller.go, the FullName is "user.Controller".
	FullName string

	// the methods names that is already binded to a handler,
	// the BeginRequest, EndRequest and OnActivate are reserved by the internal implementation.
	reservedMethods []string

	// input are always empty after the `activate`
	// are used to build the bindings, and we need this field
	// because we have 3 states (Engine.Input, OnActivate, Bind)
	// that we can add or override binding values.
	input []reflect.Value

	// the bindings that comes from input (and Engine) and can be binded to the controller's(initRef) fields.
	bindings *targetStruct
}

var emptyMethod = reflect.Method{}

func newControllerActivator(router router.Party, controller BaseController, bindValues ...reflect.Value) *ControllerActivator {
	c := &ControllerActivator{
		Router:  router,
		initRef: controller,
		reservedMethods: []string{
			"BeginRequest",
			"EndRequest",
			"OnActivate",
		},
		// the following will make sure that if
		// the controller's has set-ed pointer struct fields by the end-dev
		// we will include them to the bindings.
		// set bindings to the non-zero pointer fields' values that may be set-ed by
		// the end-developer when declaring the controller,
		// activate listeners needs them in order to know if something set-ed already or not,
		// look `BindTypeExists`.
		input: append(lookupNonZeroFieldsValues(reflect.ValueOf(controller)), bindValues...),
	}

	c.analyze()
	return c
}

func (c *ControllerActivator) isReservedMethod(name string) bool {
	for _, s := range c.reservedMethods {
		if s == name {
			return true
		}
	}

	return false
}

func (c *ControllerActivator) analyze() {
	// set full name.

	// first instance value, needed to validate
	// the actual type of the controller field
	// and to collect and save the instance's persistence fields'
	// values later on.
	typ := reflect.TypeOf(c.initRef) // type with pointer
	elemTyp := indirectTyp(typ)

	ctrlName := elemTyp.Name()
	pkgPath := elemTyp.PkgPath()
	fullName := pkgPath[strings.LastIndexByte(pkgPath, '/')+1:] + "." + ctrlName
	c.FullName = fullName
	c.Type = typ

	// register all available, exported methods to handlers if possible.
	n := typ.NumMethod()
	for i := 0; i < n; i++ {
		m := typ.Method(i)
		funcName := m.Name

		if c.isReservedMethod(funcName) {
			continue
		}

		httpMethod, httpPath, err := parse(m)
		if err != nil && err != errSkip {
			err = fmt.Errorf("MVC: fail to parse the path and method for '%s.%s': %v", c.FullName, m.Name, err)
			c.Router.GetReporter().AddErr(err)
			continue
		}

		c.Handle(httpMethod, httpPath, funcName)
	}

}

// SetBindings will override any bindings with the new "values".
func (c *ControllerActivator) SetBindings(values ...reflect.Value) {
	// set field index with matching binders, if any.
	c.input = values
	c.bindings = newTargetStruct(reflect.ValueOf(c.initRef), values...)
}

// Bind binds values to this controller, if you want to share
// binding values between controllers use the Engine's `Bind` function instead.
func (c *ControllerActivator) Bind(values ...interface{}) {
	for _, val := range values {
		if v := reflect.ValueOf(val); goodVal(v) {
			c.input = append(c.input, v)
		}
	}
}

// BindTypeExists returns true if a binder responsible to
// bind and return a type of "typ" is already registered to this controller.
func (c *ControllerActivator) BindTypeExists(typ reflect.Type) bool {
	for _, in := range c.input {
		if equalTypes(in.Type(), typ) {
			return true
		}
	}
	return false
}

func (c *ControllerActivator) activate() {
	c.SetBindings(c.input...)
}

var emptyIn = []reflect.Value{}

func (c *ControllerActivator) Handle(method, path, funcName string, middleware ...context.Handler) error {
	if method == "" || path == "" || funcName == "" ||
		c.isReservedMethod(funcName) {
		// isReservedMethod -> if it's already registered
		// by a previous Handle or analyze methods internally.
		return errSkip
	}

	m, ok := c.Type.MethodByName(funcName)
	if !ok {
		err := fmt.Errorf("MVC: function '%s' doesn't exist inside the '%s' controller",
			funcName, c.FullName)
		c.Router.GetReporter().AddErr(err)
		return err
	}

	tmpl, err := macro.Parse(path, c.Router.Macros())
	if err != nil {
		err = fmt.Errorf("MVC: fail to parse the path for '%s.%s': %v", c.FullName, funcName, err)
		c.Router.GetReporter().AddErr(err)
		return err
	}

	// add this as a reserved method name in order to
	// be sure that the same func will not be registered again, even if a custom .Handle later on.
	c.reservedMethods = append(c.reservedMethods, funcName)

	// fmt.Printf("===============%s.%s==============\n", c.FullName, funcName)

	funcIn := getInputArgsFromFunc(m.Type) // except the receiver, which is the controller pointer itself.

	pathParams := getPathParamsForInput(tmpl.Params, funcIn[1:]...)
	funcBindings := newTargetFunc(m.Func, pathParams...)

	elemTyp := indirectTyp(c.Type) // the element value, not the pointer.

	n := len(funcIn)

	handler := func(ctx context.Context) {

		// create a new controller instance of that type(>ptr).
		ctrl := reflect.New(elemTyp)
		b := ctrl.Interface().(BaseController) // the Interface(). is faster than MethodByName or pre-selected methods.
		// init the request.
		b.BeginRequest(ctx)

		// if begin request stopped the execution.
		if ctx.IsStopped() {
			return
		}

		if !c.bindings.Valid && !funcBindings.Valid {
			DispatchFuncResult(ctx, ctrl.Method(m.Index).Call(emptyIn))
		} else {
			ctxValue := reflect.ValueOf(ctx)

			if c.bindings.Valid {
				elem := ctrl.Elem()
				c.bindings.Fill(elem, ctxValue)
				if ctx.IsStopped() {
					return
				}

				// we do this in order to reduce in := make...
				// if not func input binders, we execute the handler with empty input args.
				if !funcBindings.Valid {
					DispatchFuncResult(ctx, ctrl.Method(m.Index).Call(emptyIn))
				}
			}
			// otherwise, it has one or more valid input binders,
			// make the input and call the func using those.
			if funcBindings.Valid {
				in := make([]reflect.Value, n, n)
				in[0] = ctrl
				funcBindings.Fill(&in, ctxValue)
				if ctx.IsStopped() {
					return
				}

				DispatchFuncResult(ctx, m.Func.Call(in))
			}

		}

		// end the request, don't check for stopped because this does the actual writing
		// if no response written already.
		b.EndRequest(ctx)
	}

	// register the handler now.
	c.Router.Handle(method, path, append(middleware, handler)...).
		// change the main handler's name in order to respect the controller's and give
		// a proper debug message.
		MainHandlerName = fmt.Sprintf("%s.%s", c.FullName, funcName)

	return nil
}

const (
	tokenBy       = "By"
	tokenWildcard = "Wildcard" // "ByWildcard".
)

// word lexer, not characters.
type lexer struct {
	words []string
	cur   int
}

func newLexer(s string) *lexer {
	l := new(lexer)
	l.reset(s)
	return l
}

func (l *lexer) reset(s string) {
	l.cur = -1
	var words []string
	if s != "" {
		end := len(s)
		start := -1

		for i, n := 0, end; i < n; i++ {
			c := rune(s[i])
			if unicode.IsUpper(c) {
				// it doesn't count the last uppercase
				if start != -1 {
					end = i
					words = append(words, s[start:end])
				}
				start = i
				continue
			}
			end = i + 1
		}

		if end > 0 && len(s) >= end {
			words = append(words, s[start:end])
		}
	}

	l.words = words
}

func (l *lexer) next() (w string) {
	cur := l.cur + 1

	if w = l.peek(cur); w != "" {
		l.cur++
	}

	return
}

func (l *lexer) skip() {
	if cur := l.cur + 1; cur < len(l.words) {
		l.cur = cur
	} else {
		l.cur = len(l.words) - 1
	}
}

func (l *lexer) peek(idx int) string {
	if idx < len(l.words) {
		return l.words[idx]
	}
	return ""
}

func (l *lexer) peekNext() (w string) {
	return l.peek(l.cur + 1)
}

func (l *lexer) peekPrev() (w string) {
	if l.cur > 0 {
		cur := l.cur - 1
		w = l.words[cur]
	}

	return w
}

var posWords = map[int]string{
	0: "",
	1: "first",
	2: "second",
	3: "third",
	4: "forth",
	5: "five",
	6: "sixth",
	7: "seventh",
	8: "eighth",
	9: "ninth",
}

func genParamKey(argIdx int) string {
	return "param" + posWords[argIdx] // paramfirst, paramsecond...
}

type parser struct {
	lexer *lexer
	fn    reflect.Method
}

func parse(fn reflect.Method) (method, path string, err error) {
	p := &parser{
		fn:    fn,
		lexer: newLexer(fn.Name),
	}
	return p.parse()
}

func methodTitle(httpMethod string) string {
	httpMethodFuncName := strings.Title(strings.ToLower(httpMethod))
	return httpMethodFuncName
}

var errSkip = errors.New("skip")

var allMethods = append(router.AllMethods[0:], []string{"ALL", "ANY"}...)

func (p *parser) parse() (method, path string, err error) {
	funcArgPos := 0
	path = "/"
	// take the first word and check for the method.
	w := p.lexer.next()

	for _, httpMethod := range allMethods {
		possibleMethodFuncName := methodTitle(httpMethod)
		if strings.Index(w, possibleMethodFuncName) == 0 {
			method = httpMethod
			break
		}
	}

	if method == "" {
		// this is not a valid method to parse, we just skip it,
		//  it may be used for end-dev's use cases.
		return "", "", errSkip
	}

	for {
		w := p.lexer.next()
		if w == "" {
			break
		}

		if w == tokenBy {
			funcArgPos++ // starting with 1 because in typ.NumIn() the first is the struct receiver.

			// No need for these:
			// ByBy will act like /{param:type}/{param:type} as users expected
			// if func input arguments are there, else act By like normal path /by.
			//
			// if p.lexer.peekPrev() == tokenBy || typ.NumIn() == 1 { // ByBy, then act this second By like a path
			// 	a.relPath += "/" + strings.ToLower(w)
			// 	continue
			// }

			if path, err = p.parsePathParam(path, w, funcArgPos); err != nil {
				return "", "", err
			}

			continue
		}
		// static path.
		path += "/" + strings.ToLower(w)

	}

	return
}

func (p *parser) parsePathParam(path string, w string, funcArgPos int) (string, error) {
	typ := p.fn.Type

	if typ.NumIn() <= funcArgPos {

		// By found but input arguments are not there, so act like /by path without restricts.
		path += "/" + strings.ToLower(w)
		return path, nil
	}

	var (
		paramKey  = genParamKey(funcArgPos) // paramfirst, paramsecond...
		paramType = ast.ParamTypeString     // default string
	)

	// string, int...
	goType := typ.In(funcArgPos).Name()
	nextWord := p.lexer.peekNext()

	if nextWord == tokenWildcard {
		p.lexer.skip() // skip the Wildcard word.
		paramType = ast.ParamTypePath
	} else if pType := ast.LookupParamTypeFromStd(goType); pType != ast.ParamTypeUnExpected {
		// it's not wildcard, so check base on our available macro types.
		paramType = pType
	} else {
		return "", errors.New("invalid syntax for " + p.fn.Name)
	}

	// /{paramfirst:path}, /{paramfirst:long}...
	path += fmt.Sprintf("/{%s:%s}", paramKey, paramType.String())

	if nextWord == "" && typ.NumIn() > funcArgPos+1 {
		// By is the latest word but func is expected
		// more path parameters values, i.e:
		// GetBy(name string, age int)
		// The caller (parse) doesn't need to know
		// about the incremental funcArgPos because
		// it will not need it.
		return p.parsePathParam(path, nextWord, funcArgPos+1)
	}

	return path, nil
}
