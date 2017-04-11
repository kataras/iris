package iris

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/kataras/go-errors"
)

type (
	// Policy is an interface which should be implemented by all
	// modules that can adapt a policy to the Framework.
	// With a Policy you can change the behavior of almost each of the existing Iris' features.
	Policy interface {
		// Adapt receives the main *Policies which the Policy should be attached on.
		Adapt(frame *Policies)
	}

	// Policies is the main policies list, the rest of the objects that implement the Policy
	// are adapted to the object which contains a field of type *Policies.
	//
	// Policies can have nested policies behaviors too.
	// See iris.go field: 'policies' and function 'Adapt' for more.
	Policies struct {
		LoggerPolicy
		EventPolicy
		RouterReversionPolicy
		RouterBuilderPolicy
		RouterWrapperPolicy
		RenderPolicy
		TemplateFuncsPolicy
		SessionsPolicy
	}
)

// Adapt implements the behavior in order to be valid to pass Policies as one
// useful for third-party libraries which can provide more tools in one registration.
func (p Policies) Adapt(frame *Policies) {

	// Adapt the logger (optionally, it defaults to a log.New(...).Printf)
	if p.LoggerPolicy != nil {
		p.LoggerPolicy.Adapt(frame)
	}

	// Adapt the flow callbacks (optionally)
	p.EventPolicy.Adapt(frame)

	// Adapt the reverse routing behaviors and policy
	p.RouterReversionPolicy.Adapt(frame)

	// Adapt the router builder
	if p.RouterBuilderPolicy != nil {
		p.RouterBuilderPolicy.Adapt(frame)
	}

	// Adapt any Router's wrapper (optionally)
	if p.RouterWrapperPolicy != nil {
		p.RouterWrapperPolicy.Adapt(frame)
	}

	// Adapt the render policy (both templates and rich content)
	if p.RenderPolicy != nil {
		p.RenderPolicy.Adapt(frame)
	}

	// Adapt the template funcs which can be used to register template funcs
	// from community's packages, it doesn't matters what template/view engine the user
	// uses, and if uses at all.
	if p.TemplateFuncsPolicy != nil {
		p.TemplateFuncsPolicy.Adapt(frame)
	}

	p.SessionsPolicy.Adapt(frame)

}

// LogMode is the type for the LoggerPolicy write mode.
// Two modes available:
// - ProdMode (production level mode)
// - DevMode (development level mode)
//
// The ProdMode should output only fatal errors
// The DevMode ouputs the rest of the errors
//
// Iris logs ONLY errors at both cases.
// By-default ONLY ProdMode level messages are printed to the os.Stdout.
type LogMode uint8

const (
	// ProdMode the production level logger write mode,
	// responsible to fatal errors, errors that happen which
	// your app can't continue running.
	ProdMode LogMode = iota
	// DevMode is the development level logger write mode,
	// responsible to the rest of the errors, for example
	// if you set a app.Favicon("myfav.ico"..) and that fav doesn't exists
	// in your system, then it printed by DevMode and app.Favicon simple doesn't works.
	// But the rest of the app can continue running, so it's not 'Fatal error'
	DevMode
)

// LoggerPolicy is a simple interface which is used to log mostly system panics
// exception for general debugging messages is when the `Framework.Config.IsDevelopment = true`.
// It should prints to the logger.
// Arguments should be handled in the manner of fmt.Printf.
type LoggerPolicy func(mode LogMode, log string)

// Adapt adapts a Logger to the main policies.
func (l LoggerPolicy) Adapt(frame *Policies) {
	if l != nil {
		// notes for me: comment these in order to remember
		//                     why I choose not to do that:
		// It wraps the loggers, so you can use more than one
		// when you have multiple print targets.
		// No this is not a good idea for loggers
		// the user may not expecting this behavior,
		// if the user wants multiple targets she/he
		// can wrap their loggers or use one logger to print on all targets.
		// COMMENT:
		// logger := l
		// if frame.LoggerPolicy != nil {
		// 	prevLogger := frame.LoggerPolicy
		// 	nextLogger := l
		// 	logger = func(mode LogMode, log string) {
		// 		prevLogger(mode, log)
		// 		nextLogger(mode, log)
		// 	}
		// }
		frame.LoggerPolicy = l
	}
}

// The write method exists to LoggerPolicy to be able to passed
// as a valid an io.Writer when you need it.
//
// Write writes len(p) bytes from p to the underlying data stream.
// It returns the number of bytes written from p (0 <= n <= len(p))
// and any error encountered that caused the write to stop early.
// Write must return a non-nil error if it returns n < len(p).
// Write must not modify the slice data, even temporarily.
//
// Implementations must not retain p.
//
// Note: this Write writes as the DevMode.
func (l LoggerPolicy) Write(p []byte) (n int, err error) {
	log := string(p)
	l(DevMode, log)
	return len(log), nil
}

// ToLogger returns a new *log.Logger
// which prints to the the LoggerPolicy function
// this is used when your packages needs explicit an *log.Logger.
//
// Note: Each time you call it, it returns a new *log.Logger.
func (l LoggerPolicy) ToLogger(flag int) *log.Logger {
	return log.New(l, "", flag)
}

type (
	// EventListener is the signature for type of func(*Framework),
	// which is used to register events inside an EventPolicy.
	//
	// Keep note that, inside the policy this is a wrapper
	// in order to register more than one listener without the need of slice.
	EventListener func(*Framework)

	// EventPolicy contains the available Framework's flow event callbacks.
	// Available events:
	// - Boot
	// - Build
	// - Interrupted
	// - Recover
	EventPolicy struct {
		// Boot with a listener type of EventListener.
		//   Fires when '.Boot' is called (by .Serve functions or manually),
		//   before the Build of the components and the Listen,
		//   after VHost and VSCheme configuration has been setted.
		Boot EventListener
		// Before Listen, after Boot
		Build EventListener
		// Interrupted with a listener type of EventListener.
		//   Fires after the terminal is interrupted manually by Ctrl/Cmd + C
		//   which should be used to release external resources.
		// Iris will close and os.Exit at the end of custom interrupted events.
		// If you want to prevent the default behavior just block on the custom Interrupted event.
		Interrupted EventListener
		// Recover with a listener type of func(*Framework, interface{}).
		//   Fires when an unexpected error(panic) is happening at runtime,
		//   while the server's net.Listener accepting requests
		//   or when a '.Must' call contains a filled error.
		//   Used to release external resources and '.Close' the server.
		//   Only one type of this callback is allowed.
		//
		//   If not empty then the Framework will skip its internal
		//   server's '.Close' and panic to its '.Logger' and execute that callback instaed.
		//   Differences from Interrupted:
		//    1. Fires on unexpected errors
		//    2. Only one listener is allowed.
		Recover func(*Framework, error)
	}
)

var _ Policy = EventPolicy{}

// Adapt adaps an EventPolicy object to the main *Policies.
func (e EventPolicy) Adapt(frame *Policies) {

	// Boot event listener, before the build (old: PreBuild)
	frame.EventPolicy.Boot =
		wrapEvtListeners(frame.EventPolicy.Boot, e.Boot)

		// Build event listener, after Boot and before Listen(old: PostBuild & PreListen)
	frame.EventPolicy.Build =
		wrapEvtListeners(frame.EventPolicy.Build, e.Build)

		// Interrupted event listener, when control+C or manually interrupt by os signal
	frame.EventPolicy.Interrupted =
		wrapEvtListeners(frame.EventPolicy.Interrupted, e.Interrupted)

	// Recover event listener, when panic on .Must and inside .Listen/ListenTLS/ListenUNIX/ListenLETSENCRYPT/Serve
	// only one allowed, no wrapper is used.
	if e.Recover != nil {
		frame.EventPolicy.Recover = e.Recover
	}

}

// Fire fires an EventListener with its Framework when listener is not nil.
// Returns true when fired, otherwise false.
func (e EventPolicy) Fire(ln EventListener, s *Framework) bool {
	if ln != nil {
		ln(s)
		return true
	}
	return false
}

func wrapEvtListeners(prev EventListener, next EventListener) EventListener {
	if next == nil {
		return prev
	}
	listener := next
	if prev != nil {
		listener = func(s *Framework) {
			prev(s)
			next(s)
		}
	}

	return listener
}

type (
	// RouterReversionPolicy is used for the reverse routing feature on
	// which custom routers should create and adapt to the Policies.
	RouterReversionPolicy struct {
		// StaticPath should return the static part of the route path
		// for example, with the httprouter(: and *):
		// /api/user/:userid should return /api/user
		// /api/user/:userid/messages/:messageid should return /api/user
		// /dynamicpath/*path should return /dynamicpath
		// /my/path should return /my/path
		StaticPath func(path string) string
		// WildcardPath should return a path converted to a 'dynamic' path
		// for example, with the httprouter(wildcard symbol: '*'):
		// ("/static", "path") should return /static/*path
		// ("/myfiles/assets", "anything") should return /myfiles/assets/*anything
		WildcardPath func(path string, paramName string) string
		// Param should return a named parameter as each router defines named path parameters.
		// For example, with the httprouter(: as named param symbol):
		// userid should return :userid.
		// with gorillamux, userid should return {userid}
		// or userid[1-9]+ should return {userid[1-9]+}.
		// so basically we just wrap the raw parameter name
		// with the start (and end) dynamic symbols of each router implementing the RouterReversionPolicy.
		// It's an optional functionality but it can be used to create adaptors without even know the router
		// that the user uses (which can be taken by app.Config.Other[iris.RouterNameConfigKey].
		//
		// Note: we don't need a function like WildcardParam because the developer
		// can use the Param with a combination with WildcardPath.
		Param func(paramName string) string
		// URLPath used for reverse routing on templates with {{ url }} and {{ path }} funcs.
		// Receives the route name and  arguments and returns its http path
		URLPath func(r RouteInfo, args ...string) string
	}
	// RouterBuilderPolicy is the most useful Policy for custom routers.
	// A custom router should adapt this policy which is a func
	// accepting a route repository (contains all necessary routes information)
	// and a context pool which should be used inside router's handlers.
	RouterBuilderPolicy func(repo RouteRepository, cPool ContextPool) http.Handler
	// RouterWrapperPolicy is the Policy which enables a wrapper on the top of
	// the builded Router. Usually it's useful for third-party middleware
	// when need to wrap the entire application with a middleware like CORS.
	//
	// Developers can Adapt more than one RouterWrapper
	// those wrappers' execution comes from last to first.
	// That means that the second wrapper will wrap the first, and so on.
	RouterWrapperPolicy func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)
)

func normalizePath(path string) string {
	// some users can't understand the difference between
	// request path and operating system's directory path
	// they think that "./" is the index, that's wrong, "/" is the index
	// so fix that here...
	if path[0] == '.' {
		path = path[1:]
	}
	path = strings.Replace(path, "//", "/", -1)
	if len(path) > 1 && strings.IndexByte(path, '/') == len(path)-1 {
		// if  it's not "/" and ending with slash remove that slash
		path = path[0 : len(path)-2]
	}

	return path
}

// Adapt adaps a RouterReversionPolicy object to the main *Policies.
func (r RouterReversionPolicy) Adapt(frame *Policies) {
	if r.StaticPath != nil {
		staticPathFn := r.StaticPath
		frame.RouterReversionPolicy.StaticPath = func(path string) string {
			return staticPathFn(normalizePath(path))
		}
	}

	if r.WildcardPath != nil {
		wildcardPathFn := r.WildcardPath
		frame.RouterReversionPolicy.WildcardPath = func(path string, paramName string) string {
			return wildcardPathFn(normalizePath(path), paramName)
		}
	}

	if r.Param != nil {
		frame.RouterReversionPolicy.Param = r.Param
	}

	if r.URLPath != nil {
		frame.RouterReversionPolicy.URLPath = r.URLPath
	}
}

// Adapt adaps a RouterBuilderPolicy object to the main *Policies.
func (r RouterBuilderPolicy) Adapt(frame *Policies) {
	// What is this kataras?
	// The whole design of this file is brilliant = go's power + my ideas and experience on software architecture.
	//
	// When the router decides to compile/build this behavior
	// then this overload will check for a wrapper too
	// if a wrapper exists it will wrap the result of the RouterBuilder (which is http.Handler, the Router.)
	// and return that instead.
	// I moved the logic here so we don't need a 'compile/build' method inside the routerAdaptor.
	frame.RouterBuilderPolicy = RouterBuilderPolicy(func(repo RouteRepository, cPool ContextPool) http.Handler {
		handler := r(repo, cPool)
		wrapper := frame.RouterWrapperPolicy
		if wrapper != nil {
			originalHandler := handler.ServeHTTP

			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wrapper(w, r, originalHandler)
			})
		}
		return handler
	})
}

// Adapt adaps a RouterWrapperPolicy object to the main *Policies.
func (rw RouterWrapperPolicy) Adapt(frame *Policies) {
	if rw != nil {
		wrapper := rw
		prevWrapper := frame.RouterWrapperPolicy

		if prevWrapper != nil {
			nextWrapper := rw
			wrapper = func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
				if next != nil {
					nexthttpFunc := http.HandlerFunc(func(_w http.ResponseWriter, _r *http.Request) {
						prevWrapper(_w, _r, next)
					})
					nextWrapper(w, r, nexthttpFunc)
				}

			}
		}
		frame.RouterWrapperPolicy = wrapper
	}

}

// RenderPolicy is the type which you can adapt custom renderers
// based on the 'name', simple as that.
// Note that the whole template view system and
// content negotiation works by setting this function via other adaptors.
//
// The functions are wrapped, like any other policy func, the only difference is that
// here the developer has a priority over the defaults:
//  - the last registered is trying to be executed first
//  - the first registered is executing last.
// So a custom adaptor that the community can create and share with each other
// can override the existing one with just a simple registration.
type RenderPolicy func(out io.Writer, name string, bind interface{}, options ...map[string]interface{}) (bool, error)

// Adapt adaps a RenderPolicy object to the main *Policies.
func (r RenderPolicy) Adapt(frame *Policies) {
	if r != nil {
		renderer := r
		prevRenderer := frame.RenderPolicy
		if prevRenderer != nil {
			nextRenderer := r
			renderer = func(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) (bool, error) {
				// Remember: RenderPolicy works in the opossite order of declaration,
				// the last registered is trying to be executed first,
				// the first registered is executing last.
				ok, err := nextRenderer(out, name, binding, options...)
				if !ok {

					prevOk, prevErr := prevRenderer(out, name, binding, options...)
					if err != nil {
						if prevErr != nil {
							err = errors.New(prevErr.Error()).Append(err.Error())
						}
					}
					if prevOk {
						ok = true
					}
				}
				// this renderer is responsible for this name
				// but it has an error, so don't continue to the next
				return ok, err

			}
		}

		frame.RenderPolicy = renderer
	}
}

// TemplateFuncsPolicy sets or overrides template func map.
// Defaults are the iris.URL and iris.Path, all the template engines supports the following:
// {{ url "mynamedroute" "pathParameter_ifneeded"} }
// {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
// {{ render "header.html" }}
// {{ render_r "header.html" }} // partial relative path to current page
// {{ yield }}
// {{ current }}
//
// Developers can already set the template's func map from the view adaptors, example: view.HTML(...).Funcs(...)),
// this type exists in order to be possible from third-party developers to create packages that bind template functions
// to the Iris without the need of knowing what template engine is used by the user or
// what order of declaration the user should follow.
type TemplateFuncsPolicy map[string]interface{} // interface can be: func(arguments ...string) string {}

// Adapt adaps a TemplateFuncsPolicy object to the main *Policies.
func (t TemplateFuncsPolicy) Adapt(frame *Policies) {
	if len(t) > 0 {
		if frame.TemplateFuncsPolicy == nil {
			frame.TemplateFuncsPolicy = t
			return
		}

		if frame.TemplateFuncsPolicy != nil {
			for k, v := range t {
				// set or replace the existing
				frame.TemplateFuncsPolicy[k] = v
			}
		}
	}
}

type (
	// Author's notes:
	// session manager can work as a middleware too
	// but we want an easy-api for the user
	// as we did before with: context.Session().Set/Get...
	// these things cannot be done with middleware and sessions is a critical part of an application
	// which needs attention, so far we used the kataras/go-sessions which I spent many weeks to create
	// and that time has not any known bugs or any other issues, it's fully featured.
	// BUT user may want to use other session library and in the same time users should be able to use
	// iris' api for sessions from context, so a policy is that we need, the policy will contains
	// the Start(responsewriter, request) and the Destroy(responsewriter, request)
	// (keep note that this Destroy is not called at the end of a handler, Start does its job without need to end something
	// sessions are setting in real time, when the user calls .Set ),
	// the Start(responsewriter, request) will return a 'Session' which will contain the API for context.Session() , it should be
	// rich, as before, so the interface will be a clone of the kataras/go-sessions/Session.
	// If the user wants to use other library and that library missing features that kataras/go-sesisons has
	// then the user should make an empty implementation of these calls in order to work.
	// That's no problem, before they couldn't adapt any session manager, now they will can.
	//
	// The databases or stores registration will be in the session manager's responsibility,
	// as well the DestroyByID and DestroyAll (I'm calling these with these names because
	//                                        I take as base the kataras/go-sessions,
	//                                        I have no idea if other session managers
	//                                        supports these things, if not then no problem,
	//                                        these funcs will be not required by the sessions policy)
	//
	// ok let's begin.

	// Session should expose the SessionsPolicy's end-user API.
	// This will be returned at the sess := context.Session().
	Session interface {
		ID() string
		Get(string) interface{}
		HasFlash() bool
		GetFlash(string) interface{}
		GetString(key string) string
		GetFlashString(string) string
		GetInt(key string) (int, error)
		GetInt64(key string) (int64, error)
		GetFloat32(key string) (float32, error)
		GetFloat64(key string) (float64, error)
		GetBoolean(key string) (bool, error)
		GetAll() map[string]interface{}
		GetFlashes() map[string]interface{}
		VisitAll(cb func(k string, v interface{}))
		Set(string, interface{})
		SetFlash(string, interface{})
		Delete(string)
		DeleteFlash(string)
		Clear()
		ClearFlashes()
	}

	// SessionsPolicy is the policy for a session manager.
	//
	// A SessionsPolicy should be responsible to Start a sesion based
	// on raw http.ResponseWriter and http.Request, which should return
	// a compatible iris.Session interface, type. If the external session manager
	// doesn't qualifies, then the user should code the rest of the functions with empty implementation.
	//
	// A SessionsPolicy should be responsible to Destroy a session based
	// on the http.ResponseWriter and http.Request, this function should works individually.
	//
	// No iris.Context required from users. In order to be able to adapt any external session manager.
	//
	// The SessionsPolicy should be adapted once.
	SessionsPolicy struct {
		// Start should starts the session for the particular net/http request
		Start func(http.ResponseWriter, *http.Request) Session

		// Destroy should kills the net/http session and remove the associated cookie
		// Keep note that: Destroy should not called at the end of any handler, it's an independent func.
		// Start should set
		// the values at realtime and if manager doesn't supports these
		// then the user manually have to call its 'done' func inside the handler.
		Destroy func(http.ResponseWriter, *http.Request)
	}
)

// Adapt adaps a SessionsPolicy object to the main *Policies.
//
// Remember: Each policy is an adaptor.
// An adaptor should contains one or more policies too.
func (s SessionsPolicy) Adapt(frame *Policies) {
	if s.Start != nil {
		frame.SessionsPolicy.Start = s.Start
	}
	if s.Destroy != nil {
		frame.SessionsPolicy.Destroy = s.Destroy
	}
}
