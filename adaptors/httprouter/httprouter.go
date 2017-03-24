package httprouter

//  +------------------------------------------------------------+
//  | Usage                                                      |
//  +------------------------------------------------------------+
//
//
// package main
//
// import (
// 	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
// 	"gopkg.in/kataras/iris.v6"
// )
//
// func main() {
// 	app := iris.New()
//
// 	app.Adapt(httprouter.New()) // Add this line and you're ready.
//
// 	app.Get("/api/users/:userid", func(ctx *iris.Context) {
// 		ctx.Writef("User with id: %s", ctx.Param("userid"))
// 	})
//
// 	app.Listen(":8080")
// }

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/kataras/go-errors"
	"gopkg.in/kataras/iris.v6"
)

const (
	// parameterStartByte is very used on the node, it's just contains the byte for the ':' rune/char
	parameterStartByte = byte(':')
	// slashByte is just a byte of '/' rune/char
	slashByte = byte('/')
	// slash is just a string of "/"
	slash = "/"
	// matchEverythingByte is just a byte of '*" rune/char
	matchEverythingByte = byte('*')
)

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func countParams(path string) uint8 {
	var n uint
	for i := 0; i < len(path); i++ {
		if path[i] != ':' && path[i] != '*' {
			continue
		}
		n++
	}
	if n >= 255 {
		return 255
	}
	return uint8(n)
}

type nodeType uint8

const (
	static nodeType = iota // default
	root
	param
	catchAll
)

type node struct {
	path      string
	wildChild bool
	nType     nodeType
	maxParams uint8
	indices   string
	children  []*node
	handle    iris.Middleware
	priority  uint32
}

// increments priority of the given child and reorders if necessary
func (n *node) incrementChildPrio(pos int) int {
	n.children[pos].priority++
	prio := n.children[pos].priority

	// adjust position (move to front)
	newPos := pos
	for newPos > 0 && n.children[newPos-1].priority < prio {
		// swap node positions
		n.children[newPos-1], n.children[newPos] = n.children[newPos], n.children[newPos-1]

		newPos--
	}

	// build new index char string
	if newPos != pos {
		n.indices = n.indices[:newPos] + // unchanged prefix, might be empty
			n.indices[pos:pos+1] + // the index char we move
			n.indices[newPos:pos] + n.indices[pos+1:] // rest without char at 'pos'
	}

	return newPos
}

// addRoute adds a node with the given handle to the path.
// Not concurrency-safe!
func (n *node) addRoute(path string, handle iris.Middleware) error {
	fullPath := path
	n.priority++
	numParams := countParams(path)

	// non-empty tree
	if len(n.path) > 0 || len(n.children) > 0 {
	walk:
		for {
			// Update maxParams of the current node
			if numParams > n.maxParams {
				n.maxParams = numParams
			}

			// Find the longest common prefix.
			// This also implies that the common prefix contains no ':' or '*'
			// since the existing key can't contain those chars.
			i := 0
			max := min(len(path), len(n.path))
			for i < max && path[i] == n.path[i] {
				i++
			}

			// Split edge
			if i < len(n.path) {
				child := node{
					path:      n.path[i:],
					wildChild: n.wildChild,
					nType:     static,
					indices:   n.indices,
					children:  n.children,
					handle:    n.handle,
					priority:  n.priority - 1,
				}

				// Update maxParams (max of all children)
				for i := range child.children {
					if child.children[i].maxParams > child.maxParams {
						child.maxParams = child.children[i].maxParams
					}
				}

				n.children = []*node{&child}
				// []byte for proper unicode char conversion, see #65
				n.indices = string([]byte{n.path[i]})
				n.path = path[:i]
				n.handle = nil
				n.wildChild = false
			}

			// Make new node a child of this node
			if i < len(path) {
				path = path[i:]

				if n.wildChild {
					n = n.children[0]
					n.priority++

					// Update maxParams of the child node
					if numParams > n.maxParams {
						n.maxParams = numParams
					}
					numParams--

					// Check if the wildcard matches
					if len(path) >= len(n.path) && n.path == path[:len(n.path)] &&
						// Check for longer wildcard, e.g. :name and :names
						(len(n.path) >= len(path) || path[len(n.path)] == '/') {
						continue walk
					} else {
						// Wildcard conflict
						pathSeg := strings.SplitN(path, "/", 2)[0]
						prefix := fullPath[:strings.Index(fullPath, pathSeg)] + n.path
						return errors.New("'" + pathSeg +
							"' in new path '" + fullPath +
							"' conflicts with existing wildcard '" + n.path +
							"' in existing prefix '" + prefix +
							"'")
					}
				}

				c := path[0]

				// slash after param
				if n.nType == param && c == '/' && len(n.children) == 1 {
					n = n.children[0]
					n.priority++
					continue walk
				}

				// Check if a child with the next path byte exists
				for i := 0; i < len(n.indices); i++ {
					if c == n.indices[i] {
						i = n.incrementChildPrio(i)
						n = n.children[i]
						continue walk
					}
				}

				// Otherwise insert it
				if c != ':' && c != '*' {
					// []byte for proper unicode char conversion, see #65
					n.indices += string([]byte{c})
					child := &node{
						maxParams: numParams,
					}
					n.children = append(n.children, child)
					n.incrementChildPrio(len(n.indices) - 1)
					n = child
				}
				return n.insertChild(numParams, path, fullPath, handle)

			} else if i == len(path) { // Make node a (in-path) leaf
				if n.handle != nil {
					return errors.New("a handle is already registered for path '" + fullPath + "'")
				}
				n.handle = handle
			}
			return nil
		}
	} else { // Empty tree
		n.insertChild(numParams, path, fullPath, handle)
		n.nType = root
	}
	return nil
}

func (n *node) insertChild(numParams uint8, path, fullPath string, handle iris.Middleware) error {
	var offset int // already handled bytes of the path

	// find prefix until first wildcard (beginning with ':'' or '*'')
	for i, max := 0, len(path); numParams > 0; i++ {
		c := path[i]
		if c != ':' && c != '*' {
			continue
		}

		// find wildcard end (either '/' or path end)
		end := i + 1
		for end < max && path[end] != '/' {
			switch path[end] {
			// the wildcard name must not contain ':' and '*'
			case ':', '*':
				return errors.New("only one wildcard per path segment is allowed, has: '" +
					path[i:] + "' in path '" + fullPath + "'")
			default:
				end++
			}
		}

		// check if this Node existing children which would be
		// unreachable if we insert the wildcard here
		if len(n.children) > 0 {
			return errors.New("wildcard route '" + path[i:end] +
				"' conflicts with existing children in path '" + fullPath + "'")
		}

		// check if the wildcard has a name
		if end-i < 2 {
			return errors.New("wildcards must be named with a non-empty name in path '" + fullPath + "'")
		}

		if c == ':' { // param
			// split path at the beginning of the wildcard
			if i > 0 {
				n.path = path[offset:i]
				offset = i
			}

			child := &node{
				nType:     param,
				maxParams: numParams,
			}
			n.children = []*node{child}
			n.wildChild = true
			n = child
			n.priority++
			numParams--

			// if the path doesn't end with the wildcard, then there
			// will be another non-wildcard subpath starting with '/'
			if end < max {
				n.path = path[offset:end]
				offset = end

				child := &node{
					maxParams: numParams,
					priority:  1,
				}
				n.children = []*node{child}
				n = child
			}

		} else { // catchAll
			if end != max || numParams > 1 {
				return errors.New("catch-all routes are only allowed at the end of the path in path '" + fullPath + "'")
			}

			if len(n.path) > 0 && n.path[len(n.path)-1] == '/' {
				return errors.New("catch-all conflicts with existing handle for the path segment root in path '" + fullPath + "'")
			}

			// currently fixed width 1 for '/'
			i--
			if path[i] != '/' {
				return errors.New("no / before catch-all in path '" + fullPath + "'")
			}

			n.path = path[offset:i]

			// first node: catchAll node with empty path
			child := &node{
				wildChild: true,
				nType:     catchAll,
				maxParams: 1,
			}
			n.children = []*node{child}
			n.indices = string(path[i])
			n = child
			n.priority++

			// second node: node holding the variable
			child = &node{
				path:      path[i:],
				nType:     catchAll,
				maxParams: 1,
				handle:    handle,
				priority:  1,
			}
			n.children = []*node{child}

			return nil
		}
	}

	// insert remaining path part and handle to the leaf
	n.path = path[offset:]
	n.handle = handle
	return nil
}

// Returns the handle registered with the given path (key). The values of
// wildcards are saved to a map.
// If no handle can be found, a TSR (trailing slash redirect) recommendation is
// made if a handle exists with an extra (without the) trailing slash for the
// given path.
func (n *node) getValue(path string, ctx *iris.Context) (tsr bool) {
walk: // outer loop for walking the tree
	for {
		if len(path) > len(n.path) {
			if path[:len(n.path)] == n.path {
				path = path[len(n.path):]
				// If this node does not have a wildcard (param or catchAll)
				// child,  we can just look up the next child node and continue
				// to walk down the tree
				if !n.wildChild {
					c := path[0]
					for i := 0; i < len(n.indices); i++ {
						if c == n.indices[i] {
							n = n.children[i]
							continue walk
						}
					}

					// Nothing found.
					// We can recommend to redirect to the same URL without a
					// trailing slash if a leaf exists for that path.
					tsr = (path == "/" && n.handle != nil)
					return

				}

				// handle wildcard child
				n = n.children[0]
				switch n.nType {
				case param:
					// find param end (either '/' or path end)
					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					// save param value
					ctx.Set(n.path[1:], path[:end])

					// we need to go deeper!
					if end < len(path) {
						if len(n.children) > 0 {
							path = path[end:]
							n = n.children[0]
							continue walk
						}

						// ... but we can't
						tsr = (len(path) == end+1)
						return
					}

					if ctx.Middleware = n.handle; ctx.Middleware != nil {
						return
					} else if len(n.children) == 1 {
						// No handle found. Check if a handle for this path + a
						// trailing slash exists for TSR recommendation
						n = n.children[0]
						tsr = (n.path == "/" && n.handle != nil)
					}

					return

				case catchAll:
					// save param value
					ctx.Set(n.path[2:], path)

					ctx.Middleware = n.handle
					return

				default:
					panic("invalid node type")
				}
			}
		} else if path == n.path {
			// We should have reached the node containing the handle.
			// Check if this node has a handle registered.
			if ctx.Middleware = n.handle; ctx.Middleware != nil {
				return
			}

			if path == "/" && n.wildChild && n.nType != root {
				tsr = true
				return
			}

			// No handle found. Check if a handle for this path + a
			// trailing slash exists for trailing slash recommendation
			for i := 0; i < len(n.indices); i++ {
				if n.indices[i] == '/' {
					n = n.children[i]
					tsr = (len(n.path) == 1 && n.handle != nil) ||
						(n.nType == catchAll && n.children[0].handle != nil)
					return
				}
			}

			return
		}

		// Nothing found. We can recommend to redirect to the same URL with an
		// extra trailing slash if a leaf exists for that path
		tsr = (path == "/") ||
			(len(n.path) == len(path)+1 && n.path[len(path)] == '/' &&
				path == n.path[:len(n.path)-1] && n.handle != nil)
		return
	}
}

// shift bytes in array by n bytes left
func shiftNRuneBytes(rb [4]byte, n int) [4]byte {
	switch n {
	case 0:
		return rb
	case 1:
		return [4]byte{rb[1], rb[2], rb[3], 0}
	case 2:
		return [4]byte{rb[2], rb[3]}
	case 3:
		return [4]byte{rb[3]}
	default:
		return [4]byte{}
	}
}

type (
	muxTree struct {
		method string
		// subdomain is empty for default-hostname routes,
		// ex: mysubdomain.
		subdomain string
		entry     *node
	}

	serveMux struct {
		garden        []*muxTree
		maxParameters uint8
		methodEqual   func(string, string) bool
		hosts         bool
	}
)

// path = "/api/users/:id"
// return "/api/users/%v"
//
// path = "/files/*file"
// return /files/%v
//
// path = "/:username/messages/:messageid"
// return "/%v/messages/%v"
func formatPath(path string) string {
	n1Len := strings.Count(path, ":")
	isMatchEverything := len(path) > 0 && path[len(path)-1] == matchEverythingByte
	if n1Len == 0 && !isMatchEverything {
		// its a static
		return path
	}
	if n1Len == 0 && isMatchEverything {
		//if we have something like: /mypath/anything/* -> /mypatch/anything/%v
		return path[0:len(path)-2] + "%v"

	}

	splittedN1 := strings.Split(path, "/")

	for _, v := range splittedN1 {
		if len(v) > 0 {
			if v[0] == ':' || v[0] == matchEverythingByte {
				path = strings.Replace(path, v, "%v", -1) // n1Len, but let it we don't care about performance here.
			}
		}
	}

	return path
}

// Name is the name of the router
//
// See $iris_instance.Config.Other for more.
const Name = "httprouter"

// New returns a new iris' policy to create and attach the router.
// It's based on the julienschmidt/httprouter  with more features and some iris-relative performance tips:
// subdomains(wildcard/dynamic and static) and faster parameters set (use of the already-created context's values)
// and support for reverse routing.
func New() iris.Policies {
	var logger func(iris.LogMode, string)
	mux := &serveMux{
		methodEqual: func(reqMethod string, treeMethod string) bool {
			return reqMethod == treeMethod
		},
	}
	matchEverythingString := string(matchEverythingByte)
	return iris.Policies{
		EventPolicy: iris.EventPolicy{
			Boot: func(s *iris.Framework) {
				logger = s.Log
				s.Set(iris.OptionOther(iris.RouterNameConfigKey, Name))
			}},
		RouterReversionPolicy: iris.RouterReversionPolicy{
			// path normalization done on iris' side
			StaticPath: func(path string) string {
				i := strings.IndexByte(path, parameterStartByte)
				v := strings.IndexByte(path, matchEverythingByte)
				if i > -1 || v > -1 {
					if i < v {
						return path[0:i]
					}
					// we can't return path[0:0]
					if v > 0 {
						return path[0:v]
					}

				}

				return path
			},
			WildcardPath: func(path string, paramName string) string {
				// *param
				wildcardPart := matchEverythingString + paramName

				if path[len(path)-1] != slashByte {
					// if not ending with slash then prepend the slash to the wildcard path part
					wildcardPart = slash + wildcardPart
				}
				// finally return the path given + the wildcard path part
				return path + wildcardPart
			},
			Param: func(paramName string) string {
				return string(parameterStartByte) + paramName
			},
			// path = "/api/users/:id", args = ["42"]
			// return "/api/users/42"
			//
			// path = "/files/*file", args = ["mydir","myfile.zip"]
			// return /files/mydir/myfile.zip
			//
			// path = "/:username/messages/:messageid", args = ["kataras","42"]
			// return "/kataras/messages/42"
			//
			// This policy is used for reverse routing,
			// see iris.Path/URL and ~/adaptors/view/ {{ url }} {{ urlpath }}
			URLPath: func(r iris.RouteInfo, args ...string) string {
				rpath := r.Path()
				formattedPath := formatPath(rpath)

				if rpath == formattedPath {
					// static, no need to pass args
					return rpath
				}
				// check if we have /*, if yes then join all arguments to one as path and pass that as parameter
				if formattedPath != rpath && rpath[len(rpath)-1] == matchEverythingByte {
					parameter := strings.Join(args, slash)
					return fmt.Sprintf(formattedPath, parameter)
				}
				// else return the formattedPath with its args
				for _, s := range args {
					formattedPath = strings.Replace(formattedPath, "%v", s, 1)
				}
				return formattedPath

			},
		},
		RouterBuilderPolicy: func(repo iris.RouteRepository, context iris.ContextPool) http.Handler {
			fatalErr := false
			mux.garden = mux.garden[0:0] // re-set the nodes
			mux.hosts = false
			repo.Visit(func(r iris.RouteInfo) {
				if fatalErr {
					return
				}
				// add to the registry tree
				method := r.Method()
				subdomain := r.Subdomain()
				path := r.Path()
				middleware := r.Middleware()
				tree := mux.getTree(method, subdomain)
				if tree == nil {
					//first time we register a route to this method with this domain
					tree = &muxTree{method: method, subdomain: subdomain, entry: &node{}}
					mux.garden = append(mux.garden, tree)
				}
				// I decide that it's better to explicit give subdomain and a path to it than registeredPath(mysubdomain./something) now its: subdomain: mysubdomain., path: /something
				// we have different tree for each of subdomains, now you can use everything you can use with the normal paths ( before you couldn't set /any/*path)
				if err := tree.entry.addRoute(path, middleware); err != nil {
					// while ProdMode means that the iris should not continue running
					// by-default it panics on these errors, but to make sure let's introduce the fatalErr to stop visiting
					fatalErr = true
					logger(iris.ProdMode, Name+" "+err.Error())
					return
				}

				if mp := tree.entry.maxParams; mp > mux.maxParameters {
					mux.maxParameters = mp
				}

				// check for method equality if at least one route has cors
				if r.HasCors() {
					mux.methodEqual = func(reqMethod string, treeMethod string) bool {
						// preflights
						return reqMethod == iris.MethodOptions || reqMethod == treeMethod
					}
				}

				if subdomain != "" {
					mux.hosts = true
				}
			})
			if !fatalErr {
				return mux.buildHandler(context)
			}
			return nil

		},
	}
}

func (mux *serveMux) getTree(method string, subdomain string) *muxTree {
	for i := range mux.garden {
		t := mux.garden[i]
		if t.method == method && t.subdomain == subdomain {
			return t
		}
	}
	return nil
}

func (mux *serveMux) buildHandler(pool iris.ContextPool) http.Handler {

	hostname := pool.Framework().Config.VHost
	// check if VHost is mydomain.com:80 || mydomain.com:443 before serving
	// means that the port part is optional and a valid client can make a request without the port.
	hostPort := iris.ParsePort(hostname)
	if hostPort == 80 || hostPort == 443 {
		hostname = iris.ParseHostname(hostname) // remove the port part.
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pool.Run(w, r, func(context *iris.Context) {
			routePath := context.Path()
			for i := range mux.garden {
				tree := mux.garden[i]
				if !mux.methodEqual(context.Request.Method, tree.method) {
					continue
				}

				if mux.hosts && tree.subdomain != "" {

					requestHost := context.Host()
					// println("mux are true and tree.subdomain= " + tree.subdomain + "and hostname = " + hostname + " host = " + requestHost)
					if requestHost != hostname {
						// we have a subdomain
						if strings.Contains(tree.subdomain, iris.DynamicSubdomainIndicator) {
						} else {
							if tree.subdomain+hostname != requestHost {
								// go to the next tree, we have a subdomain but it is not the correct
								continue
							}
						}
					} else {
						//("it's subdomain but the request is not the same as the vhost)
						continue
					}
				}

				mustRedirect := tree.entry.getValue(routePath, context) // pass the parameters here for 0 allocation
				if context.Middleware != nil {
					// ok we found the correct route, serve it and exit entirely from here
					//ctx.Request.Header.SetUserAgentBytes(DefaultUserAgent)
					context.Do()
					return
				} else if mustRedirect && !context.Framework().Config.DisablePathCorrection { // && context.Method() == MethodConnect {
					reqPath := routePath
					pathLen := len(reqPath)

					if pathLen > 1 {
						if reqPath[pathLen-1] == '/' {
							reqPath = reqPath[:pathLen-1] //remove the last /
						} else {
							//it has path prefix, it doesn't ends with / and it hasn't be found, then just add the slash
							reqPath = reqPath + "/"
						}

						urlToRedirect := reqPath

						statusForRedirect := iris.StatusMovedPermanently //	StatusMovedPermanently, this document is obselte, clients caches this.
						if tree.method == iris.MethodPost ||
							tree.method == iris.MethodPut ||
							tree.method == iris.MethodDelete {
							statusForRedirect = iris.StatusTemporaryRedirect //	To maintain POST data
						}

						context.Redirect(urlToRedirect, statusForRedirect)
						// RFC2616 recommends that a short note "SHOULD" be included in the
						// response because older user agents may not understand 301/307.
						// Shouldn't send the response for POST or HEAD; that leaves GET.
						if tree.method == iris.MethodGet {
							note := "<a href=\"" + HTMLEscape(urlToRedirect) + "\">Moved Permanently</a>.\n"
							// ignore error
							context.WriteString(note)
						}
						return
					}
				}
				// not found
				break
			}
			// https://github.com/kataras/iris/issues/469
			if context.Framework().Config.FireMethodNotAllowed {
				var methodAllowed string
				for i := range mux.garden {
					tree := mux.garden[i]
					methodAllowed = tree.method // keep track of the allowed method of the last checked tree
					if !mux.methodEqual(context.Method(), tree.method) {
						continue
					}
				}
				// RCF rfc2616 https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
				// The response MUST include an Allow header containing a list of valid methods for the requested resource.
				context.SetHeader("Allow", methodAllowed)
				context.EmitError(iris.StatusMethodNotAllowed)
				return
			}
			context.EmitError(iris.StatusNotFound)
		})
	})

}

//THESE ARE FROM Go Authors "html" package
var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	// "&#34;" is shorter than "&quot;".
	`"`, "&#34;",
	// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
	"'", "&#39;",
)

// HTMLEscape returns a string which has no valid html code
func HTMLEscape(s string) string {
	return htmlReplacer.Replace(s)
}
