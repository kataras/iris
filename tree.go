package iris

import (
	"bytes"
	"sync"

	"github.com/kataras/iris/utils"
	"github.com/valyala/fasthttp"
)

type (
	tree struct {
		station    *Iris
		method     string
		rootBranch *Branch
		domain     string
		hosts      bool //if domain != "" we set it directly on .Plant
		cors       bool // if cross domain allow enabled
		pool       sync.Pool
		next       *tree
	}

	// Garden is the main area which routes are planted/placed
	Garden struct {
		first *tree
	}
)

// garden

func (g *Garden) visitAll(f func(i int, tr *tree)) {
	t := g.first
	i := 0
	for t != nil {

		f(i, t)
		t = t.next
	}
}

// visitAllBreak like visitAll but if true to the function then it breaks
func (g *Garden) visitAllBreak(f func(i int, tr *tree) bool) {
	t := g.first
	i := 0
	for t != nil {

		if f(i, t) {
			break
		}
		t = t.next
	}
}

func (g *Garden) last() (t *tree) {

	t = g.first
	for t.next != nil {
		t = t.next
	}
	return
}

// getRootByMethodAndDomain returns the correct branch which it's method&domain is equal to the given method&domain, from a garden's tree
// trees with  no domain means that their domain==""
func (g *Garden) getRootByMethodAndDomain(method string, domain string) (b *Branch) {
	g.visitAll(func(i int, t *tree) {
		if t.domain == domain && t.method == method {
			b = t.rootBranch
		}
	})

	return
}

// Plant plants/adds a route to the garden
func (g *Garden) Plant(station *Iris, _route IRoute) {
	method := _route.GetMethod()
	domain := _route.GetDomain()
	path := _route.GetPath()
	theRoot := g.getRootByMethodAndDomain(method, domain)
	if theRoot == nil {
		theRoot = new(Branch)
		theNewTree := newTree(station, method, theRoot, domain, len(domain) > 0, _route.HasCors())
		if g.first == nil {
			g.first = theNewTree
		} else {
			g.last().next = theNewTree
		}

	}
	theRoot.AddBranch(domain+path, _route.GetMiddleware())
}

// tree

func newTree(station *Iris, method string, theRoot *Branch, domain string, hosts bool, hasCors bool) *tree {
	t := &tree{station: station, method: method, rootBranch: theRoot, domain: domain, hosts: hosts, cors: hasCors, pool: station.newContextPool()}
	return t
}

// serve serves the route
func (_tree *tree) serve(reqCtx *fasthttp.RequestCtx, path string) bool {
	ctx := _tree.pool.Get().(*Context)
	ctx.Reset(reqCtx)
	middleware, params, mustRedirect := _tree.rootBranch.GetBranch(path, ctx.Params) // pass the parameters here for 0 allocation
	if middleware != nil {
		ctx.Params = params
		ctx.middleware = middleware
		//ctx.Request.Header.SetUserAgentBytes(DefaultUserAgent)
		ctx.Do()
		_tree.pool.Put(ctx)
		return true
	} else if mustRedirect && _tree.station.config.PathCorrection && !bytes.Equal(reqCtx.Method(), MethodConnectBytes) {

		reqPath := path
		pathLen := len(reqPath)

		if pathLen > 1 {

			if reqPath[pathLen-1] == '/' {
				reqPath = reqPath[:pathLen-1] //remove the last /
			} else {
				//it has path prefix, it doesn't ends with / and it hasn't be found, then just add the slash
				reqPath = reqPath + "/"
			}

			ctx.Request.URI().SetPath(reqPath)
			urlToRedirect := utils.BytesToString(ctx.Request.RequestURI())

			ctx.Redirect(urlToRedirect, 301) //	StatusMovedPermanently
			// RFC2616 recommends that a short note "SHOULD" be included in the
			// response because older user agents may not understand 301/307.
			// Shouldn't send the response for POST or HEAD; that leaves GET.
			if _tree.method == MethodGet {
				note := "<a href=\"" + utils.HtmlEscape(urlToRedirect) + "\">Moved Permanently</a>.\n"
				ctx.Write(note)
			}
			_tree.pool.Put(ctx)
			return true
		}
	}

	_tree.pool.Put(ctx)
	return false
}
