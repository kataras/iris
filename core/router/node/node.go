package node

import (
	"sort"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/errors"
)

// Nodes a conversion type for []*node.
type Nodes []*node

type node struct {
	s                 string
	routeName         string
	wildcardParamName string   // name of the wildcard parameter, only one per whole Node is allowed
	paramNames        []string // only-names
	childrenNodes     Nodes
	handlers          context.Handlers
	root              bool
	rootWildcard      bool // if it's a wildcard {path} type on root, it should allow everything but it is not conflicts with
	// any other static or dynamic or wildcard paths if exists on other nodes.
}

// ErrDublicate returnned from `Add` when two or more routes have the same registered path.
var ErrDublicate = errors.New("two or more routes have the same registered path")

/// TODO: clean up needed until v8.5

// Add adds a node to the tree, returns an ErrDublicate error on failure.
func (nodes *Nodes) Add(routeName string, path string, handlers context.Handlers) error {
	// println("[Add] adding path: " + path)
	// resolve params and if that node should be added as root
	var params []string
	var paramStart, paramEnd int
	for {
		paramStart = strings.IndexByte(path[paramEnd:], ':')
		if paramStart == -1 {
			break
		}
		paramStart += paramEnd
		paramStart++
		paramEnd = strings.IndexByte(path[paramStart:], '/')

		if paramEnd == -1 {
			params = append(params, path[paramStart:])
			path = path[:paramStart]
			break
		}
		paramEnd += paramStart
		params = append(params, path[paramStart:paramEnd])
		path = path[:paramStart] + path[paramEnd:]
		paramEnd -= paramEnd - paramStart
	}

	var p []int
	for i := 0; i < len(path); i++ {
		idx := strings.IndexByte(path[i:], ':')
		if idx == -1 {
			break
		}
		p = append(p, idx+i)
		i = idx + i
	}

	for _, idx := range p {
		// print("-2 nodes.Add: path: " + path + " params len: ")
		// println(len(params))
		if err := nodes.add(routeName, path[:idx], nil, nil, true); err != nil {
			return err
		}
		// print("-1 nodes.Add: path: " + path + " params len: ")
		// println(len(params))
		if nidx := idx + 1; len(path) > nidx {
			if err := nodes.add(routeName, path[:nidx], nil, nil, true); err != nil {
				return err
			}
		}
	}

	// print("nodes.Add: path: " + path + " params len: ")
	// println(len(params))
	if err := nodes.add(routeName, path, params, handlers, true); err != nil {
		return err
	}

	// prioritize by static path remember, they were already sorted by subdomains too.
	nodes.prioritize()
	return nil
}

func (nodes *Nodes) add(routeName, path string, paramNames []string, handlers context.Handlers, root bool) (err error) {

	// println("[add] adding path: " + path)

	// wraia etsi doulevei ara
	// na to kanw na exei to node to diko tou wildcard parameter name
	// kai sto telos na pernei auto, me vasi to *paramname
	// alla edw mesa 9a ginete register vasi tou last /

	// set the wildcard param name to the root and its children.
	wildcardIdx := strings.IndexByte(path, '*')
	wildcardParamName := ""
	if wildcardIdx > 0 && len(paramNames) == 0 { // 27 Oct comment: && len(paramNames) == 0 {
		wildcardParamName = path[wildcardIdx+1:]
		path = path[0:wildcardIdx-1] + "/" // replace *paramName with single slash

		// if path[len(path)-1] == '/' {
		// if root wildcard, then add it as it's and return
		rootWildcard := path == "/"
		if rootWildcard {
			path += "/" // if root wildcard, then do it like "//" instead of simple "/"
		}

		n := &node{
			rootWildcard:      rootWildcard,
			s:                 path,
			routeName:         routeName,
			wildcardParamName: wildcardParamName,
			paramNames:        paramNames,
			handlers:          handlers,
			root:              root,
		}
		*nodes = append(*nodes, n)
		// println("1. nodes.Add path: " + path)
		return

	}

loop:
	for _, n := range *nodes {
		if n.rootWildcard {
			continue
		}

		if len(n.paramNames) == 0 && n.wildcardParamName != "" {
			continue
		}

		minlen := len(n.s)
		if len(path) < minlen {
			minlen = len(path)
		}

		for i := 0; i < minlen; i++ {
			if n.s[i] == path[i] {
				continue
			}
			if i == 0 {
				continue loop
			}

			*n = node{
				s: n.s[:i],
				childrenNodes: Nodes{
					{
						s:                 n.s[i:],
						routeName:         n.routeName,
						wildcardParamName: n.wildcardParamName, // wildcardParamName
						paramNames:        n.paramNames,
						childrenNodes:     n.childrenNodes,
						handlers:          n.handlers,
					},
					{
						s:                 path[i:],
						routeName:         routeName,
						wildcardParamName: wildcardParamName,
						paramNames:        paramNames,
						handlers:          handlers,
					},
				},
				root: n.root,
			}

			// fmt.Printf("%#v\n", n)
			// println("2. change n and return  " + n.s[:i] + " and " + path[i:])
			return
		}

		if len(path) < len(n.s) {
			// 	println("3. change n and return | n.s[:len(path)] = " + n.s[:len(path)-1] + " and child: " + n.s[len(path)-1:])

			*n = node{
				s:                 n.s[:len(path)],
				routeName:         routeName,
				wildcardParamName: wildcardParamName,
				paramNames:        paramNames,
				childrenNodes: Nodes{
					{
						s:                 n.s[len(path):],
						routeName:         n.routeName,
						wildcardParamName: n.wildcardParamName, // wildcardParamName
						paramNames:        n.paramNames,
						childrenNodes:     n.childrenNodes,
						handlers:          n.handlers,
					},
				},
				handlers: handlers,
				root:     n.root,
			}

			return
		}

		if len(path) > len(n.s) {
			if n.wildcardParamName != "" {
				n := &node{
					s:                 path,
					routeName:         routeName,
					wildcardParamName: wildcardParamName,
					paramNames:        paramNames,
					handlers:          handlers,
					root:              root,
				}
				// println("3.5. nodes.Add path: " + n.s)
				*nodes = append(*nodes, n)
				return
			}

			pathToAdd := path[len(n.s):]
			// println("4. nodes.Add path: " + pathToAdd)
			err = n.childrenNodes.add(routeName, pathToAdd, paramNames, handlers, false)
			return err
		}

		if len(handlers) == 0 { // missing handlers
			return nil
		}

		if len(n.handlers) > 0 { // n.handlers already setted
			return ErrDublicate
		}
		n.paramNames = paramNames
		n.handlers = handlers
		return
	}

	// START
	// Author's note:
	// 27 Oct 2017; fixes s|i|l+static+p
	//			    without breaking the current tests.
	if wildcardIdx > 0 {
		wildcardParamName = path[wildcardIdx+1:]
		path = path[0:wildcardIdx-1] + "/"
	}
	// END

	n := &node{
		s:                 path,
		routeName:         routeName,
		wildcardParamName: wildcardParamName,
		paramNames:        paramNames,
		handlers:          handlers,
		root:              root,
	}
	*nodes = append(*nodes, n)

	// println("5. node add on path: " + path + " n.s: " + n.s + " wildcard param: " + n.wildcardParamName)
	return
}

// Find resolves the path, fills its params
// and returns the registered to the resolved node's handlers.
func (nodes Nodes) Find(path string, params *context.RequestParams) (string, context.Handlers) {
	n, paramValues := nodes.findChild(path, nil)
	if n != nil {
		//	map the params,
		// n.params are the param names
		if len(paramValues) > 0 {
			// println("-----------")
			// print("param values returned len: ")
			// println(len(paramValues))
			// println("first value is: " + paramValues[0])
			// print("n.paramNames len: ")
			// println(len(n.paramNames))
			for i, name := range n.paramNames {
				// println("setting param name: " + name + " = " + paramValues[i])
				params.Set(name, paramValues[i])
			}
			// last is the wildcard,
			// if paramValues are exceed from the registered param names.
			// Note that n.wildcardParamName can be not empty but that doesn't meaning
			// that it contains a wildcard path, so the check is required.
			if len(paramValues) > len(n.paramNames) {
				// println("len(paramValues) > len(n.paramNames)")
				lastWildcardVal := paramValues[len(paramValues)-1]
				// println("setting wildcard param name: " + n.wildcardParamName + " = " + lastWildcardVal)
				params.Set(n.wildcardParamName, lastWildcardVal)
			}
		}
		return n.routeName, n.handlers
	}

	return "", nil
}

// Exists returns true if a node with that "path" exists,
// otherise false.
//
// We don't care about parameters here.
func (nodes Nodes) Exists(path string) bool {
	n, _ := nodes.findChild(path, nil)
	return n != nil && len(n.handlers) > 0
}

func (nodes Nodes) findChild(path string, params []string) (*node, []string) {

	for _, n := range nodes {
		if n.s == ":" {
			paramEnd := strings.IndexByte(path, '/')
			if paramEnd == -1 {
				if len(n.handlers) == 0 {
					return nil, nil
				}
				return n, append(params, path)
			}
			return n.childrenNodes.findChild(path[paramEnd:], append(params, path[:paramEnd]))
		}

		// println("n.s: " + n.s)
		// print("n.childrenNodes len: ")
		// println(len(n.childrenNodes))
		// print("n.root: ")
		// println(n.root)

		// by runtime check of:,
		// if n.s == "//" && n.root && n.wildcardParamName != "" {
		// but this will slow down, so we have a static field on the node itself:
		if n.rootWildcard {
			// println("return from n.rootWildcard")
			// single root wildcard
			if len(path) < 2 {
				// do not remove that, it seems useless but it's not,
				// we had an error while production, this fixes that.
				path = "/" + path
			}
			return n, append(params, path[1:])
		}

		// second conditional may be unnecessary
		// because of the n.rootWildcard before, but do it.
		if n.wildcardParamName != "" && len(path) > 2 {
			// println("n has wildcard n.s: " + n.s + " on path: " + path)
			// n.s = static/, path = static

			// 	println(n.s + " vs path: " + path)

			// we could have /other/ as n.s so
			// we must do this check, remember:
			// now wildcards live on their own nodes
			if len(path) == len(n.s)-1 {
				// then it's like:
				// path = /other2
				// ns = /other2/
				if path == n.s[0:len(n.s)-1] {
					return n, params
				}
			}

			// othwerwise path = /other2/dsadas
			// ns= /other2/
			if strings.HasPrefix(path, n.s) {
				if len(path) > len(n.s)+1 {
					return n, append(params, path[len(n.s):]) // without slash
				}
			}

		}

		if !strings.HasPrefix(path, n.s) {
			// fmt.Printf("---here root: %v, n.s: "+n.s+" and path: "+path+" is dynamic: %v , wildcardParamName: %s, children len: %v \n", n.root, n.isDynamic(), n.wildcardParamName, len(n.childrenNodes))
			// println(path + " n.s: " + n.s + " continue...")
			continue
		}

		if len(path) == len(n.s) {
			if len(n.handlers) == 0 {
				return nil, nil
			}
			return n, params
		}

		child, childParamNames := n.childrenNodes.findChild(path[len(n.s):], params)

		// print("childParamNames len: ")
		// println(len(childParamNames))

		// if len(childParamNames) > 0 {
		// 	println("childParamsNames[0] = " + childParamNames[0])
		// }

		if child == nil || len(child.handlers) == 0 {
			if n.s[len(n.s)-1] == '/' && !(n.root && (n.s == "/" || len(n.childrenNodes) > 0)) {
				if len(n.handlers) == 0 {
					return nil, nil
				}

				// println("if child == nil.... | n.s = " + n.s)
				// print("n.paramNames len: ")
				// println(n.paramNames)
				// print("n.wildcardParamName is: ")
				// println(n.wildcardParamName)
				// print("return n, append(params, path[len(n.s) | params: ")
				// println(path[len(n.s):])
				return n, append(params, path[len(n.s):])
			}

			continue
		}

		return child, childParamNames
	}
	return nil, nil
}

// childLen returns all the children's and their children's length.
func (n *node) childLen() (i int) {
	for _, n := range n.childrenNodes {
		i++
		i += n.childLen()
	}
	return
}

func (n *node) isDynamic() bool {
	return n.s == ":" || n.wildcardParamName != "" || n.rootWildcard
}

// prioritize sets the static paths first.
func (nodes Nodes) prioritize() {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].isDynamic() {
			return false
		}
		if nodes[j].isDynamic() {
			return true
		}

		return nodes[i].childLen() > nodes[j].childLen()
	})

	for _, n := range nodes {
		n.childrenNodes.prioritize()
	}
}
