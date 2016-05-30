package iris

import (
	"strings"
)

type (
	// IRoute is the interface which the Route should implements
	// it useful to have it as an interface because this interface is passed to the plugins
	IRoute interface {
		GetMethod() string
		GetDomain() string
		GetPath() string
		GetMiddleware() Middleware
		HasCors() bool
	}

	// Route contains basic and temporary info about the route in order to be stored to the tree
	// It's struct because we pass it ( as IRoute) to the plugins
	Route struct {
		method     string
		domain     string
		fullpath   string
		middleware Middleware
	}
)

var _ IRoute = &Route{}

// NewRoute creates, from a path string, and a slice of HandlerFunc
func NewRoute(method string, registedPath string, middleware Middleware) *Route {
	domain := ""
	//dirdy but I'm not touching this again:P
	if registedPath[0] != SlashByte && strings.Contains(registedPath, ".") && (strings.IndexByte(registedPath, SlashByte) == -1 || strings.IndexByte(registedPath, SlashByte) > strings.IndexByte(registedPath, '.')) {
		//means that is a path with domain
		//we have to extract the domain

		//find the first '/'
		firstSlashIndex := strings.IndexByte(registedPath, SlashByte)

		//firt of all remove the first '/' if that exists and we have domain
		if firstSlashIndex == 0 {
			//e.g /admin.ideopod.com/hey
			//then just remove the first slash and re-execute the NewRoute and return it
			registedPath = registedPath[1:]
			return NewRoute(method, registedPath, middleware)
		}
		//if it's just the domain, then set it(registedPath) as the domain
		//and after set the registedPath to a slash '/' for the path part
		if firstSlashIndex == -1 {
			domain = registedPath
			registedPath = Slash
		} else {
			//we have a domain + path
			domain = registedPath[0:firstSlashIndex]
			registedPath = registedPath[len(domain):]
		}

	}
	r := &Route{method: method, domain: domain, fullpath: registedPath, middleware: middleware}

	return r
}

// GetMethod returns the http method
func (r Route) GetMethod() string {
	return r.method
}

// GetDomain returns the registed domain which this route is ( if none, is "" which is means "localhost"/127.0.0.1)
func (r Route) GetDomain() string {
	return r.domain
}

// GetPath returns the full registed path
func (r Route) GetPath() string {
	return r.fullpath
}

// GetMiddleware returns the chain of the []HandlerFunc registed to this Route
func (r Route) GetMiddleware() Middleware {
	return r.middleware
}

// HasCors check if middleware passsed to a route has cors
func (r *Route) HasCors() bool {
	return RouteConflicts(r, "httpmethod")
}

// RouteConflicts checks for route's middleware conflicts
func RouteConflicts(r *Route, with string) bool {
	for _, h := range r.middleware {
		if m, ok := h.(interface {
			Conflicts() string
		}); ok {
			if c := m.Conflicts(); c == with {
				return true
			}
		}
	}
	return false
}
