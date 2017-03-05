package gorillamux

//  +------------------------------------------------------------+
//  | Usage                                                      |
//  +------------------------------------------------------------+
//
//
// package main
//
// import (
// 	"gopkg.in/kataras/iris.v6"
// 	"gopkg.in/kataras/iris.v6/adaptors/gorillamux"
// )
//
// func main() {
// 	app := iris.New()
//
// 	app.Adapt(gorillamux.New()) // Add this line and you're ready.
//
// 	app.Get("/api/users/{userid:[0-9]+}", func(ctx *iris.Context) {
// 		ctx.Writef("User with id: %s", ctx.Param("userid"))
// 	})
//
// 	app.Listen(":8080")
// }

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"gopkg.in/kataras/iris.v6"
)

const dynamicSymbol = '{'

func staticPath(path string) string {
	i := strings.IndexByte(path, dynamicSymbol)
	if i > -1 {
		return path[0:i]
	}

	return path
}

// Name is the name of the router
//
// See $iris_instance.Config.Other for more.
const Name = "gorillamux"

// New returns a new gorilla mux router which can be plugged inside iris.
// This is magic.
func New() iris.Policies {
	var router *mux.Router

	var logger func(iris.LogMode, string)
	return iris.Policies{
		EventPolicy: iris.EventPolicy{
			Boot: func(s *iris.Framework) {
				logger = s.Log
				s.Set(iris.OptionOther(iris.RouterNameConfigKey, Name))
			}},
		RouterReversionPolicy: iris.RouterReversionPolicy{
			// path normalization done on iris' side
			StaticPath: staticPath,
			WildcardPath: func(path string, paramName string) string {
				// {param:.*}
				wildcardPart := "{" + paramName + ":.*}"

				if path[len(path)-1] != '/' {
					// if not ending with slash then prepend the slash to the wildcard path part
					wildcardPart = "/" + wildcardPart
				}
				// finally return the path given + the wildcard path part
				return path + wildcardPart
			},
			Param: func(paramName string) string {
				return "{" + paramName + "}"
			},
			// 	Note: on gorilla mux the {{ url }} and {{ path}} should give the key and the value, not only the values by order.
			// 	{{ url "nameOfTheRoute" "parameterName" "parameterValue"}}.
			//
			// so: {{ url "providerLink" "facebook"}} should become
			//  {{ url "providerLink" "provider" "facebook"}}
			// 	for a path: "/auth/{provider}" with name 'providerLink'
			URLPath: func(r iris.RouteInfo, args ...string) string {
				if router == nil {
					logger(iris.ProdMode, "gorillamux' reverse routing 'URLPath' should be called after Boot/Listen/Serve")
					return ""
				}

				if r == nil {
					return ""
				}

				if gr := router.Get(r.Name()); gr != nil {
					u, err := gr.URLPath(args...)
					if err != nil {
						logger(iris.DevMode, "error on gorilla mux adaptor's URLPath(reverse routing): "+err.Error())
						return ""
					}
					return u.Path
				}
				return ""
			},
		},
		RouterBuilderPolicy: func(repo iris.RouteRepository, context iris.ContextPool) http.Handler {
			router = mux.NewRouter() // re-set the router here,
			// the RouterBuilderPolicy re-runs on every method change (route "offline/online" states mostly)

			repo.Visit(func(route iris.RouteInfo) {
				registerRoute(route, router, context)
			})

			router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.Acquire(w, r)
				// gorilla mux doesn't supports fire method not allowed like iris
				// so this is my hack to support it:
				if ctx.Framework().Config.FireMethodNotAllowed {
					stopVisitor := false
					repo.Visit(func(route iris.RouteInfo) {
						if stopVisitor {
							return
						}
						// this is not going to work 100% for all routes especially the coblex
						// but this is the best solution, to check via static path, subdomain and cors to find the 'correct' route to
						// compare its method in order to implement the status method not allowed in gorilla mux which doesn't support it
						// and if I edit its internal implementation it will be complicated for new releases to be updated.
						p := staticPath(route.Path())
						if route.Subdomain() == "" || route.Subdomain() == ctx.Subdomain() {
							if p == ctx.Path() {
								// we don't care about this route because it has cors and this method is options
								// or its method is equal with the requests but the router didn't select this route
								// that means that the dynamic path didn't match, so we skip it.
								if (route.HasCors() && ctx.Method() == iris.MethodOptions) || ctx.Method() == route.Method() {
									return
								}

								if ctx.Method() != route.Method() {
									// RCF rfc2616 https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
									// The response MUST include an Allow header containing a list of valid methods for the requested resource.
									ctx.SetHeader("Allow", route.Method())
									ctx.EmitError(iris.StatusMethodNotAllowed)
									stopVisitor = true
									return
								}
							}
						}

					})

				} else {
					// to catch custom 404 not found http errors may registered by user
					ctx.EmitError(iris.StatusNotFound)
				}
				context.Release(ctx)
			})

			return router
		},
	}
}

// so easy:
func registerRoute(route iris.RouteInfo, gorillaRouter *mux.Router, context iris.ContextPool) {

	handler := func(w http.ResponseWriter, r *http.Request) {
		context.Run(w, r, func(ctx *iris.Context) {
			if params := mux.Vars(ctx.Request); len(params) > 0 {
				// set them with ctx.Set in order to be accesible by ctx.Param in the user's handler
				for k, v := range params {
					ctx.Set(k, v)
				}
			}
			// including the global middleware, done handlers too
			ctx.Middleware = route.Middleware()
			ctx.Do()
		})
	}

	// remember, we get a new iris.Route foreach of the HTTP Methods, so this should be work
	methods := []string{route.Method()}
	// if route has cors then we register the route with the "OPTIONS" method too
	if route.HasCors() {
		methods = append(methods, http.MethodOptions)
	}
	gorillaRoute := gorillaRouter.HandleFunc(route.Path(), handler).
		Methods(methods...).
		Name(route.Name())

	subdomain := route.Subdomain()
	if subdomain != "" {
		if subdomain == iris.DynamicSubdomainIndicator {
			// it's an iris wildcard subdomain
			// so register it as wildcard on gorilla mux too
			subdomain = "{subdomain}."
		} else {
			// it's a static subdomain (which contains the dot)
		}
		// host = subdomain  + listening host
		gorillaRoute.Host(subdomain + context.Framework().Config.VHost)
	}

	// Author's notes: even if the Method is iris.MethodNone
	// the gorillamux saves the route, so we don't need to use the repo.OnMethodChanged
	// and route.IsOnline() and we don't need the RouteContextLinker, we just serve like request on Offline routes*
}
