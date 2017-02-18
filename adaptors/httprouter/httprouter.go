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

	isRoot entryCase = iota
	hasParams
	matchEverything
)

type (
	// entryCase is the type which the type of muxEntryusing in order to determinate what type (parameterized, anything, static...) is the perticular node
	entryCase uint8

	// muxEntry is the node of a tree of the routes,
	// in order to learn how this is working, google 'trie' or watch this lecture: https://www.youtube.com/watch?v=uhAUk63tLRM
	// this method is used by the BSD's kernel also
	muxEntry struct {
		part        string
		entryCase   entryCase
		hasWildNode bool
		tokens      string
		nodes       []*muxEntry
		middleware  iris.Middleware
		precedence  uint64
		paramsLen   uint8
	}
)

var (
	errMuxEntryConflictsWildcard = errors.New(`
			httprouter: '%s' in new path '%s'
							conflicts with existing wildcarded route with path: '%s'
							in existing prefix of'%s' `)

	errMuxEntryMiddlewareAlreadyExists = errors.New(`
		httprouter: Middleware were already registered for the path: '%s'`)

	errMuxEntryInvalidWildcard = errors.New(`
		httprouter: More than one wildcard found in the path part: '%s' in route's path: '%s'`)

	errMuxEntryConflictsExistingWildcard = errors.New(`
		httprouter: Wildcard for route path: '%s' conflicts with existing children in route path: '%s'`)

	errMuxEntryWildcardUnnamed = errors.New(`
		httprouter: Unnamed wildcard found in path: '%s'`)

	errMuxEntryWildcardInvalidPlace = errors.New(`
		httprouter: Wildcard is only allowed at the end of the path, in the route path: '%s'`)

	errMuxEntryWildcardConflictsMiddleware = errors.New(`
		httprouter: Wildcard  conflicts with existing middleware for the route path: '%s'`)

	errMuxEntryWildcardMissingSlash = errors.New(`
		httprouter: No slash(/) were found before wildcard in the route path: '%s'`)
)

// getParamsLen returns the parameters length from a given path
func getParamsLen(path string) uint8 {
	var n uint
	for i := 0; i < len(path); i++ {
		if path[i] != ':' && path[i] != '*' { // ParameterStartByte & MatchEverythingByte
			continue
		}
		n++
	}
	if n >= 255 {
		return 255
	}
	return uint8(n)
}

// findLower returns the smaller number between a and b
func findLower(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// add adds a muxEntry to the existing muxEntry or to the tree if no muxEntry has the prefix of
func (e *muxEntry) add(path string, middleware iris.Middleware) error {
	fullPath := path
	e.precedence++
	numParams := getParamsLen(path)

	if len(e.part) > 0 || len(e.nodes) > 0 {
	loop:
		for {
			if numParams > e.paramsLen {
				e.paramsLen = numParams
			}

			i := 0
			max := findLower(len(path), len(e.part))
			for i < max && path[i] == e.part[i] {
				i++
			}

			if i < len(e.part) {
				node := muxEntry{
					part:        e.part[i:],
					hasWildNode: e.hasWildNode,
					tokens:      e.tokens,
					nodes:       e.nodes,
					middleware:  e.middleware,
					precedence:  e.precedence - 1,
				}

				for i := range node.nodes {
					if node.nodes[i].paramsLen > node.paramsLen {
						node.paramsLen = node.nodes[i].paramsLen
					}
				}

				e.nodes = []*muxEntry{&node}
				e.tokens = string([]byte{e.part[i]})
				e.part = path[:i]
				e.middleware = nil
				e.hasWildNode = false
			}

			if i < len(path) {
				path = path[i:]

				if e.hasWildNode {
					e = e.nodes[0]
					e.precedence++

					if numParams > e.paramsLen {
						e.paramsLen = numParams
					}
					numParams--

					if len(path) >= len(e.part) && e.part == path[:len(e.part)] &&
						// Check for longer wildcard, e.g. :name and :names
						(len(e.part) >= len(path) || path[len(e.part)] == '/') {
						continue loop
					} else {
						// Wildcard conflict
						part := strings.SplitN(path, "/", 2)[0]
						prefix := fullPath[:strings.Index(fullPath, part)] + e.part
						return errMuxEntryConflictsWildcard.Format(fullPath, e.part, prefix)

					}

				}

				c := path[0]

				if e.entryCase == hasParams && c == slashByte && len(e.nodes) == 1 {
					e = e.nodes[0]
					e.precedence++
					continue loop
				}
				for i := range e.tokens {
					if c == e.tokens[i] {
						i = e.precedenceTo(i)
						e = e.nodes[i]
						continue loop
					}
				}

				if c != parameterStartByte && c != matchEverythingByte {

					e.tokens += string([]byte{c})
					node := &muxEntry{
						paramsLen: numParams,
					}
					e.nodes = append(e.nodes, node)
					e.precedenceTo(len(e.tokens) - 1)
					e = node
				}
				return e.addNode(numParams, path, fullPath, middleware)

			} else if i == len(path) {
				if e.middleware != nil {
					return errMuxEntryMiddlewareAlreadyExists.Format(fullPath)
				}
				e.middleware = middleware
			}
			return nil
		}
	} else {
		if err := e.addNode(numParams, path, fullPath, middleware); err != nil {
			return err
		}
		e.entryCase = isRoot
	}
	return nil
}

// addNode adds a muxEntry as children to other muxEntry
func (e *muxEntry) addNode(numParams uint8, path string, fullPath string, middleware iris.Middleware) error {
	var offset int

	for i, max := 0, len(path); numParams > 0; i++ {
		c := path[i]
		if c != parameterStartByte && c != matchEverythingByte {
			continue
		}

		end := i + 1
		for end < max && path[end] != slashByte {
			switch path[end] {
			case parameterStartByte, matchEverythingByte:
				return errMuxEntryInvalidWildcard.Format(path[i:], fullPath)
			default:
				end++
			}
		}

		if len(e.nodes) > 0 {
			return errMuxEntryConflictsExistingWildcard.Format(path[i:end], fullPath)
		}

		if end-i < 2 {
			return errMuxEntryWildcardUnnamed.Format(fullPath)
		}

		if c == parameterStartByte {

			if i > 0 {
				e.part = path[offset:i]
				offset = i
			}

			child := &muxEntry{
				entryCase: hasParams,
				paramsLen: numParams,
			}
			e.nodes = []*muxEntry{child}
			e.hasWildNode = true
			e = child
			e.precedence++
			numParams--

			if end < max {
				e.part = path[offset:end]
				offset = end

				child := &muxEntry{
					paramsLen:  numParams,
					precedence: 1,
				}
				e.nodes = []*muxEntry{child}
				e = child
			}

		} else {
			if end != max || numParams > 1 {
				return errMuxEntryWildcardInvalidPlace.Format(fullPath)
			}

			if len(e.part) > 0 && e.part[len(e.part)-1] == '/' {
				return errMuxEntryWildcardConflictsMiddleware.Format(fullPath)
			}

			i--
			if path[i] != slashByte {
				return errMuxEntryWildcardMissingSlash.Format(fullPath)
			}

			e.part = path[offset:i]

			child := &muxEntry{
				hasWildNode: true,
				entryCase:   matchEverything,
				paramsLen:   1,
			}
			e.nodes = []*muxEntry{child}
			e.tokens = string(path[i])
			e = child
			e.precedence++

			child = &muxEntry{
				part:       path[i:],
				entryCase:  matchEverything,
				paramsLen:  1,
				middleware: middleware,
				precedence: 1,
			}
			e.nodes = []*muxEntry{child}

			return nil
		}
	}

	e.part = path[offset:]
	e.middleware = middleware

	return nil
}

// get is used by the Router, it finds and returns the correct muxEntry for a path
func (e *muxEntry) get(path string, ctx *iris.Context) (mustRedirect bool) {
loop:
	for {
		if len(path) > len(e.part) {
			if path[:len(e.part)] == e.part {
				path = path[len(e.part):]

				if !e.hasWildNode {
					c := path[0]
					for i := range e.tokens {
						if c == e.tokens[i] {
							e = e.nodes[i]
							continue loop
						}
					}

					mustRedirect = (path == slash && e.middleware != nil)
					return
				}

				e = e.nodes[0]
				switch e.entryCase {
				case hasParams:

					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					ctx.Set(e.part[1:], path[:end])

					if end < len(path) {
						if len(e.nodes) > 0 {
							path = path[end:]
							e = e.nodes[0]
							continue loop
						}

						mustRedirect = (len(path) == end+1)
						return
					}
					if ctx.Middleware = e.middleware; ctx.Middleware != nil {
						return
					} else if len(e.nodes) == 1 {
						e = e.nodes[0]
						mustRedirect = (e.part == slash && e.middleware != nil)
					}

					return

				case matchEverything:

					ctx.Set(e.part[2:], path)
					ctx.Middleware = e.middleware
					return

				default:
					return
				}
			}
		} else if path == e.part {
			if ctx.Middleware = e.middleware; ctx.Middleware != nil {
				return
			}

			if path == slash && e.hasWildNode && e.entryCase != isRoot {
				mustRedirect = true
				return
			}

			for i := range e.tokens {
				if e.tokens[i] == slashByte {
					e = e.nodes[i]
					mustRedirect = (len(e.part) == 1 && e.middleware != nil) ||
						(e.entryCase == matchEverything && e.nodes[0].middleware != nil)
					return
				}
			}

			return
		}

		mustRedirect = (path == slash) ||
			(len(e.part) == len(path)+1 && e.part[len(path)] == slashByte &&
				path == e.part[:len(e.part)-1] && e.middleware != nil)
		return
	}
}

// precedenceTo just adds the priority of this muxEntry by an index
func (e *muxEntry) precedenceTo(index int) int {
	e.nodes[index].precedence++
	_precedence := e.nodes[index].precedence

	newindex := index
	for newindex > 0 && e.nodes[newindex-1].precedence < _precedence {
		tmpN := e.nodes[newindex-1]
		e.nodes[newindex-1] = e.nodes[newindex]
		e.nodes[newindex] = tmpN

		newindex--
	}

	if newindex != index {
		e.tokens = e.tokens[:newindex] +
			e.tokens[index:index+1] +
			e.tokens[newindex:index] + e.tokens[index+1:]
	}

	return newindex
}

type (
	muxTree struct {
		method string
		// subdomain is empty for default-hostname routes,
		// ex: mysubdomain.
		subdomain string
		entry     *muxEntry
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
			},
		},
		RouterReversionPolicy: iris.RouterReversionPolicy{
			// path normalization done on iris' side
			StaticPath: func(path string) string {

				i := strings.IndexByte(path, parameterStartByte)
				x := strings.IndexByte(path, matchEverythingByte)
				if i > -1 {
					return path[0:i]
				}
				if x > -1 {
					return path[0:x]
				}

				return path
			},
			WildcardPath: func(path string, paramName string) string {
				return path + slash + matchEverythingString + paramName
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
					tree = &muxTree{method: method, subdomain: subdomain, entry: &muxEntry{}}
					mux.garden = append(mux.garden, tree)
				}
				// I decide that it's better to explicit give subdomain and a path to it than registeredPath(mysubdomain./something) now its: subdomain: mysubdomain., path: /something
				// we have different tree for each of subdomains, now you can use everything you can use with the normal paths ( before you couldn't set /any/*path)
				if err := tree.entry.add(path, middleware); err != nil {
					// while ProdMode means that the iris should not continue running
					// by-default it panics on these errors, but to make sure let's introduce the fatalErr to stop visiting
					fatalErr = true
					logger(iris.ProdMode, err.Error())
					return
				}

				if mp := tree.entry.paramsLen; mp > mux.maxParameters {
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
					hostname := context.Framework().Config.VHost
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

				mustRedirect := tree.entry.get(routePath, context) // pass the parameters here for 0 allocation
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
				for i := range mux.garden {
					tree := mux.garden[i]
					if !mux.methodEqual(context.Method(), tree.method) {
						continue
					}
				}
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
