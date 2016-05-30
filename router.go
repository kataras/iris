package iris

import (
	"net/http/pprof"
	"strings"
	"sync"

	"github.com/kataras/iris/utils"
	"github.com/valyala/fasthttp"
)

const (

	// ParameterStartByte is very used on the node, it's just contains the byte for the ':' rune/char
	ParameterStartByte = byte(':')
	// SlashByte is just a byte of '/' rune/char
	SlashByte = byte('/')
	// Slash is just a string of "/"
	Slash = "/"
	// MatchEverythingByte is just a byte of '*" rune/char
	MatchEverythingByte = byte('*')

	// HTTP Methods(1)

	// MethodGet "GET"
	MethodGet = "GET"
	// MethodPost "POST"
	MethodPost = "POST"
	// MethodPut "PUT"
	MethodPut = "PUT"
	// MethodDelete "DELETE"
	MethodDelete = "DELETE"
	// MethodConnect "CONNECT"
	MethodConnect = "CONNECT"
	// MethodHead "HEAD"
	MethodHead = "HEAD"
	// MethodPatch "PATCH"
	MethodPatch = "PATCH"
	// MethodOptions "OPTIONS"
	MethodOptions = "OPTIONS"
	// MethodTrace "TRACE"
	MethodTrace = "TRACE"
)

var (
	// HTTP Methods(2)

	// MethodConnectBytes []byte("CONNECT")
	MethodConnectBytes = []byte(MethodConnect)
	// AllMethods "GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE"
	AllMethods = [...]string{"GET", "POST", "PUT", "DELETE", "CONNECT", "HEAD", "PATCH", "OPTIONS", "TRACE"}
)

// router internal is the route serving service, one router per server
type router struct {
	*GardenParty
	*HTTPErrorContainer
	station        *Iris
	garden         *Garden
	methodMatch    func(m1, m2 string) bool
	getRequestPath func(*fasthttp.RequestCtx) []byte
	ServeRequest   func(reqCtx *fasthttp.RequestCtx)
	// errorPool is responsible to  get the Context to handle not found errors
	errorPool sync.Pool
	//it's true when optimize already ran
	optimized bool
	mu        sync.Mutex
}

// methodMatchCorsFunc is sets the methodMatch when cors enabled (look router.optimize), it's allowing OPTIONS method to all other methods except GET
func methodMatchCorsFunc(m1, reqMethod string) bool {
	return m1 == reqMethod || reqMethod == MethodOptions //(m1 != MethodGet && reqMethod == MethodOptions)
}

// methodMatchFunc for normal method match
func methodMatchFunc(m1, m2 string) bool {
	return m1 == m2
}

func getRequestPathDefault(reqCtx *fasthttp.RequestCtx) []byte {
	// default to escape then
	return reqCtx.Path()
}

// newRouter creates and returns an empty router
func newRouter(station *Iris) *router {
	r := &router{
		station:            station,
		garden:             &Garden{},
		methodMatch:        methodMatchFunc,
		getRequestPath:     getRequestPathDefault,
		HTTPErrorContainer: defaultHTTPErrors(),
		GardenParty:        &GardenParty{relativePath: "/", station: station, root: true},
		errorPool:          station.newContextPool()}

	r.ServeRequest = r.serveFunc

	return r

}

// addRoute calls the Plant, is created to set the router's station
func (r *router) addRoute(route IRoute) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.garden.Plant(r.station, route)
}

//check if any tree has cors setted to true, means that cors middleware is added
func (r *router) cors() (has bool) {
	r.garden.visitAll(func(i int, tree *tree) {
		if tree.cors {
			has = true
		}
	})
	return
}

// check if any tree has subdomains
func (r *router) hosts() (has bool) {
	r.garden.visitAll(func(i int, tree *tree) {
		if tree.hosts {
			has = true
		}
	})
	return
}

// optimize runs once before listen, it checks if cors or hosts enabled and make the necessary changes to the Router itself
func (r *router) optimize() {
	if r.optimized {
		return
	}

	if r.cors() {
		r.methodMatch = methodMatchCorsFunc
	}

	// For performance only,in order to not check at runtime for hosts and subdomains, I think it's better to do this:
	if r.hosts() {
		r.ServeRequest = r.serveDomainFunc
	}

	//if PathEscape disabled, then take the raw URI
	if r.station.config.DisablePathEscape {
		r.getRequestPath = func(reqCtx *fasthttp.RequestCtx) []byte {
			// RequestURI fixes the https://github.com/kataras/iris/issues/135
			return reqCtx.RequestURI()
		}
	}

	// set the debug profiling handlers if Profile enabled, before the server startup, not earlier
	if r.station.config.Profile && r.station.config.ProfilePath != "" {
		debugPath := r.station.config.ProfilePath

		htmlMiddleware := func(ctx *Context) {
			ctx.SetContentType(ContentHTML + r.station.rest.CompiledCharset)
			ctx.Next()
		}

		indexHandler := ToHandlerFunc(pprof.Index)
		cmdlineHandler := ToHandlerFunc(pprof.Cmdline)
		profileHandler := ToHandlerFunc(pprof.Profile)
		symbolHandler := ToHandlerFunc(pprof.Symbol)

		goroutineHandler := ToHandlerFunc(pprof.Handler("goroutine"))
		heapHandler := ToHandlerFunc(pprof.Handler("heap"))
		threadcreateHandler := ToHandlerFunc(pprof.Handler("threadcreate"))
		debugBlockHandler := ToHandlerFunc(pprof.Handler("block"))

		r.Get(debugPath+"/*action", htmlMiddleware, func(ctx *Context) {
			action := ctx.Param("action")
			if len(action) > 1 {
				if strings.Contains(action, "cmdline") {
					cmdlineHandler.Serve((ctx))
				} else if strings.Contains(action, "profile") {
					profileHandler.Serve(ctx)
				} else if strings.Contains(action, "symbol") {
					symbolHandler.Serve(ctx)
				} else if strings.Contains(action, "goroutine") {
					goroutineHandler.Serve(ctx)
				} else if strings.Contains(action, "heap") {
					heapHandler.Serve(ctx)
				} else if strings.Contains(action, "threadcreate") {
					threadcreateHandler.Serve(ctx)
				} else if strings.Contains(action, "debug/block") {
					debugBlockHandler.Serve(ctx)
				}
			} else {
				indexHandler.Serve(ctx)
			}

		})

	}

	r.optimized = true
}

// notFound internal method, it justs takes the context from pool ( in order to have the custom errors available) and procedure a Not Found 404 error
// this is being called when no route was found used on the ServeRequest.
func (r *router) notFound(reqCtx *fasthttp.RequestCtx) {
	ctx := r.errorPool.Get().(*Context)
	ctx.Reset(reqCtx)
	ctx.NotFound()
	r.errorPool.Put(ctx)
}

//************************************************************************************
// serveFunc & serveDomainFunc selected on router.optimize, which runs before station's listen
// they are not used directly.
//************************************************************************************

// serve finds and serves a route by it's request context
// If no route found, it sends an http status 404
func (r *router) serveFunc(reqCtx *fasthttp.RequestCtx) {
	method := utils.BytesToString(reqCtx.Method())
	tree := r.garden.first
	path := utils.BytesToString(r.getRequestPath(reqCtx))
	for tree != nil {
		if r.methodMatch(tree.method, method) {
			if !tree.serve(reqCtx, path) {
				r.notFound(reqCtx)
			}
			return
		}
		tree = tree.next
	}
	//not found, get the first's pool and use that  to send a custom http error(if setted)

	r.notFound(reqCtx)

}

// serveDomainFunc finds and serves a domain tree's route by it's request context
// If no route found, it sends an http status 404
func (r *router) serveDomainFunc(reqCtx *fasthttp.RequestCtx) {
	method := utils.BytesToString(reqCtx.Method())
	domain := utils.BytesToString(reqCtx.Host())
	path := r.getRequestPath(reqCtx)
	tree := r.garden.first
	for tree != nil {
		if tree.hosts && tree.domain == domain {
			// here we have an issue, at fasthttp/uri.go 273-274 line normalize path it adds a '/' slash at the beginning, it doesn't checks for subdomains
			// I could fix it but i leave it as it is, I just create a new function inside tree named 'serveReturn' which accepts a path too. ->
			//-> reqCtx.Request.URI().SetPathBytes(append(reqCtx.Host(), reqCtx.Path()...)) <-
			path = append(reqCtx.Host(), path...)
		}
		if r.methodMatch(tree.method, method) {
			if tree.serve(reqCtx, utils.BytesToString(path)) {
				return
			}
		}
		tree = tree.next
	}
	//not found, get the first's pool and use that  to send a custom http error(if setted)
	r.notFound(reqCtx)
}
