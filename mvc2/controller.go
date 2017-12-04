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
	"github.com/kataras/iris/mvc/activator/methodfunc"
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

	// FullName it's the last package path segment + "." + the Name.
	// i.e: if login-example/user/controller.go, the FullName is "user.Controller".
	FullName string

	// key = the method's name.
	methods map[string]reflect.Method

	// services []field
	// bindServices func(elem reflect.Value)
	s services
}

func newControllerActivator(engine *Engine, router router.Party, controller BaseController) *ControllerActivator {
	c := &ControllerActivator{
		Engine:  engine,
		Router:  router,
		initRef: controller,
	}

	c.analyze()
	return c
}

var reservedMethodNames = []string{
	"BeginRequest",
	"EndRequest",
	"OnActivate",
}

func isReservedMethod(name string) bool {
	for _, s := range reservedMethodNames {
		if s == name {
			return true
		}
	}

	return false
}

func (c *ControllerActivator) analyze() {

	// set full name.
	{
		// first instance value, needed to validate
		// the actual type of the controller field
		// and to collect and save the instance's persistence fields'
		// values later on.
		val := reflect.Indirect(reflect.ValueOf(c.initRef))

		ctrlName := val.Type().Name()
		pkgPath := val.Type().PkgPath()
		fullName := pkgPath[strings.LastIndexByte(pkgPath, '/')+1:] + "." + ctrlName
		c.FullName = fullName
	}

	// set all available, exported methods.
	{
		typ := reflect.TypeOf(c.initRef) // typ, with pointer
		n := typ.NumMethod()
		c.methods = make(map[string]reflect.Method, n)
		for i := 0; i < n; i++ {
			m := typ.Method(i)
			key := m.Name

			if !isReservedMethod(key) {
				c.methods[key] = m
			}
		}
	}

	// set field index with matching service binders, if any.
	{
		// typ := indirectTyp(reflect.TypeOf(c.initRef)) // element's typ.

		c.s = getServicesFor(reflect.ValueOf(c.initRef), c.Engine.Input)
		// c.bindServices = getServicesBinderForStruct(c.Engine.binders, typ)
	}

	c.analyzeAndRegisterMethods()
}

func (c *ControllerActivator) Handle(method, path, funcName string, middleware ...context.Handler) error {
	if method == "" || path == "" || funcName == "" || isReservedMethod(funcName) {
		// isReservedMethod -> if it's already registered
		// by a previous Handle or analyze methods internally.
		return errSkip
	}

	m, ok := c.methods[funcName]
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

	fmt.Printf("===============%s.%s==============\n", c.FullName, funcName)
	funcIn := getInputArgsFromFunc(m.Type)[1:] // except the receiver, which is the controller pointer itself.

	// get any binders for this func, if any, and
	// take param binders, we can bind them because we know the path here.
	// binders := joinBindersMap(
	// 	getBindersForInput(c.Engine.binders, funcIn...),
	// 	getPathParamsBindersForInput(tmpl.Params, funcIn...))

	s := getServicesFor(m.Func, getPathParamsForInput(tmpl.Params, funcIn...))
	// s.AddSource(indirectVal(reflect.ValueOf(c.initRef)), c.Engine.Input...)

	typ := reflect.TypeOf(c.initRef)
	elem := indirectTyp(typ) // the value, not the pointer.
	hasInputBinders := len(s) > 0
	hasStructBinders := len(c.s) > 0
	n := len(funcIn) + 1

	// be, _ := typ.MethodByName("BeginRequest")
	// en, _ := typ.MethodByName("EndRequest")
	// beginIndex, endIndex := be.Index, en.Index

	handler := func(ctx context.Context) {

		// create a new controller instance of that type(>ptr).
		ctrl := reflect.New(elem)
		//ctrlAndCtxValues := []reflect.Value{ctrl, ctxValue[0]}
		// ctrl.MethodByName("BeginRequest").Call(ctxValue)
		//begin.Func.Call(ctrlAndCtxValues)
		b := ctrl.Interface().(BaseController) // the Interface(). is faster than MethodByName or pre-selected methods.
		// init the request.
		b.BeginRequest(ctx)
		//ctrl.Method(beginIndex).Call(ctxValue)
		// if begin request stopped the execution.
		if ctx.IsStopped() {
			return
		}

		if hasStructBinders {
			elem := ctrl.Elem()
			c.s.FillStructStaticValues(elem)
		}

		if !hasInputBinders {
			methodfunc.DispatchFuncResult(ctx, ctrl.Method(m.Index).Call(emptyIn))
		} else {
			in := make([]reflect.Value, n, n)
			// in[0] = ctrl.Elem()
			in[0] = ctrl
			s.FillFuncInput([]reflect.Value{reflect.ValueOf(ctx)}, &in)
			methodfunc.DispatchFuncResult(ctx, m.Func.Call(in))
			// in := make([]reflect.Value, n, n)
			// ctxValues := []reflect.Value{reflect.ValueOf(ctx)}
			// for k, v := range binders {
			// 	in[k] = v.BindFunc(ctxValues)

			// 	if ctx.IsStopped() {
			// 		return
			// 	}
			// }
			// methodfunc.DispatchFuncResult(ctx, ctrl.Method(m.Index).Call(in))
		}

		// end the request, don't check for stopped because this does the actual writing
		// if no response written already.
		b.EndRequest(ctx)
		// ctrl.MethodByName("EndRequest").Call(ctxValue)
		// end.Func.Call(ctrlAndCtxValues)
		//ctrl.Method(endIndex).Call(ctxValue)
	}

	// register the handler now.
	r := c.Router.Handle(method, path, append(middleware, handler)...)
	// change the main handler's name in order to respect the controller's and give
	// a proper debug message.
	r.MainHandlerName = fmt.Sprintf("%s.%s", c.FullName, funcName)
	// add this as a reserved method name in order to
	// be sure that the same func will not be registered again, even if a custom .Handle later on.
	reservedMethodNames = append(reservedMethodNames, funcName)
	return nil
}

func (c *ControllerActivator) analyzeAndRegisterMethods() {
	for _, m := range c.methods {
		funcName := m.Name
		httpMethod, httpPath, err := parse(m)
		if err != nil && err != errSkip {
			err = fmt.Errorf("MVC: fail to parse the path and method for '%s.%s': %v", c.FullName, m.Name, err)
			c.Router.GetReporter().AddErr(err)
			continue
		}

		c.Handle(httpMethod, httpPath, funcName)
	}
}

const (
	tokenBy       = "By"
	tokenWildcard = "Wildcard" // i.e ByWildcard
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

func (p *parser) parse() (method, path string, err error) {
	funcArgPos := 0
	path = "/"
	// take the first word and check for the method.
	w := p.lexer.next()

	for _, httpMethod := range router.AllMethods {
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
